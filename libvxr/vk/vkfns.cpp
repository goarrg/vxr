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

#include "std/stdlib.hpp"
#include "std/log.hpp"

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/vkfns.hpp"

namespace vxr::vk {
#undef VK_PROC
#define VK_PROC(FN) PFN_##FN p##FN;

#ifndef NDEBUG
#undef VK_DEBUG_PROC
#define VK_DEBUG_PROC(FN) PFN_##FN p##FN;
#endif

#include "vkfns.inc"  // IWYU pragma: keep

void initFns(instance* instance, PFN_vkGetInstanceProcAddr procAddr) {
	bool ok = true;

#undef VK_PROC
#define VK_PROC(FN)                                        \
	p##FN = (PFN_##FN)procAddr(instance->vkInstance, #FN); \
	if (!p##FN) {                                          \
		ok = false;                                        \
		vxr::std::ePrintf("[vkfn] Failed to find " #FN);   \
	}

#ifndef NDEBUG
#undef VK_DEBUG_PROC
#define VK_DEBUG_PROC(FN)                                  \
	p##FN = (PFN_##FN)procAddr(instance->vkInstance, #FN); \
	if (!p##FN) {                                          \
		ok = false;                                        \
		vxr::std::ePrintf("[vkfn] Failed to find " #FN);   \
	}
#endif

#include "vkfns.inc"  // IWYU pragma: keep

	if (!ok) {
		vxr::std::ePrintf("[vkfn] Failed to find all required functions");
		vxr::std::abortPopup(vxr::std::sourceLocation::current(),
							 "Incompatible vulkan runtime.\n"
							 "Ensure your GPU and drivers meet the minimum requirements to run this software.");
	}
}
}  // namespace vxr::vk
