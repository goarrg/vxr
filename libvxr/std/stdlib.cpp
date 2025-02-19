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

#include <stdio.h>

#include "stdlib.hpp"

// TODO: C++23 https://en.cppreference.com/w/cpp/header/stacktrace

namespace vxr::std {
namespace internal {
static loggerCallback callbackAbort = nullptr;
loggerCallback callbackAbortPopup = nullptr;

loggerCallback callbackV = nullptr;
loggerCallback callbackI = nullptr;
loggerCallback callbackW = nullptr;
loggerCallback callbackE = nullptr;
}  // namespace internal
void abort(const char* msg, sourceLocation loc) noexcept {
	if (msg != nullptr) {
		size_t sz = 0;
		for (; msg[sz] != '\0'; ++sz) {
		}
		internal::callbackE(sz, const_cast<char*>(msg));
	}

	{
		static constexpr const char* locFmt = "Fatal Error At: %s %s:%d";
		char buf[1024];	 // NOLINT(modernize-avoid-c-arrays)
		const int locSz = snprintf(buf, sizeof(buf), locFmt, loc.func, loc.file, loc.line);
		internal::callbackAbort(locSz, buf);
	}
}
}  // namespace vxr::std
extern "C" {
VXR_FN void vxr_stdlib_init(vxr_loggerCallback callbackAbort, vxr_loggerCallback callbackAbortPopup, vxr_loggerCallback callbackV,
							vxr_loggerCallback callbackI, vxr_loggerCallback callbackW, vxr_loggerCallback callbackE) {
	vxr::std::internal::callbackAbort = callbackAbort;
	vxr::std::internal::callbackAbortPopup = callbackAbortPopup;

	vxr::std::internal::callbackV = callbackV;
	vxr::std::internal::callbackI = callbackI;
	vxr::std::internal::callbackW = callbackW;
	vxr::std::internal::callbackE = callbackE;
}
}
