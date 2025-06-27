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

#include <stddef.h>

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/string.hpp"

#include "vk/vk.hpp"
#include "vk/vkfns.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"
#include "vk/device/vma/vma.hpp"

extern "C" {
VXR_FN void vxr_vk_getFormatProperties(vxr_vk_instance instanceHandle, VkFormat format, VkFormatProperties3* properties) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	VkFormatProperties2 formatProperties2 = {
		.sType = VK_STRUCTURE_TYPE_FORMAT_PROPERTIES_2,
		.pNext = properties,
	};
	VK_PROC(vkGetPhysicalDeviceFormatProperties2)
	(instance->device.vkPhysicalDevice, format, &formatProperties2);
}
VXR_FN void vxr_vk_createImage(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
							   vxr_vk_imageCreateInfo info, vxr_vk_image* t) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	{
		VkImageCreateInfo createInfo = {};
		createInfo.sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO;
		createInfo.imageType = info.type;
		createInfo.format = info.format;
		createInfo.extent = info.extent;
		createInfo.mipLevels = info.mipLevels;
		createInfo.arrayLayers = info.arrayLayers;
		createInfo.samples = VK_SAMPLE_COUNT_1_BIT;
		createInfo.tiling = VK_IMAGE_TILING_OPTIMAL;
		createInfo.initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
		createInfo.usage = info.usage;
		createInfo.flags = info.flags;

		VmaAllocationCreateInfo allocCreateInfo = {};
		allocCreateInfo.usage = VMA_MEMORY_USAGE_AUTO_PREFER_DEVICE;
		allocCreateInfo.requiredFlags = VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT;
		allocCreateInfo.memoryTypeBits = instance->device.vma.noBARMemoryTypeBits;

		const VkResult ret = vmaCreateImage(instance->device.vma.allocator, &createInfo, &allocCreateInfo, &t->vkImage,
											reinterpret_cast<VmaAllocation*>(&t->allocation), nullptr);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to create image: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}

		vxr::std::debugRun([=]() {
			vxr::std::stringbuilder builder;
			builder.write("image_").write(nameSz, name);
			vxr::vk::debugLabel(instance->device.vkDevice, t->vkImage, builder.cStr());

			builder.write("_allocation");
			vmaSetAllocationName(instance->device.vma.allocator, reinterpret_cast<VmaAllocation>(t->allocation), builder.cStr());
		});
	}
}
VXR_FN void vxr_vk_createImageMultiSampled(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
										   vxr_vk_imageMultiSampledCreateInfo info, vxr_vk_image* t) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	{
		VkImageCreateInfo createInfo = {};
		createInfo.sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO;
		createInfo.imageType = VK_IMAGE_TYPE_2D;
		createInfo.format = info.format;
		createInfo.extent = VkExtent3D{
			.width = info.extent.width,
			.height = info.extent.height,
			.depth = 1,
		};
		createInfo.mipLevels = 1;
		createInfo.arrayLayers = 1;
		createInfo.samples = info.samples;
		createInfo.tiling = VK_IMAGE_TILING_OPTIMAL;
		createInfo.initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
		createInfo.usage = info.usage;
		createInfo.flags = info.flags;

		VmaAllocationCreateInfo allocCreateInfo = {};
		allocCreateInfo.usage = VMA_MEMORY_USAGE_AUTO_PREFER_DEVICE;
		allocCreateInfo.requiredFlags = VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT;
		allocCreateInfo.memoryTypeBits = instance->device.vma.noBARMemoryTypeBits;

		const VkResult ret = vmaCreateImage(instance->device.vma.allocator, &createInfo, &allocCreateInfo, &t->vkImage,
											reinterpret_cast<VmaAllocation*>(&t->allocation), nullptr);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to create image: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}

		vxr::std::debugRun([=]() {
			vxr::std::stringbuilder builder;
			builder.write("image_multisampled_").write(nameSz, name);
			vxr::vk::debugLabel(instance->device.vkDevice, t->vkImage, builder.cStr());

			builder.write("_allocation");
			vmaSetAllocationName(instance->device.vma.allocator, reinterpret_cast<VmaAllocation>(t->allocation), builder.cStr());
		});
	}
}
VXR_FN void vxr_vk_destroyImage(vxr_vk_instance instanceHandle, vxr_vk_image t) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	vmaDestroyImage(instance->device.vma.allocator, t.vkImage, reinterpret_cast<VmaAllocation>(t.allocation));
}
VXR_FN void vxr_vk_createImageView(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
								   vxr_vk_imageViewCreateInfo info, VkImageView* view) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	{
		VkImageViewCreateInfo viewCreateInfo = {};
		viewCreateInfo.sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO;
		viewCreateInfo.flags = info.flags;
		viewCreateInfo.image = info.vkImage;
		viewCreateInfo.viewType = info.type;
		viewCreateInfo.format = info.format;
		viewCreateInfo.components.r = VK_COMPONENT_SWIZZLE_IDENTITY;
		viewCreateInfo.components.g = VK_COMPONENT_SWIZZLE_IDENTITY;
		viewCreateInfo.components.b = VK_COMPONENT_SWIZZLE_IDENTITY;
		viewCreateInfo.components.a = VK_COMPONENT_SWIZZLE_IDENTITY;

		viewCreateInfo.subresourceRange = info.range;

		const VkResult ret = VK_PROC_DEVICE(vkCreateImageView)(instance->device.vkDevice, &viewCreateInfo, nullptr, view);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to create image view: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}

		vxr::std::debugRun([=]() {
			vxr::std::stringbuilder builder;
			builder.write("image_view_").write(nameSz, name);
			vxr::vk::debugLabel(instance->device.vkDevice, *view, builder.cStr());
		});
	}
}
VXR_FN void vxr_vk_destroyImageView(vxr_vk_instance instanceHandle, VkImageView view) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	VK_PROC_DEVICE(vkDestroyImageView)(instance->device.vkDevice, view, nullptr);
}
VXR_FN void vxr_vk_createSampler(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
								 vxr_vk_samplerCreateInfo info, VkSampler* sampler) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	{
		const VkSamplerCreateInfo samplerCreateInfo = {
			.sType = VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO,
			.magFilter = info.magFilter,
			.minFilter = info.minFilter,
			.mipmapMode = info.mipmapMode,
			.addressModeU = info.borderMode,
			.addressModeV = info.borderMode,
			.addressModeW = info.borderMode,
			.mipLodBias = 0.0f,
			.anisotropyEnable = info.anisotropy > 0 ? VK_TRUE : VK_FALSE,
			.maxAnisotropy = info.anisotropy,
			.compareOp = VK_COMPARE_OP_NEVER,
			.minLod = 0.0f,
			.maxLod = 0.0f,
			.borderColor = VK_BORDER_COLOR_FLOAT_TRANSPARENT_BLACK,
			.unnormalizedCoordinates = info.unnormalizedCoordinates,
		};

		const VkResult ret = VK_PROC_DEVICE(vkCreateSampler)(instance->device.vkDevice, &samplerCreateInfo, nullptr, sampler);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to create sampler: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}

		vxr::std::debugRun([=]() {
			vxr::std::stringbuilder builder;
			builder.write("sampler_").write(nameSz, name);
			vxr::vk::debugLabel(instance->device.vkDevice, *sampler, builder.cStr());
		});
	}
}
VXR_FN void vxr_vk_destroySampler(vxr_vk_instance instanceHandle, VkSampler sampler) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	VK_PROC_DEVICE(vkDestroySampler)(instance->device.vkDevice, sampler, nullptr);
}
}
