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
#include <stdint.h>

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/string.hpp"
#include "std/vector.hpp"

#include "vk/vk.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"

extern "C" {
VXR_FN void vxr_vk_shader_createDescriptorSetLayout(vxr_vk_instance instanceHandle, size_t nameSz, const char* name, uint32_t bindingsCount,
													VkDescriptorSetLayoutBinding* bindings, VkDescriptorSetLayout* descriptorSetLayout) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	vxr::std::vector<VkDescriptorBindingFlags> descriptorLayoutBindingFlags(bindingsCount);
	for (uint32_t i = 0; i < bindingsCount; i++) {
		if (bindings[i].descriptorCount > 1) {
			descriptorLayoutBindingFlags[i] = VK_DESCRIPTOR_BINDING_UPDATE_UNUSED_WHILE_PENDING_BIT | VK_DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT;
		}
	}

	VkDescriptorSetLayoutBindingFlagsCreateInfo descriptorLayoutBindingFlagsInfo = {};
	descriptorLayoutBindingFlagsInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_BINDING_FLAGS_CREATE_INFO;
	descriptorLayoutBindingFlagsInfo.bindingCount = bindingsCount;
	descriptorLayoutBindingFlagsInfo.pBindingFlags = descriptorLayoutBindingFlags.get();

	VkDescriptorSetLayoutCreateInfo descriptorLayoutInfo = {};
	descriptorLayoutInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO;
	descriptorLayoutInfo.pNext = &descriptorLayoutBindingFlagsInfo;
	descriptorLayoutInfo.pBindings = bindings;
	descriptorLayoutInfo.bindingCount = bindingsCount;

	const VkResult ret = VK_PROC_DEVICE(vkCreateDescriptorSetLayout)(
		instance->device.vkDevice, &descriptorLayoutInfo, nullptr, descriptorSetLayout);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create descriptor set layout: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder sb;
		sb.write("descriptor_set_layout_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *descriptorSetLayout, sb.cStr());
	});
}
VXR_FN void vxr_vk_shader_destroyDescriptorSetLayout(vxr_vk_instance instanceHandle, VkDescriptorSetLayout descriptorSetLayout) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	VK_PROC_DEVICE(vkDestroyDescriptorSetLayout)(instance->device.vkDevice, descriptorSetLayout, nullptr);
}
VXR_FN void vxr_vk_shader_createDescriptorPool(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
											   VkDescriptorPoolCreateInfo info, VkDescriptorPool* descriptorPool) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	const VkResult ret = VK_PROC_DEVICE(vkCreateDescriptorPool)(instance->device.vkDevice, &info, nullptr, descriptorPool);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create descriptor pool: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder sb;
		sb.write("descriptor_pool_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *descriptorPool, sb.cStr());
	});
}
VXR_FN void vxr_vk_shader_destroyDescriptorPool(vxr_vk_instance instanceHandle, VkDescriptorPool descriptorPool) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	VK_PROC_DEVICE(vkDestroyDescriptorPool)(instance->device.vkDevice, descriptorPool, nullptr);
}
VXR_FN void vxr_vk_shader_createDescriptorSet(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
											  VkDescriptorSetAllocateInfo info, VkDescriptorSet* descriptorSet) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	const VkResult ret = VK_PROC_DEVICE(vkAllocateDescriptorSets)(instance->device.vkDevice, &info, descriptorSet);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create descriptor set: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder sb;
		sb.write("descriptor_set_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *descriptorSet, sb.cStr());
	});
}
VXR_FN void vxr_vk_shader_updateDescriptorSet(vxr_vk_instance instanceHandle, VkWriteDescriptorSet descriptorWrites) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	VK_PROC_DEVICE(vkUpdateDescriptorSets)(instance->device.vkDevice, 1, &descriptorWrites, 0, nullptr);
}
VXR_FN void vxr_vk_shader_destroyDescriptorSet(vxr_vk_instance instanceHandle, VkDescriptorPool descriptorPool, VkDescriptorSet descriptorSet) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	VK_PROC_DEVICE(vkFreeDescriptorSets)(instance->device.vkDevice, descriptorPool, 1, &descriptorSet);
}
VXR_FN void vxr_vk_shader_createPipelineLayout(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
											   vxr_vk_shader_pipelineLayoutCreateInfo info, VkPipelineLayout* layout) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	const VkPipelineLayoutCreateInfo pipelineLayoutInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
		.flags = VK_PIPELINE_LAYOUT_CREATE_INDEPENDENT_SETS_BIT_EXT,
		.setLayoutCount = info.numDescriptorSetLayouts,
		.pSetLayouts = info.descriptorSetLayouts,
		.pushConstantRangeCount = info.numPushConstantRanges,
		.pPushConstantRanges = info.pushConstantRanges,
	};

	const VkResult ret = VK_PROC_DEVICE(vkCreatePipelineLayout)(instance->device.vkDevice, &pipelineLayoutInfo, nullptr, layout);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create pipeline layout: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder sb;
		sb.write("pipelineLayout_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *layout, sb.cStr());
	});
}
VXR_FN void vxr_vk_shader_destroyPipelineLayout(vxr_vk_instance instanceHandle, VkPipelineLayout layout) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	VK_PROC_DEVICE(vkDestroyPipelineLayout)(instance->device.vkDevice, layout, nullptr);
}
VXR_FN void vxr_vk_shader_destroyPipeline(vxr_vk_instance instanceHandle, VkPipeline pipeline) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	VK_PROC_DEVICE(vkDestroyPipeline)(instance->device.vkDevice, pipeline, nullptr);
}
}
