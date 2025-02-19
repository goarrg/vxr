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
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <type_traits>

#include "concepts.hpp"	 // IWYU pragma: keep
#include "stdlib.hpp"
#include "utility.hpp"

#define MACRO_TOSTRING2(x) #x
#define MACRO_TOSTRING(x) MACRO_TOSTRING2(x)

namespace vxr::std {
// clang-format off
template <typename T>
concept character =
	::std::is_same_v<T, char> ||
	// same_as<T, signed char> ||
	// same_as<T, unsigned char> ||
	::std::is_same_v<T, char8_t>;
	// same_as<T, char16_t> ||
	// same_as<T, char32_t>;
// clang-format on

template <character T>
constexpr size_t strlen(const T* const str) noexcept {
	if (str == nullptr) {
		return 0;
	}

	size_t sz = 0;
	for (; str[sz] != '\0'; ++sz) {
	}
	return sz;
}

template <character T>
constexpr size_t strnlen(const T* const str, const size_t n) noexcept {
	if (str == nullptr) {
		return 0;
	}

	size_t sz = 0;
	for (; sz <= n && str[sz] != '\0'; ++sz) {
	}
	return sz;
}

template <character T>
constexpr T* strncpy(T* dst, const T* src, size_t n) noexcept {
	if (!(src && dst)) {
		abort("Null args");
	}

	T* ptr = dst;
	while (n-- && (*dst++ = *src++)) {	// NOLINT(clang-analyzer-core.NullDereference)
	}
	return ptr;
}

template <character T = char>
class stringbuilder;

template <character T = char>
class string {
   private:
	friend stringbuilder<T>;

	size_t len = 0;
	T* str = nullptr;

	void copy(size_t n, const T* val) noexcept {
		if (n == 0) {
			return;
		}
		if (val == nullptr) {
			abort("Unexpected nullptr");
			return;
		}
		const size_t sz = (n + 1) * sizeof(T);
		if (this->len != n) {
			T* newptr = static_cast<T*>(realloc(this->str, sz));
			if (newptr == nullptr) {
				abort("Failed realloc");
			}
			this->str = newptr;
		}
		if (this->str != nullptr) {
			memcpy(this->str, val, sz - sizeof(T));
			this->len = n;
			this->str[this->len] = 0;
		}
	}

   public:
	constexpr string() noexcept = default;
	string(const T* val) noexcept : string(strlen(val), val) {}
	string(size_t n, const T* val) noexcept { copy(n, val); }
	string(const string& other) noexcept { copy(other.len, other.str); }
	constexpr string(string&& other) noexcept { *this = vxr::std::move(other); }
	~string() noexcept {
		free(this->str);
		this->len = 0;
		this->str = nullptr;
	}
	[[nodiscard]] constexpr operator const T*() const { return this->str; }
	[[nodiscard]] constexpr size_t size() const noexcept { return this->len; }
	[[nodiscard]] constexpr const T* cStr() const noexcept { return this->str; }
	string& operator=(const T* val) noexcept {
		copy(strlen(val), val);
		return *this;
	}
	string& operator=(const string& val) noexcept {
		if (this != &val) {
			copy(val.len, val.str);
		}
		return *this;
	}
	constexpr string& operator=(string&& other) noexcept {
		this->len = other.len;
		this->str = other.str;

		other.len = 0;
		other.str = nullptr;
		return *this;
	}
	template <character U>
	[[nodiscard]] bool operator==(const U* u) const {
		return strncmp(this->str, u, this->len) == 0;
	}
	template <character U>
	[[nodiscard]] bool operator!=(const U* u) const {
		return strncmp(this->str, u, this->len) != 0;
	}
};

template <character T>
class stringbuilder {
   private:
	size_t len = 0;
	size_t cap = 0;
	T* ptr = nullptr;

	void grow(size_t n) noexcept {
		static constexpr size_t blockSize = 256;
		this->cap = ((this->cap + n + blockSize - 1) / blockSize) * blockSize;
		const size_t sz = this->cap * sizeof(T);
		T* newptr = static_cast<T*>(realloc(this->ptr, sz));
		if (newptr == nullptr) {
			abort("Failed realloc");
			return;
		}
		this->ptr = newptr;
	}

