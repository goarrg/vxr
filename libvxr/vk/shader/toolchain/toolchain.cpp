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
#include <new>

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/utility.hpp"
#include "std/algorithm.hpp"

#include "vk/shader/toolchain/toolchain.hpp"

extern "C" {
VXR_FN void vxr_vk_shader_initToolchain(vxr_vk_shader_toolchainOptions options, vxr_vk_shader_toolchain* toolchainHandle) {
	auto* toolchain = new (::std::nothrow) vxr::vk::shader::toolchain(options);
	*toolchainHandle = toolchain->handle();
}
VXR_FN void vxr_vk_shader_destroyToolchain(vxr_vk_shader_toolchain handle) {
	auto* toolchain = vxr::vk::shader::toolchain::fromHandle(handle);
	delete toolchain;
}
VXR_FN void vxr_vk_shader_compile(vxr_vk_shader_toolchain toolchainHandle, vxr_vk_shader_compileInfo info,
								  vxr_vk_shader_compileResult* resultHandle, vxr_vk_shader_reflectResult* reflectionHandle) {
	auto* toolchain = vxr::vk::shader::toolchain::fromHandle(toolchainHandle);
	auto* result = toolchain->compile(info);
	(*resultHandle) = result->handle();
	(*reflectionHandle) = result->reflection.handle();
}
VXR_FN void vxr_vk_shader_destroyCompileResult(vxr_vk_shader_compileResult resultHandle) {
	auto* result = vxr::vk::shader::toolchain::compileResult::fromHandle(resultHandle);
	delete result;
}
VXR_FN void vxr_vk_shader_compileResult_getSPIRV(vxr_vk_shader_compileResult resultHandle, vxr_vk_shader_spirv* spirv) {
	auto* result = vxr::vk::shader::toolchain::compileResult::fromHandle(resultHandle);

	spirv->len = result->spirv.len();
	spirv->data = result->spirv.get();
}
VXR_FN void vxr_vk_shader_reflect(vxr_vk_shader_toolchain, vxr_vk_shader_spirv spirv, vxr_vk_shader_reflectResult* resultHandle) {
	auto* result = new (::std::nothrow) vxr::vk::shader::reflector(spirv);
	(*resultHandle) = result->handle();
}
VXR_FN void vxr_vk_shader_destroyReflectResult(vxr_vk_shader_reflectResult resultHandle) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);
	delete result;
}
VXR_FN void vxr_vk_shader_reflectResult_getEntryPoints(vxr_vk_shader_reflectResult resultHandle,
													   size_t* entryPointsCount, vxr_vk_shader_entryPoint* entryPoints) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);

	if (entryPoints != nullptr) {
		for (size_t i = 0; i < vxr::std::min(*entryPointsCount, result->getEntryPoints().size()); i++) {
			entryPoints[i] = vxr_vk_shader_entryPoint{
				result->getEntryPoints()[i].name.size(),
				result->getEntryPoints()[i].name.cStr(),
				result->getEntryPoints()[i].stage,
			};
		}
	} else {
		*entryPointsCount = result->getEntryPoints().size();
	}
}
VXR_FN void vxr_vk_shader_reflectResult_getSpecConstants(vxr_vk_shader_reflectResult resultHandle, uint32_t* specConstantCount,
														 vxr_vk_shader_reflectResult_specConstant* specConstants) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);

	if (specConstants != nullptr) {
		for (uint32_t i = 0; i < vxr::std::min<uint32_t>(*specConstantCount, result->getSpecConstants().size()); i++) {
			specConstants[i] = vxr_vk_shader_reflectResult_specConstant{
				result->getSpecConstants()[i].name.cStr(),
				result->getSpecConstants()[i].value,
			};
		}
	} else {
		*specConstantCount = result->getSpecConstants().size();
	}
}
VXR_FN void vxr_vk_shader_reflectResult_getLocalSize(vxr_vk_shader_reflectResult resultHandle,
													 vxr_vk_shader_reflectResult_constant (*output)[3]) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);
	auto sizes = result->getLocalSize();
	vxr::std::copy(sizes.begin(), sizes.end(), *output);
}
VXR_FN void vxr_vk_shader_reflectResult_getNumOutputs(vxr_vk_shader_reflectResult resultHandle, size_t e, uint32_t* numOutputs) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);
	*numOutputs = result->getNumOutputs(e);
}
VXR_FN void vxr_vk_shader_reflectResult_getPushConstantRange(vxr_vk_shader_reflectResult resultHandle, VkPushConstantRange* range) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);
	*range = result->getPushConstantRange();
}
VXR_FN void vxr_vk_shader_reflectResult_getDescriptorSetSizes(vxr_vk_shader_reflectResult resultHandle, uint32_t* sz, uint32_t* setSizes) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);

	if (setSizes != nullptr) {
		for (uint32_t i = 0; i < vxr::std::min<uint32_t>(*sz, result->getDescriptorSets().size()); i++) {
			setSizes[i] = result->getDescriptorSets()[i].size();
		}
	} else {
		*sz = result->getDescriptorSets().size();
	}
}
VXR_FN void vxr_vk_shader_reflectResult_getDescriptorSetBinding(vxr_vk_shader_reflectResult resultHandle, uint32_t set, uint32_t binding,
																vxr_vk_shader_reflectResult_descriptorSetBinding* info) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);
	const auto& b = result->getDescriptorSets()[set][binding];

	info->type = b.type;
	info->count = b.count;
	info->numAliases = b.aliases.size();
}
VXR_FN void vxr_vk_shader_reflectResult_getBufferMetadata(vxr_vk_shader_reflectResult resultHandle, uint32_t set, uint32_t binding,
														  uint32_t alias, vxr_vk_shader_reflectResult_bufferMetadata* info) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);
	const auto& b = result->getDescriptorSets()[set][binding];

	switch (b.type) {
		case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
		case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
			break;

		default:
			vxr::std::ePrintf("Set: %d binding: %d is not a buffer", set, binding);
			vxr::std::abort();
			break;
	}

	*info = b.aliases[alias].buffer;
}
VXR_FN void vxr_vk_shader_reflectResult_getSamplerMetadata(vxr_vk_shader_reflectResult resultHandle, uint32_t set, uint32_t binding,
														   uint32_t alias, vxr_vk_shader_reflectResult_samplerMetadata* info) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);
	const auto& b = result->getDescriptorSets()[set][binding];

	switch (b.type) {
		case VK_DESCRIPTOR_TYPE_SAMPLER:
			break;

		default:
			vxr::std::ePrintf("Set: %d binding: %d is not a image", set, binding);
			vxr::std::abort();
			break;
	}

	*info = b.aliases[alias].sampler;
}
VXR_FN void vxr_vk_shader_reflectResult_getImageMetadata(vxr_vk_shader_reflectResult resultHandle, uint32_t set, uint32_t binding,
														 uint32_t alias, vxr_vk_shader_reflectResult_imageMetadata* info) {
	auto* result = vxr::vk::shader::reflector::fromHandle(resultHandle);
	const auto& b = result->getDescriptorSets()[set][binding];

	switch (b.type) {
		case VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
		case VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
		case VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
			break;

		default:
			vxr::std::ePrintf("Set: %d binding: %d is not a image", set, binding);
			vxr::std::abort();
			break;
	}

	*info = b.aliases[alias].image;
}
}
