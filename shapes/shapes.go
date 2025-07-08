//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go  -O -Os -strip main.comp
//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go -skip-metadata -O -Os -strip main.vert
//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go -skip-metadata -O -Os -strip main.frag

//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go -O -Os -strip pipeline_poly.vert
//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go -O -Os -strip pipeline_line.vert
//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go -O -Os -strip pipeline_linestrip.vert

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

package shapes

import (
	"unsafe"

	"goarrg.com/debug"
	"goarrg.com/rhi/vxr"
	"goarrg.com/rhi/vxr/internal/util"
)

func RequiredVkFeatureStructs() []vxr.VkFeatureStruct {
	return []vxr.VkFeatureStruct{
		vxr.VkPhysicalDeviceFeatures{
			ShaderStorageImageReadWithoutFormat:  true,
			ShaderStorageImageWriteWithoutFormat: true,
			// "shaderUniformBufferArrayDynamicIndexing": true,
			ShaderSampledImageArrayDynamicIndexing:  true,
			ShaderStorageBufferArrayDynamicIndexing: true,
			ShaderStorageImageArrayDynamicIndexing:  true,
		},
		vxr.VkPhysicalDeviceVulkan12Features{
			StorageBuffer8BitAccess: true,
			ShaderInt8:              true,
			// "shaderUniformBufferArrayNonUniformIndexing": true,
			ShaderSampledImageArrayNonUniformIndexing:  true,
			ShaderStorageBufferArrayNonUniformIndexing: true,
			ShaderStorageImageArrayNonUniformIndexing:  true,
			ScalarBlockLayout:                          true,
			VulkanMemoryModel:                          true,
		},
	}
}

var instance = struct {
	logger *debug.Logger

	dispatcher                    *vxr.ComputePipeline
	solid2DPipeline               vxr.GraphicsPipelineLibrary
	solid2DObjectBufferMetadata   vxr.ShaderBindingTypeBufferMetadata
	solid2DTriangleBufferMetadata vxr.ShaderBindingTypeBufferMetadata

	poly2DVertexInputPipeline        *vxr.VertexInputPipeline
	poly2DVertexShader               *vxr.Shader
	poly2DVertexShaderLayout         *vxr.ShaderLayout
	poly2DVertexShaderObjectMetadata vxr.ShaderBindingTypeBufferMetadata

	line2DVertexInputPipeline        *vxr.VertexInputPipeline
	line2DVertexShader               *vxr.Shader
	line2DVertexShaderLayout         *vxr.ShaderLayout
	line2DVertexShaderObjectMetadata vxr.ShaderBindingTypeBufferMetadata

	lineStrip2DVertexInputPipeline        *vxr.VertexInputPipeline
	lineStrip2DVertexShader               *vxr.Shader
	lineStrip2DVertexShaderLayout         *vxr.ShaderLayout
	lineStrip2DVertexShaderObjectMetadata vxr.ShaderBindingTypeBufferMetadata
}{
	logger: debug.NewLogger("vxr", "shapes"),
}

func Init() {
	properties := vxr.DeviceProperties()

	// solid2DPipeline
	{
		cs, cl, m := vxrcLoad_main_comp()
		vs, vl := vxrcLoad_main_vert()
		fs, fl := vxrcLoad_main_frag()
		instance.solid2DObjectBufferMetadata = m.DescriptorSetBindings["Objects"].(vxr.ShaderBindingTypeBufferMetadata)
		instance.solid2DTriangleBufferMetadata = m.DescriptorSetBindings["Triangles"].(vxr.ShaderBindingTypeBufferMetadata)

		instance.solid2DPipeline.Layout = vxr.NewPipelineLayout(
			vxr.PipelineLayoutCreateInfo{
				ShaderLayout: cl, ShaderStage: vxr.ShaderStageCompute,
			},
			vxr.PipelineLayoutCreateInfo{
				ShaderLayout: vl, ShaderStage: vxr.ShaderStageVertex,
			},
			vxr.PipelineLayoutCreateInfo{
				ShaderLayout: fl, ShaderStage: vxr.ShaderStageFragment,
			},
		)
		instance.dispatcher = vxr.NewComputePipeline(instance.solid2DPipeline.Layout, cs, cl.EntryPoints["main"], vxr.ComputePipelineCreateInfo{
			SpecConstants: []uint32{properties.Compute.SubgroupSize, 1, 1},
		})
		instance.solid2DPipeline.VertexInput = vxr.NewVertexInputPipeline(vxr.VertexInputPipelineCreateInfo{
			Topology: vxr.VertexTopologyTriangleList,
		})
		instance.solid2DPipeline.VertexShader = vxr.NewGraphicsShaderPipeline(instance.solid2DPipeline.Layout,
			vs, vl.EntryPoints["main"], vxr.GraphicsShaderPipelineCreateInfo{})
		instance.solid2DPipeline.FragmentShader = vxr.NewGraphicsShaderPipeline(instance.solid2DPipeline.Layout,
			fs, fl.EntryPoints["main"], vxr.GraphicsShaderPipelineCreateInfo{})

		// we are packing the indirect buffer into the triangle buffer, do this after creating the layouts
		// as we may use the size there in the future
		instance.solid2DTriangleBufferMetadata.Size = uint64(unsafe.Sizeof(uint32(0)) * 4)
	}

	// poly2DPipeline
	{
		instance.poly2DVertexInputPipeline = vxr.NewVertexInputPipeline(vxr.VertexInputPipelineCreateInfo{
			Topology: vxr.VertexTopologyTriangleList,
		})
		var m *vxr.ShaderMetadata
		instance.poly2DVertexShader, instance.poly2DVertexShaderLayout, m = vxrcLoad_pipeline_poly_vert()
		instance.poly2DVertexShaderObjectMetadata = m.DescriptorSetBindings["Objects"].(vxr.ShaderBindingTypeBufferMetadata)
	}

	// line2DPipeline
	{
		instance.line2DVertexInputPipeline = vxr.NewVertexInputPipeline(vxr.VertexInputPipelineCreateInfo{
			Topology: vxr.VertexTopologyLineList,
		})
		var m *vxr.ShaderMetadata
		instance.line2DVertexShader, instance.line2DVertexShaderLayout, m = vxrcLoad_pipeline_line_vert()
		instance.line2DVertexShaderObjectMetadata = m.DescriptorSetBindings["Objects"].(vxr.ShaderBindingTypeBufferMetadata)
	}

	// lineStrip2DPipeline
	{
		instance.lineStrip2DVertexInputPipeline = vxr.NewVertexInputPipeline(vxr.VertexInputPipelineCreateInfo{
			Topology: vxr.VertexTopologyLineStrip,
		})
		var m *vxr.ShaderMetadata
		instance.lineStrip2DVertexShader, instance.lineStrip2DVertexShaderLayout, m = vxrcLoad_pipeline_linestrip_vert()
		instance.lineStrip2DVertexShaderObjectMetadata = m.DescriptorSetBindings["Objects"].(vxr.ShaderBindingTypeBufferMetadata)
	}
}

func Destroy() {
	instance.dispatcher.Destroy()
	instance.dispatcher = nil

	instance.solid2DPipeline.VertexShader.Destroy()
	instance.solid2DPipeline.FragmentShader.Destroy()
	instance.solid2DPipeline = vxr.GraphicsPipelineLibrary{}

	instance.poly2DVertexInputPipeline = nil
	instance.line2DVertexInputPipeline = nil
	instance.lineStrip2DVertexInputPipeline = nil
}

func abort(fmt string, args ...any) {
	instance.logger.EPrintf(fmt, args...)
	util.Abort()
}
