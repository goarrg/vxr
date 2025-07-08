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

#version 450
#pragma shader_stage(fragment)

#extension GL_ARB_shading_language_include : enable
#extension GL_EXT_nonuniform_qualifier : enable

#include "common.glsl"

layout(constant_id = 0) const int maxTextureCount = 1;

layout(location = 0) in flat uint objectID;
layout(location = 1) in vec2 uv;

layout(location = 0) out vec4 outColor;

layout(set = 0, binding = 0, scalar) buffer readonly restrict Objects {
	uint numObjects;
	layout(row_major) object objects[];
};

layout(set = 1, binding = 0) uniform sampler textureSampler;
layout(set = 1, binding = 1) uniform texture2D textures[maxTextureCount];

#define HAS_BIT(x, y) ((x & y) == y)

void main() {
	vec4 color;
	const object o = objects[objectID];
	if (HAS_BIT(o.polygonMode, POLYGON_MODE_TEXTURED_BIT)) {
		color = texture(sampler2D(textures[nonuniformEXT(o.color)], textureSampler), uv);
	} else {
		color = unpackUnorm4x8(o.color);
	}
	outColor = vec4(color.rgb / color.a, color.a);
}
