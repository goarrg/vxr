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

#include "vk/shader/toolchain/compiler.hpp"

using spvOptimizer = struct spv_optimizer_t*;
using spvOptimizerOptions = struct spv_optimizer_options_t*;
using spvValidatorOptions = struct spv_validator_options_t*;
using spvBinary = struct spv_binary_t*;

namespace vxr::vk::shader {
class optimizer {
   private:
	::spvOptimizer spvOptimizer;
	::spvOptimizerOptions spvOptions;
	::spvValidatorOptions spvValidatorOptions;

   public:
	class result {
	   private:
		spvBinary spvResult;

	   public:
		result() noexcept = default;
		result(spvBinary spvResult) noexcept : spvResult(spvResult) {}
		constexpr result(result&& other) noexcept {
			this->spvResult = other.spvResult;
			other.spvResult = nullptr;
		}
		~result() noexcept;

		[[nodiscard]] size_t len() const noexcept;
		[[nodiscard]] const uint32_t* get() const noexcept;
	};

	optimizer(vxr_vk_shader_toolchainOptions) noexcept;
	~optimizer() noexcept;

	[[nodiscard]] result optimize(compiler::result&) const noexcept;
};
}  // namespace vxr::vk::shader
