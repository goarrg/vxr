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

#include <spirv_cross/spirv.h>
#include <spirv_cross/spirv_cross_c.h>

#include <stddef.h>
#include <stdint.h>

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/string.hpp"
#include "std/array.hpp"
#include "std/vector.hpp"
#include "std/utility.hpp"

#include "vxr/vxr.h"
#include "vk/shader/toolchain/reflector.hpp"

inline static VkDescriptorType spvcResrouceToVkDescriptor(spvc_resource_type tid, spvc_compiler compiler, spvc_variable_id vid) {
	switch (tid) {
		case SPVC_RESOURCE_TYPE_UNIFORM_BUFFER:
			return VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;

		case SPVC_RESOURCE_TYPE_STORAGE_BUFFER:
			return VK_DESCRIPTOR_TYPE_STORAGE_BUFFER;

		case SPVC_RESOURCE_TYPE_STORAGE_IMAGE:
			if (spvc_type_get_image_dimension(spvc_compiler_get_type_handle(compiler, vid)) == SpvDimBuffer) {
				return VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER;
			}
			return VK_DESCRIPTOR_TYPE_STORAGE_IMAGE;

		case SPVC_RESOURCE_TYPE_SAMPLED_IMAGE:
			return VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;

		case SPVC_RESOURCE_TYPE_SEPARATE_IMAGE:
			if (spvc_type_get_image_dimension(spvc_compiler_get_type_handle(compiler, vid)) == SpvDimBuffer) {
				return VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER;
			}
			return VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE;

		case SPVC_RESOURCE_TYPE_SEPARATE_SAMPLERS:
			return VK_DESCRIPTOR_TYPE_SAMPLER;

		default:
			vxr::std::ePrintf("Unknown spvc resource type: %d", tid);
			vxr::std::abort();
			return VK_DESCRIPTOR_TYPE_MAX_ENUM;
	}
}

inline static vxr_vk_shader_reflectResult_bufferMetadata reflectBuffer(spvc_context context, spvc_compiler compiler, spvc_reflected_resource r) {
	vxr_vk_shader_reflectResult_bufferMetadata result = {
		.name = r.name,
	};

	const spvc_type t = spvc_compiler_get_type_handle(compiler, r.base_type_id);
	const uint32_t numMembers = spvc_type_get_num_member_types(t);
	const spvc_type lastMember = spvc_compiler_get_type_handle(compiler, spvc_type_get_member_type(t, numMembers - 1));
	const uint32_t numDimensions = spvc_type_get_num_array_dimensions(lastMember);
	const bool isRuntimeArray =
		(numDimensions > 0) && (spvc_type_array_dimension_is_literal(lastMember, numDimensions - 1) == SPVC_TRUE) &&
		(spvc_type_get_array_dimension(lastMember, numDimensions - 1) == 0);
	if (isRuntimeArray) {
		if (numDimensions > 1) {
			vxr::std::abort("Variable length multi dimensional arrays are not implemented");
		}

		uint32_t sz;
		uint32_t stride;

		spvc_result ret = spvc_compiler_type_struct_member_offset(compiler, t, numMembers - 1, &sz);
		if (ret != SPVC_SUCCESS) {
			vxr::std::ePrintf("Failed to get buffer size: %s", spvc_context_get_last_error_string(context));
			vxr::std::abort();
		}
		ret = spvc_compiler_type_struct_member_array_stride(compiler, t, numMembers - 1, &stride);
		if (ret != SPVC_SUCCESS) {
			vxr::std::ePrintf("Failed to get buffer size: %s", spvc_context_get_last_error_string(context));
			vxr::std::abort();
		}

		result.size = sz;
		result.runtimeArrayStride = stride;
	} else {
		size_t sz;
		const spvc_result ret = spvc_compiler_get_declared_struct_size(compiler, t, &sz);
		if (ret != SPVC_SUCCESS) {
			vxr::std::ePrintf("Failed to get buffer size: %s", spvc_context_get_last_error_string(context));
			vxr::std::abort();
		}
		result.size = sz;
	}

	return result;
}

