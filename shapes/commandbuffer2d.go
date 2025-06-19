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
	"goarrg.com/gmath/color"
	"goarrg.com/rhi/vxr"
	"goarrg.com/rhi/vxr/internal/util"
)

type cbState uint

const (
	cbIdle cbState = iota
	cbRecording
)

type shape2d struct {
	polygonMode   uint32
	triangleCount uint32
	layer         uint32
	parameter1    float32
	color         color.UNorm[uint8]
	modelMatrix   [2][3]float32
}

type CommandBuffer2D struct {
	noCopy             noCopy
	cbState            cbState
	shapesColored      []shape2d
	shapesColoredAlpha []shape2d

	objectCount    uint32
	objectCapacity uint32
	objectBuffer   *vxr.DeviceBuffer

	triangleCount    uint32
	triangleCapacity uint32
	triangleBuffer   *vxr.DeviceBuffer

	depthImage *vxr.DeviceDepthStencilImage
}

func (cb *CommandBuffer2D) Begin() {
	if cb.noCopy.addr == nil {
		cb.noCopy.init()
	}
	cb.noCopy.check()
	if cb.cbState != cbIdle {
		abort("Begin() called while CommandBuffer2D is not idle")
	}
	cb.shapesColored = cb.shapesColored[:0]
	cb.shapesColoredAlpha = cb.shapesColoredAlpha[:0]

	cb.objectCount = 0
	cb.triangleCount = 0

	cb.cbState = cbRecording
}

func (cb *CommandBuffer2D) End() {
	cb.noCopy.check()
	if cb.cbState != cbRecording {
		abort("End() called while CommandBuffer2D is not in a recording state")
	}
	cb.cbState = cbIdle
}

/*
Destroy will destroy all persistent objects, caller is responsible for synchronization.
*/
func (cb *CommandBuffer2D) Destroy() {
	cb.noCopy.check()
	cb.objectBuffer.Destroy()
	cb.triangleBuffer.Destroy()
	cb.depthImage.Destroy()
	cb.noCopy.close()
	*cb = CommandBuffer2D{}
}

func (cb *CommandBuffer2D) PreExecuteDstImageBarrierInfo() vxr.ImageBarrierInfo {
	cb.noCopy.check()
	return vxr.ImageBarrierInfo{
		Stage:  vxr.PipelineStageRenderAttachmentWrite,
		Access: vxr.AccessFlagMemoryWrite,
		Layout: vxr.ImageLayoutAttachmentOptimal,
	}
}

