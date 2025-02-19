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
#include <stddef.h>
#include <string.h>
#include <new>

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/array.hpp"
#include "std/vector.hpp"
#include "std/time.hpp"
#include "std/utility.hpp"
#include "std/string.hpp"

#include "vk/vk.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"
#include "vk/device/vma/vma.hpp"
#include "vk/graphics/graphics.hpp"
#include "vk/graphics/swapchain/swapchain.hpp"

namespace vxr::vk::graphics {
frame::frame(vxr::vk::instance* instance, size_t nameSz, const char* name) noexcept
	: vkDevice(instance->device.vkDevice) {
	{
		VkSemaphoreCreateInfo semaphoreInfo = {};
		semaphoreInfo.sType = VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO;

		VkResult ret = VK_PROC_DEVICE(vkCreateSemaphore)(instance->device.vkDevice, &semaphoreInfo, nullptr, &this->surfaceAcquireSemaphore);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to create semaphore: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}
		vxr::std::debugRun([=, this]() {
			vxr::std::stringbuilder builder;
			builder.write("semaphore_binary_surface_acquire_frame_").write(nameSz, name);
			vxr::vk::debugLabel(instance->device.vkDevice, this->surfaceAcquireSemaphore, builder.cStr());
		});
		ret = VK_PROC_DEVICE(vkCreateSemaphore)(instance->device.vkDevice, &semaphoreInfo, nullptr, &this->surfaceReleaseSemaphore);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to create semaphore: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}
		vxr::std::debugRun([=, this]() {
			vxr::std::stringbuilder builder;
			builder.write("semaphore_binary_surface_release_frame_").write(nameSz, name);
			vxr::vk::debugLabel(instance->device.vkDevice, this->surfaceReleaseSemaphore, builder.cStr());
		});
	}

	{
		VkFenceCreateInfo fenceInfo = {};
		fenceInfo.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO;
		fenceInfo.flags = VK_FENCE_CREATE_SIGNALED_BIT;

		const VkResult ret = VK_PROC_DEVICE(vkCreateFence)(instance->device.vkDevice, &fenceInfo, nullptr, &this->fence);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to create fence: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}
		vxr::std::debugRun([=, this]() {
			vxr::std::stringbuilder builder;
			builder.write("graphics_fence_frame_").write(nameSz, name);
			vxr::vk::debugLabel(instance->device.vkDevice, this->fence, builder.cStr());
		});
	}

	{
		this->vmaAllocator = instance->device.vma.allocator;

		VkBufferCreateInfo bufferCreateInfo = {};
		bufferCreateInfo.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
		// bufferCreateInfo.size = 1024;
		bufferCreateInfo.usage = VK_BUFFER_USAGE_TRANSFER_SRC_BIT | VK_BUFFER_USAGE_TRANSFER_DST_BIT;

		VmaAllocationCreateInfo allocCreateInfo = {};
		allocCreateInfo.usage = VMA_MEMORY_USAGE_AUTO_PREFER_HOST;
		allocCreateInfo.flags = VMA_ALLOCATION_CREATE_HOST_ACCESS_SEQUENTIAL_WRITE_BIT;
		allocCreateInfo.requiredFlags = VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_CACHED_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT;
		allocCreateInfo.memoryTypeBits = instance->device.vma.noBARMemoryTypeBits;

		uint32_t memTypeIndex;
		VkResult ret = vmaFindMemoryTypeIndexForBufferInfo(this->vmaAllocator, &bufferCreateInfo, &allocCreateInfo, &memTypeIndex);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to find host memory type for scratch buffers: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}

		VmaPoolCreateInfo poolCreateInfo = {};
		poolCreateInfo.memoryTypeIndex = memTypeIndex;
		poolCreateInfo.flags = VMA_POOL_CREATE_IGNORE_BUFFER_IMAGE_GRANULARITY_BIT | VMA_POOL_CREATE_LINEAR_ALGORITHM_BIT;
		poolCreateInfo.minBlockCount = 0;
		poolCreateInfo.maxBlockCount = 0;
		poolCreateInfo.blockSize = 0;

		ret = vmaCreatePool(this->vmaAllocator, &poolCreateInfo, &this->vmaPool);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to create VmaPool: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}
		vxr::std::debugRun([=, this]() {
			vxr::std::stringbuilder builder;
			builder.write("graphics_pool_frame_").write(nameSz, name);
			vmaSetPoolName(this->vmaAllocator, this->vmaPool, builder.cStr());
		});
	}

	{
		VkCommandPoolCreateInfo poolInfo = {};
		poolInfo.sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO;
		poolInfo.queueFamilyIndex = instance->device.graphicsQueue.family;
		poolInfo.flags = VK_COMMAND_POOL_CREATE_TRANSIENT_BIT;

		const VkResult ret = VK_PROC_DEVICE(vkCreateCommandPool)(instance->device.vkDevice, &poolInfo, nullptr, &this->vkCommandPool);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to create graphics commandpool: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}
		vxr::std::debugRun([=, this]() {
			vxr::std::stringbuilder builder;
			builder.write("graphics_cmd_pool_frame_").write(nameSz, name);
			vxr::vk::debugLabel(instance->device.vkDevice, this->vkCommandPool, builder.cStr());
		});
		this->allocatedCommandBuffers = 0;
	}
}

