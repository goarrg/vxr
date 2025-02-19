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

#include "concepts.hpp"

namespace vxr::std {
template <typename T, typename U>
constexpr void copy(T first, T last, U dst) noexcept
	requires assignable<U, T>
{
	for (; first != last; (void)++first, (void)++dst) {
		*dst = *first;
	}
}
}  // namespace vxr::std
