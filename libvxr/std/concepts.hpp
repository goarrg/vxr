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

#include <type_traits>

namespace vxr::std {
template <typename T>
concept pointer = ::std::is_pointer_v<T>;

template <typename T>
concept complete = requires(T v) { sizeof(T); };

template <typename T>
concept constructible = (!::std::is_pointer_v<T>) && complete<T> && requires(T* t) { t = new T(); };

template <typename T>
concept copy_constructible = requires(T* t, ::std::remove_reference_t<T> u) { t = new T(u); };

template <typename T>
concept destructible = (!::std::is_pointer_v<T>) && complete<T> && requires(T t) { t.~T(); };

template <typename T, typename U>
concept assignable = (complete<T> && complete<U>) && requires(T t, U u) { t = u; };

template <class T>
concept integral = ::std::is_integral_v<T> || ::std::is_enum_v<T>;
}  // namespace vxr::std
