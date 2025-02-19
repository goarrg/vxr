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

#include "vxr/vxr.h"
#include "vk/device/vma/vma.hpp"

namespace vxr::vk::device {

struct queue {
	uint32_t family = 0;
	uint32_t index = 0;
	VkQueue vkQueue = VK_NULL_HANDLE;
};

using bindIndexBuffer = void (*)(VkCommandBuffer, vxr_vk_graphics_indexBufferInfo);

struct instance {
	VkPhysicalDevice vkPhysicalDevice;
	VkDevice vkDevice;

	struct vma vma;

	struct queue computeQueue;
	struct queue graphicsQueue;
	struct queue transferQueue;

	vxr_vk_device_properties properties;
	// table of function pointers for functions that vary behaviour depending on features enabled
	// this is to not pay the cost of ifs
	struct {
		::vxr::vk::device::bindIndexBuffer bindIndexBuffer;
	} fnTable;
};

#define VK_PROC_DEVICE(FN) extern PFN_##FN p##FN;
#define VK_TRY_PROC_DEVICE(FN) extern PFN_##FN p##FN;

#ifndef NDEBUG
#define VK_DEBUG_PROC_DEVICE(FN) extern PFN_##FN p##FN;
#else
#define VK_DEBUG_PROC_DEVICE(FN)
#endif

#include "device_vkfns.inc"	 // IWYU pragma: keep

#undef VK_PROC_DEVICE
#define VK_PROC_DEVICE(FN) ::vxr::vk::device::p##FN

#undef VK_TRY_PROC_DEVICE
#define VK_TRY_PROC_DEVICE(FN) ::vxr::vk::device::p##FN

#ifndef NDEBUG
#undef VK_DEBUG_PROC_DEVICE
#define VK_DEBUG_PROC_DEVICE(FN) ::vxr::vk::device::p##FN
#endif
}  // namespace vxr::vk::device
