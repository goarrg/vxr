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

#include "vk/graphics/swapchain/swapchain.hpp"

#include <stddef.h>
#include <stdint.h>

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/array.hpp"
#include "std/vector.hpp"
#include "std/utility.hpp"
#include "std/string.hpp"

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/vkfns.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"

static constexpr vxr::std::array wantFormats = {
	vxr::std::pair(VK_FORMAT_B8G8R8A8_SRGB, VK_COLOR_SPACE_SRGB_NONLINEAR_KHR),
	vxr::std::pair(VK_FORMAT_R8G8B8A8_SRGB, VK_COLOR_SPACE_SRGB_NONLINEAR_KHR),
};

inline static bool findFormat(vxr::vk::graphics::swapchain* swapchain, vxr::std::vector<VkSurfaceFormatKHR>& surfaceFormats) {
	for (auto want : wantFormats) {
		for (size_t i = 0; i < surfaceFormats.size(); i++) {
			auto& surfaceFormat = surfaceFormats[i];
			if (surfaceFormat.format == want.first && surfaceFormat.colorSpace == want.second) {
				swapchain->surfaceFormat = surfaceFormat;
				vxr::std::iPrintf("Selected format: [%d]", i);
				return true;
			}
		}
	}

	return false;
}

inline static void printFormats(vxr::std::vector<VkSurfaceFormatKHR>& surfaceFormats) {
	vxr::std::stringbuilder builder;

	for (size_t i = 0; auto& surfaceFormat : surfaceFormats) {
		builder << "\n[" << i++ << "] ";
		builder.writef("Format: %d Color Space: %d", surfaceFormat.format, surfaceFormat.colorSpace);
	}

	vxr::std::iPrintf("Found surface formats: %s", builder.cStr());
}

#define HANDLE_SURFACE_ERROR(ret, fmt, ...)      \
	switch (ret) {                               \
		case VK_SUCCESS:                         \
			break;                               \
                                                 \
		case VK_ERROR_SURFACE_LOST_KHR:          \
			return ret;                          \
                                                 \
		default:                                 \
			vxr::std::ePrintf(fmt, __VA_ARGS__); \
			vxr::std::abort();                   \
			break;                               \
	}

