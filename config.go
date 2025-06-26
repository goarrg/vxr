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

/*
	#cgo pkg-config: vxr

	#include "vxr/vxr.h"
*/
import "C"

import (
	"bytes"
	"fmt"
	"runtime"
	"slices"
	"unsafe"

	"goarrg.com/gmath"
	"golang.org/x/exp/maps"
)

type Config struct {
	PreferredVkPhysicalDevice uintptr
	API                       uint32
	MaxFramesInFlight         int32
	DescriptorPoolBankSize    int32

	RequiredExtensions []string
	OptionalExtensions []string

	RequiredFeatures []VkFeatureStruct
	OptionalFeatures []VkFeatureStruct

	RequiredFormatFeatures             map[Format]FormatFeatureFlags
	RequiredDepthStencilFormatFeatures map[DepthStencilFormat]FormatFeatureFlags
}

func vkAPI2String(api uint32) string {
	return fmt.Sprintf("%d.%d.%d", ((api >> 22) & 0x7F), ((api >> 12) & 0x3FF), (api & 0xFFF))
}

func (c *Config) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	buff.WriteString(fmt.Sprintf("\"PreferredVkPhysicalDevice\": %q,", toHex(c.PreferredVkPhysicalDevice)))
	buff.WriteString(fmt.Sprintf("\"API\": %q,", vkAPI2String(c.API)))
	buff.WriteString(fmt.Sprintf("\"MaxFramesInFlight\": %d,", c.MaxFramesInFlight))
	buff.WriteString(fmt.Sprintf("\"DescriptorPoolBankSize\": %d,", c.DescriptorPoolBankSize))

	buff.WriteString(fmt.Sprintf("\"RequiredExtensions\": %s,", jsonString(c.RequiredExtensions)))
	buff.WriteString(fmt.Sprintf("\"OptionalExtensions\": %s,", jsonString(c.OptionalExtensions)))

	buff.WriteString(fmt.Sprintf("\"RequiredFeatures\": %s,", jsonString(c.RequiredFeatures)))
	buff.WriteString(fmt.Sprintf("\"OptionalFeatures\": %s,", jsonString(c.OptionalFeatures)))

	{
		buff.WriteString("\"RequiredFormatFeatures\": {")
		err := mapRunFuncStringSorted(c.RequiredFormatFeatures, func(k Format, v FormatFeatureFlags) error {
			buff.WriteString(fmt.Sprintf("%q: %q,", k.String(), v.String()))
			return nil
		})
		if err == nil {
			buff.Truncate(buff.Len() - 1)
		}
		buff.WriteString("},")
	}
	{
		buff.WriteString("\"RequiredDepthStencilFormatFeatures\": {")
		err := mapRunFuncStringSorted(c.RequiredDepthStencilFormatFeatures, func(k DepthStencilFormat, v FormatFeatureFlags) error {
			buff.WriteString(fmt.Sprintf("%q: %q,", k.String(), v.String()))
			return nil
		})
		if err == nil {
			buff.Truncate(buff.Len() - 1)
		}
		buff.WriteString("},")
	}

	buff.Truncate(buff.Len() - 1)
	buff.WriteString("}")
	return buff.Bytes(), nil
}

func (c *Config) validate() {
	if c.API == 0 {
		c.API = C.VXR_VK_MIN_API
	} else if !gmath.InRange(c.API, C.VXR_VK_MIN_API, C.VXR_VK_MAX_API) {
		abort("Config.API is outside of valid api range [%q, %q]", vkAPI2String(C.VXR_VK_MIN_API), vkAPI2String(C.VXR_VK_MAX_API))
	}
	if c.MaxFramesInFlight <= 0 {
		abort("Config.MaxFramesInFlight must be >= 1")
	}
	if c.DescriptorPoolBankSize <= 0 {
		abort("Config.DescriptorPoolBankSize must be >= 1")
	}
}

