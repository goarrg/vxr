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
#include <new>	// IWYU pragma: keep

#include "stdlib.hpp"
#include "utility.hpp"
#include "concepts.hpp"	 // IWYU pragma: keep
#include "array.hpp"

namespace vxr::std {
template <typename T>
class vector {
   private:
	size_t len = 0;
	size_t cap = 0;
	T* ptr = nullptr;

	static_assert((pointer<T>) || (constructible<T> && destructible<T>));

	void clear() noexcept {
		if constexpr (destructible<T>) {
			for (size_t i = 0; i < this->len; i++) {
				this->ptr[i].~T();
			}
		}
		this->len = 0;
	}

	void grow(size_t n) noexcept {
		const size_t targetCap = this->cap + n;
		const size_t doubleCap = this->cap + this->cap;

		if (targetCap > doubleCap) {
			return reserve(targetCap);
		}

		static constexpr size_t threshold = 256;
		if (targetCap < threshold) {
			return reserve(doubleCap);
		}

		size_t newCap = this->cap;
		do {
			newCap += (newCap + (3 * threshold)) >> 2;
		} while (newCap < targetCap);

		return reserve(newCap);
	}

   public:
	vector(const vector&) = delete;
	vector& operator=(const vector&) = delete;

	constexpr vector() noexcept = default;
	template <size_t N>
	constexpr vector(vxr::std::array<T, N>&& args) noexcept {
		this->reserve(N);
		for (auto& arg : args) {
			new (reinterpret_cast<void*>(&this->ptr[this->len++])) T(vxr::std::move(arg));
		}
	}
	template <typename U>
	constexpr vector(vector<U>& other) noexcept
		requires assignable<T, U>
	{
		this->reserve(other.size());
		for (auto& arg : other) {
			new (reinterpret_cast<void*>(&this->ptr[this->len++])) T(arg);
		}
	}
	constexpr vector(vector&& other) noexcept { *this = vxr::std::move(other); }
	vector(size_t len) noexcept : vector(len, len) {}
	vector(size_t len, size_t cap) noexcept {
		if (cap < len) {
			abort("cap < len");
		}
		reserve(cap);
		resize(len);
	}
	~vector() noexcept {
		clear();
		free(reinterpret_cast<void*>(this->ptr));
		this->cap = 0;
		this->ptr = nullptr;
	}

	void reserve(size_t n) noexcept {
		if (this->cap >= n) {
			return;
		}
		this->cap = n;
		const size_t sz = this->cap * sizeof(T);  // NOLINT(bugprone-sizeof-expression)
		T* newptr = static_cast<T*>(realloc(reinterpret_cast<void*>(this->ptr), sz));
		if (newptr == nullptr) {
			abort("Failed realloc");
		}
		this->ptr = newptr;
	}
	void resize(size_t n) noexcept {
		if constexpr (destructible<T>) {
			for (size_t i = n; i < this->len; i++) {
				this->ptr[i].~T();
			}
		}
		reserve(n);
		if constexpr (constructible<T>) {
			for (size_t i = this->len; i < n; i++) {
				new (reinterpret_cast<void*>(&this->ptr[i])) T();
			}
		}
		this->len = n;
	}
	void resize(size_t n, T value) noexcept
		requires vxr::std::copy_constructible<T>
	{
		if constexpr (destructible<T>) {
			for (size_t i = len; i < this->len; i++) {
				this->ptr[i].~T();
			}
		}
		reserve(n);
		for (size_t i = this->len; i < n; i++) {
			new (reinterpret_cast<void*>(&this->ptr[i])) T(value);
		}
		this->len = n;
	}

	T& operator[](size_t i) noexcept {
		if (!inRange<size_t>(i, 0, len - 1)) {
			abort("Index out of bounds");
		}

		return ptr[i];
	}
	const T& operator[](size_t i) const noexcept {
		if (!inRange<size_t>(i, 0, len - 1)) {
			abort("Index out of bounds");
		}

		return ptr[i];
	}

	template <size_t N>
	vector& operator=(vxr::std::array<T, N>&& args) noexcept {
		this->clear();
		this->reserve(N);
		for (auto& arg : args) {
			new (reinterpret_cast<void*>(&this->ptr[this->len++])) T(vxr::std::move(arg));
		}
		return *this;
	}
	vector& operator=(vector&& other) noexcept {
		this->len = other.len;
		this->cap = other.cap;
		this->ptr = other.ptr;

		other.len = other.cap = 0;
		other.ptr = nullptr;

		return *this;
	}

	[[nodiscard]] size_t size() const noexcept { return len; }
	[[nodiscard]] size_t capacity() const noexcept { return cap; }
	[[nodiscard]] const T* get() const noexcept { return ptr; }
	[[nodiscard]] const T* begin() const noexcept { return ptr; }
	[[nodiscard]] const T* end() const noexcept { return begin() + size(); }

	[[nodiscard]] T* get() noexcept { return ptr; }
	[[nodiscard]] T* begin() noexcept { return ptr; }
	[[nodiscard]] T* end() noexcept { return begin() + size(); }

	void pushBack(T&& val) noexcept {
		if (this->len == this->cap) {
			grow(1);
		}
		new (reinterpret_cast<void*>(&this->ptr[this->len++])) T(vxr::std::move(val));
	}
	void pushBack(const T& val) noexcept {
		if (this->len == this->cap) {
			grow(1);
		}
		new (reinterpret_cast<void*>(&this->ptr[this->len++])) T(val);
	}
};
}  // namespace vxr::std
