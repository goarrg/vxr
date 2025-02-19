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
#include "vk/vkfns.hpp"
#include "vk/device/device.hpp"

namespace vxr::vk::device {
#undef VK_PROC_DEVICE
#define VK_PROC_DEVICE(FN) PFN_##FN p##FN;

#undef VK_TRY_PROC_DEVICE
#define VK_TRY_PROC_DEVICE(FN) PFN_##FN p##FN;

#ifndef NDEBUG
#undef VK_DEBUG_PROC_DEVICE
#define VK_DEBUG_PROC_DEVICE(FN) PFN_##FN p##FN;
#endif

#include "device_vkfns.inc"	 // IWYU pragma: keep
}  // namespace vxr::vk::device

inline static void setupVKFNs(vxr::vk::instance* instance) {
	bool ok = true;

#undef VK_PROC_DEVICE
#define VK_PROC_DEVICE(FN)                                                                             \
	::vxr::vk::device::p##FN = (PFN_##FN)VK_PROC(vkGetDeviceProcAddr)(instance->device.vkDevice, #FN); \
	if (!::vxr::vk::device::p##FN) {                                                                   \
		vxr::std::ePrintf("[device_vkfn] Failed to find: " #FN);                                       \
		ok = false;                                                                                    \
	}

#undef VK_TRY_PROC_DEVICE
#define VK_TRY_PROC_DEVICE(FN) \
	::vxr::vk::device::p##FN = (PFN_##FN)VK_PROC(vkGetDeviceProcAddr)(instance->device.vkDevice, #FN);

#ifndef NDEBUG
#undef VK_DEBUG_PROC_DEVICE
#define VK_DEBUG_PROC_DEVICE(FN)                                                                       \
	::vxr::vk::device::p##FN = (PFN_##FN)VK_PROC(vkGetDeviceProcAddr)(instance->device.vkDevice, #FN); \
	if (!::vxr::vk::device::p##FN) {                                                                   \
		vxr::std::ePrintf("[device_vkfn] Failed to find: " #FN);                                       \
		ok = false;                                                                                    \
	}
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

	if (!ok) {
		vxr::std::ePrintf("[device_vkfn] Failed to find all required functions");
		vxr::std::abortPopup(vxr::std::sourceLocation::current(),
							 "Incompatible vulkan runtime.\n"
							 "Ensure your GPU and drivers meet the minimum requirements to run this software.");
	}
}