func (cb *CommandBuffer2D) Execute(frame *vxr.Frame, vcb *vxr.GraphicsCommandBuffer, output vxr.ColorImage) {
	cb.noCopy.check()
	if cb.cbState != cbIdle {
		abort("Execute(...) called while CommandBuffer2D is not idle")
	}
	vcb.BeginNamedRegion("shapes2d")

	dsDispatch := instance.dispatcherLayout.NewDescriptorSet(0)
	frame.QueueDestory(dsDispatch)
	dsDraw := instance.solid2DPipeline.Layout.NewDescriptorSet(0)
	frame.QueueDestory(dsDraw)

	if cb.objectCapacity < cb.objectCount {
		const objectCapacityIncrement = 128
		cb.objectCapacity = ((cb.objectCount + objectCapacityIncrement) / objectCapacityIncrement) * objectCapacityIncrement
		frame.QueueDestory(cb.objectBuffer)
		cb.objectBuffer = vxr.NewDeviceBuffer("vxr/shapes/object",
			(instance.solid2DObjectBufferMetadata.Size + (instance.solid2DObjectBufferMetadata.RuntimeArrayStride * uint64(cb.objectCapacity))),
			vxr.BufferUsageStorageBuffer|vxr.BufferUsageTransferDst)
	}
	if cb.triangleCapacity < cb.triangleCount {
		const triangleCapacityIncrement = 512
		cb.triangleCapacity = ((cb.triangleCapacity + triangleCapacityIncrement) / triangleCapacityIncrement) * triangleCapacityIncrement
		frame.QueueDestory(cb.triangleBuffer)
		cb.triangleBuffer = vxr.NewDeviceBuffer("vxr/shapes/triangle",
			(instance.solid2DTriangleBufferMetadata.Size + (instance.solid2DTriangleBufferMetadata.RuntimeArrayStride * uint64(cb.triangleCapacity))),
			vxr.BufferUsageStorageBuffer|vxr.BufferUsageIndirectBuffer|vxr.BufferUsageTransferDst)
	}
	if !output.Extent().InRange(gmath.Extent3i32{}, cb.depthImage.Extent()) {
		frame.QueueDestory(cb.depthImage)
		cb.depthImage = vxr.NewDepthStencilImageWithAtMostBits("vxr/shapes/depth", 32, 0, vxr.ImageCreateInfo{
			Usage:          vxr.ImageUsageDepthStencilAttachment,
			Flags:          0,
			Extent:         output.Extent(),
			NumMipLevels:   1,
			NumArrayLayers: 1,
		})
	}

	dsDispatch.Bind(0, 0, vxr.DescriptorBufferInfo{
		Buffer: cb.objectBuffer,
	})
	dsDispatch.Bind(1, 0, vxr.DescriptorBufferInfo{
		Buffer: cb.triangleBuffer,
	})

	dsDraw.Bind(0, 0, vxr.DescriptorBufferInfo{
		Buffer: cb.objectBuffer,
	})
	dsDraw.Bind(1, 0, vxr.DescriptorBufferInfo{
		Buffer: cb.triangleBuffer,
		Offset: instance.solid2DTriangleBufferMetadata.Size,
	})

	vcb.BufferBarrier(vxr.BufferBarrier{
		Buffer: cb.objectBuffer,
		Src: vxr.BufferBarrierInfo{
			Stage:  vxr.PipelineStageFragmentShader,
			Access: vxr.AccessFlagMemoryRead,
		},
		Dst: vxr.BufferBarrierInfo{
			Stage:  vxr.PipelineStageTransfer,
			Access: vxr.AccessFlagMemoryWrite,
		},
	}, vxr.BufferBarrier{
		Buffer: cb.triangleBuffer,
		Src: vxr.BufferBarrierInfo{
			Stage:  vxr.PipelineStageFragmentShader,
			Access: vxr.AccessFlagMemoryRead,
		},
		Dst: vxr.BufferBarrierInfo{
			Stage:  vxr.PipelineStageTransfer,
			Access: vxr.AccessFlagMemoryWrite,
		},
	})

	obj := frame.NewHostScratchBuffer("vxr/shapes/object", cb.objectBuffer.Size(), vxr.BufferUsageTransferSrc)
	off := util.HostWrite(obj, 0, cb.objectCount)
	{
		extent := output.Extent()
		for _, s := range cb.shapesColored {
			s.modelMatrix[0] = [3]float32{
				s.modelMatrix[0][0] * (2 / float32(extent.X)),
				s.modelMatrix[0][1] * (2 / float32(extent.X)),
				s.modelMatrix[0][2] * (2 / float32(extent.X)),
			}
			s.modelMatrix[1] = [3]float32{
				s.modelMatrix[1][0] * (2 / float32(extent.Y)),
				s.modelMatrix[1][1] * (2 / float32(extent.Y)),
				s.modelMatrix[1][2] * (2 / float32(extent.Y)),
			}

			off += util.HostWrite(obj, off, s)
		}
		for _, s := range cb.shapesColoredAlpha {
			s.modelMatrix[0] = [3]float32{
				s.modelMatrix[0][0] * (2 / float32(extent.X)),
				s.modelMatrix[0][1] * (2 / float32(extent.X)),
				s.modelMatrix[0][2] * (2 / float32(extent.X)),
			}
			s.modelMatrix[1] = [3]float32{
				s.modelMatrix[1][0] * (2 / float32(extent.Y)),
				s.modelMatrix[1][1] * (2 / float32(extent.Y)),
				s.modelMatrix[1][2] * (2 / float32(extent.Y)),
			}

			off += util.HostWrite(obj, off, s)
		}
	}

	vcb.CopyBuffer(obj, cb.objectBuffer, []vxr.BufferCopyRegion{
		{
			Size: cb.objectBuffer.Size(),
		},
	})
	vcb.UpdateBuffer(cb.triangleBuffer, 0, []byte{0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0})

	vcb.BufferBarrier(vxr.BufferBarrier{
		Buffer: cb.objectBuffer,
		Src: vxr.BufferBarrierInfo{
			Stage:  vxr.PipelineStageTransfer,
			Access: vxr.AccessFlagMemoryWrite,
		},
		Dst: vxr.BufferBarrierInfo{
			Stage:  vxr.PipelineStageCompute,
			Access: vxr.AccessFlagMemoryRead | vxr.AccessFlagMemoryWrite,
		},
	}, vxr.BufferBarrier{
		Buffer: cb.triangleBuffer,
		Src: vxr.BufferBarrierInfo{
			Stage:  vxr.PipelineStageTransfer,
			Access: vxr.AccessFlagMemoryWrite,
		},
		Dst: vxr.BufferBarrierInfo{
			Stage:  vxr.PipelineStageCompute,
			Access: vxr.AccessFlagMemoryRead | vxr.AccessFlagMemoryWrite,
		},
	})

	vcb.Dispatch(instance.dispatcher, vxr.DispatchInfo{
		DescriptorSets: []*vxr.DescriptorSet{dsDispatch},
		ThreadCount:    gmath.Extent3u32{X: cb.objectCount, Y: 1, Z: 1},
	})

	vcb.CompoundBarrier(
		nil,
		[]vxr.BufferBarrier{
			{
				Buffer: cb.objectBuffer,
				Src: vxr.BufferBarrierInfo{
					Stage:  vxr.PipelineStageCompute,
					Access: vxr.AccessFlagMemoryWrite,
				},
				Dst: vxr.BufferBarrierInfo{
					Stage:  vxr.PipelineStageVertexShader,
					Access: vxr.AccessFlagMemoryRead,
				},
			},
			{
				Buffer: cb.triangleBuffer,
				Src: vxr.BufferBarrierInfo{
					Stage:  vxr.PipelineStageCompute,
					Access: vxr.AccessFlagMemoryWrite,
				},
				Dst: vxr.BufferBarrierInfo{
					Stage:  vxr.PipelineStageIndirect | vxr.PipelineStageVertexShader,
					Access: vxr.AccessFlagMemoryRead,
				},
			},
		},
		[]vxr.ImageBarrier{
			{
				Image: cb.depthImage,
				Src: vxr.ImageBarrierInfo{
					Stage:  vxr.PipelineStageFragmentTests,
					Access: vxr.AccessFlagMemoryWrite,
					Layout: vxr.ImageLayoutUndefined,
				},
				Dst: vxr.ImageBarrierInfo{
					Stage:  vxr.PipelineStageFragmentTests,
					Access: vxr.AccessFlagMemoryWrite,
					Layout: vxr.ImageLayoutAttachmentOptimal,
				},
				Range: vxr.ImageSubresourceRange{BaseMipLevel: 0, NumMipLevels: 1, BaseArrayLayer: 0, NumArrayLayers: 1},
			},
		},
	)

	vcb.RenderPassBegin("vxr/shapes",
		gmath.Recti32{W: output.Extent().X, H: output.Extent().Y},
		vxr.RenderParameters{
			FlipViewport: true,
		},
		vxr.RenderAttachments{
			Color: []vxr.RenderColorAttachment{
				{
					Image:               output,
					Layout:              vxr.ImageLayoutAttachmentOptimal,
					LoadOp:              vxr.RenderAttachmentLoadOpClear,
					StoreOp:             vxr.RenderAttachmentStoreOpStore,
					ColorBlendEnable:    true,
					ColorBlendEquation:  vxr.RenderColorAttachmentBlendPremultipliedAlpha(),
					ColorComponentFlags: output.Format().ColorComponentFlags(),
				},
			},
			Depth: vxr.RenderDepthAttachment{
				Image:   cb.depthImage,
				Layout:  vxr.ImageLayoutAttachmentOptimal,
				LoadOp:  vxr.RenderAttachmentLoadOpClear,
				StoreOp: vxr.RenderAttachmentStoreOpStore,
			},
		})
	vcb.DrawIndirect(instance.solid2DPipeline, vxr.DrawIndirectInfo{
		DrawParameters: vxr.DrawParameters{
			DescriptorSets:   []*vxr.DescriptorSet{dsDraw},
			DepthTestEnable:  true,
			DepthWriteEnable: true,
			DepthCompareOp:   vxr.CompareOpGreaterOrEqual,
		},
		IndirectBuffer: vxr.DrawIndirectBufferInfo{
			Buffer:    cb.triangleBuffer,
			DrawCount: 1,
		},
	})
	vcb.RenderPassEnd()

	vcb.EndNamedRegion()
}

