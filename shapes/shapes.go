//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go  -O -Os main.comp
//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go -skip-metadata -O -Os -strip main.vert
//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go -skip-metadata -O -Os -strip main.frag

//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go -O -Os pipeline.vert

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

	"goarrg.com"
	"goarrg.com/debug"
	"goarrg.com/rhi/vxr"
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

type platform struct{}

func (platform) Abort()                           { panic("Fatal Error") }
func (platform) AbortPopup(f string, args ...any) { panic("Fatal Error") }

var instance = struct {
	platform goarrg.PlatformInterface
	logger   *debug.Logger

	dispatcher                    *vxr.ComputePipeline
	solid2DPipeline               vxr.GraphicsPipelineLibrary
	solid2DObjectBufferMetadata   vxr.ShaderBindingTypeBufferMetadata
	solid2DTriangleBufferMetadata vxr.ShaderBindingTypeBufferMetadata

	custom2DVertexInputPipeline        *vxr.VertexInputPipeline
	custom2DVertexShader               *vxr.Shader
	custom2DVertexShaderLayout         *vxr.ShaderLayout
	custom2DVertexShaderObjectMetadata vxr.ShaderBindingTypeBufferMetadata
}{
	platform: platform{},
	logger:   debug.NewLogger("vxr", "shapes"),
}

func Init(platform goarrg.PlatformInterface) {
	instance.platform = platform
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

	// custom2DPipeline
	{
		instance.custom2DVertexInputPipeline = vxr.NewVertexInputPipeline(vxr.VertexInputPipelineCreateInfo{
			Topology: vxr.VertexTopologyTriangleList,
		})
		var m *vxr.ShaderMetadata
		instance.custom2DVertexShader, instance.custom2DVertexShaderLayout, m = vxrcLoad_pipeline_vert()
		instance.custom2DVertexShaderObjectMetadata = m.DescriptorSetBindings["Objects"].(vxr.ShaderBindingTypeBufferMetadata)
	}
}

func Destroy() {
	instance.dispatcher.Destroy()
	instance.dispatcher = nil

	instance.solid2DPipeline.VertexShader.Destroy()
	instance.solid2DPipeline.FragmentShader.Destroy()
	instance.solid2DPipeline = vxr.GraphicsPipelineLibrary{}

	instance.custom2DVertexInputPipeline = nil
}

func abort(fmt string, args ...any) {
	instance.logger.EPrintf(fmt, args...)
	instance.platform.Abort()
}