frame::~frame() noexcept {
	while (this->freeCommandBuffers.size() != 0u) {
		auto* cb = this->freeCommandBuffers.popFront();
		VK_PROC_DEVICE(vkFreeCommandBuffers)(this->vkDevice, this->vkCommandPool, 1, &cb);
	}
	for (auto& cb : this->pendingCommandBuffers) {
		VK_PROC_DEVICE(vkFreeCommandBuffers)(this->vkDevice, this->vkCommandPool, 1, &cb);
	}
	for (auto& b : this->pendingScratchBuffers) {
		vmaUnmapMemory(this->vmaAllocator, b.second);
		vmaDestroyBuffer(this->vmaAllocator, b.first, b.second);
	}

	this->allocatedCommandBuffers = 0;

	VK_PROC_DEVICE(vkDestroySemaphore)(this->vkDevice, this->surfaceAcquireSemaphore, nullptr);
	VK_PROC_DEVICE(vkDestroySemaphore)(this->vkDevice, this->surfaceReleaseSemaphore, nullptr);
	VK_PROC_DEVICE(vkDestroyFence)(this->vkDevice, this->fence, nullptr);
	VK_PROC_DEVICE(vkDestroyCommandPool)(this->vkDevice, this->vkCommandPool, nullptr);

	vmaDestroyPool(this->vmaAllocator, this->vmaPool);
}
}  // namespace vxr::vk::graphics