inline static vxr_vk_shader_reflectResult_imageMetadata reflectImage([[maybe_unused]] spvc_context context,
																	 spvc_compiler compiler, spvc_reflected_resource r) {
	vxr_vk_shader_reflectResult_imageMetadata result = {
		.name = r.name,
	};

	const spvc_type t = spvc_compiler_get_type_handle(compiler, r.base_type_id);
	switch (spvc_type_get_image_dimension(t)) {
		case SpvDim1D:
			if (spvc_type_get_image_arrayed(t) == SPVC_TRUE) {
				result.viewType = VK_IMAGE_VIEW_TYPE_1D_ARRAY;
			} else {
				result.viewType = VK_IMAGE_VIEW_TYPE_1D;
			}
			break;

		case SpvDim2D:
			if (spvc_type_get_image_arrayed(t) == SPVC_TRUE) {
				result.viewType = VK_IMAGE_VIEW_TYPE_2D_ARRAY;
			} else {
				result.viewType = VK_IMAGE_VIEW_TYPE_2D;
			}
			break;

		case SpvDim3D:
			result.viewType = VK_IMAGE_VIEW_TYPE_3D;
			break;

		case SpvDimCube:
			if (spvc_type_get_image_arrayed(t) == SPVC_TRUE) {
				result.viewType = VK_IMAGE_VIEW_TYPE_CUBE_ARRAY;
			} else {
				result.viewType = VK_IMAGE_VIEW_TYPE_CUBE;
			}
			break;

		default:
			vxr::std::ePrintf("Invalid/Unknown SpvDim [%d] for image descriptors", spvc_type_get_image_dimension(t));
			vxr::std::abort();
			break;
	}

	return result;
}

