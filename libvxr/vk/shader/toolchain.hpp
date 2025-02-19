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

#include <stdint.h>
#include <new>	// IWYU pragma: keep

#include "std/utility.hpp"

#include "vk/vk.hpp"
#include "vk/shader/compiler.hpp"
#include "vk/shader/reflector.hpp"
#include "vk/shader/optimizer.hpp"

namespace vxr::vk::shader {
class toolchain {
   private:
	::vxr::vk::shader::compiler compiler;
	::vxr::vk::shader::optimizer optimizer;

   public:
	struct compileResult {
		optimizer::result spirv;
		reflector reflection;

		compileResult(vxr_vk_shader_spirv src, optimizer::result&& optimized) noexcept
			: spirv(vxr::std::move(optimized)), reflection(src) {}

		[[nodiscard]] vxr_vk_shader_compileResult handle() noexcept {
			return reinterpret_cast<vxr_vk_shader_compileResult>(this);
		}
		[[nodiscard]] static compileResult* fromHandle(vxr_vk_shader_compileResult handle) noexcept {
			return reinterpret_cast<compileResult*>(handle);
		}
	};

	toolchain(uint32_t vkVersion) noexcept : compiler(vkVersion), optimizer(vkVersion) {}
	~toolchain() noexcept = default;

	[[nodiscard]] compileResult* compile(vxr_vk_shader_compileInfo info) const noexcept {
		auto src = this->compiler.compile(info);

		return new (::std::nothrow) compileResult(
			vxr_vk_shader_spirv{
				.len = src.len(),
				.data = src.get(),
			},
			this->optimizer.optimize(src));
	}

	[[nodiscard]] vxr_vk_shader_toolchain handle() noexcept { return reinterpret_cast<vxr_vk_shader_toolchain>(this); }
	[[nodiscard]] static toolchain* fromHandle(vxr_vk_shader_toolchain handle) noexcept {
		return reinterpret_cast<toolchain*>(handle);
	}
};
}  // namespace vxr::vk::shader
