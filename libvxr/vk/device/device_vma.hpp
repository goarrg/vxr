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

#pragma once

#ifndef __cplusplus
#error C++ only header
#endif

#include <stdint.h>

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/utility.hpp"

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/vkfns.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"
#include "vk/device/vma/vma.hpp"

inline static void setupVMA(vxr::vk::instance* instance) {
	// VMA version check to make sure we update the function struct.
	static_assert(VMA_VULKAN_VERSION == 1004000);
	const VmaVulkanFunctions vkFns = {
		VK_PROC(vkGetInstanceProcAddr),
		VK_PROC(vkGetDeviceProcAddr),
		VK_PROC(vkGetPhysicalDeviceProperties),
		VK_PROC(vkGetPhysicalDeviceMemoryProperties),
		VK_PROC_DEVICE(vkAllocateMemory),
		VK_PROC_DEVICE(vkFreeMemory),
		VK_PROC_DEVICE(vkMapMemory),
		VK_PROC_DEVICE(vkUnmapMemory),
		VK_PROC_DEVICE(vkFlushMappedMemoryRanges),
		VK_PROC_DEVICE(vkInvalidateMappedMemoryRanges),
		VK_PROC_DEVICE(vkBindBufferMemory),
		VK_PROC_DEVICE(vkBindImageMemory),
		VK_PROC_DEVICE(vkGetBufferMemoryRequirements),
		VK_PROC_DEVICE(vkGetImageMemoryRequirements),
		VK_PROC_DEVICE(vkCreateBuffer),
		VK_PROC_DEVICE(vkDestroyBuffer),
		VK_PROC_DEVICE(vkCreateImage),
		VK_PROC_DEVICE(vkDestroyImage),
		VK_PROC_DEVICE(vkCmdCopyBuffer),
		VK_PROC_DEVICE(vkGetBufferMemoryRequirements2),
		VK_PROC_DEVICE(vkGetImageMemoryRequirements2),
		VK_PROC_DEVICE(vkBindBufferMemory2),
		VK_PROC_DEVICE(vkBindImageMemory2),
		VK_PROC(vkGetPhysicalDeviceMemoryProperties2),
		VK_PROC_DEVICE(vkGetDeviceBufferMemoryRequirements),
		VK_PROC_DEVICE(vkGetDeviceImageMemoryRequirements),
	};

	VmaAllocatorCreateInfo allocatorInfo = {};
	allocatorInfo.physicalDevice = instance->device.vkPhysicalDevice;
	allocatorInfo.device = instance->device.vkDevice;
	allocatorInfo.pVulkanFunctions = &vkFns;
	allocatorInfo.instance = instance->vkInstance;
	allocatorInfo.vulkanApiVersion = instance->device.properties.api;
	allocatorInfo.flags = VMA_ALLOCATOR_CREATE_EXT_MEMORY_BUDGET_BIT | VMA_ALLOCATOR_CREATE_KHR_MAINTENANCE4_BIT;

	const VkResult ret = vmaCreateAllocator(&allocatorInfo, &instance->device.vma.allocator);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to init VMA: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	instance->device.vma.noBARMemoryTypeBits = 0;
	instance->device.vma.barMemoryTypeBits = 0;
	VkPhysicalDeviceMemoryProperties2 properties = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_MEMORY_PROPERTIES_2};
	VK_PROC(vkGetPhysicalDeviceMemoryProperties2)(instance->device.vkPhysicalDevice, &properties);
	for (uint32_t i = 0; i < properties.memoryProperties.memoryTypeCount; i++) {
		if (vxr::std::cmpBitFlagsContains(
				properties.memoryProperties.memoryTypes[i].propertyFlags,
				VkMemoryPropertyFlags(VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT | VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT))) {
			vxr::std::vPrintf("Found BAR memory type: %d", i);
			instance->device.vma.barMemoryTypeBits |= 1 << i;
		} else {
			instance->device.vma.noBARMemoryTypeBits |= 1 << i;
		}
	}
}

inline static void destroyVMA(vxr::vk::instance* instance) {
	vmaDestroyAllocator(instance->device.vma.allocator);
}
