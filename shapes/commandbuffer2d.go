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
	"encoding/binary"
	"unsafe"

	"goarrg.com/gmath"
	"goarrg.com/gmath/color"
	"goarrg.com/rhi/vxr"
	"goarrg.com/rhi/vxr/internal/util"
	"goarrg.com/rhi/vxr/managed"
)

type cbState uint

const (
	cbIdle cbState = iota
	cbRecording
	cbExecutedPrePass
	cbExecutedDraw
)

type shape2d struct {
	polygonMode    uint32
	triangleOffset uint32
	triangleCount  uint32
	layer          uint32
	parameter1     float32
	color          uint32
	modelMatrix    [2][3]float32
}

type CommandBuffer2D struct {
	noCopy  util.NoCopy
	cbState cbState
	shapes  []shape2d

	descriptorSetDrawInfo  *vxr.DescriptorSet
	descriptorSetTextures  *vxr.DescriptorSet
	managedTextureBindings *managed.DescriptorArrayImage

	objectCount    uint32
	objectCapacity uint32
	objectBuffer   *vxr.DeviceBuffer

	triangleCount    uint32
	triangleCapacity uint32
	triangleBuffer   *vxr.DeviceBuffer
}

func (cb *CommandBuffer2D) Begin() {
	if cb.noCopy.InitLazy() {
		cb.descriptorSetTextures = instance.solid2DPipeline.Layout.NewDescriptorSet(1)
		cb.descriptorSetTextures.Bind(0, 0, instance.linearSampler)
		cb.managedTextureBindings = managed.NewDescriptorArrayImage(cb.descriptorSetTextures, 1)
	}
	if cb.cbState == cbRecording {
		abort("Begin() called while CommandBuffer2D is not idle")
	}

	cb.shapes = cb.shapes[:0]
	cb.objectCount = 0
	cb.triangleCount = 0

	cb.cbState = cbRecording
}

func (cb *CommandBuffer2D) End() {
	cb.noCopy.Check()
	if cb.cbState != cbRecording {
		abort("End() called while CommandBuffer2D is not in a recording state")
	}
	cb.cbState = cbIdle
}

/*
Destroy will destroy all persistent objects, caller is responsible for synchronization.
*/
func (cb *CommandBuffer2D) Destroy() {
	cb.noCopy.Check()
	cb.descriptorSetTextures.Destroy()
	cb.objectBuffer.Destroy()
	cb.triangleBuffer.Destroy()
	cb.noCopy.Close()
	*cb = CommandBuffer2D{}
}

func (cb *CommandBuffer2D) BindTexture(img vxr.DescriptorImageInfo) int {
	return cb.managedTextureBindings.Push(img)
}

func (cb *CommandBuffer2D) UnBindTexture(f *vxr.Frame, img vxr.Image) {
	cb.managedTextureBindings.Pop(f, img)
}

