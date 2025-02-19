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

#include <stddef.h>
#include <stdint.h>

#include "std/vector.hpp"
#include "std/array.hpp"
#include "std/utility.hpp"

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/vkfns.hpp"
#include "vk/device/device.hpp"
#include "vk/device/selector/selector.hpp"

inline static bool findComputeQueue(vxr::vk::device::instance* device, const auto& queueFamilies) {
	// Best is isolated queue
	for (uint32_t i = 0; i < queueFamilies.size(); i++) {
		if (vxr::std::cmpBitFlags(queueFamilies[i].queueFlags, VK_QUEUE_COMPUTE_BIT, VK_QUEUE_GRAPHICS_BIT)) {
			device->computeQueue.family = i;
			return true;
		}
	}

	return false;
}

inline static bool findGraphicsQueue(vxr::vk::device::instance* device, VkSurfaceKHR vkSurface, const auto& queueFamilies) {
	for (uint32_t i = 0; i < queueFamilies.size(); i++) {
		VkBool32 presentSupport = VK_FALSE;
		const VkResult ret = VK_PROC(vkGetPhysicalDeviceSurfaceSupportKHR)(device->vkPhysicalDevice, i, vkSurface, &presentSupport);
		if (ret != VK_SUCCESS) {
			return false;
		}

		if (vxr::std::cmpBitFlagsContains(queueFamilies[i].queueFlags, VkQueueFlags(VK_QUEUE_GRAPHICS_BIT | VK_QUEUE_COMPUTE_BIT)) &&
			(presentSupport != 0u)) {
			device->graphicsQueue.family = i;
			return true;
		}
	}

	return false;
}

inline static bool findTransferQueue(vxr::vk::device::instance* device, const auto& queueFamilies) {
	// Best is isolated queue
	for (uint32_t i = 0; i < queueFamilies.size(); i++) {
		if (vxr::std::cmpBitFlags(queueFamilies[i].queueFlags, VK_QUEUE_TRANSFER_BIT, VK_QUEUE_COMPUTE_BIT | VK_QUEUE_GRAPHICS_BIT)) {
			device->transferQueue.family = i;
			return true;
		}
	}

	return false;
}

bool vxr::vk::device::selector::selector::findQueues(vxr::vk::instance* instance) {
	uint32_t haveQueueFamilies = 0;
	VK_PROC(vkGetPhysicalDeviceQueueFamilyProperties)
	(instance->device.vkPhysicalDevice, &haveQueueFamilies, nullptr);
	vxr::std::vector<VkQueueFamilyProperties> queueFamilies(haveQueueFamilies);
	VK_PROC(vkGetPhysicalDeviceQueueFamilyProperties)
	(instance->device.vkPhysicalDevice, &haveQueueFamilies, queueFamilies.get());

	if (!findGraphicsQueue(&instance->device, this->targetSurface, queueFamilies)) {
		return false;
	}
	queueCreateInfos.resize(0);
	queuePriorities.resize(0);
	queueCreateInfos.pushBack(VkDeviceQueueCreateInfo{
		.sType = VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
		.queueFamilyIndex = instance->device.graphicsQueue.family,
		.queueCount = 1,
	});
	queuePriorities.pushBack(vxr::std::vector<float>(vxr::std::array{1.0f}));

	if (findComputeQueue(&instance->device, queueFamilies)) {
		queueCreateInfos.pushBack(VkDeviceQueueCreateInfo{
			.sType = VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
			.queueFamilyIndex = instance->device.computeQueue.family,
			.queueCount = 1,
		});
		queuePriorities.pushBack(vxr::std::vector<float>(vxr::std::array{0.5f}));
	} else if (queueFamilies[queueCreateInfos[0].queueFamilyIndex].queueCount > queueCreateInfos[0].queueCount) {
		instance->device.computeQueue = {
			.family = queueCreateInfos[0].queueFamilyIndex,
			.index = queueCreateInfos[0].queueCount,
		};
		queueCreateInfos[0].queueCount++;
		queuePriorities[0].pushBack(0.5f);
	} else {
		return false;
	}

	if (findTransferQueue(&instance->device, queueFamilies)) {
		queueCreateInfos.pushBack(VkDeviceQueueCreateInfo{
			.sType = VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
			.queueFamilyIndex = instance->device.transferQueue.family,
			.queueCount = 1,
		});
		queuePriorities.pushBack(vxr::std::vector<float>(vxr::std::array{0.0f}));
	} else if (queueFamilies[queueCreateInfos[0].queueFamilyIndex].queueCount > queueCreateInfos[0].queueCount) {
		instance->device.transferQueue = {
			.family = queueCreateInfos[0].queueFamilyIndex,
			.index = queueCreateInfos[0].queueCount,
		};
		queueCreateInfos[0].queueCount++;
		queuePriorities[0].pushBack(0.0f);
	} else {
		return false;
	}

	return true;
}
