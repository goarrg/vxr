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

#include "vxr/vxr.h"

#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include "std/stdlib.hpp"
#include "std/array.hpp"
#include "std/memory.hpp"
#include "std/utility.hpp"
#include "std/log.hpp"

namespace vxr::vk::device::reflect {
struct vkStructureChain {
	VkStructureType sType;
	vkStructureChain* pNext;
};
struct type {
   public:
	enum id : uint8_t {
		vkStructureType,
		voidPtr,
		vkBool32,
		maxID,
	};

   private:
	static constexpr vxr::std::array<const char*, maxID> names = {
		"VkStructureType",
		"VoidPtr",
		"VkBool32",
	};

   public:
	id id;

	constexpr type() noexcept = default;
	constexpr type(enum id id) noexcept : id(id) {}
	virtual constexpr ~type() noexcept = default;

	[[nodiscard]] constexpr const char* name() const { return names[id]; }
	[[nodiscard]] constexpr size_t size() const {
		switch (this->id) {
			case id::vkStructureType:
				return sizeof(vkStructureType);
			case id::voidPtr:
				return sizeof(void*);
			case id::vkBool32:
				return sizeof(vkBool32);

			case id::maxID:
				break;
		}

		vxr::std::ePrintf("Unknown vxr::vk::reflect::Type::id");
		vxr::std::abort();
		return 0;
	}
};
struct value {
	void* ptr = nullptr;

	constexpr value() noexcept = default;
	constexpr value(void* ptr) noexcept : ptr(ptr) {}
	virtual constexpr ~value() noexcept = default;
};

struct structField {
	::vxr::vk::device::reflect::type type;
	size_t offset;
	const char* name;

	constexpr structField() noexcept = default;
	constexpr structField(struct type type, size_t offset, const char* name) noexcept
		: type(vxr::std::move(type)), offset(offset), name(name) {};
	virtual constexpr ~structField() noexcept = default;
};
struct structType {
	const char* name;
	size_t size;

	constexpr structType(const char* name, size_t size) noexcept : name(name), size(size) {}
	virtual constexpr ~structType() noexcept = default;

	[[nodiscard]] virtual constexpr size_t numField() const noexcept = 0;
	[[nodiscard]] virtual constexpr structField field(size_t i) const noexcept = 0;
	[[nodiscard]] virtual constexpr const structField* begin() const noexcept = 0;
	[[nodiscard]] virtual constexpr const structField* end() const noexcept = 0;
	[[nodiscard]] virtual vxr::std::smartPtr<vkStructureChain> allocate() const noexcept = 0;
};

struct structFieldValue : structField, value {
	constexpr structFieldValue() noexcept = default;
	constexpr structFieldValue(const structField& type, void* ptr) noexcept : structField(type), value{ptr} {}
	constexpr ~structFieldValue() noexcept override = default;
};
struct structValue : value {
	const structType* type;

	constexpr structValue() noexcept = default;
	constexpr structValue(const structType* type, void* ptr) noexcept : value{ptr}, type(type) {}
	constexpr ~structValue() noexcept override = default;

	[[nodiscard]] virtual constexpr size_t numField() const noexcept = 0;
	[[nodiscard]] virtual constexpr structFieldValue field(size_t i) noexcept = 0;
	[[nodiscard]] virtual constexpr structFieldValue* begin() noexcept = 0;
	[[nodiscard]] virtual constexpr structFieldValue* end() noexcept = 0;
	[[nodiscard]] virtual vxr::std::smartPtr<vkStructureChain> clone() const noexcept = 0;
};

namespace internal {
template <size_t N>
struct structTypeImpl : structType {
   private:
	const vxr::std::array<structField, N> fields;

   public:
	template <typename... Args>
	constexpr structTypeImpl(const char* name, size_t size, Args&&... fields) noexcept
		: structType(name, size), fields(vxr::std::move(fields)...) {
		static_assert(sizeof...(Args) == N);
	}
	constexpr ~structTypeImpl() noexcept override = default;

	[[nodiscard]] constexpr size_t numField() const noexcept override { return N; }
	[[nodiscard]] constexpr structField field(size_t i) const noexcept override { return fields[i]; }
	[[nodiscard]] constexpr const structField* begin() const noexcept override { return &fields[0]; }
	[[nodiscard]] constexpr const structField* end() const noexcept override { return this->begin() + N; }
	[[nodiscard]] vxr::std::smartPtr<vkStructureChain> allocate() const noexcept override {
		auto* tmp = calloc(1, size);
		if (tmp == nullptr) {
			vxr::std::ePrintf("Failed to allocate memory");
			vxr::std::abort();
		}
		return {static_cast<vkStructureChain*>(tmp), [](vkStructureChain* ptr) { free(ptr); }};
	}
};
template <size_t N>
struct structChainTypeImpl : structTypeImpl<N> {
   private:
	VkStructureType sType;

   public:
	template <typename... Args>
	constexpr structChainTypeImpl(VkStructureType sType, const char* name, size_t size, Args&&... fields) noexcept
		: structTypeImpl<N>(name, size, vxr::std::move(fields)...), sType(sType) {
		static_assert(sizeof...(Args) == N);
	}
	constexpr ~structChainTypeImpl() noexcept override = default;

	[[nodiscard]] vxr::std::smartPtr<vkStructureChain> allocate() const noexcept override {
		auto tmp = structTypeImpl<N>::allocate();
		tmp.get()->sType = this->sType;
		return tmp;
	}
};

template <size_t N>
struct structValueImpl : structValue {
   private:
	vxr::std::array<structFieldValue, N> fields;

	[[nodiscard]] vxr::std::array<structFieldValue, N> makeFields() const noexcept {
		vxr::std::array<structFieldValue, N> fields;
		for (size_t i = 0; i < N; i++) {
			auto t = type->field(i);
			fields[i] = structFieldValue(t, (reinterpret_cast<uint8_t*>(ptr)) + t.offset);
		}
		return fields;
	}

   public:
	structValueImpl(const structType* type, void* ptr) noexcept : structValue(type, ptr), fields(makeFields()) {}
	constexpr ~structValueImpl() noexcept override = default;
	[[nodiscard]] constexpr size_t numField() const noexcept override { return N; }
	[[nodiscard]] constexpr structFieldValue field(size_t i) noexcept override { return fields[i]; };
	[[nodiscard]] constexpr structFieldValue* begin() noexcept override { return &fields[0]; }
	[[nodiscard]] constexpr structFieldValue* end() noexcept override { return this->begin() + N; }
	[[nodiscard]] vxr::std::smartPtr<vkStructureChain> clone() const noexcept override {
		auto tmp = this->type->allocate();
		memcpy(tmp.get(), this->ptr, this->type->size);
		return vxr::std::move(tmp);
	}
};
}  // namespace internal

#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wswitch"

#include "device_features_reflection.inc"  // IWYU pragma: keep

#pragma GCC diagnostic pop
}  // namespace vxr::vk::device::reflect
