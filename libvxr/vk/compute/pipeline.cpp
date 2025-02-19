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
VXR_FN void vxr_vk_compute_createShaderPipeline(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
												vxr_vk_compute_shaderPipelineCreateInfo shader, VkPipeline* pipeline) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	const VkShaderModuleCreateInfo shaderModuleCreateInfo = {
		.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
		.codeSize = shader.spirv.len * sizeof(uint32_t),
		.pCode = shader.spirv.data,
	};

	const VkPipelineShaderStageRequiredSubgroupSizeCreateInfo requiredSubgroupSizeInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_REQUIRED_SUBGROUP_SIZE_CREATE_INFO,
		.pNext = (void*)&shaderModuleCreateInfo,
		.requiredSubgroupSize = shader.requiredSubgroupSize,
	};

	vxr::std::vector<VkSpecializationMapEntry> specializationEntries(shader.numSpecConstants);
	for (uint32_t i = 0; i < shader.numSpecConstants; i++) {
		specializationEntries[i] = VkSpecializationMapEntry{
			.constantID = i,
			.offset = (uint32_t)sizeof(uint32_t) * i,
			.size = sizeof(uint32_t),
		};
	}
	const VkSpecializationInfo specializationInfo = {
		.mapEntryCount = static_cast<uint32_t>(specializationEntries.size()),
		.pMapEntries = specializationEntries.get(),

		.dataSize = sizeof(uint32_t) * specializationEntries.size(),
		.pData = shader.specConstants,
	};

	const vxr::std::string entryPoint(shader.entryPointSize, shader.entryPoint);
	VkPipelineShaderStageCreateInfo stageCreateInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
		.pNext = &shaderModuleCreateInfo,
		.flags = shader.stageFlags,
		.stage = VK_SHADER_STAGE_COMPUTE_BIT,
		.pName = entryPoint.cStr(),
	};
	if (shader.requiredSubgroupSize > 0) {
		stageCreateInfo.pNext = &requiredSubgroupSizeInfo;
	}
	if (specializationEntries.size() > 0) {
		stageCreateInfo.pSpecializationInfo = &specializationInfo;
	}

	const VkComputePipelineCreateInfo computePipelineCreateInfo = {
		.sType = VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO,
		.stage = stageCreateInfo,
		.layout = shader.layout,
	};

	const VkResult ret = VK_PROC_DEVICE(vkCreateComputePipelines)(
		instance->device.vkDevice, VK_NULL_HANDLE, 1, &computePipelineCreateInfo, nullptr, pipeline);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create compute shader pipeline: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder sb;
		sb.write("pipeline_compute_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *pipeline, sb.cStr());
	});
}
}
