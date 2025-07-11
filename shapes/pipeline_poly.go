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

/*
#include "polygonmode.h"
*/
import "C"

import (
	"goarrg.com/gmath"
	"goarrg.com/rhi/vxr"
	"goarrg.com/rhi/vxr/internal/util"
)

type Pipeline2D struct {
	noCopy        util.NoCopy
	gpl           vxr.GraphicsPipelineLibrary
	mode          uint32
	triangleCount uint32
	vertexCount   uint32
}

/*
NewPipeline2DTriangle create a new Pipeline2D that can only generate triangles.
It creates a triangle that fits within a square with length 1.0 before transformation.
*/
func NewPipeline2DTriangle(fragmentLayout *vxr.ShaderLayout, specConstants []uint32) *Pipeline2D {
	p := Pipeline2D{
		gpl: vxr.GraphicsPipelineLibrary{
			Layout: vxr.NewPipelineLayout(
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: instance.poly2DVertexShaderLayout, ShaderStage: vxr.ShaderStageVertex,
				},
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: fragmentLayout, ShaderStage: vxr.ShaderStageFragment,
					SpecConstants: specConstants,
				},
			),
			VertexInput: instance.poly2DVertexInputPipeline,
		},
		mode:          C.POLYGON_MODE_REGULAR_CONCAVE,
		triangleCount: 1,
		vertexCount:   3,
	}
	p.gpl.VertexShader = vxr.NewGraphicsShaderPipeline(p.gpl.Layout,
		instance.poly2DVertexShader, instance.poly2DVertexShaderLayout.EntryPoints["main"], vxr.GraphicsShaderPipelineCreateInfo{
			SpecConstants: []uint32{
				C.POLYGON_MODE_REGULAR_CONCAVE,
				1,
			},
		})
	p.noCopy.Init()
	return &p
}

/*
NewPipeline2DSquare create a new Pipeline2D that can only generate squares.
It creates a square with length 1.0 before transformation.
*/
func NewPipeline2DSquare(fragmentLayout *vxr.ShaderLayout, specConstants []uint32) *Pipeline2D {
	p := Pipeline2D{
		gpl: vxr.GraphicsPipelineLibrary{
			Layout: vxr.NewPipelineLayout(
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: instance.poly2DVertexShaderLayout, ShaderStage: vxr.ShaderStageVertex,
				},
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: fragmentLayout, ShaderStage: vxr.ShaderStageFragment,
					SpecConstants: specConstants,
				},
			),
			VertexInput: instance.poly2DVertexInputPipeline,
		},
		mode:          C.POLYGON_MODE_REGULAR_CONCAVE,
		triangleCount: 2,
		vertexCount:   6,
	}
	p.gpl.VertexShader = vxr.NewGraphicsShaderPipeline(p.gpl.Layout,
		instance.poly2DVertexShader, instance.poly2DVertexShaderLayout.EntryPoints["main"], vxr.GraphicsShaderPipelineCreateInfo{
			SpecConstants: []uint32{
				C.POLYGON_MODE_REGULAR_CONCAVE,
				2,
			},
		})
	p.noCopy.Init()
	return &p
}

/*
NewPipeline2DRegularNGon create a new Pipeline2D that can only draw a single shape with the given side count.
It creates an n-gon that fits within a circle with radius 0.5 before transformation.
*/
func NewPipeline2DRegularNGon(fragmentLayout *vxr.ShaderLayout, specConstants []uint32, sides uint32) *Pipeline2D {
	if sides < 3 {
		abort("The smallest possible shape is 3 sides")
	}
	p := Pipeline2D{
		gpl: vxr.GraphicsPipelineLibrary{
			Layout: vxr.NewPipelineLayout(
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: instance.poly2DVertexShaderLayout, ShaderStage: vxr.ShaderStageVertex,
				},
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: fragmentLayout, ShaderStage: vxr.ShaderStageFragment,
					SpecConstants: specConstants,
				},
			),
			VertexInput: instance.poly2DVertexInputPipeline,
		},
		mode:          C.POLYGON_MODE_REGULAR_CONCAVE,
		triangleCount: sides,
	}
	if sides <= 4 {
		sides = sides - 2
	}
	p.vertexCount = sides * 3
	p.gpl.VertexShader = vxr.NewGraphicsShaderPipeline(p.gpl.Layout,
		instance.poly2DVertexShader, instance.poly2DVertexShaderLayout.EntryPoints["main"], vxr.GraphicsShaderPipelineCreateInfo{
			SpecConstants: []uint32{
				C.POLYGON_MODE_REGULAR_CONCAVE,
				sides,
			},
		})
	p.noCopy.Init()
	return &p
}

