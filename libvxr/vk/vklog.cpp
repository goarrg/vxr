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

#include "std/stdlib.hpp"  // IWYU pragma: keep
#include "std/log.hpp"	   // IWYU pragma: keep

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/vklog.hpp"
#include "vk/vkfns.hpp"

namespace vxr::vk {
void initMessenger([[maybe_unused]] instance* instance, [[maybe_unused]] PFN_vkDebugUtilsMessengerCallbackEXT callback) {
#ifndef NDEBUG
	VkDebugUtilsMessengerCreateInfoEXT pCreateInfo = {};
	pCreateInfo.sType = VK_STRUCTURE_TYPE_DEBUG_UTILS_MESSENGER_CREATE_INFO_EXT;
	pCreateInfo.messageSeverity = VK_DEBUG_UTILS_MESSAGE_SEVERITY_ERROR_BIT_EXT;

#if VXR_LOG_LEVEL <= VXR_LOG_LEVEL_VERBOSE
	pCreateInfo.messageSeverity |= VK_DEBUG_UTILS_MESSAGE_SEVERITY_VERBOSE_BIT_EXT;
#endif
#if VXR_LOG_LEVEL <= VXR_LOG_LEVEL_INFO
	pCreateInfo.messageSeverity |= VK_DEBUG_UTILS_MESSAGE_SEVERITY_INFO_BIT_EXT;
#endif
#if VXR_LOG_LEVEL <= VXR_LOG_LEVEL_WARN
	pCreateInfo.messageSeverity |= VK_DEBUG_UTILS_MESSAGE_SEVERITY_WARNING_BIT_EXT;
#endif

	pCreateInfo.messageType = VK_DEBUG_UTILS_MESSAGE_TYPE_GENERAL_BIT_EXT | VK_DEBUG_UTILS_MESSAGE_TYPE_VALIDATION_BIT_EXT
							  | VK_DEBUG_UTILS_MESSAGE_TYPE_PERFORMANCE_BIT_EXT;
	pCreateInfo.pfnUserCallback = callback;

	const VkResult ret = VK_DEBUG_PROC(vkCreateDebugUtilsMessengerEXT)(
		instance->vkInstance, &pCreateInfo, nullptr, &instance->vkMessenger);
	if (ret != VK_SUCCESS) {
		vxr::std::abortPopup(
			vxr::std::sourceLocation::current(), "Failed to init VkDebugUtilsMessenger: %s", vkResultStr(ret).cStr());
	}
#endif
}

void destroyMessenger([[maybe_unused]] instance* instance) {
#ifndef NDEBUG
	VK_DEBUG_PROC(vkDestroyDebugUtilsMessengerEXT)(instance->vkInstance, instance->vkMessenger, nullptr);
	instance->vkMessenger = VK_NULL_HANDLE;
#endif
}
}  // namespace vxr::vk
