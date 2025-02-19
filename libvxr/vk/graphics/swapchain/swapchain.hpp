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

#include "vxr/vxr.h"

#include <stddef.h>
#include <stdint.h>

#include "std/vector.hpp"
#include "std/utility.hpp"

namespace vxr::vk {

struct instance;

namespace graphics {
struct swapchain {
	VkExtent2D extent = {};
	VkSurfaceFormatKHR surfaceFormat = {};
	VkSwapchainKHR vkSwapchain = VK_NULL_HANDLE;
	vxr::std::vector<vxr::std::pair<VkImage, VkImageView>> images;

	[[nodiscard]] size_t size() const noexcept { return this->images.size(); }
};

VkResult initSwapchain(instance*, VkSurfaceKHR, uint32_t);
void destroySwapchain(instance*);
}  // namespace graphics
}  // namespace vxr::vk
