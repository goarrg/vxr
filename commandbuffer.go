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
	"goarrg.com/rhi/vxr/internal/vk"
)

type commandBuffer struct {
	noCopy          noCopy
	vkCommandBuffer C.VkCommandBuffer
}

func (cb *commandBuffer) BeginNamedRegion(name string) {
	cb.noCopy.check()
	C.vxr_vk_commandBuffer_beginNamedRegion(instance.cInstance, cb.vkCommandBuffer, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))))
	runtime.KeepAlive(name)
}

func (cb *commandBuffer) EndNamedRegion() {
	cb.noCopy.check()
	C.vxr_vk_commandBuffer_endNamedRegion(instance.cInstance, cb.vkCommandBuffer)
}

type AccessFlags C.VkAccessFlags2

const (
	AccessFlagNone        AccessFlags = vk.ACCESS_2_NONE
	AccessFlagMemoryRead  AccessFlags = vk.ACCESS_2_MEMORY_READ_BIT
	AccessFlagMemoryWrite AccessFlags = vk.ACCESS_2_MEMORY_WRITE_BIT
)

type MemoryBarrierInfo struct {
	Stage  PipelineStage
	Access AccessFlags
}

type MemoryBarrier struct {
	Src MemoryBarrierInfo
	Dst MemoryBarrierInfo
}

type BufferBarrierInfo struct {
	Stage  PipelineStage
	Access AccessFlags
}

type BufferBarrier struct {
	Buffer Buffer
	Src    BufferBarrierInfo
	Dst    BufferBarrierInfo
}

type ImageBarrierInfo struct {
	Stage  PipelineStage
	Access AccessFlags
	Layout ImageLayout
}

type ImageSubresourceRange struct {
	BaseMipLevel uint32
	NumMipLevels uint32

	BaseArrayLayer uint32
	NumArrayLayers uint32
}

type ImageBarrier struct {
	Image Image
	Src   ImageBarrierInfo
	Dst   ImageBarrierInfo
	Range ImageSubresourceRange
}

func (cb *commandBuffer) CompoundBarrier(memoryBarriers []MemoryBarrier, bufferBarriers []BufferBarrier, imageBarriers []ImageBarrier) {
	cb.noCopy.check()

	memoryBarrierInfos := make([]C.VkMemoryBarrier2, 0, len(memoryBarriers))
	for _, barrier := range memoryBarriers {
		memoryBarrierInfos = append(memoryBarrierInfos,
			C.VkMemoryBarrier2{
				sType:         vk.STRUCTURE_TYPE_MEMORY_BARRIER_2,
				srcStageMask:  C.VkPipelineStageFlags2(barrier.Src.Stage),
				srcAccessMask: C.VkAccessFlags2(barrier.Src.Access),
				dstStageMask:  C.VkPipelineStageFlags2(barrier.Dst.Stage),
				dstAccessMask: C.VkAccessFlags2(barrier.Dst.Access),
			},
		)
	}

	bufferBarrierInfos := make([]C.VkBufferMemoryBarrier2, 0, len(bufferBarriers))
	for _, barrier := range bufferBarriers {
		bufferBarrierInfos = append(bufferBarrierInfos,
			C.VkBufferMemoryBarrier2{
				sType:         vk.STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER_2,
				srcStageMask:  C.VkPipelineStageFlags2(barrier.Src.Stage),
				srcAccessMask: C.VkAccessFlags2(barrier.Src.Access),
				dstStageMask:  C.VkPipelineStageFlags2(barrier.Dst.Stage),
				dstAccessMask: C.VkAccessFlags2(barrier.Dst.Access),
				buffer:        barrier.Buffer.vkBuffer(),
				offset:        0,
				size:          vk.WHOLE_SIZE,
			},
		)
	}

	imageBarrierInfos := make([]C.VkImageMemoryBarrier2, 0, len(imageBarriers))
	for _, barrier := range imageBarriers {
		imageBarrierInfos = append(imageBarrierInfos,
			C.VkImageMemoryBarrier2{
				sType:         vk.STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER_2,
				srcStageMask:  C.VkPipelineStageFlags2(barrier.Src.Stage),
				srcAccessMask: C.VkAccessFlags2(barrier.Src.Access),
				dstStageMask:  C.VkPipelineStageFlags2(barrier.Dst.Stage),
				dstAccessMask: C.VkAccessFlags2(barrier.Dst.Access),
				oldLayout:     C.VkImageLayout(barrier.Src.Layout),
				newLayout:     C.VkImageLayout(barrier.Dst.Layout),
				image:         barrier.Image.vkImage(),
				subresourceRange: C.VkImageSubresourceRange{
					aspectMask:   barrier.Image.vkImageAspectFlags(),
					baseMipLevel: C.uint32_t(barrier.Range.BaseMipLevel), levelCount: C.uint32_t(barrier.Range.NumMipLevels),
					baseArrayLayer: C.uint32_t(barrier.Range.BaseArrayLayer), layerCount: C.uint32_t(barrier.Range.NumArrayLayers),
				},
			},
		)
	}

	C.vxr_vk_commandBuffer_barrier(instance.cInstance, cb.vkCommandBuffer,
		C.VkDependencyInfo{
			sType:              vk.STRUCTURE_TYPE_DEPENDENCY_INFO,
			memoryBarrierCount: C.uint32_t(len(memoryBarrierInfos)), pMemoryBarriers: unsafe.SliceData(memoryBarrierInfos),
			bufferMemoryBarrierCount: C.uint32_t(len(bufferBarriers)), pBufferMemoryBarriers: unsafe.SliceData(bufferBarrierInfos),
			imageMemoryBarrierCount: C.uint32_t(len(imageBarrierInfos)), pImageMemoryBarriers: unsafe.SliceData(imageBarrierInfos),
		})
	runtime.KeepAlive(memoryBarrierInfos)
	runtime.KeepAlive(bufferBarrierInfos)
	runtime.KeepAlive(imageBarrierInfos)
}

