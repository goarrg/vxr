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

#include "vxr/vxr.h"  // IWYU pragma: associated

#include <stddef.h>
#include <stdint.h>
#include <string.h>

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/string.hpp"

#include "vk/vk.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"
#include "vk/device/vma/vma.hpp"

VXR_FN void vxr_vk_createHostBuffer(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
									vxr_vk_bufferCreateInfo info, vxr_vk_hostBuffer* b) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	VkBufferCreateInfo bufferInfo = {};
	bufferInfo.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
	bufferInfo.size = info.size;
	bufferInfo.usage = info.usage;
	bufferInfo.sharingMode = VK_SHARING_MODE_EXCLUSIVE;

	VmaAllocationCreateInfo allocCreateInfo = {};
	allocCreateInfo.usage = VMA_MEMORY_USAGE_AUTO_PREFER_HOST;
	allocCreateInfo.flags = VMA_ALLOCATION_CREATE_HOST_ACCESS_SEQUENTIAL_WRITE_BIT;
	allocCreateInfo.requiredFlags = VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_CACHED_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT;
	allocCreateInfo.memoryTypeBits = instance->device.vma.noBARMemoryTypeBits;

	VkResult ret = vmaCreateBuffer(instance->device.vma.allocator, &bufferInfo, &allocCreateInfo, &b->vkBuffer,
								   reinterpret_cast<VmaAllocation*>(&b->allocation), nullptr);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create buffer: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	ret = vmaMapMemory(instance->device.vma.allocator, reinterpret_cast<VmaAllocation>(b->allocation), &b->ptr);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to map buffer: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder builder;
		builder.write("buffer_host_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, b->vkBuffer, builder.cStr());

		builder.write("_allocation");
		vmaSetAllocationName(instance->device.vma.allocator, reinterpret_cast<VmaAllocation>(b->allocation), builder.cStr());
	});
}
VXR_FN void vxr_vk_destroyHostBuffer(vxr_vk_instance instanceHandle, vxr_vk_hostBuffer b) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	vmaUnmapMemory(instance->device.vma.allocator, reinterpret_cast<VmaAllocation>(b.allocation));
	vmaDestroyBuffer(instance->device.vma.allocator, b.vkBuffer, reinterpret_cast<VmaAllocation>(b.allocation));
}
VXR_FN void vxr_vk_hostBuffer_write(vxr_vk_instance, vxr_vk_hostBuffer buffer, size_t offset, size_t sz, void* data) {
	memcpy(static_cast<void*>(static_cast<uint8_t*>(buffer.ptr) + offset), data, sz);
}
VXR_FN void vxr_vk_hostBuffer_read(vxr_vk_instance, vxr_vk_hostBuffer buffer, size_t offset, size_t sz, void* data) {
	memcpy(data, static_cast<void*>(static_cast<uint8_t*>(buffer.ptr) + offset), sz);
}
VXR_FN void vxr_vk_createDeviceBuffer(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
									  vxr_vk_bufferCreateInfo info, vxr_vk_deviceBuffer* b) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	VkBufferCreateInfo bufferInfo = {};
	bufferInfo.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
	bufferInfo.size = info.size;
	bufferInfo.usage = info.usage;
	bufferInfo.sharingMode = VK_SHARING_MODE_EXCLUSIVE;

	VmaAllocationCreateInfo allocCreateInfo = {};
	allocCreateInfo.usage = VMA_MEMORY_USAGE_AUTO_PREFER_DEVICE;
	allocCreateInfo.requiredFlags = VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT;
	allocCreateInfo.memoryTypeBits = instance->device.vma.noBARMemoryTypeBits;

	const VkResult ret = vmaCreateBuffer(instance->device.vma.allocator, &bufferInfo, &allocCreateInfo, &b->vkBuffer,
										 reinterpret_cast<VmaAllocation*>(&b->allocation), nullptr);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create buffer: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder builder;
		builder.write("buffer_device_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, b->vkBuffer, builder.cStr());

		builder.write("_allocation");
		vmaSetAllocationName(instance->device.vma.allocator, reinterpret_cast<VmaAllocation>(b->allocation), builder.cStr());
	});
}
VXR_FN void vxr_vk_destroyDeviceBuffer(vxr_vk_instance instanceHandle, vxr_vk_deviceBuffer b) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	vmaDestroyBuffer(instance->device.vma.allocator, b.vkBuffer, reinterpret_cast<VmaAllocation>(b.allocation));
}
