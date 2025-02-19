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

#define VXR_LOG_LEVEL_VERBOSE 0
#define VXR_LOG_LEVEL_INFO 1
#define VXR_LOG_LEVEL_WARN 2
#define VXR_LOG_LEVEL_ERROR 3

#ifndef VXR_LOG_LEVEL
#ifdef NDEBUG
#define VXR_LOG_LEVEL VXR_LOG_LEVEL_WARN
#else
#define VXR_LOG_LEVEL VXR_LOG_LEVEL_VERBOSE
#endif
#endif

#include <stdio.h>

#include "stdlib.hpp"
#include "array.hpp"
#include "vector.hpp"
#include "string.hpp"

namespace vxr::std {
template <typename... T>
inline static void vPrintf([[maybe_unused]] const char* fmt, [[maybe_unused]] T... args) noexcept {
#if VXR_LOG_LEVEL <= VXR_LOG_LEVEL_VERBOSE
	if constexpr (sizeof...(T) > 0) {
		//+1 for null terminator
		const int n = snprintf(nullptr, 0, fmt, args...) + 1;
		vxr::std::vector<char> buf(n);
		snprintf(buf.get(), n, fmt, args...);

		//-1 to get strlen
		internal::callbackV(n - 1, buf.get());
	} else {
		internal::callbackV(strlen(fmt), const_cast<char*>(fmt));
	}
#endif
}

template <typename... T>
inline static void iPrintf([[maybe_unused]] const char* fmt, [[maybe_unused]] T... args) noexcept {
#if VXR_LOG_LEVEL <= VXR_LOG_LEVEL_INFO
	if constexpr (sizeof...(T) > 0) {
		//+1 for null terminator
		const int n = snprintf(nullptr, 0, fmt, args...) + 1;
		vxr::std::vector<char> buf(n);
		snprintf(buf.get(), n, fmt, args...);

		//-1 to get strlen
		internal::callbackI(n - 1, buf.get());
	} else {
		internal::callbackI(strlen(fmt), const_cast<char*>(fmt));
	}
#endif
}

template <typename... T>
inline static void wPrintf([[maybe_unused]] const char* fmt, [[maybe_unused]] T... args) noexcept {
#if VXR_LOG_LEVEL <= VXR_LOG_LEVEL_WARN
	if constexpr (sizeof...(T) > 0) {
		//+1 for null terminator
		const int n = snprintf(nullptr, 0, fmt, args...) + 1;
		vxr::std::vector<char> buf(n);
		snprintf(buf.get(), n, fmt, args...);

		//-1 to get strlen
		internal::callbackW(n - 1, buf.get());
	} else {
		internal::callbackW(strlen(fmt), const_cast<char*>(fmt));
	}
#endif
}

template <typename... T>
inline static void ePrintf(const char* fmt, T... args) noexcept {
	if constexpr (sizeof...(T) > 0) {
		//+1 for null terminator
		const int n = snprintf(nullptr, 0, fmt, args...) + 1;
		vxr::std::vector<char> buf(n);
		snprintf(buf.get(), n, fmt, args...);

		//-1 to get strlen
		internal::callbackE(n - 1, buf.get());
	} else {
		internal::callbackE(strlen(fmt), const_cast<char*>(fmt));
	}
}

template <typename... T>
inline static void abortPopup(sourceLocation loc, const char* fmt, T... args) noexcept {
	static constexpr const char* locFmt = "Fatal Error At: %s %s:%d";

	if constexpr (sizeof...(T) > 0) {
		const int locSz = snprintf(nullptr, 0, locFmt, loc.func, loc.file, loc.line);
		const int msgSz = snprintf(nullptr, 0, fmt, args...);
		vxr::std::vector<char> buf(locSz + msgSz + 1);
		snprintf(buf.get(), locSz + 1, locFmt, loc.func, loc.file, loc.line);
		snprintf(buf.get() + locSz, msgSz + 1, fmt, args...);

		internal::callbackE(locSz, buf.get());
		internal::callbackAbortPopup(msgSz, buf.get() + locSz);
	} else {
		vxr::std::array<char, 1024> buf;
		const int locSz = snprintf(buf.get(), buf.size(), locFmt, loc.func, loc.file, loc.line);
		internal::callbackE(locSz, buf.get());
		internal::callbackAbortPopup(strlen(fmt), const_cast<char*>(fmt));
	}
}
}  // namespace vxr::std
