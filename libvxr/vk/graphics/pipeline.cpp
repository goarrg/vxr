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
#include "std/array.hpp"
#include "std/vector.hpp"

#include "vk/vk.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"

extern "C" {
VXR_FN void vxr_vk_graphics_createVertexInputPipeline(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
													  VkPrimitiveTopology topology, VkPipeline* pipeline) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	static constexpr vxr::std::array dynamicStates = {
		// VK_DYNAMIC_STATE_VERTEX_INPUT_EXT,
		VK_DYNAMIC_STATE_PRIMITIVE_TOPOLOGY,
	};
	static constexpr VkPipelineDynamicStateCreateInfo dynamicInfo{
		.sType = VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO,
		.dynamicStateCount = dynamicStates.size(),
		.pDynamicStates = dynamicStates.get(),
	};
	static constexpr VkPipelineVertexInputStateCreateInfo inputStateInfo{
		.sType = VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO,
	};

	static constexpr VkGraphicsPipelineLibraryCreateInfoEXT libraryInfo = {
		.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_LIBRARY_CREATE_INFO_EXT,
		.flags = VK_GRAPHICS_PIPELINE_LIBRARY_VERTEX_INPUT_INTERFACE_BIT_EXT,
	};

	const VkPipelineInputAssemblyStateCreateInfo inputAssemblyInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO,
		.topology = topology,
		.primitiveRestartEnable = VK_FALSE,
	};

	const VkGraphicsPipelineCreateInfo pipelineCreateInfo = {
		.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
		.pNext = &libraryInfo,
		.flags = VK_PIPELINE_CREATE_LIBRARY_BIT_KHR | VK_PIPELINE_CREATE_RETAIN_LINK_TIME_OPTIMIZATION_INFO_BIT_EXT,
		.pVertexInputState = &inputStateInfo,
		.pInputAssemblyState = &inputAssemblyInfo,
		.pDynamicState = &dynamicInfo,
	};

	const VkResult ret = VK_PROC_DEVICE(vkCreateGraphicsPipelines)(
		instance->device.vkDevice, nullptr, 1, &pipelineCreateInfo, nullptr, pipeline);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create vertex input pipeline: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder sb;
		sb.write("pipeline_vertex_input_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *pipeline, sb.cStr());
	});
}
VXR_FN void vxr_vk_graphics_createShaderPipeline(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
												 vxr_vk_graphics_shaderPipelineCreateInfo shader, VkPipeline* pipeline) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	static constexpr vxr::std::array vertexDynamicStates = {
		VK_DYNAMIC_STATE_VIEWPORT_WITH_COUNT,
		VK_DYNAMIC_STATE_SCISSOR_WITH_COUNT,
		VK_DYNAMIC_STATE_CULL_MODE,
		VK_DYNAMIC_STATE_FRONT_FACE,
	};
	static constexpr VkPipelineDynamicStateCreateInfo vertexDynamicInfo{
		.sType = VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO,
		.dynamicStateCount = vertexDynamicStates.size(),
		.pDynamicStates = vertexDynamicStates.get(),
	};

	static constexpr vxr::std::array fragmentDynamicStates = {
		VK_DYNAMIC_STATE_DEPTH_TEST_ENABLE,
		VK_DYNAMIC_STATE_DEPTH_WRITE_ENABLE,
	};
	static constexpr VkPipelineDynamicStateCreateInfo fragmentDynamicInfo{
		.sType = VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO,
		.dynamicStateCount = fragmentDynamicStates.size(),
		.pDynamicStates = fragmentDynamicStates.get(),
	};

	// vertex states
	static constexpr VkPipelineViewportStateCreateInfo viewportState = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO,
	};
	const VkPipelineRasterizationStateCreateInfo rasterizationState = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO,
		.depthClampEnable = VK_FALSE,
		.rasterizerDiscardEnable = VK_FALSE,
		.polygonMode = VK_POLYGON_MODE_FILL,
		.depthBiasEnable = VK_FALSE,
		.lineWidth = 1.0f,
		// .cullMode = VK_CULL_MODE_NONE,
		// .frontFace = VK_FRONT_FACE_CLOCKWISE,
	};

	// fragment states
	static constexpr VkPipelineMultisampleStateCreateInfo multisampleInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO,
		.rasterizationSamples = VK_SAMPLE_COUNT_1_BIT,
		.sampleShadingEnable = VK_FALSE,
	};
	static constexpr VkPipelineDepthStencilStateCreateInfo depthStencilInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO,
		// .depthTestEnable = VK_TRUE,
		// .depthWriteEnable = VK_TRUE,
		.depthCompareOp = VK_COMPARE_OP_GREATER_OR_EQUAL,
		.depthBoundsTestEnable = VK_FALSE,
		.stencilTestEnable = VK_FALSE,
		.minDepthBounds = 0.0f,
		.maxDepthBounds = 1.0f,
	};

	const VkShaderModuleCreateInfo shaderModuleCreateInfo = {
		.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
		.codeSize = shader.spirv.len * sizeof(uint32_t),
		.pCode = shader.spirv.data,
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
		.stage = shader.stage,
		.pName = entryPoint.cStr(),
	};

	// without this, it returns VK_ERROR_OUT_OF_HOST_MEMORY when we use
	// VK_PIPELINE_CREATE_RETAIN_LINK_TIME_OPTIMIZATION_INFO_BIT_EXT
	if (specializationEntries.size() > 0) {
		stageCreateInfo.pSpecializationInfo = &specializationInfo;
	}

	VkGraphicsPipelineLibraryCreateInfoEXT libraryInfo = {.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_LIBRARY_CREATE_INFO_EXT};
	VkGraphicsPipelineCreateInfo pipelineCreateInfo = {
		.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
		.pNext = &libraryInfo,
		.flags = VK_PIPELINE_CREATE_LIBRARY_BIT_KHR | VK_PIPELINE_CREATE_RETAIN_LINK_TIME_OPTIMIZATION_INFO_BIT_EXT,
		.stageCount = 1,
		.pStages = &stageCreateInfo,
		.layout = shader.layout,
	};

	if (shader.stage == VK_SHADER_STAGE_FRAGMENT_BIT) {
		libraryInfo.flags = VK_GRAPHICS_PIPELINE_LIBRARY_FRAGMENT_SHADER_BIT_EXT;
		pipelineCreateInfo.pMultisampleState = &multisampleInfo;
		pipelineCreateInfo.pDepthStencilState = &depthStencilInfo;
		pipelineCreateInfo.pDynamicState = &fragmentDynamicInfo;
	} else {
		libraryInfo.flags = VK_GRAPHICS_PIPELINE_LIBRARY_PRE_RASTERIZATION_SHADERS_BIT_EXT;
		pipelineCreateInfo.pViewportState = &viewportState;
		pipelineCreateInfo.pRasterizationState = &rasterizationState;
		pipelineCreateInfo.pDynamicState = &vertexDynamicInfo;
	}

	const VkResult ret = VK_PROC_DEVICE(vkCreateGraphicsPipelines)(
		instance->device.vkDevice, nullptr, 1, &pipelineCreateInfo, nullptr, pipeline);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create graphics shader pipeline: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder sb;
		if (shader.stage == VK_SHADER_STAGE_FRAGMENT_BIT) {
			sb.write("pipeline_fragment_");
		} else {
			sb.write("pipeline_vertex_");
		}
		sb.write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *pipeline, sb.cStr());
	});
}
VXR_FN void vxr_vk_graphics_createFragmentOutputPipeline(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
														 vxr_vk_graphics_fragmentOutputPipelineCreateInfo info, VkPipeline* pipeline) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	static constexpr vxr::std::array dynamicStates = {
		VK_DYNAMIC_STATE_COLOR_BLEND_ENABLE_EXT,
		VK_DYNAMIC_STATE_COLOR_BLEND_EQUATION_EXT,
		VK_DYNAMIC_STATE_COLOR_WRITE_MASK_EXT,
	};
	static constexpr VkPipelineDynamicStateCreateInfo dynamicInfo{
		.sType = VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO,
		.dynamicStateCount = dynamicStates.size(),
		.pDynamicStates = dynamicStates.get(),
	};
	static constexpr VkPipelineMultisampleStateCreateInfo multisampleInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO,
		.rasterizationSamples = VK_SAMPLE_COUNT_1_BIT,
		.sampleShadingEnable = VK_FALSE,
	};
	static constexpr VkPipelineColorBlendStateCreateInfo colorBlendInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO,
	};

	const VkPipelineRenderingCreateInfo renderingInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_RENDERING_CREATE_INFO,
		.colorAttachmentCount = info.numColorAttachments,
		.pColorAttachmentFormats = info.colorAttachmentFormats,
		.depthAttachmentFormat = info.depthFormat,
		.stencilAttachmentFormat = info.stencilFormat,
	};

	const VkGraphicsPipelineLibraryCreateInfoEXT libraryInfo = {
		.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_LIBRARY_CREATE_INFO_EXT,
		.pNext = &renderingInfo,
		.flags = VK_GRAPHICS_PIPELINE_LIBRARY_FRAGMENT_OUTPUT_INTERFACE_BIT_EXT,
	};

	const VkGraphicsPipelineCreateInfo pipelineCreateInfo = {
		.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
		.pNext = &libraryInfo,
		.flags = VK_PIPELINE_CREATE_LIBRARY_BIT_KHR | VK_PIPELINE_CREATE_RETAIN_LINK_TIME_OPTIMIZATION_INFO_BIT_EXT,
		.pMultisampleState = &multisampleInfo,
		.pColorBlendState = &colorBlendInfo,
		.pDynamicState = info.numColorAttachments > 0 ? &dynamicInfo : nullptr,
	};

	const VkResult ret = VK_PROC_DEVICE(vkCreateGraphicsPipelines)(
		instance->device.vkDevice, nullptr, 1, &pipelineCreateInfo, nullptr, pipeline);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create fragment output pipeline: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}
	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder sb;
		sb.write("pipeline_fragment_output_");
		sb.write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *pipeline, sb.cStr());
	});
}
VXR_FN VkBool32 vxr_vk_graphics_linkPipelines(vxr_vk_instance instanceHandle, size_t nameSz, const char* name, VkPipelineLayout layout,
											  uint32_t numPipelines, VkPipeline* pipelines, VkPipeline* executable) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	const VkPipelineLibraryCreateInfoKHR linkingInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_LIBRARY_CREATE_INFO_KHR,
		.libraryCount = numPipelines,
		.pLibraries = pipelines,
	};

	VkGraphicsPipelineCreateInfo executablePipelineCreateInfo = {
		.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
		.pNext = &linkingInfo,
		.flags = static_cast<VkPipelineCreateFlags>(
			VK_PIPELINE_CREATE_LINK_TIME_OPTIMIZATION_BIT_EXT | VK_PIPELINE_CREATE_FAIL_ON_PIPELINE_COMPILE_REQUIRED_BIT),
		.layout = layout,
	};
	VkResult ret = VK_PROC_DEVICE(vkCreateGraphicsPipelines)(
		instance->device.vkDevice, nullptr, 1, &executablePipelineCreateInfo, nullptr, executable);
	if (ret == VK_SUCCESS) {
		vxr::std::vPrintf("Loaded cached executable optimized pipeline");
		vxr::std::debugRun([=]() {
			vxr::std::stringbuilder sb;
			sb.write("pipeline_executable_optimized_").write(nameSz, name);
			vxr::vk::debugLabel(instance->device.vkDevice, *executable, sb.cStr());
		});
		return VK_TRUE;
	}

	if (ret == VK_PIPELINE_COMPILE_REQUIRED) {
		vxr::std::vPrintf("Executable pipeline not cached, fast linking pipeline");
		executablePipelineCreateInfo.flags = VK_PIPELINE_CREATE_DISABLE_OPTIMIZATION_BIT;
		ret = VK_PROC_DEVICE(vkCreateGraphicsPipelines)(
			instance->device.vkDevice, nullptr, 1, &executablePipelineCreateInfo, nullptr, executable);
	}
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create executable pipeline: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder sb;
		sb.write("pipeline_executable_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *executable, sb.cStr());
	});

	return VK_FALSE;
}
VXR_FN void vxr_vk_graphics_linkOptimizePipelines(vxr_vk_instance instanceHandle, size_t nameSz, const char* name, VkPipelineLayout layout,
												  uint32_t numPipelines, VkPipeline* pipelines, VkPipeline* executable) {
	vxr::std::vPrintf("Linking optimized executable pipeline");
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	const VkPipelineLibraryCreateInfoKHR linkingInfo = {
		.sType = VK_STRUCTURE_TYPE_PIPELINE_LIBRARY_CREATE_INFO_KHR,
		.libraryCount = numPipelines,
		.pLibraries = pipelines,
	};

	const VkGraphicsPipelineCreateInfo executablePipelineCreateInfo = {
		.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
		.pNext = &linkingInfo,
		.flags = static_cast<VkPipelineCreateFlags>(VK_PIPELINE_CREATE_LINK_TIME_OPTIMIZATION_BIT_EXT),
		.layout = layout,
	};

	const VkResult ret = VK_PROC_DEVICE(vkCreateGraphicsPipelines)(
		instance->device.vkDevice, nullptr, 1, &executablePipelineCreateInfo, nullptr, executable);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create executable pipeline: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder sb;
		sb.write("pipeline_executable_optimized_").write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *executable, sb.cStr());
	});
}
}