func (c *Config) createDeviceSelector(surface uint64) C.vxr_vk_device_selector {
	// process required
	{
		c.RequiredExtensions = append([]string{
			C.VK_KHR_SWAPCHAIN_EXTENSION_NAME,
			C.VK_EXT_MEMORY_BUDGET_EXTENSION_NAME,
		}, c.RequiredExtensions...)
		c.RequiredFeatures = append([]VkFeatureStruct{
			VkPhysicalDeviceFeatures{
				FillModeNonSolid: true,
			},
			VkPhysicalDeviceVulkan11Features{},
			VkPhysicalDeviceVulkan12Features{
				DescriptorIndexing:                        true,
				DescriptorBindingUpdateUnusedWhilePending: true,
				DescriptorBindingPartiallyBound:           true,
				TimelineSemaphore:                         true,
			},
			VkPhysicalDeviceVulkan13Features{
				PipelineCreationCacheControl: true,
				SubgroupSizeControl:          true,
				ComputeFullSubgroups:         true,
				Synchronization2:             true,
				DynamicRendering:             true,
				Maintenance4:                 true,
			},
			VkPhysicalDeviceGraphicsPipelineLibraryFeaturesEXT{
				GraphicsPipelineLibrary: true,
			},
			VkPhysicalDeviceExtendedDynamicState3FeaturesEXT{
				ExtendedDynamicState3PolygonMode:        true,
				ExtendedDynamicState3ColorBlendEnable:   true,
				ExtendedDynamicState3ColorBlendEquation: true,
				ExtendedDynamicState3ColorWriteMask:     true,
			},
			VkPhysicalDeviceSwapchainMaintenance1FeaturesEXT{
				SwapchainMaintenance1: true,
			},
			/*
				VkPhysicalDeviceMaintenance5FeaturesKHR{
					Maintenance5: true,
				},
			*/
		}, c.RequiredFeatures...)
		if c.API < C.VK_API_VERSION_1_4 {
			c.RequiredFeatures = append(c.RequiredFeatures, VkPhysicalDeviceLineRasterizationFeaturesEXT{
				BresenhamLines: true,
			})
		} else {
			c.RequiredFeatures = append(c.RequiredFeatures, VkPhysicalDeviceVulkan14Features{
				BresenhamLines: true,
			})
		}
		for _, s := range c.RequiredFeatures {
			if s.extension() != "" {
				c.RequiredExtensions = append(c.RequiredExtensions, s.extension())
			}
		}
		slices.Sort(c.RequiredExtensions)
		c.RequiredExtensions = slices.Compact(c.RequiredExtensions)

		depends := []string{}
		for _, e := range c.RequiredExtensions {
			d := getExtensionDependencies(e)
			d = filterCorePromotedExtensions(c.API, d)
			depends = append(depends, d...)
		}
		c.RequiredExtensions = append(c.RequiredExtensions, depends...)
		slices.Sort(c.RequiredExtensions)
		c.RequiredExtensions = slices.Compact(c.RequiredExtensions)
	}

	// process optional
	{
		c.OptionalExtensions = append([]string{}, c.OptionalExtensions...)
		c.OptionalFeatures = append([]VkFeatureStruct{
			VkPhysicalDeviceFeatures{
				WideLines: true,
			},
		}, c.OptionalFeatures...)
		for _, s := range c.OptionalFeatures {
			if s.extension() != "" {
				c.OptionalExtensions = append(c.OptionalExtensions, s.extension())
			}
		}
		slices.Sort(c.OptionalExtensions)
		c.OptionalExtensions = slices.Compact(c.OptionalExtensions)

		depends := []string{}
		for _, e := range c.OptionalExtensions {
			d := getExtensionDependencies(e)
			d = filterCorePromotedExtensions(c.API, d)
			depends = append(depends, d...)
		}
		c.OptionalExtensions = append(c.OptionalExtensions, depends...)
		slices.Sort(c.OptionalExtensions)
		c.OptionalExtensions = slices.Compact(c.OptionalExtensions)
	}

	var selector C.vxr_vk_device_selector
	{
		C.vxr_vk_device_createSelector(C.uintptr_t(c.PreferredVkPhysicalDevice), C.uint32_t(c.API), C.uint64_t(surface), &selector)

		// extensions
		{
			for _, s := range c.RequiredExtensions {
				C.vxr_vk_device_selector_appendRequiredExtension(selector, C.size_t(len(s)), (*C.char)(unsafe.Pointer(unsafe.StringData(s))))
				runtime.KeepAlive(s)
			}
			for _, s := range c.OptionalExtensions {
				if _, found := slices.BinarySearch(c.RequiredExtensions, s); !found {
					C.vxr_vk_device_selector_appendOptionalExtension(selector, C.size_t(len(s)), (*C.char)(unsafe.Pointer(unsafe.StringData(s))))
					runtime.KeepAlive(s)
				}
			}
		}

		// features
		{
			// combine the struct lists as both require and optional must have the same chain
			{
				commonFeatureStructs := make([]C.VkStructureType, 0, max(len(c.RequiredFeatures), len(c.OptionalFeatures)))
				for _, s := range c.RequiredFeatures {
					commonFeatureStructs = append(commonFeatureStructs, s.sType())
				}
				for _, s := range c.OptionalFeatures {
					commonFeatureStructs = append(commonFeatureStructs, s.sType())
				}
				slices.Sort(commonFeatureStructs)
				commonFeatureStructs = slices.Compact(commonFeatureStructs)
				C.vxr_vk_device_selector_initFeatureChain(selector, C.size_t(len(commonFeatureStructs)), unsafe.SliceData(commonFeatureStructs))
				runtime.KeepAlive(commonFeatureStructs)
			}
			// required features
			{
				for _, s := range c.RequiredFeatures {
					list := s.enabledList()
					C.vxr_vk_device_selector_appendRequiredFeature(selector, s.sType(), C.size_t(len(list)), unsafe.SliceData(list))
					runtime.KeepAlive(list)
				}
			}
			// optional features
			{
				for _, s := range c.OptionalFeatures {
					list := s.enabledList()
					C.vxr_vk_device_selector_appendOptionalFeature(selector, s.sType(), C.size_t(len(list)), unsafe.SliceData(list))
					runtime.KeepAlive(list)
				}
			}
		}

		// format features
		{
			keys := maps.Keys(c.RequiredFormatFeatures)
			slices.Sort(keys)
			for _, k := range keys {
				C.vxr_vk_device_selector_appendRequiredFormatFeature(selector, C.VkFormat(k), C.VkFormatFeatureFlags2(c.RequiredFormatFeatures[k]))
			}
		}
		// depth format features
		{
			keys := maps.Keys(c.RequiredDepthStencilFormatFeatures)
			slices.Sort(keys)
			for _, k := range keys {
				C.vxr_vk_device_selector_appendRequiredFormatFeature(selector, C.VkFormat(k), C.VkFormatFeatureFlags2(c.RequiredDepthStencilFormatFeatures[k]))
			}
		}
	}

	return selector
}

type config struct {
	maxFramesInFlight      int32
	descriptorPoolBankSize int32
}

func (c *config) use(user Config) {
	c.maxFramesInFlight = user.MaxFramesInFlight
	c.descriptorPoolBankSize = user.DescriptorPoolBankSize

	for k := range user.RequiredFormatFeatures {
		_ = instance.formatProperties.colorFeatures(k)
	}
	for k := range user.RequiredDepthStencilFormatFeatures {
		_ = instance.formatProperties.depthFeatures(k)
	}
}
