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

#include <stdint.h>

#include "std/utility.hpp"
#include "std/log.hpp"

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/vkfns.hpp"
#include "vk/device/selector/selector.hpp"

// NOLINTNEXTLINE(readability-make-member-function-const)
bool vxr::vk::device::selector::selector::findProperties(vxr::vk::instance* instance) {
	VkPhysicalDeviceVulkan13Properties device13Properties = {
		.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_3_PROPERTIES,
	};
	VkPhysicalDeviceVulkan12Properties device12Properties = {
		.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_2_PROPERTIES,
		.pNext = &device13Properties,
	};
	VkPhysicalDeviceVulkan11Properties device11Properties = {
		.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_1_PROPERTIES,
		.pNext = &device12Properties,
	};
	VkPhysicalDeviceProperties2 deviceProperties2 = {
		.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_PROPERTIES_2,
		.pNext = &device11Properties,
	};
	VK_PROC(vkGetPhysicalDeviceProperties2)(instance->device.vkPhysicalDevice, &deviceProperties2);

	{
		auto properties = deviceProperties2.properties;
		if (properties.apiVersion < this->requiredAPI) {
			vxr::std::vPrintf("Device API %d.%d < required API %d.%d",	//
							  VK_API_VERSION_MAJOR(properties.apiVersion), VK_API_VERSION_MINOR(properties.apiVersion),
							  VK_API_VERSION_MAJOR(this->requiredAPI), VK_API_VERSION_MINOR(this->requiredAPI));
			return false;
		}

		instance->device.properties.vendorID = properties.vendorID;
		instance->device.properties.deviceID = properties.deviceID;
		instance->device.properties.driverVersion = properties.driverVersion;
		instance->device.properties.api = vxr::std::min(properties.apiVersion, this->requiredAPI);

		// compute Properties
		{
			instance->device.properties.compute.subgroupSize = device11Properties.subgroupSize;	 //
		}
	}

	{
		const VkPhysicalDeviceLimits device10Proprties = deviceProperties2.properties.limits;
		auto* limits = &instance->device.properties.limits;

		{
			limits->minLineWidth = device10Proprties.lineWidthRange[0];
			limits->maxLineWidth = device10Proprties.lineWidthRange[1];

			limits->minPointSize = device10Proprties.pointSizeRange[0];
			limits->maxPointSize = device10Proprties.pointSizeRange[1];
		}

		// global limits
		{
			limits->global.maxAllocationSize = device11Properties.maxMemoryAllocationSize;
			limits->global.maxMemoryAllocationCount = device10Proprties.maxMemoryAllocationCount;
			limits->global.maxSamplerAllocationCount = device10Proprties.maxSamplerAllocationCount;
		}

		// per descriptor limits
		{
			limits->perDescriptor.maxImageDimension1D =
				int32_t(vxr::std::min(device10Proprties.maxImageDimension1D, uint32_t(INT32_MAX)));
			limits->perDescriptor.maxImageDimension2D =
				int32_t(vxr::std::min(device10Proprties.maxImageDimension2D, uint32_t(INT32_MAX)));
			limits->perDescriptor.maxImageDimension3D =
				int32_t(vxr::std::min(device10Proprties.maxImageDimension3D, uint32_t(INT32_MAX)));
			limits->perDescriptor.maxImageDimensionCube =
				int32_t(vxr::std::min(device10Proprties.maxImageDimensionCube, uint32_t(INT32_MAX)));
			limits->perDescriptor.maxImageArrayLayers =
				int32_t(vxr::std::min(device10Proprties.maxImageArrayLayers, uint32_t(INT32_MAX)));

			limits->perDescriptor.maxSamplerAnisotropy = device10Proprties.maxSamplerAnisotropy;
			limits->perDescriptor.maxUBOSize = device10Proprties.maxUniformBufferRange;
			limits->perDescriptor.maxSBOSize = device10Proprties.maxStorageBufferRange;
		}

		// per stage limits
		{
			limits->perStage.maxSamplerCount = device10Proprties.maxPerStageDescriptorSamplers;
			limits->perStage.maxSampledImageCount = device10Proprties.maxPerStageDescriptorSampledImages;
			limits->perStage.maxCombinedImageSamplerCount = vxr::std::min(
				device10Proprties.maxPerStageDescriptorSamplers, device10Proprties.maxPerStageDescriptorSampledImages);
			limits->perStage.maxStorageImageCount = device10Proprties.maxPerStageDescriptorStorageImages;
			limits->perStage.maxUBOCount = device10Proprties.maxPerStageDescriptorUniformBuffers;
			limits->perStage.maxSBOCount = device10Proprties.maxPerStageDescriptorStorageBuffers;
			limits->perStage.maxResourceCount = device10Proprties.maxPerStageResources;
		}

		// per pipeline limits
		{
			limits->perPipeline.maxSamplerCount = device10Proprties.maxDescriptorSetSamplers;
			limits->perPipeline.maxSampledImageCount = device10Proprties.maxDescriptorSetSampledImages;
			limits->perPipeline.maxCombinedImageSamplerCount = vxr::std::min(
				device10Proprties.maxDescriptorSetSamplers, device10Proprties.maxDescriptorSetSampledImages);
			limits->perPipeline.maxStorageImageCount = device10Proprties.maxDescriptorSetStorageImages;

			limits->perPipeline.maxUBOCount = device10Proprties.maxDescriptorSetUniformBuffers;
			limits->perPipeline.maxSBOCount = device10Proprties.maxDescriptorSetStorageBuffers;

			limits->perPipeline.maxBoundDescriptorSets = device10Proprties.maxBoundDescriptorSets;
			limits->perPipeline.maxPushConstantsSize = device10Proprties.maxPushConstantsSize;
		}

		// compute Limits
		{
			limits->compute.maxDispatchSize = VkExtent3D{
				device10Proprties.maxComputeWorkGroupCount[0],
				device10Proprties.maxComputeWorkGroupCount[1],
				device10Proprties.maxComputeWorkGroupCount[2],
			};
			limits->compute.maxLocalSize = VkExtent3D{
				device10Proprties.maxComputeWorkGroupSize[0],
				device10Proprties.maxComputeWorkGroupSize[1],
				device10Proprties.maxComputeWorkGroupSize[2],
			};

			limits->compute.workgroup = {
				.maxInvocations = device10Proprties.maxComputeWorkGroupInvocations,
				.maxSubgroupCount = device13Properties.maxComputeWorkgroupSubgroups,
			};

			limits->compute.minSubgroupSize = device13Properties.minSubgroupSize;
			limits->compute.maxSubgroupSize = device13Properties.maxSubgroupSize;
		}
	}

	return true;
}
