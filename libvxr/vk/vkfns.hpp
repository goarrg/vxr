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

#include "vk/vk.hpp"

namespace vxr::vk {
#define VK_PROC(FN) extern PFN_##FN p##FN;

#ifndef NDEBUG
#define VK_DEBUG_PROC(FN) extern PFN_##FN p##FN;
#else
#define VK_DEBUG_PROC(FN)
#endif

#include "vkfns.inc"  // IWYU pragma: keep

#undef VK_PROC
#define VK_PROC(FN) ::vxr::vk::p##FN

#ifndef NDEBUG
#undef VK_DEBUG_PROC
#define VK_DEBUG_PROC(FN) ::vxr::vk::p##FN
#endif

extern void initFns(instance*, PFN_vkGetInstanceProcAddr);
}  // namespace vxr::vk