namespace vxr::vk::shader {
reflector::reflector(vxr_vk_shader_spirv spirv) noexcept {
	auto ret = spvc_context_create(&this->spvcContext);
	if (ret != SPVC_SUCCESS) {
		vxr::std::ePrintf("Failed to create spvc context: %d", ret);
		vxr::std::abort();
	}

	spvc_parsed_ir parsedIR;
	ret = spvc_context_parse_spirv(this->spvcContext, reinterpret_cast<const SpvId*>(spirv.data), spirv.len, &parsedIR);
	if (ret != SPVC_SUCCESS) {
		vxr::std::ePrintf("Failed to prase spv: %s", spvc_context_get_last_error_string(this->spvcContext));
		vxr::std::abort();
	}

	ret = spvc_context_create_compiler(
		this->spvcContext, SPVC_BACKEND_NONE, parsedIR, SPVC_CAPTURE_MODE_TAKE_OWNERSHIP, &this->spvcCompiler);
	if (ret != SPVC_SUCCESS) {
		vxr::std::ePrintf("Failed to create spvc compiler: %s", spvc_context_get_last_error_string(this->spvcContext));
		vxr::std::abort();
	}

	this->spvcResources = nullptr;
}
reflector::~reflector() noexcept {
	spvc_context_destroy(this->spvcContext);
}

[[nodiscard]] const vxr::std::vector<reflector::entryPoint>& reflector::getEntryPoints() noexcept {
	if (this->entryPoints.size() == 0) {
		size_t sz;
		const spvc_entry_point* spvEntryPoints;
		auto ret = spvc_compiler_get_entry_points(this->spvcCompiler, &spvEntryPoints, &sz);
		if (ret != SPVC_SUCCESS) {
			vxr::std::ePrintf("Failed to get entry points: %s", spvc_context_get_last_error_string(this->spvcContext));
			vxr::std::abort();
		}

		for (size_t i = 0; i < sz; i++) {
			auto stage = VkShaderStageFlagBits(0);
			switch (spvEntryPoints[i].execution_model) {
				case SpvExecutionModelVertex:
					stage = VK_SHADER_STAGE_VERTEX_BIT;
					break;

				case SpvExecutionModelFragment:
					stage = VK_SHADER_STAGE_FRAGMENT_BIT;
					break;

				case SpvExecutionModelGLCompute:
					stage = VK_SHADER_STAGE_COMPUTE_BIT;
					break;

				default:
					vxr::std::ePrintf("Failed to get entry points: Unknown stage [%d] for entry point %s",
									  spvEntryPoints[i].execution_model, spvEntryPoints[i].name);
					vxr::std::abort();
					break;
			}
			this->entryPoints.pushBack(reflector::entryPoint{.name = spvEntryPoints[i].name, .stage = stage});
		}
	}

	return this->entryPoints;
}

[[nodiscard]] const vxr::std::vector<reflector::specConstant>& reflector::getSpecConstants() noexcept {
	if (this->specConstants.size() == 0) {
		size_t sz;
		const spvc_specialization_constant* spvConstants;
		auto ret = spvc_compiler_get_specialization_constants(this->spvcCompiler, &spvConstants, &sz);
		if (ret != SPVC_SUCCESS) {
			vxr::std::ePrintf("Failed to get specialization constants: %s", spvc_context_get_last_error_string(this->spvcContext));
			vxr::std::abort();
		}
		for (size_t i = 0; i < sz; i++) {
			this->specConstants.resize(vxr::std::max<size_t>(this->specConstants.size(), spvConstants[i].constant_id + 1));
			const char* name = spvc_compiler_get_name(this->spvcCompiler, spvConstants[i].id);
			this->specConstants[spvConstants[i].constant_id].name = name;
			this->specConstants[spvConstants[i].constant_id].value = spvc_constant_get_scalar_u32(
				spvc_compiler_get_constant_handle(this->spvcCompiler, spvConstants[i].id), 0, 0);
		}

		spvc_specialization_constant x, y, z;
		spvc_compiler_get_work_group_size_specialization_constants(this->spvcCompiler, &x, &y, &z);

		if (x.id != 0) {
			this->specConstants[x.constant_id].name = "local_size_x";
		}
		if (y.id != 0) {
			this->specConstants[y.constant_id].name = "local_size_y";
		}
		if (z.id != 0) {
			this->specConstants[z.constant_id].name = "local_size_z";
		}
	}

	return this->specConstants;
}

[[nodiscard]] vxr::std::array<vxr_vk_shader_reflectResult_constant, 3> reflector::getLocalSize() const noexcept {
	spvc_specialization_constant x, y, z;
	spvc_compiler_get_work_group_size_specialization_constants(this->spvcCompiler, &x, &y, &z);

	vxr::std::array<vxr_vk_shader_reflectResult_constant, 3> localSize = {};

	if (x.id != 0) {
		localSize[0].value = x.constant_id;
		localSize[0].isSpecConstant = VK_TRUE;
	} else {
		localSize[0].value = spvc_compiler_get_execution_mode_argument_by_index(this->spvcCompiler, SpvExecutionModeLocalSize, 0);
	}
	if (y.id != 0) {
		localSize[1].value = y.constant_id;
		localSize[1].isSpecConstant = VK_TRUE;
	} else {
		localSize[1].value = spvc_compiler_get_execution_mode_argument_by_index(this->spvcCompiler, SpvExecutionModeLocalSize, 1);
	}
	if (z.id != 0) {
		localSize[2].value = z.constant_id;
		localSize[2].isSpecConstant = VK_TRUE;
	} else {
		localSize[2].value = spvc_compiler_get_execution_mode_argument_by_index(this->spvcCompiler, SpvExecutionModeLocalSize, 2);
	}

	return localSize;
}

[[nodiscard]] uint32_t reflector::getNumOutputs(size_t i) noexcept {
	auto& entryPoint = this->entryPoints[i];
	switch (entryPoint.stage) {
		case VK_SHADER_STAGE_FRAGMENT_BIT: {
			auto ret = spvc_compiler_set_entry_point(this->spvcCompiler, entryPoint.name.cStr(), SpvExecutionModelFragment);
			if (ret != SPVC_SUCCESS) {
				vxr::std::ePrintf("Failed to set entry point: %s", spvc_context_get_last_error_string(this->spvcContext));
				vxr::std::abort();
			}
		} break;

		default:
			vxr::std::ePrintf("Failed to get stage outputs: unknown/unimplemented shader stage: %d", entryPoint.stage);
			vxr::std::abort();
			break;
	}

	if (entryPoint.spvcResources == nullptr) {
		spvc_set set;
		auto ret = spvc_compiler_get_active_interface_variables(this->spvcCompiler, &set);
		if (ret != SPVC_SUCCESS) {
			vxr::std::ePrintf("Failed to create spvc resources: %s", spvc_context_get_last_error_string(this->spvcContext));
			vxr::std::abort();
		}

		ret = spvc_compiler_create_shader_resources_for_active_variables(this->spvcCompiler, &entryPoint.spvcResources, set);
		if (ret != SPVC_SUCCESS) {
			vxr::std::ePrintf("Failed to create spvc resources: %s", spvc_context_get_last_error_string(this->spvcContext));
			vxr::std::abort();
		}
	}

	const spvc_reflected_resource* resource;
	size_t count;
	auto ret = spvc_resources_get_resource_list_for_type(
		entryPoint.spvcResources, SPVC_RESOURCE_TYPE_STAGE_OUTPUT, &resource, &count);
	if (ret != SPVC_SUCCESS) {
		vxr::std::ePrintf("Failed to get stage outputs: %s", spvc_context_get_last_error_string(this->spvcContext));
		vxr::std::abort();
	}

	uint32_t maxLocation = 0;
	for (size_t i = 0; i < count; i++) {
		maxLocation = vxr::std::max(
			maxLocation, spvc_compiler_get_decoration(this->spvcCompiler, resource[i].id, SpvDecorationLocation) + 1);
	}
	return maxLocation;
}

[[nodiscard]] VkPushConstantRange reflector::getPushConstantRange() noexcept {
	if (this->spvcResources == nullptr) {
		auto ret = spvc_compiler_create_shader_resources(this->spvcCompiler, &this->spvcResources);
		if (ret != SPVC_SUCCESS) {
			vxr::std::ePrintf("Failed to create spvc resources: %s", spvc_context_get_last_error_string(this->spvcContext));
			vxr::std::abort();
		}
	}

	const spvc_reflected_resource* resource;
	size_t count;
	auto ret = spvc_resources_get_resource_list_for_type(this->spvcResources, SPVC_RESOURCE_TYPE_PUSH_CONSTANT, &resource, &count);
	if (ret != SPVC_SUCCESS) {
		vxr::std::ePrintf("Failed to get push constants: %s", spvc_context_get_last_error_string(this->spvcContext));
		vxr::std::abort();
	}
	if (count != 0) {
		size_t sz;
		ret = spvc_compiler_get_declared_struct_size(
			this->spvcCompiler, spvc_compiler_get_type_handle(this->spvcCompiler, resource->base_type_id), &sz);
		if (ret != SPVC_SUCCESS) {
			vxr::std::ePrintf("Failed to get push constants: %s", spvc_context_get_last_error_string(this->spvcContext));
			vxr::std::abort();
		}
		const size_t off = spvc_compiler_get_member_decoration(this->spvcCompiler, resource->base_type_id, 0, SpvDecorationOffset);

		return {.offset = static_cast<uint32_t>(off), .size = static_cast<uint32_t>(sz - off)};
	}

	return {};
}

// NOLINTNEXTLINE(readability-function-cognitive-complexity)
[[nodiscard]] const vxr::std::vector<vxr::std::vector<reflector::binding>>& reflector::getDescriptorSets() noexcept {
	if (this->spvcResources == nullptr) {
		auto ret = spvc_compiler_create_shader_resources(this->spvcCompiler, &this->spvcResources);
		if (ret != SPVC_SUCCESS) {
			vxr::std::ePrintf("Failed to create spvc resources: %s", spvc_context_get_last_error_string(this->spvcContext));
			vxr::std::abort();
		}
	}

	if (this->descriptorSets.size() == 0) {
		static constexpr vxr::std::array resourceBindingTypes = {
			SPVC_RESOURCE_TYPE_UNIFORM_BUFFER, SPVC_RESOURCE_TYPE_STORAGE_BUFFER, SPVC_RESOURCE_TYPE_STORAGE_IMAGE,
			SPVC_RESOURCE_TYPE_SAMPLED_IMAGE,  SPVC_RESOURCE_TYPE_SEPARATE_IMAGE, SPVC_RESOURCE_TYPE_SEPARATE_SAMPLERS,
		};
		for (auto resourceType : resourceBindingTypes) {
			const spvc_reflected_resource* resourceList;
			size_t count;
			auto ret = spvc_resources_get_resource_list_for_type(this->spvcResources, resourceType, &resourceList, &count);
			if (ret != SPVC_SUCCESS) {
				vxr::std::ePrintf("Failed to get resource list [%d]: %s", resourceType,
								  spvc_context_get_last_error_string(this->spvcContext));
				vxr::std::abort();
			}

			for (size_t i = 0; i < count; i++) {
				const auto r = resourceList[i];
				const uint32_t set = spvc_compiler_get_decoration(this->spvcCompiler, r.id, SpvDecorationDescriptorSet);
				const uint32_t binding = spvc_compiler_get_decoration(this->spvcCompiler, r.id, SpvDecorationBinding);

				this->descriptorSets.resize(vxr::std::max<size_t>(this->descriptorSets.size(), set + 1));
				this->descriptorSets[set].resize(vxr::std::max<size_t>(this->descriptorSets[set].size(), binding + 1));
				const auto type = spvcResrouceToVkDescriptor(resourceType, this->spvcCompiler, r.type_id);
				if (this->descriptorSets[set][binding].type != VK_DESCRIPTOR_TYPE_MAX_ENUM &&
					this->descriptorSets[set][binding].type != type) {
					vxr::std::ePrintf("Aliased binding [set: %d, binding: %d] must have a consistent VkDescriptorType", set, binding);
					vxr::std::abort();
				}
				this->descriptorSets[set][binding].type = type;

				const spvc_type t = spvc_compiler_get_type_handle(this->spvcCompiler, r.type_id);
				const uint32_t d = spvc_type_get_num_array_dimensions(t);

				switch (d) {
					case 0: {
						const auto currentValue = this->descriptorSets[set][binding].count.value;
						if ((currentValue != 0) && (currentValue != 1)) {
							vxr::std::ePrintf("Aliased binding [set: %d, binding: %d] must have a consistent length", set, binding);
							vxr::std::abort();
						}
						this->descriptorSets[set][binding].count.value = 1;
					} break;

					case 1: {
						const auto currentValue = this->descriptorSets[set][binding].count.value;
						const auto count = spvc_type_get_array_dimension(t, 0);
						if (spvc_type_array_dimension_is_literal(t, 0) == SPVC_TRUE) {
							if (count == 0) {
								vxr::std::ePrintf("Variable descriptor count bindings are not implemented");
								vxr::std::abort();
							}
							if ((currentValue != 0) &&
								((currentValue != count) || (this->descriptorSets[set][binding].count.isSpecConstant != VK_FALSE))) {
								vxr::std::ePrintf(
									"Aliased binding [set: %d, binding: %d] must have a consistent length", set, binding);
								vxr::std::abort();
							}
							this->descriptorSets[set][binding].count.value = count;
						} else {
							const auto sid = spvc_compiler_get_decoration(this->spvcCompiler, count, SpvDecorationSpecId);
							if ((currentValue != 0) &&
								((currentValue != sid) || (this->descriptorSets[set][binding].count.isSpecConstant != VK_TRUE))) {
								vxr::std::ePrintf(
									"Aliased binding [set: %d, binding: %d] must have a consistent spec constant id", set, binding);
								vxr::std::abort();
							}
							this->descriptorSets[set][binding].count.value = sid;
							this->descriptorSets[set][binding].count.isSpecConstant = VK_TRUE;
						}
					} break;

					default: {
						vxr::std::ePrintf("Multi dimensional descriptor arrays are not implemented");
						vxr::std::abort();
					}
				}

				switch (this->descriptorSets[set][binding].type) {
					case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
					case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
						this->descriptorSets[set][binding].aliases.pushBack({
							.buffer = reflectBuffer(this->spvcContext, this->spvcCompiler, r),
						});
						break;

					case VK_DESCRIPTOR_TYPE_SAMPLER:
						this->descriptorSets[set][binding].aliases.pushBack({
							.sampler = {.name = r.name},
						});
						break;

					case VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
					case VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
					case VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
						this->descriptorSets[set][binding].aliases.pushBack({
							.image = reflectImage(this->spvcContext, this->spvcCompiler, r),
						});
						break;

					default:
						break;
				}
			}
		}
	}

	return this->descriptorSets;
}
}  // namespace vxr::vk::shader
