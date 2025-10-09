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
#include <new>
#include <shaderc/env.h>
#include <shaderc/status.h>
#include <shaderc/shaderc.h>

#include "std/stdlib.hpp"
#include "std/defer.hpp"
#include "std/log.hpp"

#include "vk/vk.hpp"
#include "vk/shader/toolchain/compiler.hpp"

static shaderc_include_result* shadercIncludeResolver(void* data, const char* requestedSource, int type,
													  const char* requestingSource, size_t /*include_depth*/) {
	auto* info = reinterpret_cast<vxr_vk_shader_compileInfo*>(data);
	auto result = info->includeResolver(info->userdata, const_cast<char*>(requestedSource),
										static_cast<vxr_vk_shader_includeType>(type), const_cast<char*>(requestingSource));
	auto* shadercResult = new (::std::nothrow) shaderc_include_result();
	shadercResult->source_name = result.name;
	shadercResult->source_name_length = result.nameSize;
	shadercResult->content = reinterpret_cast<const char*>(result.content);	 // NOLINT(performance-no-int-to-ptr)
	shadercResult->content_length = result.contentSize;
	shadercResult->user_data = reinterpret_cast<void*>(result.userdata);  // NOLINT(performance-no-int-to-ptr)
	return shadercResult;
}

static void shadercIncludeResultRelease(void* data, shaderc_include_result* shadercResult) {
	auto* info = reinterpret_cast<vxr_vk_shader_compileInfo*>(data);
	const vxr_vk_shader_includeResult result = {
		.nameSize = shadercResult->source_name_length,
		.name = shadercResult->source_name,
		.contentSize = shadercResult->content_length,
		.content = reinterpret_cast<uintptr_t>(shadercResult->content),
		.userdata = reinterpret_cast<uintptr_t>(shadercResult->user_data),
	};
	info->resultReleaser(info->userdata, result);
	delete shadercResult;
}

namespace vxr::vk::shader {
compiler::compiler(vxr_vk_shader_toolchainOptions options) noexcept {
	this->shadercCompiler = shaderc_compiler_initialize();
	this->shadercOptions = shaderc_compile_options_initialize();

	shaderc_compile_options_set_target_env(this->shadercOptions, shaderc_target_env_vulkan, options.api);
	shaderc_compile_options_set_warnings_as_errors(this->shadercOptions);
	shaderc_compile_options_set_preserve_bindings(this->shadercOptions, true);

	if (options.strip == VK_FALSE) {
		shaderc_compile_options_set_generate_debug_info(this->shadercOptions);
	}
}

compiler::~compiler() noexcept {
	shaderc_compiler_release(this->shadercCompiler);
	shaderc_compile_options_release(this->shadercOptions);
}

compiler::result compiler::compile(vxr_vk_shader_compileInfo info) const noexcept {
	auto* options = shaderc_compile_options_clone(this->shadercOptions);
	DEFER([&] { shaderc_compile_options_release(options); });
	shaderc_compile_options_set_include_callbacks(options, shadercIncludeResolver, shadercIncludeResultRelease, &info);
	for (size_t i = 0; i < info.numMacros; i++) {
		const auto& m = info.macros[i];
		shaderc_compile_options_add_macro_definition(options, m.name, m.nameSize, m.value, m.valueSize);
	}
	shaderc_compilation_result_t result = shaderc_compile_into_spv(
		// NOLINTNEXTLINE(performance-no-int-to-ptr)
		this->shadercCompiler, reinterpret_cast<const char*>(info.content), info.contentSize,
		shaderc_glsl_infer_from_source, info.name, "main", options);
	const shaderc_compilation_status status = shaderc_result_get_compilation_status(result);
	const char* err = "unknown_error";
	switch (status) {
		case shaderc_compilation_status_success:
			return result;

		case shaderc_compilation_status_invalid_stage:
			err = "invalid_stage";
			break;
		case shaderc_compilation_status_compilation_error:
			err = "compilation_error";
			break;
		case shaderc_compilation_status_internal_error:
			err = "internal_error";
			break;
		case shaderc_compilation_status_null_result_object:
			err = "null_result_object";
			break;
		case shaderc_compilation_status_invalid_assembly:
			err = "invalid_assembly";
			break;
		case shaderc_compilation_status_validation_error:
			err = "validation_error";
			break;
		case shaderc_compilation_status_transformation_error:
			err = "transformation_error";
			break;
		case shaderc_compilation_status_configuration_error:
			err = "configuration_error";
			break;
	}
	vxr::std::ePrintf("Failed to compile shader: (%d: %s) %s", status, err, shaderc_result_get_error_message(result));
	vxr::std::abort();
	return result;
}

compiler::result::~result() noexcept {
	shaderc_result_release(this->shadercResult);
}

size_t compiler::result::len() const noexcept {
	return shaderc_result_get_length(this->shadercResult) / sizeof(uint32_t);
}

const uint32_t* compiler::result::get() const noexcept {
	return reinterpret_cast<const uint32_t*>(shaderc_result_get_bytes(this->shadercResult));
}
}  // namespace vxr::vk::shader
