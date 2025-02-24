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

	#include <string.h>
	#include "vxr/vxr.h"
*/
import "C"

import (
	"runtime"
	"strings"
	"unsafe"

	"goarrg.com/debug"
	"goarrg.com/rhi/vxr/internal/vk"
)

type BufferUsageFlags C.VkBufferUsageFlags

const (
	BufferUsageTransferSrc        BufferUsageFlags = vk.BUFFER_USAGE_TRANSFER_SRC_BIT
	BufferUsageTransferDst        BufferUsageFlags = vk.BUFFER_USAGE_TRANSFER_DST_BIT
	BufferUsageUniformBuffer      BufferUsageFlags = vk.BUFFER_USAGE_UNIFORM_BUFFER_BIT
	BufferUsageUniformTexelBuffer BufferUsageFlags = vk.BUFFER_USAGE_UNIFORM_TEXEL_BUFFER_BIT
	BufferUsageStorageBuffer      BufferUsageFlags = vk.BUFFER_USAGE_STORAGE_BUFFER_BIT
	BufferUsageStorageTexelBuffer BufferUsageFlags = vk.BUFFER_USAGE_STORAGE_TEXEL_BUFFER_BIT
	BufferUsageIndexBuffer        BufferUsageFlags = vk.BUFFER_USAGE_INDEX_BUFFER_BIT
	BufferUsageVertexBuffer       BufferUsageFlags = vk.BUFFER_USAGE_VERTEX_BUFFER_BIT
	BufferUsageIndirectBuffer     BufferUsageFlags = vk.BUFFER_USAGE_INDIRECT_BUFFER_BIT
)

func (u BufferUsageFlags) HasBits(want BufferUsageFlags) bool {
	return (u & want) == want
}

func (u BufferUsageFlags) String() string {
	str := ""
	if u.HasBits(BufferUsageTransferSrc) {
		str += "TransferSrc|"
	}
	if u.HasBits(BufferUsageTransferDst) {
		str += "TransferDst|"
	}
	if u.HasBits(BufferUsageUniformBuffer) {
		str += "UniformBuffer|"
	}
	if u.HasBits(BufferUsageUniformTexelBuffer) {
		str += "UniformTexelBuffer|"
	}
	if u.HasBits(BufferUsageStorageBuffer) {
		str += "StorageBuffer|"
	}
	if u.HasBits(BufferUsageStorageTexelBuffer) {
		str += "StorageTexelBuffer|"
	}
	if u.HasBits(BufferUsageIndexBuffer) {
		str += "IndexBuffer|"
	}
	if u.HasBits(BufferUsageVertexBuffer) {
		str += "VertexBuffer|"
	}
	if u.HasBits(BufferUsageIndirectBuffer) {
		str += "IndirectBuffer|"
	}
	return strings.TrimSuffix(str, "|")
}

type Buffer interface {
	Usage() BufferUsageFlags
	Size() uint64

	vkBuffer() C.VkBuffer
}

type HostBuffer struct {
	noCopy     noCopy
	bufferSize uint64
	usageFlags BufferUsageFlags
	cBuffer    C.vxr_vk_hostBuffer
}

var _ interface {
	Buffer
	Destroyer
} = (*HostBuffer)(nil)

func validateBufferCreation(size uint64, usage BufferUsageFlags) error {
	switch usage {
	case BufferUsageUniformBuffer:
		if size > uint64(instance.deviceProperties.Limits.PerDesctiptor.MaxUBOSize) {
			return debug.Errorf("Buffer size [%d] is larger than DeviceProperties.Limits.PerDesctiptor.MaxUBOSize [%d]",
				size, instance.deviceProperties.Limits.PerDesctiptor.MaxUBOSize)
		}
	case BufferUsageStorageBuffer:
		if size > uint64(instance.deviceProperties.Limits.PerDesctiptor.MaxSBOSize) {
			return debug.Errorf("Buffer size [%d] is larger than DeviceProperties.Limits.PerDesctiptor.MaxSBOSize [%d]",
				size, instance.deviceProperties.Limits.PerDesctiptor.MaxSBOSize)
		}
	}

	return nil
}

