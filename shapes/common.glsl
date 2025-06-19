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

#extension GL_EXT_shader_explicit_arithmetic_types_int8 : enable
#extension GL_ARB_shading_language_include : enable

#include "polygonmode.h"

struct object {
	uint polygonMode;
	uint triangleCount;
	uint layer;
	float parameter1;
	u8vec4 color;
	mat3x2 matrix;
};

struct triangle {
	uint oID;
	vec2 vertices[3];
};
