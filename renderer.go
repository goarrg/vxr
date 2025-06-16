/*
Copyright 2025 The goARRG Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vxr

//go:generate go run ./libvxr/vk/vkfns_gen.go
//go:generate go run ./libvxr/vk/device/device_vkfns_gen.go
//go:generate go run ./vk_gen.go

/*
	#cgo pkg-config: vxr

	#include "vxr/vxr.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"
	"unsafe"

	"goarrg.com"
	"goarrg.com/debug"
	"golang.org/x/exp/maps"

	"goarrg.com/rhi/vxr/internal/vk"
)

const (
	MinAPI uint32 = C.VXR_VK_MIN_API
	MaxAPI uint32 = C.VXR_VK_MAX_API
)

type Destroyer interface {
	Destroy()
}

type destroyFunc struct {
	f func()
}

func (d destroyFunc) Destroy() {
	d.f()
}

type state struct {
	platform        goarrg.PlatformInterface
	logger          *debug.Logger
	vkInstance      goarrg.VkInstance
	config          config
	cSurface        uint64
	cInstance       C.vxr_vk_instance
	cShaderCompiler C.vxr_vk_shader_toolchain

	deviceProperties Properties
	formatProperties formatProperties

	descriptorSetLayoutCache descriptorSetLayoutCache
	pipelineLayoutCache      pipelineLayoutCache
	descriptorSetCache       descriptorSetCache

	graphics graphicsState

	sleep bool
	sizeX float64
	sizeY float64
}

type platform struct{}

func (platform) Abort()                           { panic("Fatal Error") }
func (platform) AbortPopup(f string, args ...any) { panic("Fatal Error") }

var instanceInitOnce sync.Once

var instance = state{
	platform: platform{},
	logger:   debug.NewLogger("vxr"),

	formatProperties: formatProperties{
		color: colorFormatProperties{
			optimalTilingFeatures: map[Format]FormatFeatureFlags{},
		},
		depth: depthFormatProperties{
			optimalTilingFeatures: map[DepthStencilFormat]FormatFeatureFlags{},
		},
	},

	descriptorSetLayoutCache: descriptorSetLayoutCache{cache: map[string]C.VkDescriptorSetLayout{}},
	pipelineLayoutCache:      pipelineLayoutCache{cache: map[string]C.VkPipelineLayout{}},
	descriptorSetCache:       descriptorSetCache{descriptorPools: map[string]*descriptorPool{}},

	graphics: graphicsState{
		pipelineCache: graphicsPipelineCache{
			cache:       map[string]C.VkPipeline{},
			cacheShader: map[string]*GraphicsShaderPipeline{},
		},
		destroyerChan: make(chan Destroyer),
	},
}

func VkConfig() goarrg.VkConfig {
	if C.VXR_DEBUG == 1 {
		return goarrg.VkConfig{
			API:    C.VXR_VK_MIN_API,
			Layers: []string{},
			Extensions: []string{
				C.VK_EXT_DEBUG_UTILS_EXTENSION_NAME,
				C.VK_KHR_GET_SURFACE_CAPABILITIES_2_EXTENSION_NAME,
				C.VK_EXT_SURFACE_MAINTENANCE_1_EXTENSION_NAME,
			},
		}
	}
	return goarrg.VkConfig{
		API:    C.VXR_VK_MIN_API,
		Layers: []string{},
		Extensions: []string{
			C.VK_KHR_GET_SURFACE_CAPABILITIES_2_EXTENSION_NAME,
			C.VK_EXT_SURFACE_MAINTENANCE_1_EXTENSION_NAME,
		},
	}
}

func initFramesInFlight(num int32) {
	for i := int(num); i < len(instance.graphics.framesInFlight); i++ {
		instance.graphics.framesInFlight[i].destroy()
	}
	for i := len(instance.graphics.framesInFlight); i < int(num); i++ {
		var f frame
		name := fmt.Sprintf("%d", i)
		C.vxr_vk_graphics_createFrame(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))), &f.cFrame)
		instance.graphics.framesInFlight = append(instance.graphics.framesInFlight, f)
	}
	instance.graphics.framesInFlight = instance.graphics.framesInFlight[:num]
}

func SetLogLevel(l uint32) {
	instance.logger.SetLevel(l)
}

func InitInstance(platform goarrg.PlatformInterface, vkInstance goarrg.VkInstance) {
	instanceInitOnce.Do(func() {
		instance.platform = platform
		instance.vkInstance = vkInstance
		instance.logger.IPrintf("vxr_vk_init")
		C.vxr_vk_init(C.uintptr_t(instance.vkInstance.Uintptr()), C.uintptr_t(instance.vkInstance.ProcAddr()),
			cGoVkLog, &instance.cInstance)
	})
}

type ErrorDeviceNotFound struct{}

func (ErrorDeviceNotFound) Is(target error) bool {
	_, ok := target.(ErrorDeviceNotFound)
	return ok
}

func (ErrorDeviceNotFound) Error() string {
	return "Device Not Found"
}

// Searches for and returns a VkPhysicalDevice with a matching UUID or error if none found.
// The input UUID must have come from Properties.UUID as the UUID is non standard
// and may not have been provided by a vulkan function call.
func LookupVkPhysicalDeviceFromUUID(uuid UUID) (uintptr, error) {
	var device C.uintptr_t
	ret := C.vxr_vk_device_vkPhysicalDeviceFromUUID(instance.cInstance, (*[unsafe.Sizeof(UUID{})]C.uint8_t)(unsafe.Pointer(&uuid)), &device)
	switch ret {
	case vk.SUCCESS:
		return uintptr(device), nil

	case vk.ERROR_DEVICE_LOST:
		return 0, debug.ErrorWrapf(ErrorDeviceNotFound{}, "Failed to lookup UUID")

	default:
		return 0, debug.Errorf("Failed to get list of devices: %s", vkResultStr(ret))
	}
}

func InitDevice(config Config) {
	config.validate()
	instance.logger.IPrintf("User requested config: %s", prettyString(&config))

	var err error
	instance.logger.IPrintf("CreateSurface")
	if instance.cSurface, err = instance.vkInstance.CreateSurface(); err != nil {
		abort("Failed to create surface: %v", err)
	}

	instance.logger.IPrintf("vxr_vk_device_init")
	selector := config.createDeviceSelector(instance.cSurface)
	defer C.vxr_vk_device_destroySelector(selector)
	C.vxr_vk_device_init(instance.cInstance, selector)

	{
		instance.logger.IPrintf("vxr_vk_device_getProperties")
		C.vxr_vk_device_getProperties(instance.cInstance, (*C.vxr_vk_device_properties)(unsafe.Pointer(&instance.deviceProperties)))

		{
			var numExtensions C.size_t
			C.vxr_vk_device_selector_getEnabledExtensions(selector, &numExtensions, nil)
			extensions := make([]*C.char, numExtensions)
			C.vxr_vk_device_selector_getEnabledExtensions(selector, &numExtensions, unsafe.SliceData(extensions))

			for _, e := range extensions {
				instance.deviceProperties.EnabledExtensions = append(instance.deviceProperties.EnabledExtensions, C.GoString(e))
			}
		}
		{
			var enabledFeatures *C.char
			C.vxr_vk_device_selector_getEnabledFeatures(selector, &enabledFeatures)
			err := json.Unmarshal([]byte(C.GoString(enabledFeatures)), &instance.deviceProperties.EnabledFeatures)
			if err != nil {
				abort("Failed to get enabled features: %v", err)
			}
		}
		instance.logger.IPrintf("%s", prettyString(&instance.deviceProperties))
	}

	instance.logger.IPrintf("Initializing Configuration")
	instance.config.use(config)
	initFramesInFlight(1)
	instance.logger.IPrintf("Initialization Completed")
}

func DeviceProperties() Properties {
	ret := instance.deviceProperties
	ret.EnabledExtensions = slices.Clone(instance.deviceProperties.EnabledExtensions)
	ret.EnabledFeatures = make(map[string]map[string]bool)
	for k, v := range instance.deviceProperties.EnabledFeatures {
		ret.EnabledFeatures[k] = maps.Clone(v)
	}
	return ret
}

func Resize(w int, h int) {
	instance.sleep = true
	instance.sizeX = float64(w)
	instance.sizeY = float64(h)

	if w != 0 && h != 0 {
		instance.sleep = false
		start := time.Now()

		for i := 0; i < len(instance.graphics.framesInFlight); i++ {
			instance.graphics.framesInFlight[i].waitSurface()
		}

		instance.logger.IPrintf("vxr_vk_graphics_init")
		ret := C.vxr_vk_graphics_init(instance.cInstance, C.uint64_t(instance.cSurface), C.uint32_t(instance.config.maxFramesInFlight))
		if ret != vk.SUCCESS {
			abort("Failed to initialize graphics system: %s", vkResultStr(ret))
		}

		info := CurrentSurfaceInfo()
		initFramesInFlight(info.NumFramesInFlight)
		instance.logger.IPrintf("Resize took: %v", time.Since(start))
	}
}

func Destroy() {
	C.vxr_vk_waitIdle(instance.cInstance)

loop:
	for {
		select {
		case j := <-instance.graphics.destroyerChan:
			j.Destroy()
		default:
			break loop
		}
	}

	instance.logger.VPrintf("formatProperties: %s", prettyString(&instance.formatProperties))

	instance.logger.VPrintf("pipelineCache: %s", prettyString(&instance.graphics.pipelineCache))
	for _, p := range instance.graphics.pipelineCache.cache {
		C.vxr_vk_shader_destroyPipeline(instance.cInstance, p)
	}
	for _, s := range instance.graphics.pipelineCache.cacheShader {
		C.vxr_vk_shader_destroyPipeline(instance.cInstance, s.vkPipeline)
	}

	instance.logger.VPrintf("descriptorSetLayoutCache: %s", prettyString(&instance.descriptorSetLayoutCache))
	instance.logger.VPrintf("pipelineLayoutCache: %s", prettyString(&instance.pipelineLayoutCache))
	instance.logger.VPrintf("descriptorSetCache: %s", prettyString(&instance.descriptorSetCache))

	for _, l := range instance.descriptorSetLayoutCache.cache {
		C.vxr_vk_shader_destroyDescriptorSetLayout(instance.cInstance, l)
	}
	for _, l := range instance.pipelineLayoutCache.cache {
		C.vxr_vk_shader_destroyPipelineLayout(instance.cInstance, l)
	}
	for _, p := range instance.descriptorSetCache.descriptorPools {
		p.destroy()
	}

	for _, f := range instance.graphics.framesInFlight {
		f.destroy()
	}

	instance.logger.IPrintf("vxr_vk_graphics_destroy")
	C.vxr_vk_graphics_destroy(instance.cInstance)
	instance.logger.IPrintf("vxr_vk_device_destroy")
	C.vxr_vk_device_destroy(instance.cInstance)
	instance.logger.IPrintf("vxr_vk_destroy")
	C.vxr_vk_destroy(instance.cInstance)

	instance.cInstance = nil
}
