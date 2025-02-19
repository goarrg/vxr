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

#include "std/string.hpp"
#include "std/array.hpp"
#include "std/vector.hpp"

#include "vxr/vxr.h"

using spvcContext = struct spvc_context_s*;
using spvcCompiler = struct spvc_compiler_s*;
using spvcResources = struct spvc_resources_s*;

namespace vxr::vk::shader {
class reflector {
   protected:
	::spvcContext spvcContext;
	::spvcCompiler spvcCompiler;
	::spvcResources spvcResources;

   public:
	struct entryPoint {
		vxr::std::string<char> name;
		VkShaderStageFlagBits stage;
		::spvcResources spvcResources;
	};
	struct specConstant {
		vxr::std::string<char> name;
		uint32_t value;
	};
	struct binding {
		VkDescriptorType type;
		vxr_vk_shader_reflectResult_constant count;

		binding() {
			type = VK_DESCRIPTOR_TYPE_MAX_ENUM;
			count = {};
		}

		union metadata {
			vxr_vk_shader_reflectResult_bufferMetadata buffer;
			vxr_vk_shader_reflectResult_imageMetadata image;
			vxr_vk_shader_reflectResult_samplerMetadata sampler;
		};

		vxr::std::vector<metadata> aliases;
	};

   private:
	vxr::std::vector<entryPoint> entryPoints;
	vxr::std::vector<specConstant> specConstants;
	vxr::std::vector<vxr::std::vector<binding>> descriptorSets;

   public:
	reflector() noexcept = delete;
	reflector(reflector&&) noexcept = delete;

	reflector(vxr_vk_shader_spirv) noexcept;
	~reflector() noexcept;

	[[nodiscard]] const vxr::std::vector<entryPoint>& getEntryPoints() noexcept;
	[[nodiscard]] const vxr::std::vector<specConstant>& getSpecConstants() noexcept;
	[[nodiscard]] vxr::std::array<vxr_vk_shader_reflectResult_constant, 3> getLocalSize() const noexcept;
	[[nodiscard]] uint32_t getNumOutputs(size_t) noexcept;
	[[nodiscard]] VkPushConstantRange getPushConstantRange() noexcept;
	[[nodiscard]] const vxr::std::vector<vxr::std::vector<binding>>& getDescriptorSets() noexcept;

	[[nodiscard]] vxr_vk_shader_reflectResult handle() noexcept {
		return reinterpret_cast<vxr_vk_shader_reflectResult>(this);
	}
	[[nodiscard]] static reflector* fromHandle(vxr_vk_shader_reflectResult handle) noexcept {
		return reinterpret_cast<reflector*>(handle);
	}
};
}  // namespace vxr::vk::shader
