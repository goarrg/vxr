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

#include "std/stdlib.hpp"
#include "std/log.hpp"

#include "vk/vk.hpp"
#include "vk/device/device.hpp"

static void bindIndexBuffer(VkCommandBuffer cb, vxr_vk_graphics_indexBufferInfo info) {
	VK_PROC_DEVICE(vkCmdBindIndexBuffer)(cb, info.vkBuffer, info.offset, info.indexType);
}
static void bindIndexBufferMaint5(VkCommandBuffer cb, vxr_vk_graphics_indexBufferInfo info) {
	VK_TRY_PROC_DEVICE(vkCmdBindIndexBuffer2KHR)(cb, info.vkBuffer, info.offset, info.size, info.indexType);
}
static void bindIndexBufferVK14(VkCommandBuffer cb, vxr_vk_graphics_indexBufferInfo info) {
	VK_TRY_PROC_DEVICE(vkCmdBindIndexBuffer2)(cb, info.vkBuffer, info.offset, info.size, info.indexType);
}

inline static void setupFNTable(vxr::vk::instance* instance) {
	instance->device.fnTable = {};

#define BEGIN_LOAD(var) auto& ptr = (var);
#define TRY_LOAD(test, fn)                                           \
	if ((ptr == nullptr) && (VK_TRY_PROC_DEVICE(test) != nullptr)) { \
		vxr::std::iPrintf("FN Table Using: " #fn);                   \
		ptr = fn;                                                    \
	}
#define END_LOAD(name)                                                    \
	if ((ptr == nullptr)) {                                               \
		vxr::std::ePrintf("Failed to init function pointer for: " #name); \
		vxr::std::abort();                                                \
	}

	{
		BEGIN_LOAD(instance->device.fnTable.bindIndexBuffer)
		TRY_LOAD(vkCmdBindIndexBuffer2, bindIndexBufferVK14)
		TRY_LOAD(vkCmdBindIndexBuffer2KHR, bindIndexBufferMaint5)
		TRY_LOAD(vkCmdBindIndexBuffer, bindIndexBuffer)
		END_LOAD(bindIndexBuffer)
	}

#undef END_LOAD
#undef TRY_LOAD
#undef BEGIN_LOAD
}
