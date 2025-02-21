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
	"fmt"
	"runtime"
	"unsafe"

	"goarrg.com/gmath"
	"goarrg.com/rhi/vxr/internal/vk"
)

type ComputeStageFlags C.VkPipelineShaderStageCreateFlags

const (
	ComputeStageVaryingSubgroupSize ComputeStageFlags = vk.PIPELINE_SHADER_STAGE_CREATE_ALLOW_VARYING_SUBGROUP_SIZE_BIT
	ComputeStageRequireFullSubgroup ComputeStageFlags = vk.PIPELINE_SHADER_STAGE_CREATE_REQUIRE_FULL_SUBGROUPS_BIT
)

type ComputePipeline struct {
	noCopy     noCopy
	id         string
	name       string
	layout     *PipelineLayout
	vkPipeline C.VkPipeline
	localSize  gmath.Extent3u32
}

type ComputePipelineCreateInfo struct {
	StageFlags           ComputeStageFlags
	RequiredSubgroupSize uint32
	SpecConstants        []uint32
}

func NewComputePipeline(pipelineLayout *PipelineLayout, s *Shader, entryPoint ShaderEntryPointComputeLayout, info ComputePipelineCreateInfo) *ComputePipeline {
	localSize := gmath.Extent3u32{X: entryPoint.LocalSize[0].Value, Y: entryPoint.LocalSize[1].Value, Z: entryPoint.LocalSize[2].Value}
	if entryPoint.LocalSize[0].IsSpecConstant {
		localSize.X = info.SpecConstants[localSize.X]
	}
	if entryPoint.LocalSize[1].IsSpecConstant {
		localSize.Y = info.SpecConstants[localSize.Y]
	}
	if entryPoint.LocalSize[2].IsSpecConstant {
		localSize.Z = info.SpecConstants[localSize.Z]
	}
	if !localSize.InRange(gmath.Extent3[uint32]{}, instance.deviceProperties.Limits.Compute.MaxLocalSize) {
		limit := instance.deviceProperties.Limits.Compute.MaxLocalSize
		abort("Shader's local sizes [%d,%d,%d] is greater than Properties.Limits.Compute.MaxLocalSize [%d,%d,%d]",
			localSize.X, localSize.Y, localSize.Z, limit.X, limit.Y, limit.Z)
	}
	if info.RequiredSubgroupSize > 0 {
		if !instance.deviceProperties.Limits.Compute.SubgroupSize.CheckValue(info.RequiredSubgroupSize) {
			abort("ComputePipelineCreateInfo.RequiredSubgroupSize [%d] is not within Properties.Limits.Compute.SubgroupSize [%+v] ",
				info.RequiredSubgroupSize, instance.deviceProperties.Limits.Compute.SubgroupSize)
		}
		if maxWorkgroupThreads := info.RequiredSubgroupSize * instance.deviceProperties.Limits.Compute.Workgroup.MaxSubgroupCount; localSize.Volume() > maxWorkgroupThreads {
			abort("Shader's local sizes [%d*%d*%d] is greater than ComputePipelineCreateInfo.RequiredSubgroupSize * Properties.Limits.Compute..Workgroup.MaxSubgroupCount [%d]",
				localSize.X, localSize.Y, localSize.Z, maxWorkgroupThreads)
		}
	} else if localSize.Volume() > instance.deviceProperties.Limits.Compute.Workgroup.MaxInvocations {
		abort("Shader's local sizes [%d*%d*%d] is greater than Properties.Limits.Compute..Workgroup.MaxInvocations [%d]",
			localSize.X, localSize.Y, localSize.Z, instance.deviceProperties.Limits.Compute.Workgroup.MaxInvocations)
	}

	entryPointName := entryPoint.EntryPointName()
	pipeline := &ComputePipeline{
		name:      fmt.Sprintf("[%q,%s,%s]", s.ID, entryPointName, jsonString(info.SpecConstants)),
		layout:    pipelineLayout,
		localSize: localSize,
	}
	pipeline.noCopy.init()

	pipelineInfo := C.vxr_vk_compute_shaderPipelineCreateInfo{
		stageFlags:     C.VkPipelineShaderStageCreateFlags(info.StageFlags),
		layout:         pipelineLayout.vkPipelinelayout,
		entryPointSize: C.size_t(len(entryPointName)),
		entryPoint:     (*C.char)(unsafe.Pointer(unsafe.StringData(entryPointName))),
		spirv: C.vxr_vk_shader_spirv{
			len:  C.size_t(len(s.SPIRV)),
			data: (*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(s.SPIRV))),
		},
		requiredSubgroupSize: C.uint32_t(info.RequiredSubgroupSize),
		numSpecConstants:     C.uint32_t(len(info.SpecConstants)),
		specConstants:        (*C.uint32_t)(unsafe.SliceData(info.SpecConstants)),
	}
	C.vxr_vk_compute_createShaderPipeline(instance.cInstance, C.size_t(len(pipeline.name)), (*C.char)(unsafe.Pointer(unsafe.StringData(pipeline.name))),
		pipelineInfo, &pipeline.vkPipeline)
	runtime.KeepAlive(entryPointName)
	runtime.KeepAlive(s.SPIRV)
	runtime.KeepAlive(info.SpecConstants)

	pipeline.id = genID(pipeline.vkPipeline)
	return pipeline
}

func (p *ComputePipeline) Destroy() {
	p.noCopy.check()
	C.vxr_vk_shader_destroyPipeline(instance.cInstance, p.vkPipeline)
	p.noCopy.close()
}