/*
ExecutePrePass records commands that has to happen before ExecuteDraw.
It must be called outside a renderpass.
*/
func (cb *CommandBuffer2D) ExecutePrePass(frame *vxr.Frame, vcb *vxr.GraphicsCommandBuffer, viewport gmath.Extent2i32) {
	cb.noCopy.Check()
	switch cb.cbState {
	case cbRecording:
		abort("ExecutePrePass(...) called while CommandBuffer2D is not idle")
	case cbExecutedPrePass:
		abort("ExecutePrePass(...) called twice without ExecuteDraw(...)")
	}
	cb.descriptorSetDrawInfo = instance.solid2DPipeline.Layout.NewDescriptorSet(0)
	frame.QueueDestory(cb.descriptorSetDrawInfo)
	vcb.BeginNamedRegion("shapes2d-prepass")
	{
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

		cb.descriptorSetDrawInfo.Bind(0, 0, vxr.DescriptorBufferInfo{
			Buffer: cb.objectBuffer,
		})
		cb.descriptorSetDrawInfo.Bind(1, 0, vxr.DescriptorBufferInfo{
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
			for _, s := range cb.shapes {
				s.modelMatrix[0] = [3]float32{
					s.modelMatrix[0][0] * (2 / float32(viewport.X)),
					s.modelMatrix[0][1] * (2 / float32(viewport.X)),
					s.modelMatrix[0][2] * (2 / float32(viewport.X)),
				}
				s.modelMatrix[1] = [3]float32{
					s.modelMatrix[1][0] * (2 / float32(viewport.Y)),
					s.modelMatrix[1][1] * (2 / float32(viewport.Y)),
					s.modelMatrix[1][2] * (2 / float32(viewport.Y)),
				}

				off += util.HostWrite(obj, off, s)
			}
		}

		vcb.CopyBuffer(obj, cb.objectBuffer, []vxr.BufferCopyRegion{
			{
				Size: cb.objectBuffer.Size(),
			},
		})
		{
			indirect := []byte{0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}
			_, _ = binary.Append(indirect[:0], binary.NativeEndian, cb.triangleCount*3)
			vcb.UpdateBuffer(cb.triangleBuffer, 0, indirect)
		}

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
			DescriptorSets: []*vxr.DescriptorSet{cb.descriptorSetDrawInfo, cb.descriptorSetTextures},
			ThreadCount:    gmath.Extent3u32{X: cb.objectCount, Y: 1, Z: 1},
		})

		vcb.BufferBarrier(
			vxr.BufferBarrier{
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
			vxr.BufferBarrier{
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
		)
	}
	vcb.EndNamedRegion()

	cb.cbState = cbExecutedPrePass
}

/*
ExecuteDraw records draw commands that has to happen after ExecutePrePass.
It must be called inside a renderpass.
*/
func (cb *CommandBuffer2D) ExecuteDraw(frame *vxr.Frame, vcb *vxr.GraphicsCommandBuffer) {
	cb.noCopy.Check()
	if cb.cbState != cbExecutedPrePass {
		abort("ExecuteDraw(...) called before ExecutePrePass(...)")
	}
	vcb.BeginNamedRegion("shapes2d-draw")

	vcb.DrawIndirect(instance.solid2DPipeline, vxr.DrawIndirectInfo{
		DrawParameters: vxr.DrawParameters{
			DescriptorSets:   []*vxr.DescriptorSet{cb.descriptorSetDrawInfo, cb.descriptorSetTextures},
			DepthTestEnable:  true,
			DepthWriteEnable: true,
			DepthCompareOp:   vxr.CompareOpGreaterOrEqual,
		},
		IndirectBuffer: vxr.DrawIndirectBufferInfo{
			Buffer:    cb.triangleBuffer,
			DrawCount: 1,
		},
	})

	vcb.EndNamedRegion()

	cb.cbState = cbExecutedDraw
}

func (cb *CommandBuffer2D) drawTriangle(t Transform2D, c uint32, flags uint32) {
	cb.noCopy.Check()
	if cb.cbState != cbRecording {
		abort("Draw*() called while CommandBuffer2D is not in a recording state")
	}
	shape := shape2d{
		polygonMode:    C.POLYGON_MODE_REGULAR_CONCAVE | flags,
		triangleOffset: cb.triangleCount,
		triangleCount:  1,
		layer:          cb.objectCount,
		color:          c,
		modelMatrix:    t.modelMatrix(C.POLYGON_MODE_REGULAR_CONCAVE, 1),
	}

	cb.shapes = append(cb.shapes, shape)
	cb.objectCount++
	cb.triangleCount += shape.triangleCount
}

func (cb *CommandBuffer2D) DrawTriangle(t Transform2D, c color.UNorm[uint8]) {
	cb.drawTriangle(t, *(*uint32)(unsafe.Pointer(&c)), 0)
}

func (cb *CommandBuffer2D) DrawTriangleTextured(t Transform2D, texture uint32) {
	cb.drawTriangle(t, texture, C.POLYGON_MODE_TEXTURED_BIT)
}

func (cb *CommandBuffer2D) drawSquare(t Transform2D, c uint32, flags uint32) {
	cb.noCopy.Check()
	if cb.cbState != cbRecording {
		abort("Draw*() called while CommandBuffer2D is not in a recording state")
	}
	shape := shape2d{
		polygonMode:    C.POLYGON_MODE_REGULAR_CONCAVE | flags,
		triangleOffset: cb.triangleCount,
		triangleCount:  2,
		layer:          cb.objectCount,
		color:          c,
		modelMatrix:    t.modelMatrix(C.POLYGON_MODE_REGULAR_CONCAVE, 2),
	}

	cb.shapes = append(cb.shapes, shape)
	cb.objectCount++
	cb.triangleCount += shape.triangleCount
}

func (cb *CommandBuffer2D) DrawSquare(t Transform2D, c color.UNorm[uint8]) {
	cb.drawSquare(t, *(*uint32)(unsafe.Pointer(&c)), 0)
}

func (cb *CommandBuffer2D) DrawSquareTextured(t Transform2D, texture uint32) {
	cb.drawSquare(t, texture, C.POLYGON_MODE_TEXTURED_BIT)
}

func (cb *CommandBuffer2D) drawRegularNGon(sides uint32, t Transform2D, c uint32, flags uint32) {
	cb.noCopy.Check()
	if cb.cbState != cbRecording {
		abort("Draw*() called while CommandBuffer2D is not in a recording state")
	}
	if sides < 3 {
		abort("The smallest possible shape is 3 sides")
	}

	shape := shape2d{
		polygonMode:    C.POLYGON_MODE_REGULAR_CONCAVE | flags,
		triangleOffset: cb.triangleCount,
		triangleCount:  sides,
		layer:          cb.objectCount,
		color:          c,
		modelMatrix:    t.modelMatrix(C.POLYGON_MODE_REGULAR_CONCAVE, sides),
	}

	cb.shapes = append(cb.shapes, shape)
	cb.objectCount++
	if sides > 4 {
		cb.triangleCount += shape.triangleCount
	} else {
		cb.triangleCount += shape.triangleCount - 2
	}
}

func (cb *CommandBuffer2D) DrawRegularNGon(sides uint32, t Transform2D, c color.UNorm[uint8]) {
	cb.drawRegularNGon(sides, t, *(*uint32)(unsafe.Pointer(&c)), 0)
}

func (cb *CommandBuffer2D) DrawRegularNGonTextured(sides uint32, t Transform2D, texture uint32) {
	cb.drawRegularNGon(sides, t, texture, C.POLYGON_MODE_TEXTURED_BIT)
}

func (cb *CommandBuffer2D) drawRegularNGonStar(sides uint32, thickness float32, t Transform2D, c uint32, flags uint32) {
	cb.noCopy.Check()
	if cb.cbState != cbRecording {
		abort("Draw*() called while CommandBuffer2D is not in a recording state")
	}
	if sides < 4 {
		abort("The smallest possible shape is 4 sides")
	}
	if !gmath.InRange(thickness, 0, 1) {
		abort("Thickness must be between 0 and 1")
	}

	shape := shape2d{
		polygonMode:    C.POLYGON_MODE_REGULAR_STAR | flags,
		triangleOffset: cb.triangleCount,
		triangleCount:  sides,
		layer:          cb.objectCount,
		parameter1:     thickness,
		color:          c,
		modelMatrix:    t.modelMatrix(C.POLYGON_MODE_REGULAR_STAR, sides),
	}

	cb.shapes = append(cb.shapes, shape)
	cb.objectCount++
	cb.triangleCount += shape.triangleCount * 2
}

func (cb *CommandBuffer2D) DrawRegularNGonStar(sides uint32, thickness float32, t Transform2D, c color.UNorm[uint8]) {
	cb.drawRegularNGonStar(sides, thickness, t, *(*uint32)(unsafe.Pointer(&c)), 0)
}

func (cb *CommandBuffer2D) DrawRegularNGonStarTextured(sides uint32, thickness float32, t Transform2D, texture uint32) {
	cb.drawRegularNGonStar(sides, thickness, t, texture, C.POLYGON_MODE_TEXTURED_BIT)
}