func NewHostBuffer(name string, size uint64, usage BufferUsageFlags) *HostBuffer {
	if err := validateBufferCreation(size, usage); err != nil {
		abort("Failed trying to create HostBuffer with size [%d] and usage [%s]: %s", size, usage.String(), err)
	}
	b := HostBuffer{bufferSize: size, usageFlags: usage}
	b.noCopy.init()
	info := C.vxr_vk_bufferCreateInfo{
		size:  C.VkDeviceSize(size),
		usage: C.VkBufferUsageFlags(usage),
	}
	C.vxr_vk_createHostBuffer(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
		info, &b.cBuffer)
	runtime.KeepAlive(name)
	return &b
}

func (b *HostBuffer) HostWrite(offset uintptr, data []byte) {
	b.noCopy.check()
	if (uint64(len(data)) + uint64(offset)) > b.bufferSize {
		abort("HostWrite(%d, len(data): %d) will overflow buffer of size %d", offset, len(data), b.bufferSize)
	}

	C.vxr_vk_hostBuffer_write(instance.cInstance, b.cBuffer, C.size_t(offset), C.size_t(len(data)), unsafe.Pointer(unsafe.SliceData(data)))
	runtime.KeepAlive(data)
}

func (b *HostBuffer) HostRead(offset uintptr, data []byte) {
	b.noCopy.check()
	if (uint64(len(data)) + uint64(offset)) > b.bufferSize {
		abort("HostRead(%d, len(data): %d) will overflow buffer of size %d", offset, len(data), b.bufferSize)
	}

	C.vxr_vk_hostBuffer_read(instance.cInstance, b.cBuffer, C.size_t(offset), C.size_t(len(data)), unsafe.Pointer(unsafe.SliceData(data)))
	runtime.KeepAlive(data)
}

func (b *HostBuffer) Usage() BufferUsageFlags {
	b.noCopy.check()
	return b.usageFlags
}

func (b *HostBuffer) Size() uint64 {
	b.noCopy.check()
	return b.bufferSize
}

func (b *HostBuffer) Destroy() {
	b.noCopy.check()
	C.vxr_vk_destroyHostBuffer(instance.cInstance, b.cBuffer)
	b.noCopy.close()
}

func (b *HostBuffer) vkBuffer() C.VkBuffer {
	b.noCopy.check()
	return b.cBuffer.vkBuffer
}

type DeviceBuffer struct {
	noCopy     noCopy
	bufferSize uint64
	usageFlags BufferUsageFlags
	cBuffer    C.vxr_vk_deviceBuffer
}

var _ interface {
	Buffer
	Destroyer
} = (*DeviceBuffer)(nil)

func NewDeviceBuffer(name string, size uint64, usage BufferUsageFlags) *DeviceBuffer {
	if err := validateBufferCreation(size, usage); err != nil {
		abort("Failed trying to create DeviceBuffer with size [%d] and usage [%s]: %s", size, usage.String(), err)
	}
	b := DeviceBuffer{bufferSize: size, usageFlags: usage}
	b.noCopy.init()
	info := C.vxr_vk_bufferCreateInfo{
		size:  C.VkDeviceSize(size),
		usage: C.VkBufferUsageFlags(usage),
	}
	C.vxr_vk_createDeviceBuffer(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
		info, &b.cBuffer)
	runtime.KeepAlive(name)
	return &b
}

func (b *DeviceBuffer) Usage() BufferUsageFlags {
	b.noCopy.check()
	return b.usageFlags
}

func (b *DeviceBuffer) Size() uint64 {
	b.noCopy.check()
	return b.bufferSize
}

func (b *DeviceBuffer) Destroy() {
	b.noCopy.check()
	C.vxr_vk_destroyDeviceBuffer(instance.cInstance, b.cBuffer)
	b.noCopy.close()
}

func (b *DeviceBuffer) vkBuffer() C.VkBuffer {
	b.noCopy.check()
	return b.cBuffer.vkBuffer
}