namespace vxr::vk::graphics {
VkResult initSwapchain(instance* instance, VkSurfaceKHR surface, uint32_t wantNumImages) {
	auto* swapchain = &instance->graphics.swapchain;

	{
		for (auto& image : swapchain->images) {
			VK_PROC_DEVICE(vkDestroyImageView)(instance->device.vkDevice, image.second, nullptr);
		}
		swapchain->images.resize(0);
	}

	{
		VkBool32 presentSupport = VK_FALSE;
		const VkResult ret = VK_PROC(vkGetPhysicalDeviceSurfaceSupportKHR)(
			instance->device.vkPhysicalDevice, instance->device.graphicsQueue.family, surface, &presentSupport);
		if (ret != VK_SUCCESS) {
			return ret;
		}
		if (presentSupport == VK_FALSE) {
			return VK_ERROR_INCOMPATIBLE_DRIVER;
		}
	}

	{
		uint32_t numSurfaceFormats = 0;
		VkResult ret = VK_PROC(vkGetPhysicalDeviceSurfaceFormatsKHR)(
			instance->device.vkPhysicalDevice, surface, &numSurfaceFormats, nullptr);
		HANDLE_SURFACE_ERROR(ret, "Failed to get surface formats: %s", vxr::vk::vkResultStr(ret).cStr());

		vxr::std::vector<VkSurfaceFormatKHR> surfaceFormats(numSurfaceFormats);
		ret = VK_PROC(vkGetPhysicalDeviceSurfaceFormatsKHR)(
			instance->device.vkPhysicalDevice, surface, &numSurfaceFormats, surfaceFormats.get());
		HANDLE_SURFACE_ERROR(ret, "Failed to get surface formats: %s", vxr::vk::vkResultStr(ret).cStr());

		printFormats(surfaceFormats);
		if (!findFormat(swapchain, surfaceFormats)) {
			swapchain->surfaceFormat = surfaceFormats[0];
			vxr::std::wPrintf("No known surface formats found");
			vxr::std::iPrintf("Selected format: [0]");
		}
	}

	VkSurfaceCapabilitiesKHR surfaceCapabilities;
	VkPresentModeKHR presentMode = VK_PRESENT_MODE_FIFO_KHR;
	vxr::std::vector<VkPresentModeKHR> compatiblePresentModes;

	{
		uint32_t numPresentModes = 0;
		VkResult ret = VK_PROC(vkGetPhysicalDeviceSurfacePresentModesKHR)(
			instance->device.vkPhysicalDevice, surface, &numPresentModes, nullptr);
		HANDLE_SURFACE_ERROR(ret, "Failed to get surface present modes: %s", vxr::vk::vkResultStr(ret).cStr());

		vxr::std::vector<VkPresentModeKHR> presentModes(numPresentModes);
		ret = VK_PROC(vkGetPhysicalDeviceSurfacePresentModesKHR)(
			instance->device.vkPhysicalDevice, surface, &numPresentModes, presentModes.get());
		HANDLE_SURFACE_ERROR(ret, "Failed to get surface present modes: %s", vxr::vk::vkResultStr(ret).cStr());

		for (uint32_t i = 0; i < numPresentModes; i++) {
			if (presentModes[i] == VK_PRESENT_MODE_FIFO_RELAXED_KHR) {
				presentMode = presentModes[i];
				break;
			}
		}
	}

	{
		const VkSurfacePresentModeEXT surfacePresentModeInfo = {
			.sType = VK_STRUCTURE_TYPE_SURFACE_PRESENT_MODE_EXT,
			.presentMode = presentMode,
		};
		const VkPhysicalDeviceSurfaceInfo2KHR surfaceInfo2 = {
			.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SURFACE_INFO_2_KHR,
			.pNext = &surfacePresentModeInfo,
			.surface = surface,
		};

		VkSurfacePresentModeCompatibilityEXT surfacePresentModeCompatibilityInfo = {
			.sType = VK_STRUCTURE_TYPE_SURFACE_PRESENT_MODE_COMPATIBILITY_EXT,
		};
		VkSurfaceCapabilities2KHR surfaceCapabilities2 = {
			.sType = VK_STRUCTURE_TYPE_SURFACE_CAPABILITIES_2_KHR,
			.pNext = &surfacePresentModeCompatibilityInfo,
		};

		VkResult ret = VK_PROC(vkGetPhysicalDeviceSurfaceCapabilities2KHR)(
			instance->device.vkPhysicalDevice, &surfaceInfo2, &surfaceCapabilities2);
		HANDLE_SURFACE_ERROR(ret, "Failed to get surface capabilities: %s", vxr::vk::vkResultStr(ret).cStr());

		compatiblePresentModes.resize(surfacePresentModeCompatibilityInfo.presentModeCount);
		surfacePresentModeCompatibilityInfo.pPresentModes = compatiblePresentModes.get();

		ret = VK_PROC(vkGetPhysicalDeviceSurfaceCapabilities2KHR)(instance->device.vkPhysicalDevice, &surfaceInfo2, &surfaceCapabilities2);
		HANDLE_SURFACE_ERROR(ret, "Failed to get surface capabilities: %s", vxr::vk::vkResultStr(ret).cStr());

		surfaceCapabilities = surfaceCapabilities2.surfaceCapabilities;
	}

	{
		if (surfaceCapabilities.currentExtent.width == UINT32_MAX || surfaceCapabilities.currentExtent.height == UINT32_MAX) {
			vxr::std::ePrintf("Wayland is currently unimplemented");
			vxr::std::abort();
		}

		if (!vxr::std::cmpBitFlagsContains(surfaceCapabilities.supportedCompositeAlpha, VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR)) {
			vxr::std::ePrintf("Failed to create swapchain: VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR is unsupported");
			vxr::std::abort();
		}

		swapchain->extent = {surfaceCapabilities.currentExtent.width, surfaceCapabilities.currentExtent.height};
	}

	{
		VkSwapchainPresentModesCreateInfoEXT presentModesCreateInfo = {
			.sType = VK_STRUCTURE_TYPE_SWAPCHAIN_PRESENT_MODES_CREATE_INFO_EXT,
			.presentModeCount = static_cast<uint32_t>(compatiblePresentModes.size()),
			.pPresentModes = compatiblePresentModes.get(),
		};

		VkSwapchainCreateInfoKHR createInfo = {
			.sType = VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR,
			.pNext = &presentModesCreateInfo,
			.surface = surface,
			.minImageCount = vxr::std::max(surfaceCapabilities.minImageCount, wantNumImages),
			.imageFormat = swapchain->surfaceFormat.format,
			.imageColorSpace = swapchain->surfaceFormat.colorSpace,
			.imageExtent = surfaceCapabilities.currentExtent,
			.imageArrayLayers = 1,
			.imageUsage = VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
			.imageSharingMode = VK_SHARING_MODE_EXCLUSIVE,
			.preTransform = surfaceCapabilities.currentTransform,
			.compositeAlpha = VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR,
			.presentMode = presentMode,
			.oldSwapchain = swapchain->vkSwapchain,
		};

		if ((surfaceCapabilities.maxImageCount != 0u) && (createInfo.minImageCount > surfaceCapabilities.maxImageCount)) {
			createInfo.minImageCount = surfaceCapabilities.maxImageCount;
		}

		const VkResult ret = VK_PROC_DEVICE(vkCreateSwapchainKHR)(
			instance->device.vkDevice, &createInfo, nullptr, &swapchain->vkSwapchain);
		HANDLE_SURFACE_ERROR(ret, "Failed to create swapchain: %s", vxr::vk::vkResultStr(ret).cStr());
		VK_PROC_DEVICE(vkDestroySwapchainKHR)(instance->device.vkDevice, createInfo.oldSwapchain, nullptr);
	}

	{
		uint32_t numImages = 0;
		VkResult ret = VK_PROC_DEVICE(vkGetSwapchainImagesKHR)(instance->device.vkDevice, swapchain->vkSwapchain, &numImages, nullptr);
		HANDLE_SURFACE_ERROR(ret, "Failed to get swapchain images: %s", vxr::vk::vkResultStr(ret).cStr());

		vxr::std::vector<VkImage> swapChainImages(numImages);
		swapchain->images.resize(numImages);
		ret = VK_PROC_DEVICE(vkGetSwapchainImagesKHR)(
			instance->device.vkDevice, swapchain->vkSwapchain, &numImages, swapChainImages.get());
		HANDLE_SURFACE_ERROR(ret, "Failed to get swapchain images: %s", vxr::vk::vkResultStr(ret).cStr());

		for (uint32_t i = 0; i < swapchain->images.size(); i++) {
			const VkImageViewCreateInfo createInfo = {
				.sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
				.image = swapchain->images[i].first = swapChainImages[i],
				.viewType = VK_IMAGE_VIEW_TYPE_2D,
				.format = swapchain->surfaceFormat.format,
				.components =
					VkComponentMapping{
						.r = VK_COMPONENT_SWIZZLE_IDENTITY,
						.g = VK_COMPONENT_SWIZZLE_IDENTITY,
						.b = VK_COMPONENT_SWIZZLE_IDENTITY,
						.a = VK_COMPONENT_SWIZZLE_IDENTITY,
					},
				.subresourceRange =
					VkImageSubresourceRange{
						.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT,
						.baseMipLevel = 0,
						.levelCount = 1,
						.baseArrayLayer = 0,
						.layerCount = 1,
					},
			};

			ret = VK_PROC_DEVICE(vkCreateImageView)(
				instance->device.vkDevice, &createInfo, nullptr, &swapchain->images[i].second);
			HANDLE_SURFACE_ERROR(ret, "Failed to create swapchain image view: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::vk::debugLabel(instance->device.vkDevice, swapchain->images[i].first, "swapchain_image_%d", i);
			vxr::vk::debugLabel(instance->device.vkDevice, swapchain->images[i].second, "swapchain_image_view_%d", i);
		}
	}

	return VK_SUCCESS;
}

void destroySwapchain(instance* instance) {
	auto* swapchain = &instance->graphics.swapchain;

	for (auto& image : swapchain->images) {
		VK_PROC_DEVICE(vkDestroyImageView)(instance->device.vkDevice, image.second, nullptr);
	}
	swapchain->images.resize(0);

	VK_PROC_DEVICE(vkDestroySwapchainKHR)(instance->device.vkDevice, swapchain->vkSwapchain, nullptr);
	swapchain->vkSwapchain = VK_NULL_HANDLE;
}
}  // namespace vxr::vk::graphics