func (cb *CommandBuffer2D) PostExecuteSrcImageBarrierInfo() vxr.ImageBarrierInfo {
	cb.noCopy.check()
	return vxr.ImageBarrierInfo{
		Stage:  vxr.PipelineStageRenderAttachmentWrite,
		Access: vxr.AccessFlagMemoryWrite,
		Layout: vxr.ImageLayoutAttachmentOptimal,
	}
}

func (cb *CommandBuffer2D) DrawTriangle(t Transform2D, c color.UNorm[uint8]) {
	cb.noCopy.check()
	shape := shape2d{
		polygonMode:   C.POLYGON_MODE_REGULAR_CONCAVE,
		triangleCount: 1,
		layer:         cb.objectCount,
		color:         c,
		modelMatrix:   t.modelMatrix(C.POLYGON_MODE_REGULAR_CONCAVE, 1),
	}

	if c.A == 255 {
		cb.shapesColored = append(cb.shapesColored, shape)
	} else {
		cb.shapesColoredAlpha = append(cb.shapesColoredAlpha, shape)
	}

	cb.objectCount++
	cb.triangleCount += shape.triangleCount
}

func (cb *CommandBuffer2D) DrawSquare(t Transform2D, c color.UNorm[uint8]) {
	cb.noCopy.check()
	shape := shape2d{
		polygonMode:   C.POLYGON_MODE_REGULAR_CONCAVE,
		triangleCount: 2,
		layer:         cb.objectCount,
		color:         c,
		modelMatrix:   t.modelMatrix(C.POLYGON_MODE_REGULAR_CONCAVE, 2),
	}

	if c.A == 255 {
		cb.shapesColored = append(cb.shapesColored, shape)
	} else {
		cb.shapesColoredAlpha = append(cb.shapesColoredAlpha, shape)
	}

	cb.objectCount++
	cb.triangleCount += shape.triangleCount
}

