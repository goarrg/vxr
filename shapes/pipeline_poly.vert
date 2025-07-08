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

#define M_PI 3.1415926535897932384626433832795
#define M_SQRT2 1.4142135623730950488016887242096

#extension GL_EXT_scalar_block_layout : enable
#extension GL_ARB_shading_language_include : enable

#include "polygonmode.h"

layout(constant_id = 0) const uint polygonMode = POLYGON_MODE_REGULAR_CONCAVE;
layout(constant_id = 1) const uint triangleCount = 1;

layout(location = 0) out flat uint instanceID;
layout(location = 1) out vec2 uv;

struct object {
	float parameter1;
	mat3x2 matrix;
};

layout(set = 0, binding = 0, scalar) buffer readonly restrict Objects {
	layout(row_major) object objects[];
};

void main() {
	instanceID = gl_InstanceIndex;
	const object obj = objects[gl_InstanceIndex];
	switch (polygonMode & POLYGON_MODE_MASK) {
		case POLYGON_MODE_REGULAR_CONCAVE: {
			if (triangleCount == 3) {
				const vec2 verts[] = vec2[](vec2(0.0, -0.5), vec2(0.43301, 0.25), vec2(-0.43301, 0.25));
				uv = verts[gl_VertexIndex] + vec2(0.5);
				gl_Position = vec4(-1, -1, 0, 0) + vec4(obj.matrix * vec3(verts[gl_VertexIndex], 1), 0, 1.0);
				return;
			} else if (triangleCount == 4) {
				const float d = 0.25 * M_SQRT2;
				const vec2 verts[] = vec2[](vec2(-d, -d), vec2(d, -d), vec2(d, d), vec2(d, d), vec2(-d, d), vec2(-d, -d));
				uv = verts[gl_VertexIndex] + vec2(0.5);
				gl_Position = vec4(-1, -1, 0, 0) + vec4(obj.matrix * vec3(verts[gl_VertexIndex], 1), 0, 1.0);
				return;
			}
			switch (triangleCount) {
				case 1: {
					const vec2 verts[] = vec2[](vec2(0.0, -0.5), vec2(0.5, 0.5), vec2(-0.5, 0.5));
					uv = verts[gl_VertexIndex] + vec2(0.5);
					gl_Position = vec4(-1, -1, 0, 0) + vec4(obj.matrix * vec3(verts[gl_VertexIndex], 1), 0, 1.0);
					return;
				}

				case 2: {
					const vec2 verts[] = vec2[](
						vec2(-0.5, -0.5), vec2(0.5, -0.5), vec2(0.5, 0.5), vec2(0.5, 0.5), vec2(-0.5, 0.5), vec2(-0.5, -0.5));
					uv = verts[gl_VertexIndex] + vec2(0.5);
					gl_Position = vec4(-1, -1, 0, 0) + vec4(obj.matrix * vec3(verts[gl_VertexIndex], 1), 0, 1.0);
					return;
				}

				default: {
					const uint i = gl_VertexIndex / 3;
					const float lastAngle = 2.0 * M_PI * (float(i) / float(triangleCount));
					const vec2 lastVertex = vec2(sin(lastAngle), -cos(lastAngle)) * 0.5;
					const float nextAngle = 2.0 * M_PI * (float(i + 1) / float(triangleCount));
					const vec2 nextVertex = vec2(sin(nextAngle), -cos(nextAngle)) * 0.5;
					const vec2 verts[] = vec2[](vec2(0), lastVertex, nextVertex);
					uv = verts[gl_VertexIndex % 3] + vec2(0.5);
					gl_Position = vec4(-1, -1, 0, 0) + vec4(obj.matrix * vec3(verts[gl_VertexIndex % 3], 1), 0, 1.0);
					return;
				}
			}
		}

		case POLYGON_MODE_REGULAR_STAR: {
			const uint i = gl_VertexIndex / 6;
			const float lastAngle = 2.0 * M_PI * ((float(i) - 0.5) / float(triangleCount));
			const vec2 lastVertex = vec2(sin(lastAngle), -cos(lastAngle)) * (0.5 * obj.parameter1);
			const float nextAngle = 2.0 * M_PI * (float(i) + 0.5) / float(triangleCount);
			const vec2 nextVertex = vec2(sin(nextAngle), -cos(nextAngle)) * (0.5 * obj.parameter1);

			const float starAngle = 2.0 * M_PI * (float(i) / float(triangleCount));
			const vec2 starVertex = vec2(sin(starAngle), -cos(starAngle)) * 0.5;

			const vec2 verts[] = vec2[](vec2(0), lastVertex, nextVertex, nextVertex, lastVertex, starVertex);

			uv = verts[gl_VertexIndex % 6] + vec2(0.5);
			gl_Position = vec4(-1, -1, 0, 0) + vec4(obj.matrix * vec3(verts[gl_VertexIndex % 6], 1), 0, 1.0);
		}
	}
}