extern "C" {
VXR_FN void vxr_vk_graphics_createFrame(vxr_vk_instance instanceHandle, size_t nameSz, const char* name, vxr_vk_graphics_frame* frameHandle) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	// auto* graphics = &instance->graphics;
	auto* frame = new (::std::nothrow) vxr::vk::graphics::frame(instance, nameSz, name);
	*frameHandle = frame->handle();
}
VXR_FN void vxr_vk_graphics_destroyFrame(vxr_vk_graphics_frame frameHandle) {
	auto* frame = vxr::vk::graphics::frame::fromHandle(frameHandle);
	delete frame;
}
VXR_FN void vxr_vk_graphics_frame_begin(vxr_vk_instance instanceHandle, size_t nameSz, const char* name, vxr_vk_graphics_frame frameHandle) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	// auto* graphics = &instance->graphics;
	auto* frame = vxr::vk::graphics::frame::fromHandle(frameHandle);

	{
		const VkResult ret = VK_PROC_DEVICE(vkResetCommandPool)(instance->device.vkDevice, frame->vkCommandPool, 0);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to reset graphics command pool: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}

		for (auto& cb : frame->pendingCommandBuffers) {
			frame->freeCommandBuffers.pushBack(cb);
		}
		frame->pendingCommandBuffers.resize(0);

		if (frame->freeCommandBuffers.size() != frame->allocatedCommandBuffers) {
			vxr::std::ePrintf("Allocated %zu command buffers but submitted %zu", frame->allocatedCommandBuffers,
							  frame->freeCommandBuffers.size());
			vxr::std::abort();
		}
	}

	{
		for (auto& b : frame->pendingScratchBuffers) {
			vmaUnmapMemory(instance->device.vma.allocator, b.second);
			vmaDestroyBuffer(instance->device.vma.allocator, b.first, b.second);
		}
		frame->pendingScratchBuffers.resize(0);
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder builder;
		builder.write("graphics_").write(nameSz, name);
		vxr::vk::debugLabelBegin(instance->device.graphicsQueue.vkQueue, builder.cStr());
	});
}
VXR_FN void vxr_vk_graphics_frame_cancel(vxr_vk_instance instanceHandle, vxr_vk_graphics_frame frameHandle) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	auto* graphics = &instance->graphics;
	auto* frame = vxr::vk::graphics::frame::fromHandle(frameHandle);

	vxr::std::abort("DO NOT USE");

	const vxr::std::array indices = {
		frame->imageIndex,
	};
	const VkReleaseSwapchainImagesInfoEXT releaseInfo = {
		.sType = VK_STRUCTURE_TYPE_RELEASE_SWAPCHAIN_IMAGES_INFO_EXT,
		.swapchain = graphics->swapchain.vkSwapchain,
		.imageIndexCount = indices.size(),
		.pImageIndices = indices.get(),
	};

	const VkResult ret = VK_TRY_PROC_DEVICE(vkReleaseSwapchainImagesEXT)(instance->device.vkDevice, &releaseInfo);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to release image: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}
	vxr::vk::debugLabelEnd(instance->device.graphicsQueue.vkQueue);
}
VXR_FN void vxr_vk_graphics_frame_end(vxr_vk_instance instanceHandle, vxr_vk_graphics_frame) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	// auto* graphics = &instance->graphics;
	// auto* frame = vxr::vk::graphics::frame::fromHandle(frameHandle);

	vxr::vk::debugLabelEnd(instance->device.graphicsQueue.vkQueue);
}
VXR_FN VkResult vxr_vk_graphics_frame_acquireSurface(vxr_vk_instance instanceHandle, vxr_vk_graphics_frame frameHandle, vxr_vk_surface* surface) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	auto* graphics = &instance->graphics;
	auto* frame = vxr::vk::graphics::frame::fromHandle(frameHandle);

	if (graphics->swapchain.vkSwapchain == VK_NULL_HANDLE) {
		return VK_ERROR_SURFACE_LOST_KHR;
	}

	{
		const VkResult ret = VK_PROC_DEVICE(vkAcquireNextImageKHR)(
			instance->device.vkDevice, graphics->swapchain.vkSwapchain, vxr::std::time::second,
			frame->surfaceAcquireSemaphore, VK_NULL_HANDLE, &frame->imageIndex);
		switch (ret) {
			case VK_SUCCESS:
			case VK_SUBOPTIMAL_KHR:
				break;

			case VK_ERROR_OUT_OF_DATE_KHR:
			case VK_ERROR_SURFACE_LOST_KHR:
				return ret;

			default:
				vxr::std::ePrintf("Failed to acquire image: %s", vxr::vk::vkResultStr(ret).cStr());
				vxr::std::abort();
				break;
		}
	}

	{
		vxr_vk_graphics_getSurfaceInfo(instanceHandle, &surface->info);
		surface->vkImage = graphics->swapchain.images[frame->imageIndex].first;
		surface->vkImageView = graphics->swapchain.images[frame->imageIndex].second;
		surface->acquireSemaphore = frame->surfaceAcquireSemaphore;
		surface->releaseSemaphore = frame->surfaceReleaseSemaphore;
	}

	return VK_SUCCESS;
}
VXR_FN void vxr_vk_graphics_frame_createHostScratchBuffer(vxr_vk_instance instanceHandle, vxr_vk_graphics_frame frameHandle, size_t nameSz,
														  const char* name, vxr_vk_bufferCreateInfo info, vxr_vk_hostBuffer* b) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	// auto* graphics = &instance->graphics;
	auto* frame = vxr::vk::graphics::frame::fromHandle(frameHandle);

	VkBufferCreateInfo bufferInfo = {};
	bufferInfo.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
	bufferInfo.size = info.size;
	bufferInfo.usage = info.usage;
	bufferInfo.sharingMode = VK_SHARING_MODE_EXCLUSIVE;

	VmaAllocationCreateInfo allocCreateInfo = {};
	allocCreateInfo.usage = VMA_MEMORY_USAGE_AUTO_PREFER_HOST;
	allocCreateInfo.flags = VMA_ALLOCATION_CREATE_HOST_ACCESS_SEQUENTIAL_WRITE_BIT;
	allocCreateInfo.requiredFlags = VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT;
	allocCreateInfo.memoryTypeBits = instance->device.vma.noBARMemoryTypeBits;
	allocCreateInfo.pool = frame->vmaPool;

	VkResult ret = vmaCreateBuffer(instance->device.vma.allocator, &bufferInfo, &allocCreateInfo, &b->vkBuffer,
								   reinterpret_cast<VmaAllocation*>(&b->allocation), nullptr);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create buffer: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	frame->pendingScratchBuffers.pushBack(vxr::std::pair{b->vkBuffer, reinterpret_cast<VmaAllocation>(b->allocation)});
	ret = vmaMapMemory(instance->device.vma.allocator, reinterpret_cast<VmaAllocation>(b->allocation), &b->ptr);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to map buffer: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder builder;
		builder.write("buffer_hostScratch_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, b->vkBuffer, builder.cStr());

		builder.write("_allocation");
		vmaSetAllocationName(instance->device.vma.allocator, reinterpret_cast<VmaAllocation>(b->allocation), builder.cStr());
	});
}
VXR_FN VkResult vxr_vk_graphics_frame_submit(vxr_vk_instance instanceHandle, vxr_vk_graphics_frame frameHandle) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	auto* graphics = &instance->graphics;
	auto* frame = vxr::vk::graphics::frame::fromHandle(frameHandle);

	VkResult status = VK_SUCCESS;
	{
		VkSwapchainPresentFenceInfoEXT presentFenceInfo = {};
		presentFenceInfo.sType = VK_STRUCTURE_TYPE_SWAPCHAIN_PRESENT_FENCE_INFO_EXT;

		VkPresentInfoKHR presentInfo = {};
		presentInfo.sType = VK_STRUCTURE_TYPE_PRESENT_INFO_KHR;
		presentInfo.pNext = &presentFenceInfo;

		const vxr::std::array waitSemaphores = {
			frame->surfaceReleaseSemaphore,
		};
		presentInfo.waitSemaphoreCount = waitSemaphores.size();
		presentInfo.pWaitSemaphores = waitSemaphores.get();

		const vxr::std::array swapchains = {
			graphics->swapchain.vkSwapchain,
		};
		const vxr::std::array<uint32_t, swapchains.size()> indices = {
			frame->imageIndex,
		};
		const vxr::std::array<VkFence, swapchains.size()> fences = {
			frame->fence,
		};
		presentFenceInfo.swapchainCount = presentInfo.swapchainCount = swapchains.size();
		presentInfo.pSwapchains = swapchains.get();
		presentInfo.pImageIndices = indices.get();
		presentFenceInfo.pFences = fences.get();

		{
			const VkResult ret = VK_PROC_DEVICE(vkResetFences)(
				instance->device.vkDevice, presentFenceInfo.swapchainCount, presentFenceInfo.pFences);
			if (ret != VK_SUCCESS) {
				vxr::std::ePrintf("Failed to reset fence on frame: %s", vxr::vk::vkResultStr(ret).cStr());
				vxr::std::abort();
			}
		}

		{
			const VkResult ret = VK_PROC_DEVICE(vkQueuePresentKHR)(instance->device.graphicsQueue.vkQueue, &presentInfo);
			switch (ret) {
				case VK_SUCCESS:
				case VK_SUBOPTIMAL_KHR:
				case VK_ERROR_OUT_OF_DATE_KHR:
					// case VK_ERROR_SURFACE_LOST_KHR:
					status = ret;
					break;

				default:
					vxr::std::ePrintf("Failed to present frame: %s", vxr::vk::vkResultStr(ret).cStr());
					vxr::std::abort();
					break;
			}
		}
	}

	return status;
}
VXR_FN void vxr_vk_graphics_frame_wait(vxr_vk_instance instanceHandle, vxr_vk_graphics_frame frameHandle) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	// auto* graphics = &instance->graphics;
	auto* frame = vxr::vk::graphics::frame::fromHandle(frameHandle);

	{
		const VkResult ret = VK_PROC_DEVICE(vkWaitForFences)(
			instance->device.vkDevice, 1, &frame->fence, VK_TRUE, vxr::std::time::second);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to wait on frame: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}
	}
}
}
