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

#include <stddef.h>
#include <stdint.h>

#include "std/utility.hpp"
#include "std/vector.hpp"
#include "std/ringbuffer.hpp"

#include "vxr/vxr.h"
#include "vk/device/vma/vma.hpp"
#include "vk/graphics/swapchain/swapchain.hpp"

namespace vxr::vk {
struct instance;
namespace graphics {
struct frame {
	frame() noexcept = delete;
	frame(frame&) = delete;
	frame& operator=(const frame&) = delete;

	frame(vxr::vk::instance*, size_t, const char*) noexcept;
	~frame() noexcept;

	VkDevice vkDevice;

	uint32_t imageIndex;
	VkSemaphore surfaceAcquireSemaphore;
	VkSemaphore surfaceReleaseSemaphore;
	VkFence fence;

	VmaAllocator vmaAllocator;
	VmaPool vmaPool;
	vxr::std::vector<vxr::std::pair<VkBuffer, VmaAllocation>> pendingScratchBuffers;

	VkCommandPool vkCommandPool;
	size_t allocatedCommandBuffers;
	vxr::std::ringbuffer<VkCommandBuffer> freeCommandBuffers;
	vxr::std::vector<VkCommandBuffer> pendingCommandBuffers;

	[[nodiscard]] vxr_vk_graphics_frame handle() noexcept { return reinterpret_cast<vxr_vk_graphics_frame>(this); }
	[[nodiscard]] static frame* fromHandle(vxr_vk_graphics_frame handle) noexcept {
		return reinterpret_cast<frame*>(handle);
	}
};

struct system {
	struct swapchain swapchain;
};
}  // namespace graphics
}  // namespace vxr::vk
