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

#include "vk/vk.hpp"

using shadercCompilationResult = struct shaderc_compilation_result*;
using shadercCompiler = struct shaderc_compiler*;
using shadercCompileOptions = struct shaderc_compile_options*;

namespace vxr::vk::shader {
class compiler {
   private:
	::shadercCompiler shadercCompiler;
	::shadercCompileOptions shadercOptions;

   public:
	class result {
	   private:
		shadercCompilationResult shadercResult = nullptr;

	   public:
		result() noexcept = default;
		result(shadercCompilationResult shaderc) noexcept : shadercResult(shaderc) {}
		constexpr result(result&& other) noexcept {
			this->shadercResult = other.shadercResult;
			other.shadercResult = nullptr;
		}
		~result() noexcept;

		[[nodiscard]] size_t len() const noexcept;
		[[nodiscard]] const uint32_t* get() const noexcept;
	};

	compiler(uint32_t) noexcept;
	~compiler() noexcept;

	[[nodiscard]] result compile(vxr_vk_shader_compileInfo) const noexcept;
};
}  // namespace vxr::vk::shader
