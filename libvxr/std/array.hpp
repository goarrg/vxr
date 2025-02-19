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
#include "utility.hpp"
#include "stdlib.hpp"
#include "concepts.hpp"

namespace vxr::std {
template <typename T, size_t N>
class array {
   private:
	T data[N] = {};	 // NOLINT(modernize-avoid-c-arrays)

   public:
	constexpr array() noexcept = default;
	constexpr array(const array& other) noexcept = default;
	constexpr array(array&& other) noexcept = default;
	constexpr array& operator=(const array& other) noexcept = default;
	constexpr array& operator=(array&& other) noexcept = default;

	constexpr array(const T (&data)[N]) noexcept {	// NOLINT(modernize-avoid-c-arrays)
		for (size_t i = 0; i < N; i++) {
			this->data[i] = data[i];
		}
	}

	template <typename... Args>
	constexpr array(Args&&... args) noexcept
		requires(assignable<T, Args> && ...)
		: data{move(args)...} {
		static_assert(sizeof...(Args) == N);
	}

	[[nodiscard]] constexpr size_t size() const noexcept { return N; }
	[[nodiscard]] constexpr const T* get() const noexcept { return &data[0]; }
	[[nodiscard]] constexpr const T* begin() const noexcept { return &data[0]; }
	[[nodiscard]] constexpr const T* end() const noexcept { return begin() + size(); }

	[[nodiscard]] T* get() noexcept { return &data[0]; }
	[[nodiscard]] T* begin() noexcept { return &data[0]; }
	[[nodiscard]] T* end() noexcept { return begin() + size(); }

	[[nodiscard]] T& operator[](size_t i) noexcept {
		if (!inRange<size_t>(i, 0, N - 1)) {
			abort("Index out of bounds");
		}

		return data[i];
	}
	[[nodiscard]] constexpr const T& operator[](size_t i) const noexcept {
		if (!inRange<size_t>(i, 0, N - 1)) {
			abort("Index out of bounds");
		}

		return data[i];
	}
};

template <typename T, typename... Ts>
array(T, Ts...) -> array<T, sizeof...(Ts) + 1>;
}  // namespace vxr::std
