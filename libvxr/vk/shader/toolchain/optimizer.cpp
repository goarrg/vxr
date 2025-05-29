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

#include <stddef.h>
#include <stdint.h>
#include <spirv-tools/libspirv.h>

#include "std/stdlib.hpp"
#include "std/log.hpp"

#include "vxr/vxr.h"
#include "vk/shader/toolchain/optimizer.hpp"
#include "vk/shader/toolchain/compiler.hpp"

static void spvLogger(spv_message_level_t level, const char* /*file*/, const spv_position_t* /*pos*/, const char* msg) {
	switch (level) {
		case SPV_MSG_FATAL:
		case SPV_MSG_INTERNAL_ERROR:
		case SPV_MSG_ERROR:
			vxr::std::ePrintf(msg);
			break;
		case SPV_MSG_WARNING:
			vxr::std::wPrintf(msg);
			break;
		case SPV_MSG_INFO:
			vxr::std::iPrintf(msg);
			break;
		case SPV_MSG_DEBUG:
			vxr::std::vPrintf(msg);
			break;
	}
}

namespace vxr::vk::shader {
optimizer::optimizer(vxr_vk_shader_toolchainOptions options) noexcept {
	// Check to make sure we update the switch below.
	static_assert(VK_API_VERSION_1_4 == VXR_VK_MAX_API);  // NOLINT(misc-redundant-expression)

	spv_target_env env;
	switch (VK_VERSION_MINOR(options.api)) {
		case 0:
			env = SPV_ENV_VULKAN_1_0;
			break;
		case 1:
			env = SPV_ENV_VULKAN_1_1;
			break;
		case 2:
			env = SPV_ENV_VULKAN_1_2;
			break;
		case 3:
			env = SPV_ENV_VULKAN_1_3;
			break;
		case 4:
		default:
			env = SPV_ENV_VULKAN_1_4;
			break;
	}

	this->spvOptimizer = spvOptimizerCreate(env);
	this->spvOptions = spvOptimizerOptionsCreate();
	this->spvValidatorOptions = spvValidatorOptionsCreate();

	spvOptimizerSetMessageConsumer(this->spvOptimizer, spvLogger);

	spvValidatorOptionsSetSkipBlockLayout(this->spvValidatorOptions, true);
	spvValidatorOptionsSetRelaxLogicalPointer(this->spvValidatorOptions, true);
	spvValidatorOptionsSetBeforeHlslLegalization(this->spvValidatorOptions, true);

	spvOptimizerOptionsSetValidatorOptions(this->spvOptions, this->spvValidatorOptions);
	spvOptimizerOptionsSetPreserveBindings(this->spvOptions, true);

	if (options.strip == VK_TRUE) {
		if (!spvOptimizerRegisterPassFromFlag(this->spvOptimizer, "--strip-debug")) {
			vxr::std::ePrintf("Failed to add strip-debug optimization pass");
			vxr::std::abort();
		}
		if (!spvOptimizerRegisterPassFromFlag(this->spvOptimizer, "--strip-nonsemantic")) {
			vxr::std::ePrintf("Failed to add strip-nonsemantic optimization pass");
			vxr::std::abort();
		}
	}
	if (options.optimizePerformance == VK_TRUE) {
		spvOptimizerRegisterPerformancePasses(this->spvOptimizer);
	}
	if (options.optimizeSize == VK_TRUE) {
		spvOptimizerRegisterSizePasses(this->spvOptimizer);
	}
}

optimizer::~optimizer() noexcept {
	spvOptimizerDestroy(this->spvOptimizer);
	spvOptimizerOptionsDestroy(this->spvOptions);
	spvValidatorOptionsDestroy(this->spvValidatorOptions);
}

optimizer::result optimizer::optimize(compiler::result& src) const noexcept {
	spv_binary result;
	auto ret = spvOptimizerRun(this->spvOptimizer, src.get(), src.len(), &result, this->spvOptions);
	if (ret != SPV_SUCCESS) {
		vxr::std::ePrintf("Failed to optimize shader: %d", ret);
		vxr::std::abort();
	}
	return result;
}

optimizer::result::~result() noexcept {
	spvBinaryDestroy(this->spvResult);
}

size_t optimizer::result::len() const noexcept {
	return this->spvResult->wordCount;
}

const uint32_t* optimizer::result::get() const noexcept {
	return this->spvResult->code;
}
}  // namespace vxr::vk::shader