   public:
	stringbuilder(const stringbuilder&) = delete;
	stringbuilder& operator=(const stringbuilder&) = delete;

	constexpr stringbuilder() noexcept = default;
	constexpr stringbuilder(string<T>&& value) noexcept {
		this->len = this->cap = value.len;
		this->ptr = value.str;

		value.len = 0;
		value.str = nullptr;
	}
	~stringbuilder() noexcept { free(this->ptr); }
	[[nodiscard]] constexpr string<T> str() const noexcept { return string<T>(this->len, this->ptr); }
	[[nodiscard]] constexpr const T* cStr() const noexcept { return this->ptr; }

	stringbuilder& reset() noexcept {
		this->len = 0;
		this->ptr[this->len] = 0;
		return *this;
	}

	stringbuilder& write(const size_t valueLen, const T* value) noexcept {
		if (valueLen == 0) {
			return *this;
		}

		const ptrdiff_t space = (this->cap - this->len);
		const ptrdiff_t diff = valueLen + 1 - space;
		if (diff > 0) {
			grow(diff);
		}
		if (this->ptr == nullptr) {
			abort("Unexpected nullptr");
			return *this;
		}
		strncpy(this->ptr + this->len, value, valueLen);
		this->len += valueLen;
		this->ptr[this->len] = 0;
		return *this;
	}
	stringbuilder& write(const T* value) noexcept {
		const size_t len = strlen(value);
		return this->write(len, value);
	}
	template <typename... Args>
	stringbuilder& writef(const T* fmt, Args... args) noexcept {
		const ptrdiff_t space = (this->cap - this->len);
		const size_t n = snprintf(this->ptr + this->len, space, fmt, args...) + 1;
		const ptrdiff_t diff = n - space;
		if (diff > 0) {
			grow(diff);
			snprintf(this->ptr + this->len, n, fmt, args...);
		}
		this->len += n - 1;
		return *this;
	}
	stringbuilder& backspace(size_t n = 1) noexcept {
		if (this->len == 0) {
			return *this;
		}
		if (this->ptr == nullptr) {
			abort("Unexpected nullptr");
			return *this;
		}
		this->len -= min(n, this->len);
		this->ptr[this->len] = 0;
		return *this;
	}

	stringbuilder& operator<<(const T* value) noexcept {
		writef("%s", value);
		return *this;
	}
	stringbuilder& operator<<(const string<T>&& value) noexcept {
		writef("%s", value.c_str());
		return *this;
	}

	stringbuilder& operator<<(char value) noexcept {
		writef("%hhd", value);
		return *this;
	}
	stringbuilder& operator<<(unsigned char value) noexcept {
		writef("%hhu", value);
		return *this;
	}

	stringbuilder& operator<<(short value) noexcept {
		writef("%hd", value);
		return *this;
	}
	stringbuilder& operator<<(unsigned short value) noexcept {
		writef("%hu", value);
		return *this;
	}

	stringbuilder& operator<<(int value) noexcept {
		writef("%d", value);
		return *this;
	}
	stringbuilder& operator<<(unsigned int value) noexcept {
		writef("%u", value);
		return *this;
	}

	stringbuilder& operator<<(long value) noexcept {
		writef("%ld", value);
		return *this;
	}
	stringbuilder& operator<<(unsigned long value) noexcept {
		writef("%lu", value);
		return *this;
	}

	stringbuilder& operator<<(long long value) noexcept {
		writef("%lld", value);
		return *this;
	}
	stringbuilder& operator<<(unsigned long long value) noexcept {
		writef("%llu", value);
		return *this;
	}

	stringbuilder& operator<<(float value) noexcept {
		writef("%f", value);
		return *this;
	}
	stringbuilder& operator<<(double value) noexcept {
		writef("%lf", value);
		return *this;
	}
	stringbuilder& operator<<(long double value) noexcept {
		writef("%Lf", value);
		return *this;
	}

	stringbuilder& operator<<(bool value) noexcept {
		return value ? this << static_cast<const T*>("true") : this << static_cast<const T*>("false");
	}
};
}  // namespace vxr::std
