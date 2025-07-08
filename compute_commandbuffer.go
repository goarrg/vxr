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
	"runtime"
	"unsafe"

	"goarrg.com/gmath"
)

type ComputeCommandBuffer struct {
	commandBuffer
}

type DispatchInfo struct {
	PushConstants  []byte
	DescriptorSets []*DescriptorSet
	ThreadCount    gmath.Extent3u32
}

func (cb *ComputeCommandBuffer) Dispatch(p *ComputePipeline, info DispatchInfo) {
	cb.noCopy.Check()

	if err := p.layout.cmdValidate(info.PushConstants, info.DescriptorSets); err != nil {
		abort("Failed to validate DispatchInfo: %s", err)
	}

	descriptorSets := make([]C.VkDescriptorSet, 0, len(info.DescriptorSets))
	for _, s := range info.DescriptorSets {
		s.noCopy.Check()
		descriptorSets = append(descriptorSets, s.cDescriptorSet)
	}

	cInfo := C.vxr_vk_compute_dispatchInfo{
		layout:   p.layout.vkPipelinelayout,
		pipeline: p.vkPipeline,

		numDescriptorSets: C.uint32_t(len(descriptorSets)),
		descriptorSets:    unsafe.SliceData(descriptorSets),

		groupCount: C.VkExtent3D{
			width:  C.uint32_t((info.ThreadCount.X + p.localSize.X - 1) / p.localSize.X),
			height: C.uint32_t((info.ThreadCount.Y + p.localSize.Y - 1) / p.localSize.Y),
			depth:  C.uint32_t((info.ThreadCount.Z + p.localSize.Z - 1) / p.localSize.Z),
		},
	}

	if (cInfo.groupCount.width > C.uint32_t(instance.deviceProperties.Limits.Compute.MaxDispatchSize.X)) ||
		(cInfo.groupCount.height > C.uint32_t(instance.deviceProperties.Limits.Compute.MaxDispatchSize.Y)) ||
		(cInfo.groupCount.depth > C.uint32_t(instance.deviceProperties.Limits.Compute.MaxDispatchSize.Z)) {
		abort("DispatchInfo.ThreadCount/localSize %+v would exceed DeviceProperties.Limits.Compute.MaxDispatchSize %+v",
			cInfo.groupCount, instance.deviceProperties.Limits.Compute.MaxDispatchSize)
	}

	if p.layout.pushConstantRange.size > 0 {
		cInfo.pushConstantRange = p.layout.pushConstantRange
		cInfo.pushConstantData = unsafe.Pointer(unsafe.SliceData(info.PushConstants))
	}

	C.vxr_vk_compute_dispatch(instance.cInstance, cb.vkCommandBuffer, cInfo)
	runtime.KeepAlive(info.PushConstants)
	runtime.KeepAlive(descriptorSets)
}

type DispatchIndirectInfo struct {
	PushConstants  []byte
	DescriptorSets []*DescriptorSet
	Buffer         Buffer
	Offset         uint64
}

func (cb *ComputeCommandBuffer) DispatchIndirect(p *ComputePipeline, info DispatchIndirectInfo) {
	cb.noCopy.Check()

	if err := p.layout.cmdValidate(info.PushConstants, info.DescriptorSets); err != nil {
		abort("Failed to validate DispatchIndirectInfo: %s", err)
	}
	if (info.Offset % 4) != 0 {
		abort("DispatchIndirectInfo.Offset [%d] is not a multiple of 4", info.Offset)
	}
	if (info.Buffer.Size() - info.Offset) < uint64(unsafe.Sizeof(C.VkDispatchIndirectCommand{})) {
		abort("DispatchIndirectInfo.Offset + sizeof(VkDrawIndexedIndirectCommand) [%d + %d] overflows buffer [%d]",
			info.Offset, unsafe.Sizeof(C.VkDispatchIndirectCommand{}), info.Buffer.Size())
	}

	descriptorSets := make([]C.VkDescriptorSet, 0, len(info.DescriptorSets))
	for _, s := range info.DescriptorSets {
		s.noCopy.Check()
		descriptorSets = append(descriptorSets, s.cDescriptorSet)
	}

	cInfo := C.vxr_vk_compute_dispatchIndirectInfo{
		layout:   p.layout.vkPipelinelayout,
		pipeline: p.vkPipeline,

		numDescriptorSets: C.uint32_t(len(descriptorSets)),
		descriptorSets:    unsafe.SliceData(descriptorSets),

		buffer: info.Buffer.vkBuffer(),
		offset: C.VkDeviceSize(info.Offset),
	}

	if p.layout.pushConstantRange.size > 0 {
		cInfo.pushConstantRange = p.layout.pushConstantRange
		cInfo.pushConstantData = unsafe.Pointer(unsafe.SliceData(info.PushConstants))
	}

	C.vxr_vk_compute_dispatchIndirect(instance.cInstance, cb.vkCommandBuffer, cInfo)
	runtime.KeepAlive(info.PushConstants)
	runtime.KeepAlive(descriptorSets)
}