/*
NewPipeline2DRegularNGonStar create a new Pipeline2D that can only draw a single shape with the given point count.
It creates an n-gon star that fits within a circle with radius 0.5 before transformation.
*/
func NewPipeline2DRegularNGonStar(fragmentLayout *vxr.ShaderLayout, specConstants []uint32, points uint32) *Pipeline2D {
	if points < 4 {
		abort("The smallest possible shape is 4 sides")
	}
	p := Pipeline2D{
		gpl: vxr.GraphicsPipelineLibrary{
			Layout: vxr.NewPipelineLayout(
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: instance.poly2DVertexShaderLayout, ShaderStage: vxr.ShaderStageVertex,
				},
				vxr.PipelineLayoutCreateInfo{
					ShaderLayout: fragmentLayout, ShaderStage: vxr.ShaderStageFragment,
					SpecConstants: specConstants,
				},
			),
			VertexInput: instance.poly2DVertexInputPipeline,
		},
		mode:          C.POLYGON_MODE_REGULAR_STAR,
		triangleCount: points,
		vertexCount:   points * 6,
	}
	p.gpl.VertexShader = vxr.NewGraphicsShaderPipeline(p.gpl.Layout,
		instance.poly2DVertexShader, instance.poly2DVertexShaderLayout.EntryPoints["main"], vxr.GraphicsShaderPipelineCreateInfo{
			SpecConstants: []uint32{
				C.POLYGON_MODE_REGULAR_STAR,
				points,
			},
		})
	p.noCopy.Init()
	return &p
}

func (p *Pipeline2D) Destroy() {
	p.noCopy.Check()
	p.gpl.VertexShader.Destroy()
	p.noCopy.Close()
}

type InstanceData2D struct {
	Transform Transform2D
	// Parameter1 is a polygon parameter that changes what it represents depending on which New* function you used.
	// Currently only NGonStar shapes uses this parameter to mean thickness of the polygon in the middle of the star,
	// it is an unsigned normalized value between 0 and 1 inclusive.
	Parameter1 float32
}

/*
Draw must be called within an active RenderPass.
DescriptorSets passed in parameters will have their indexes offset by 1 as set 0 is reserved for the vertex shader.
All vertices will have a Z value of 0.
*/
func (p *Pipeline2D) Draw(f *vxr.Frame, cb *vxr.GraphicsCommandBuffer, frag *vxr.GraphicsShaderPipeline,
	viewport gmath.Extent2i32, parameters vxr.DrawParameters,
	instances ...InstanceData2D,
) {
	p.noCopy.Check()
	ds := p.gpl.Layout.NewDescriptorSet(0)
	f.QueueDestory(ds)
	{
		b := f.NewHostScratchBuffer("vxr/shapes/customVertexShaderObjectData",
			instance.poly2DVertexShaderObjectMetadata.Size+
				(instance.poly2DVertexShaderObjectMetadata.RuntimeArrayStride*uint64(len(instances))),
			vxr.BufferUsageStorageBuffer,
		)
		ds.Bind(0, 0, vxr.DescriptorBufferInfo{
			Buffer: b,
		})
		var off uintptr
		for _, i := range instances {
			m := i.Transform.modelMatrix(p.mode, p.triangleCount)
			m[0] = [3]float32{
				m[0][0] * (2 / float32(viewport.X)),
				m[0][1] * (2 / float32(viewport.X)),
				m[0][2] * (2 / float32(viewport.X)),
			}
			m[1] = [3]float32{
				m[1][0] * (2 / float32(viewport.Y)),
				m[1][1] * (2 / float32(viewport.Y)),
				m[1][2] * (2 / float32(viewport.Y)),
			}
			off += util.HostWrite(b, off, struct {
				parameter1 float32
				matrix     [2][3]float32
			}{
				parameter1: i.Parameter1,
				matrix:     m,
			})
		}
	}
	parameters.DescriptorSets = append([]*vxr.DescriptorSet{ds}, parameters.DescriptorSets...)
	cb.Draw(vxr.GraphicsPipelineLibrary{
		Layout:         p.gpl.Layout,
		VertexInput:    p.gpl.VertexInput,
		VertexShader:   p.gpl.VertexShader,
		FragmentShader: frag,
	}, vxr.DrawInfo{
		DrawParameters: parameters,
		VertexCount:    p.vertexCount,
		InstanceCount:  uint32(len(instances)),
	})
}