func (cb *commandBuffer) ExecutionBarrier(src, dst PipelineStage) {
	cb.CompoundBarrier([]MemoryBarrier{{Src: MemoryBarrierInfo{Stage: src}, Dst: MemoryBarrierInfo{Stage: dst}}}, nil, nil)
}

func (cb *commandBuffer) MemoryBarrier(barriers ...MemoryBarrier) {
	cb.CompoundBarrier(barriers, nil, nil)
}

func (cb *commandBuffer) ImageBarrier(barriers ...ImageBarrier) {
	cb.CompoundBarrier(nil, nil, barriers)
}

func (cb *commandBuffer) FillBuffer(buffer Buffer, offset, size uint64, value uint32) {
	cb.noCopy.check()
	C.vxr_vk_commandBuffer_fillBuffer(instance.cInstance, cb.vkCommandBuffer, buffer.vkBuffer(), C.VkDeviceSize(offset), C.VkDeviceSize(size), C.uint32_t(value))
}

func (cb *commandBuffer) UpdateBuffer(buffer Buffer, offset uint64, data []byte) {
	cb.noCopy.check()
	if len(data) > 65536 {
		abort("UpdateBuffer is limited to 65536 bytes")
	}
	C.vxr_vk_commandBuffer_updateBuffer(instance.cInstance, cb.vkCommandBuffer, buffer.vkBuffer(), C.VkDeviceSize(offset),
		C.VkDeviceSize(len(data)), unsafe.Pointer(unsafe.SliceData(data)))
}

func (cb *commandBuffer) ClearColorImage(img ColorImage, layout ImageLayout, value ColorImageClearValue, imgRange ImageSubresourceRange) {
	cRange := C.VkImageSubresourceRange{
		aspectMask:   img.vkImageAspectFlags(),
		baseMipLevel: C.uint32_t(imgRange.BaseMipLevel), levelCount: C.uint32_t(imgRange.NumMipLevels),
		baseArrayLayer: C.uint32_t(imgRange.BaseArrayLayer), layerCount: C.uint32_t(imgRange.NumArrayLayers),
	}

	C.vxr_vk_commandBuffer_clearColorImage(instance.cInstance, cb.vkCommandBuffer, img.vkImage(), C.VkImageLayout(layout), value.vkClearValue(),
		1, &cRange)
}

type BufferCopyRegion struct {
	SrcBufferOffset uint64
	DstBufferOffset uint64
	Size            uint64
}

func (cb *commandBuffer) CopyBuffer(bIn, bOut Buffer, regions []BufferCopyRegion) {
	cb.noCopy.check()

	cRegions := make([]C.VkBufferCopy, len(regions))
	for i, r := range regions {
		cRegions[i] = C.VkBufferCopy{
			srcOffset: C.VkDeviceSize(r.SrcBufferOffset),
			dstOffset: C.VkDeviceSize(r.DstBufferOffset),
			size:      C.VkDeviceSize(r.Size),
		}
	}
	C.vxr_vk_commandBuffer_copyBuffer(instance.cInstance, cb.vkCommandBuffer, bIn.vkBuffer(), bOut.vkBuffer(),
		C.uint32_t(len(cRegions)), unsafe.SliceData(cRegions))
	runtime.KeepAlive(cRegions)
}

type ImageSubresourceLayers struct {
	MipLevel uint32

	BaseArrayLayer uint32
	NumArrayLayers uint32
}

type BufferImageCopyRegion struct {
	BufferOffset     uint64
	ImageSubresource ImageSubresourceLayers
	ImageOffset      gmath.Vector3i32
	ImageExtent      gmath.Extent3i32
}

func (cb *commandBuffer) CopyBufferToImage(buffer Buffer, image Image, layout ImageLayout, regions []BufferImageCopyRegion) {
	cb.noCopy.check()

	cRegions := make([]C.VkBufferImageCopy, len(regions))
	for i, r := range regions {
		cRegions[i] = C.VkBufferImageCopy{
			bufferOffset: C.VkDeviceSize(r.BufferOffset),
			imageSubresource: C.VkImageSubresourceLayers{
				aspectMask:     image.vkImageAspectFlags(),
				mipLevel:       C.uint32_t(r.ImageSubresource.MipLevel),
				baseArrayLayer: C.uint32_t(r.ImageSubresource.BaseArrayLayer),
				layerCount:     C.uint32_t(r.ImageSubresource.NumArrayLayers),
			},
			imageOffset: C.VkOffset3D{
				x: C.int32_t(r.ImageOffset.X),
				y: C.int32_t(r.ImageOffset.Y),
				z: C.int32_t(r.ImageOffset.Z),
			},
			imageExtent: C.VkExtent3D{
				width:  C.uint32_t(r.ImageExtent.X),
				height: C.uint32_t(r.ImageExtent.Y),
				depth:  C.uint32_t(r.ImageExtent.Z),
			},
		}
	}
	C.vxr_vk_commandBuffer_copyBufferToImage(instance.cInstance, cb.vkCommandBuffer, buffer.vkBuffer(), image.vkImage(), C.VkImageLayout(layout),
		C.uint32_t(len(cRegions)), unsafe.SliceData(cRegions))
	runtime.KeepAlive(cRegions)
}
