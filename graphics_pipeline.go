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

package vxr

/*
	#cgo pkg-config: vxr

	#include "vxr/vxr.h"
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"

	"goarrg.com/debug"
	"goarrg.com/rhi/vxr/internal/vk"
)

type VertexTopology uint32

const (
	VertexTopologyPointList                  VertexTopology = vk.PRIMITIVE_TOPOLOGY_POINT_LIST
	VertexTopologyLineList                   VertexTopology = vk.PRIMITIVE_TOPOLOGY_LINE_LIST
	VertexTopologyLineStrip                  VertexTopology = vk.PRIMITIVE_TOPOLOGY_LINE_STRIP
	VertexTopologyTriangleList               VertexTopology = vk.PRIMITIVE_TOPOLOGY_TRIANGLE_LIST
	VertexTopologyTriangleStrip              VertexTopology = vk.PRIMITIVE_TOPOLOGY_TRIANGLE_STRIP
	VertexTopologyTriangleFan                VertexTopology = vk.PRIMITIVE_TOPOLOGY_TRIANGLE_FAN
	VertexTopologyLineListWithAdjacency      VertexTopology = vk.PRIMITIVE_TOPOLOGY_LINE_LIST_WITH_ADJACENCY
	VertexTopologyLineStripWithAdjacency     VertexTopology = vk.PRIMITIVE_TOPOLOGY_LINE_STRIP_WITH_ADJACENCY
	VertexTopologyTriangleListWithAdjacency  VertexTopology = vk.PRIMITIVE_TOPOLOGY_TRIANGLE_LIST_WITH_ADJACENCY
	VertexTopologyTriangleStripWithAdjacency VertexTopology = vk.PRIMITIVE_TOPOLOGY_TRIANGLE_STRIP_WITH_ADJACENCY
	VertexTopologyPatchList                  VertexTopology = vk.PRIMITIVE_TOPOLOGY_PATCH_LIST
)

func (t VertexTopology) String() string {
	switch t {
	case VertexTopologyPointList:
		return "PointList"
	case VertexTopologyLineList:
		return "LintList"
	case VertexTopologyLineStrip:
		return "LineStrip"
	case VertexTopologyTriangleList:
		return "TriangleList"
	case VertexTopologyTriangleStrip:
		return "TriangleStrip"
	case VertexTopologyTriangleFan:
		return "TriangleFan"
	case VertexTopologyLineListWithAdjacency:
		return "LineListWithAdjacency"
	case VertexTopologyLineStripWithAdjacency:
		return "LineStripWithAdjacency"
	case VertexTopologyTriangleListWithAdjacency:
		return "TriangleListWithAdjacency"
	case VertexTopologyTriangleStripWithAdjacency:
		return "TriangleStripWithAdjacency:"
	case VertexTopologyPatchList:
		return "PatchList"

	default:
		abort("Unknown VertexTopology: %d", t)
		return ""
	}
}

type VertexInputPipelineCreateInfo struct {
	Topology               VertexTopology
	PrimitiveRestartEnable bool
}

type VertexInputPipeline struct {
	id         string
	name       string
	vkPipeline C.VkPipeline
	topology   C.VkPrimitiveTopology

	// bindings   []C.VkVertexInputBindingDescription2EXT
	// attributes []C.VkVertexInputAttributeDescription2EXT
}

func NewVertexInputPipeline(info VertexInputPipelineCreateInfo) *VertexInputPipeline {
	p := &VertexInputPipeline{
		topology: C.VkPrimitiveTopology(info.Topology),
	}

	if info.PrimitiveRestartEnable {
		switch info.Topology {
		case VertexTopologyLineStrip, VertexTopologyLineStripWithAdjacency:
			p.name = "[vertex_input:line,restart]"
		case VertexTopologyTriangleStrip, VertexTopologyTriangleFan,
			VertexTopologyTriangleStripWithAdjacency:
			p.name = "[vertex_input:triangle,restart]"
		case VertexTopologyPatchList:
			p.name = "[vertex_input:patch,restart]"
		default:
			abort("PrimitiveRestart is invalid for topology: %s", info.Topology.String())
		}

		p.vkPipeline = instance.graphics.pipelineCache.createOrRetrievePipeline(p.name, func() C.VkPipeline {
			C.vxr_vk_graphics_createVertexInputPipeline(instance.cInstance, C.size_t(len(p.name)), (*C.char)(unsafe.Pointer(unsafe.StringData(p.name))),
				C.VkPrimitiveTopology(info.Topology), vk.TRUE, &p.vkPipeline)
			return p.vkPipeline
		})
	} else {
		switch info.Topology {
		case VertexTopologyPointList:
			p.name = "[vertex_input:point]"
		case VertexTopologyLineList, VertexTopologyLineStrip, VertexTopologyLineListWithAdjacency, VertexTopologyLineStripWithAdjacency:
			p.name = "[vertex_input:line]"
		case VertexTopologyTriangleList, VertexTopologyTriangleStrip, VertexTopologyTriangleFan, VertexTopologyTriangleListWithAdjacency,
			VertexTopologyTriangleStripWithAdjacency:
			p.name = "[vertex_input:triangle]"
		case VertexTopologyPatchList:
			p.name = "[vertex_input:patch]"
		}

		p.vkPipeline = instance.graphics.pipelineCache.createOrRetrievePipeline(p.name, func() C.VkPipeline {
			C.vxr_vk_graphics_createVertexInputPipeline(instance.cInstance, C.size_t(len(p.name)), (*C.char)(unsafe.Pointer(unsafe.StringData(p.name))),
				C.VkPrimitiveTopology(info.Topology), vk.FALSE, &p.vkPipeline)
			return p.vkPipeline
		})
	}
	p.id = genID(p.vkPipeline)

	return p
}

type GraphicsShaderPipeline struct {
	noCopy noCopy
	id     string
	name   string
	stage  ShaderStage

	vkPipeline C.VkPipeline
}

type GraphicsShaderPipelineCreateInfo struct {
	SpecConstants []uint32
}

func NewGraphicsShaderPipeline(pipelineLayout *PipelineLayout, s *Shader, entryPoint ShaderEntryPointLayout, info GraphicsShaderPipelineCreateInfo) *GraphicsShaderPipeline {
	entryPointName := entryPoint.EntryPointName()
	shaderName := fmt.Sprintf("[%q,%s,%s]", s.ID, entryPointName, jsonString(info.SpecConstants))
	pipelineInfo := C.vxr_vk_graphics_shaderPipelineCreateInfo{
		layout:         pipelineLayout.vkPipelinelayout,
		entryPointSize: C.size_t(len(entryPointName)),
		entryPoint:     (*C.char)(unsafe.Pointer(unsafe.StringData(entryPointName))),
		stage:          C.VkShaderStageFlagBits(entryPoint.ShaderStage()),
		spirv: C.vxr_vk_shader_spirv{
			len:  C.size_t(len(s.SPIRV)),
			data: (*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(s.SPIRV))),
		},
		numSpecConstants: C.uint32_t(len(info.SpecConstants)),
		specConstants:    (*C.uint32_t)(unsafe.SliceData(info.SpecConstants)),
	}
	var vkPipeline C.VkPipeline
	C.vxr_vk_graphics_createShaderPipeline(instance.cInstance, C.size_t(len(shaderName)), (*C.char)(unsafe.Pointer(unsafe.StringData(shaderName))),
		pipelineInfo, &vkPipeline)
	runtime.KeepAlive(entryPointName)
	runtime.KeepAlive(s.SPIRV)
	runtime.KeepAlive(info.SpecConstants)
	return instance.graphics.pipelineCache.getShader(shaderName, entryPoint.ShaderStage(), vkPipeline)
}

func (s *GraphicsShaderPipeline) Destroy() {
	s.noCopy.check()
	instance.graphics.pipelineCache.destroyShader(s.name)
	s.noCopy.close()
}

type GraphicsPipelineLibrary struct {
	Layout                       *PipelineLayout
	VertexInput                  *VertexInputPipeline
	VertexShader, FragmentShader *GraphicsShaderPipeline
}

func (gp *GraphicsPipelineLibrary) validate() error {
	gp.VertexShader.noCopy.check()
	gp.FragmentShader.noCopy.check()
	if gp.VertexShader.stage != ShaderStageVertex {
		return debug.Errorf("Vertex shader does not contain a vertex shader entry point")
	}
	if gp.FragmentShader.stage != ShaderStageFragment {
		return debug.Errorf("Fragment shader does not contain a fragment shader entry point")
	}
	return nil
}
