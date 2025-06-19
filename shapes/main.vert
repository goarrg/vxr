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
#pragma shader_stage(vertex)

#extension GL_EXT_scalar_block_layout : enable
#extension GL_ARB_shading_language_include : enable

#include "common.glsl"

layout(set = 0, binding = 0, scalar) buffer readonly restrict Objects {
	uint numObjects;
	layout(row_major) object objects[];
};
layout(set = 0, binding = 1, scalar) buffer readonly restrict Triangles {
	triangle triangles[];
};

layout(location = 0) out vec4 fragColor;

void main() {
	const uint tID = gl_VertexIndex / 3;
	const uint vID = gl_VertexIndex % 3;
	const triangle t = triangles[tID];
	const object o = objects[t.oID];
	gl_Position = vec4(-1, -1, 0, 0) + vec4(o.matrix * vec3(t.vertices[vID], 1), float(o.layer + 1) / float(numObjects), 1.0);
	const vec4 color = vec4(o.color) / vec4(255);
	fragColor = vec4(color.rgb * color.a, color.a);
}
