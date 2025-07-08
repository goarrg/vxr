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
	"goarrg.com/gmath"
	"goarrg.com/rhi/vxr"
	"goarrg.com/rhi/vxr/internal/util"
)

type Pipeline2DLine struct {
	noCopy util.NoCopy
	gpl    vxr.GraphicsPipelineLibrary
}

func NewPipeline2DLine(fragmentLayout *vxr.ShaderLayout, specConstants []uint32) *Pipeline2DLine {
	p := Pipeline2DLine{
		gpl: vxr.GraphicsPipelineLibrary{
			Layout: vxr.NewPipelineLayout(
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: instance.line2DVertexShaderLayout, ShaderStage: vxr.ShaderStageVertex,
				},
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: fragmentLayout, ShaderStage: vxr.ShaderStageFragment,
					SpecConstants: specConstants,
				},
			),
			VertexInput: instance.line2DVertexInputPipeline,
		},
	}
	p.gpl.VertexShader = vxr.NewGraphicsShaderPipeline(p.gpl.Layout,
		instance.line2DVertexShader, instance.line2DVertexShaderLayout.EntryPoints["main"], vxr.GraphicsShaderPipelineCreateInfo{
			SpecConstants: []uint32{},
		})
	p.noCopy.Init()
	return &p
}

func (p *Pipeline2DLine) Destroy() {
	p.noCopy.Check()
	p.gpl.VertexShader.Destroy()
	p.noCopy.Close()
}

type InstanceData2DLine struct {
	P0, P1 gmath.Vector2f32
}

func (p *Pipeline2DLine) Draw(f *vxr.Frame, cb *vxr.GraphicsCommandBuffer, frag *vxr.GraphicsShaderPipeline,
	viewport gmath.Extent2i32, parameters vxr.DrawParameters, width float32,
	instances ...InstanceData2DLine,
) {
	p.noCopy.Check()
	ds := p.gpl.Layout.NewDescriptorSet(0)
	f.QueueDestory(ds)
	{
		b := f.NewHostScratchBuffer("vxr/shapes/customVertexShaderLineObjectData",
			instance.line2DVertexShaderObjectMetadata.Size+
				(instance.line2DVertexShaderObjectMetadata.RuntimeArrayStride*uint64(len(instances))),
			vxr.BufferUsageStorageBuffer,
		)
		ds.Bind(0, 0, vxr.DescriptorBufferInfo{
			Buffer: b,
		})

		var off uintptr
		s := gmath.Vector2f32{X: 2 / float32(viewport.X), Y: 2 / float32(viewport.Y)}
		for _, i := range instances {
			off += util.HostWrite(b, off, struct {
				p0, p1 [2]float32
			}{
				p0: gmath.Vector2f32{X: -1, Y: -1}.Add(i.P0.Scale(s)).ToArrayf32(),
				p1: gmath.Vector2f32{X: -1, Y: -1}.Add(i.P1.Scale(s)).ToArrayf32(),
			})
		}
	}
	parameters.DescriptorSets = append([]*vxr.DescriptorSet{ds}, parameters.DescriptorSets...)
	cb.RenderPassSetLineWidth(width)
	cb.Draw(vxr.GraphicsPipelineLibrary{
		Layout:         p.gpl.Layout,
		VertexInput:    p.gpl.VertexInput,
		VertexShader:   p.gpl.VertexShader,
		FragmentShader: frag,
	}, vxr.DrawInfo{
		DrawParameters: parameters,
		VertexCount:    uint32(len(instances) * 2),
		InstanceCount:  1,
	})
}

type Pipeline2DLineStrip struct {
	noCopy util.NoCopy
	gpl    vxr.GraphicsPipelineLibrary
}

func NewPipeline2DLineStrip(fragmentLayout *vxr.ShaderLayout, specConstants []uint32) *Pipeline2DLineStrip {
	p := Pipeline2DLineStrip{
		gpl: vxr.GraphicsPipelineLibrary{
			Layout: vxr.NewPipelineLayout(
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: instance.lineStrip2DVertexShaderLayout, ShaderStage: vxr.ShaderStageVertex,
				},
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: fragmentLayout, ShaderStage: vxr.ShaderStageFragment,
					SpecConstants: specConstants,
				},
			),
			VertexInput: instance.lineStrip2DVertexInputPipeline,
		},
	}
	p.gpl.VertexShader = vxr.NewGraphicsShaderPipeline(p.gpl.Layout,
		instance.lineStrip2DVertexShader, instance.lineStrip2DVertexShaderLayout.EntryPoints["main"], vxr.GraphicsShaderPipelineCreateInfo{
			SpecConstants: []uint32{},
		})
	p.noCopy.Init()
	return &p
}

func (p *Pipeline2DLineStrip) Destroy() {
	p.noCopy.Check()
	p.gpl.VertexShader.Destroy()
	p.noCopy.Close()
}

func (p *Pipeline2DLineStrip) Draw(f *vxr.Frame, cb *vxr.GraphicsCommandBuffer, frag *vxr.GraphicsShaderPipeline,
	viewport gmath.Extent2i32, parameters vxr.DrawParameters, width float32,
	points ...gmath.Vector2f32,
) {
	p.noCopy.Check()
	ds := p.gpl.Layout.NewDescriptorSet(0)
	f.QueueDestory(ds)
	{
		b := f.NewHostScratchBuffer("vxr/shapes/customVertexShaderLineObjectData",
			instance.lineStrip2DVertexShaderObjectMetadata.Size+
				(instance.lineStrip2DVertexShaderObjectMetadata.RuntimeArrayStride*uint64(len(points))),
			vxr.BufferUsageStorageBuffer,
		)
		ds.Bind(0, 0, vxr.DescriptorBufferInfo{
			Buffer: b,
		})

		var off uintptr
		s := gmath.Vector2f32{X: 2 / float32(viewport.X), Y: 2 / float32(viewport.Y)}
		for _, p := range points {
			off += util.HostWrite(b, off, struct {
				p0 [2]float32
			}{
				p0: gmath.Vector2f32{X: -1, Y: -1}.Add(p.Scale(s)).ToArrayf32(),
			})
		}
	}
	parameters.DescriptorSets = append([]*vxr.DescriptorSet{ds}, parameters.DescriptorSets...)
	cb.RenderPassSetLineWidth(width)
	cb.Draw(vxr.GraphicsPipelineLibrary{
		Layout:         p.gpl.Layout,
		VertexInput:    p.gpl.VertexInput,
		VertexShader:   p.gpl.VertexShader,
		FragmentShader: frag,
	}, vxr.DrawInfo{
		DrawParameters: parameters,
		VertexCount:    uint32(len(points)),
		InstanceCount:  1,
	})
}
