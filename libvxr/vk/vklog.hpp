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

#include "std/stdlib.hpp"  // IWYU pragma: keep
#include "std/log.hpp"	   // IWYU pragma: keep
#include "std/string.hpp"
#include "std/vector.hpp"  // IWYU pragma: keep

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/vkfns.hpp"

namespace vxr::vk {
extern void initMessenger(instance*, PFN_vkDebugUtilsMessengerCallbackEXT callback);
extern void destroyMessenger(instance*);

inline static vxr::std::string<char> vkResultStr(VkResult code) {
	switch (code) {
		case VK_SUCCESS:
			return "VK_SUCCESS";
		case VK_NOT_READY:
			return "VK_NOT_READY";
		case VK_TIMEOUT:
			return "VK_TIMEOUT";
		case VK_EVENT_SET:
			return "VK_EVENT_SET";
		case VK_EVENT_RESET:
			return "VK_EVENT_RESET";
		case VK_INCOMPLETE:
			return "VK_INCOMPLETE";
		case VK_ERROR_OUT_OF_HOST_MEMORY:
			return "VK_ERROR_OUT_OF_HOST_MEMORY";
		case VK_ERROR_OUT_OF_DEVICE_MEMORY:
			return "VK_ERROR_OUT_OF_DEVICE_MEMORY";
		case VK_ERROR_INITIALIZATION_FAILED:
			return "VK_ERROR_INITIALIZATION_FAILED";
		case VK_ERROR_DEVICE_LOST:
			return "VK_ERROR_DEVICE_LOST";
		case VK_ERROR_MEMORY_MAP_FAILED:
			return "VK_ERROR_MEMORY_MAP_FAILED";
		case VK_ERROR_LAYER_NOT_PRESENT:
			return "VK_ERROR_LAYER_NOT_PRESENT";
		case VK_ERROR_EXTENSION_NOT_PRESENT:
			return "VK_ERROR_EXTENSION_NOT_PRESENT";
		case VK_ERROR_FEATURE_NOT_PRESENT:
			return "VK_ERROR_FEATURE_NOT_PRESENT";
		case VK_ERROR_INCOMPATIBLE_DRIVER:
			return "VK_ERROR_INCOMPATIBLE_DRIVER";
		case VK_ERROR_TOO_MANY_OBJECTS:
			return "VK_ERROR_TOO_MANY_OBJECTS";
		case VK_ERROR_FORMAT_NOT_SUPPORTED:
			return "VK_ERROR_FORMAT_NOT_SUPPORTED";
		case VK_ERROR_FRAGMENTED_POOL:
			return "VK_ERROR_FRAGMENTED_POOL";
		case VK_ERROR_OUT_OF_POOL_MEMORY:
			return "VK_ERROR_OUT_OF_POOL_MEMORY";
		case VK_ERROR_INVALID_EXTERNAL_HANDLE:
			return "VK_ERROR_INVALID_EXTERNAL_HANDLE";
		case VK_ERROR_SURFACE_LOST_KHR:
			return "VK_ERROR_SURFACE_LOST_KHR";
		case VK_ERROR_NATIVE_WINDOW_IN_USE_KHR:
			return "VK_ERROR_NATIVE_WINDOW_IN_USE_KHR";
		case VK_SUBOPTIMAL_KHR:
			return "VK_SUBOPTIMAL_KHR";
		case VK_ERROR_OUT_OF_DATE_KHR:
			return "VK_ERROR_OUT_OF_DATE_KHR";
		case VK_ERROR_INCOMPATIBLE_DISPLAY_KHR:
			return "VK_ERROR_INCOMPATIBLE_DISPLAY_KHR";
		case VK_ERROR_VALIDATION_FAILED_EXT:
			return "VK_ERROR_VALIDATION_FAILED_EXT";
		case VK_ERROR_INVALID_SHADER_NV:
			return "VK_ERROR_INVALID_SHADER_NV";
		case VK_ERROR_IMAGE_USAGE_NOT_SUPPORTED_KHR:
			return "VK_ERROR_IMAGE_USAGE_NOT_SUPPORTED_KHR";
		case VK_ERROR_VIDEO_PICTURE_LAYOUT_NOT_SUPPORTED_KHR:
			return "VK_ERROR_VIDEO_PICTURE_LAYOUT_NOT_SUPPORTED_KHR";
		case VK_ERROR_VIDEO_PROFILE_OPERATION_NOT_SUPPORTED_KHR:
			return "VK_ERROR_VIDEO_PROFILE_OPERATION_NOT_SUPPORTED_KHR";
		case VK_ERROR_VIDEO_PROFILE_FORMAT_NOT_SUPPORTED_KHR:
			return "VK_ERROR_VIDEO_PROFILE_FORMAT_NOT_SUPPORTED_KHR";
		case VK_ERROR_VIDEO_PROFILE_CODEC_NOT_SUPPORTED_KHR:
			return "VK_ERROR_VIDEO_PROFILE_CODEC_NOT_SUPPORTED_KHR";
		case VK_ERROR_VIDEO_STD_VERSION_NOT_SUPPORTED_KHR:
			return "VK_ERROR_VIDEO_STD_VERSION_NOT_SUPPORTED_KHR";
		case VK_ERROR_INVALID_DRM_FORMAT_MODIFIER_PLANE_LAYOUT_EXT:
			return "VK_ERROR_INVALID_DRM_FORMAT_MODIFIER_PLANE_LAYOUT_EXT";
		case VK_ERROR_FRAGMENTATION:
			return "VK_ERROR_FRAGMENTATION";
		case VK_ERROR_NOT_PERMITTED_KHR:
			return "VK_ERROR_NOT_PERMITTED_KHR";
		case VK_ERROR_INVALID_OPAQUE_CAPTURE_ADDRESS:
			return "VK_ERROR_INVALID_OPAQUE_CAPTURE_ADDRESS";
		case VK_ERROR_FULL_SCREEN_EXCLUSIVE_MODE_LOST_EXT:
			return "VK_ERROR_FULL_SCREEN_EXCLUSIVE_MODE_LOST_EXT";
		case VK_ERROR_UNKNOWN:
			return "VK_ERROR_UNKNOWN";
		case VK_PIPELINE_COMPILE_REQUIRED:
			return "VK_PIPELINE_COMPILE_REQUIRED";
		case VK_THREAD_IDLE_KHR:
			return "VK_THREAD_IDLE_KHR";
		case VK_THREAD_DONE_KHR:
			return "VK_THREAD_DONE_KHR";
		case VK_OPERATION_DEFERRED_KHR:
			return "VK_OPERATION_DEFERRED_KHR";
		case VK_OPERATION_NOT_DEFERRED_KHR:
			return "VK_OPERATION_NOT_DEFERRED_KHR";
		case VK_ERROR_INVALID_VIDEO_STD_PARAMETERS_KHR:
			return "VK_ERROR_INVALID_VIDEO_STD_PARAMETERS_KHR";
		case VK_ERROR_COMPRESSION_EXHAUSTED_EXT:
			return "VK_ERROR_COMPRESSION_EXHAUSTED_EXT";
		case VK_ERROR_INCOMPATIBLE_SHADER_BINARY_EXT:
			return "VK_ERROR_INCOMPATIBLE_SHADER_BINARY_EXT";
		case VK_PIPELINE_BINARY_MISSING_KHR:
			return "VK_PIPELINE_BINARY_MISSING_KHR";
		case VK_ERROR_NOT_ENOUGH_SPACE_KHR:
			return "VK_ERROR_NOT_ENOUGH_SPACE_KHR";
		case VK_RESULT_MAX_ENUM:
			break;
	}

	return (vxr::std::stringbuilder<char>() << "Unknown VkResult: " << code).str();
}

template <typename... Args>
inline static void debugLabelBegin([[maybe_unused]] VkQueue q, [[maybe_unused]] const char* fmt, [[maybe_unused]] Args... args) {
#ifndef NDEBUG
	VkDebugUtilsLabelEXT nameInfo = {};
	nameInfo.sType = VK_STRUCTURE_TYPE_DEBUG_UTILS_LABEL_EXT;

	vxr::std::vector<char> buf;	 // NOLINT(misc-const-correctness)

	if constexpr (sizeof...(Args) > 0) {
		const int n = snprintf(nullptr, 0, fmt, args...) + 1;
		buf.resize(n);
		snprintf(buf.get(), n, fmt, args...);
		nameInfo.pLabelName = buf.get();
	} else {
		nameInfo.pLabelName = fmt;
	}

	VK_DEBUG_PROC(vkQueueBeginDebugUtilsLabelEXT)(q, &nameInfo);
#endif
}
inline static void debugLabelEnd([[maybe_unused]] VkQueue q) {
#ifndef NDEBUG
	VK_DEBUG_PROC(vkQueueEndDebugUtilsLabelEXT)(q);
#endif
}

template <typename... Args>
inline static void debugLabelBegin([[maybe_unused]] VkCommandBuffer cb, [[maybe_unused]] const char* fmt, [[maybe_unused]] Args... args) {
#ifndef NDEBUG
	VkDebugUtilsLabelEXT nameInfo = {};
	nameInfo.sType = VK_STRUCTURE_TYPE_DEBUG_UTILS_LABEL_EXT;

	vxr::std::vector<char> buf;	 // NOLINT(misc-const-correctness)

	if constexpr (sizeof...(Args) > 0) {
		int n = snprintf(nullptr, 0, fmt, args...) + 1;
		buf.resize(n);
		snprintf(buf.get(), n, fmt, args...);
		nameInfo.pLabelName = buf.get();
	} else {
		nameInfo.pLabelName = fmt;
	}

	VK_DEBUG_PROC(vkCmdBeginDebugUtilsLabelEXT)(cb, &nameInfo);
#endif
}
inline static void debugLabelEnd([[maybe_unused]] VkCommandBuffer cb) {
#ifndef NDEBUG
	VK_DEBUG_PROC(vkCmdEndDebugUtilsLabelEXT)(cb);
#endif
}

template <typename... Args>
inline static void debugLabel([[maybe_unused]] VkDevice vkDevice, [[maybe_unused]] VkObjectType type,
							  [[maybe_unused]] uint64_t handle, [[maybe_unused]] const char* fmt, [[maybe_unused]] Args... args) {
#ifndef NDEBUG
	VkDebugUtilsObjectNameInfoEXT nameInfo = {};
	nameInfo.sType = VK_STRUCTURE_TYPE_DEBUG_UTILS_OBJECT_NAME_INFO_EXT;
	nameInfo.objectType = type;
	nameInfo.objectHandle = handle;

	vxr::std::vector<char> buf;	 // NOLINT(misc-const-correctness)

	if constexpr (sizeof...(Args) > 0) {
		const int n = snprintf(nullptr, 0, fmt, args...) + 1;
		buf.resize(n);
		snprintf(buf.get(), n, fmt, args...);
		nameInfo.pObjectName = buf.get();
	} else {
		nameInfo.pObjectName = fmt;
	}

	const VkResult ret = VK_DEBUG_PROC(vkSetDebugUtilsObjectNameEXT)(vkDevice, &nameInfo);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to debug label: %s", vkResultStr(ret).cStr());
		vxr::std::abort();
	}
#endif
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkQueue queue, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_QUEUE, reinterpret_cast<uint64_t>(queue), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkFence fence, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_FENCE, reinterpret_cast<uint64_t>(fence), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkCommandPool pool, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_COMMAND_POOL, reinterpret_cast<uint64_t>(pool), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkBuffer buffer, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_BUFFER, reinterpret_cast<uint64_t>(buffer), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkImage image, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_IMAGE, reinterpret_cast<uint64_t>(image), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkImageView view, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_IMAGE_VIEW, reinterpret_cast<uint64_t>(view), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkSampler sampler, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_SAMPLER, reinterpret_cast<uint64_t>(sampler), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkSemaphore semaphore, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_SEMAPHORE, reinterpret_cast<uint64_t>(semaphore), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkDescriptorSetLayout layout, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_DESCRIPTOR_SET_LAYOUT, reinterpret_cast<uint64_t>(layout), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkDescriptorPool pool, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_DESCRIPTOR_POOL, reinterpret_cast<uint64_t>(pool), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkDescriptorSet set, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_DESCRIPTOR_SET, reinterpret_cast<uint64_t>(set), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkPipelineLayout layout, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_PIPELINE_LAYOUT, reinterpret_cast<uint64_t>(layout), fmt, args...);
}

template <typename... Args>
inline static void debugLabel(VkDevice vkDevice, VkPipeline pipeline, const char* fmt, Args... args) {
	debugLabel(vkDevice, VK_OBJECT_TYPE_PIPELINE, reinterpret_cast<uint64_t>(pipeline), fmt, args...);
}
}  // namespace vxr::vk
