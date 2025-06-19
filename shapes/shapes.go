//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go  -O -Os main.comp
//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go -skip-metadata -O -Os -strip main.vert
//go:generate go run goarrg.com/rhi/vxr/cmd/vxrc -id-prefix=vxr/shapes/ -dir=./ -generator=go -skip-metadata -O -Os -strip main.frag

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

	dispatcherLayout              *vxr.PipelineLayout
	dispatcher                    *vxr.ComputePipeline
	solid2DPipeline               vxr.GraphicsPipelineLibrary
	solid2DObjectBufferMetadata   vxr.ShaderBindingTypeBufferMetadata
	solid2DTriangleBufferMetadata vxr.ShaderBindingTypeBufferMetadata
}{
	platform: platform{},
	logger:   debug.NewLogger("vxr", "shapes"),
}

func Init(platform goarrg.PlatformInterface) {
	instance.platform = platform
	properties := vxr.DeviceProperties()

	cs, cl, m := vxrcLoad_main_comp()
	vs, vl := vxrcLoad_main_vert()
	fs, fl := vxrcLoad_main_frag()
	instance.solid2DObjectBufferMetadata = m.DescriptorSetBindings["Objects"].(vxr.ShaderBindingTypeBufferMetadata)
	instance.solid2DTriangleBufferMetadata = m.DescriptorSetBindings["Triangles"].(vxr.ShaderBindingTypeBufferMetadata)

	instance.dispatcherLayout = vxr.NewPipelineLayout(
		vxr.PipelineLayoutCreateInfo{
			ShaderLayout: cl, ShaderStage: vxr.ShaderStageCompute,
		},
	)
	instance.dispatcher = vxr.NewComputePipeline(instance.dispatcherLayout, cs, cl.EntryPoints["main"], vxr.ComputePipelineCreateInfo{
		SpecConstants: []uint32{properties.Compute.SubgroupSize, 1, 1},
	})

	instance.solid2DPipeline.Layout = vxr.NewPipelineLayout(
		vxr.PipelineLayoutCreateInfo{
			ShaderLayout: vl, ShaderStage: vxr.ShaderStageVertex,
		},
		vxr.PipelineLayoutCreateInfo{
			ShaderLayout: fl, ShaderStage: vxr.ShaderStageFragment,
		},
	)
	instance.solid2DPipeline.VertexInput = vxr.NewVertexInputPipeline(vxr.VertexInputPipelineCreateInfo{
		Topology: vxr.VertexTopologyTriangleList,
	})
	instance.solid2DPipeline.VertexShader = vxr.NewGraphicsShaderPipeline(instance.solid2DPipeline.Layout,
		vs, vl.EntryPoints["main"], vxr.GraphicsShaderPipelineCreateInfo{})
	instance.solid2DPipeline.FragmentShader = vxr.NewGraphicsShaderPipeline(instance.solid2DPipeline.Layout,
		fs, fl.EntryPoints["main"], vxr.GraphicsShaderPipelineCreateInfo{})
}

func Destroy() {
	instance.dispatcher.Destroy()
	instance.dispatcher = nil

	instance.solid2DPipeline.VertexShader.Destroy()
	instance.solid2DPipeline.FragmentShader.Destroy()
	instance.solid2DPipeline = vxr.GraphicsPipelineLibrary{}
}

func abort(fmt string, args ...any) {
	instance.logger.EPrintf(fmt, args...)
	instance.platform.Abort()
}
