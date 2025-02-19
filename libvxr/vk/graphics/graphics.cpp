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

#include "vxr/vxr.h"  // IWYU pragma: associated

#include <stdint.h>

#include "std/utility.hpp"

#include "vk/vk.hpp"
#include "vk/graphics/graphics.hpp"
#include "vk/graphics/swapchain/swapchain.hpp"

extern "C" {
VXR_FN VkResult vxr_vk_graphics_init(vxr_vk_instance instanceHandle, uint64_t vkSurface, uint32_t wantNumImages) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	// auto* graphics = &instance->graphics;

	{
		const VkResult ret = vxr::vk::graphics::initSwapchain(
			instance, reinterpret_cast<VkSurfaceKHR>(vkSurface), wantNumImages);  // NOLINT(performance-no-int-to-ptr)
		if (ret != VK_SUCCESS) {
			return ret;
		}
	}

	return VK_SUCCESS;
}
VXR_FN void vxr_vk_graphics_getSurfaceInfo(vxr_vk_instance instanceHandle, vxr_vk_surfaceInfo* info) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	auto* graphics = &instance->graphics;

	info->format = graphics->swapchain.surfaceFormat.format;
	info->extent = graphics->swapchain.extent;
	// -1 to get useable count, TODO: handle mailbox
	info->numImages = graphics->swapchain.size() > 0 ? (vxr::std::max<uint32_t>(2, graphics->swapchain.size()) - 1) : 0;
}
VXR_FN void vxr_vk_graphics_destroy(vxr_vk_instance instanceHandle) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	// auto* graphics = &instance->graphics;

	vxr::vk::graphics::destroySwapchain(instance);
}
}
