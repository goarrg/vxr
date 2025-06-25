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
	"strings"
	"sync"
	"unsafe"

	"goarrg.com/rhi/vxr/internal/vk"
	"golang.org/x/exp/maps"
)

type graphicsPipelineCache struct {
	mtx   sync.RWMutex
	cache map[string]C.VkPipeline

	mtxShader   sync.Mutex
	cacheShader map[string]*GraphicsShaderPipeline
}

type graphicsState struct {
	pipelineCache graphicsPipelineCache

	frameStarted   bool
	frameIndex     int
	framesInFlight []frame

	destroyerChan chan Destroyer
}

func (c *graphicsPipelineCache) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	{
		buff.WriteString("\"cache\": {")
		err := mapRunFuncSorted(c.cache, func(k string, v C.VkPipeline) error {
			buff.WriteString(fmt.Sprintf("%q: %q,", k, toHex(v)))
			return nil
		})
		if err == nil {
			buff.Truncate(buff.Len() - 1)
		}
		buff.WriteString("},")
	}
	{
		buff.WriteString("\"cacheShader\": {")
		err := mapRunFuncSorted(c.cacheShader, func(k string, v *GraphicsShaderPipeline) error {
			buff.WriteString(fmt.Sprintf("%q: %q,", k, toHex(v.vkPipeline)))
			return nil
		})
		if err == nil {
			buff.Truncate(buff.Len() - 1)
		}
		buff.WriteString("}")
	}

	buff.WriteString("}")
	return buff.Bytes(), nil
}

func (c *graphicsPipelineCache) destroyPipeline(id string, pipeline C.VkPipeline) {
	c.mtx.Lock()
	maps.DeleteFunc(c.cache, func(k string, v C.VkPipeline) bool {
		destroy := strings.Contains(k, id)
		if destroy {
			go func() {
				instance.graphics.destroyerChan <- destroyFunc{
					func() {
						instance.logger.VPrintf("Destroying pipeline: %s", k)
						C.vxr_vk_shader_destroyPipeline(instance.cInstance, v)
					},
				}
			}()
		}
		return destroy
	})
	c.mtx.Unlock()

	go func() {
		instance.graphics.destroyerChan <- destroyFunc{
			func() {
				instance.logger.VPrintf("Destroying pipeline: %s", id)
				C.vxr_vk_shader_destroyPipeline(instance.cInstance, pipeline)
			},
		}
	}()
}

func (c *graphicsPipelineCache) createOrRetrievePipeline(id string, f func() C.VkPipeline) C.VkPipeline {
	c.mtx.RLock()
	pipeline, ok := c.cache[id]
	c.mtx.RUnlock()
	if ok {
		return pipeline
	}

	pipeline = f()
	c.mtx.Lock()
	if c.cache[id] == nil {
		c.cache[id] = pipeline
	} else {
		defer C.vxr_vk_shader_destroyPipeline(instance.cInstance, pipeline)
		pipeline = c.cache[id]
	}
	c.mtx.Unlock()
	return pipeline
}

func (c *graphicsPipelineCache) linkOrRetrieveExecutablePipeline(id, name string, layout C.VkPipelineLayout, pipelines []C.VkPipeline) C.VkPipeline {
	c.mtx.RLock()
	pipeline, ok := c.cache[id]
	c.mtx.RUnlock()
	if ok {
		return pipeline
	}

	optimized := C.vxr_vk_graphics_linkPipelines(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
		layout, C.uint32_t(len(pipelines)), unsafe.SliceData(pipelines), &pipeline,
	)
	runtime.KeepAlive(name)
	runtime.KeepAlive(pipelines)
	c.mtx.Lock()
	if c.cache[id] == nil {
		c.cache[id] = pipeline
		if optimized == vk.FALSE {
			go func() {
				var optimized C.VkPipeline
				C.vxr_vk_graphics_linkOptimizePipelines(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
					layout, C.uint32_t(len(pipelines)), unsafe.SliceData(pipelines), &optimized,
				)
				runtime.KeepAlive(name)
				runtime.KeepAlive(pipelines)
				c.mtx.Lock()
				old := c.cache[id]
				c.cache[id] = optimized
				c.mtx.Unlock()
				instance.graphics.destroyerChan <- destroyFunc{
					func() {
						instance.logger.VPrintf("Destroying unoptimized pipeline: %s", id)
						C.vxr_vk_shader_destroyPipeline(instance.cInstance, old)
					},
				}
			}()
		}
	} else {
		defer C.vxr_vk_shader_destroyPipeline(instance.cInstance, pipeline)
		pipeline = c.cache[id]
	}
	c.mtx.Unlock()
	return pipeline
}