func (cb *CommandBuffer2D) DrawRegularNGon(sides uint32, t Transform2D, c color.UNorm[uint8]) {
	cb.noCopy.check()
	if sides < 3 {
		abort("The smallest possible shape is 3 sides")
	}

	shape := shape2d{
		polygonMode:   C.POLYGON_MODE_REGULAR_CONCAVE,
		triangleCount: sides,
		layer:         cb.objectCount,
		color:         c,
		modelMatrix:   t.modelMatrix(C.POLYGON_MODE_REGULAR_CONCAVE, sides),
	}

	if c.A == 255 {
		cb.shapesColored = append(cb.shapesColored, shape)
	} else {
		cb.shapesColoredAlpha = append(cb.shapesColoredAlpha, shape)
	}

	cb.objectCount++
	cb.triangleCount += shape.triangleCount
}

func (cb *CommandBuffer2D) DrawRegularNGonStar(sides uint32, thickness float32, t Transform2D, c color.UNorm[uint8]) {
	cb.noCopy.check()
	if sides < 4 {
		abort("The smallest possible shape is 4 sides")
	}
	if !gmath.InRange(thickness, 0, 1) {
		abort("Thickness must be between 0 and 1")
	}

	shape := shape2d{
		polygonMode:   C.POLYGON_MODE_REGULAR_STAR,
		triangleCount: sides,
		layer:         cb.objectCount,
		parameter1:    thickness,
		color:         c,
		modelMatrix:   t.modelMatrix(C.POLYGON_MODE_REGULAR_STAR, sides),
	}

	if c.A == 255 {
		cb.shapesColored = append(cb.shapesColored, shape)
	} else {
		cb.shapesColoredAlpha = append(cb.shapesColoredAlpha, shape)
	}

	cb.objectCount++
	cb.triangleCount += shape.triangleCount
}
