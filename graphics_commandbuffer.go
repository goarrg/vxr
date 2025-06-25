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
	"strings"
	"unsafe"

	"goarrg.com/gmath"
	"goarrg.com/rhi/vxr/internal/vk"
)

type renderPass struct {
	id                     string
	name                   string
	numColorAttachments    int
	fragmentOutputPipeline C.VkPipeline
}

type GraphicsCommandBuffer struct {
	ComputeCommandBuffer

	cFrame            C.vxr_vk_graphics_frame
	currentRenderPass renderPass
}

type RenderParameters struct {
	// FlipViewport flips the viewport's origin.
	// goarrg assumes a bottom left origin, so viewports are flipped by default,
	// enable to flip again to use vulkan's default top left origin.
	FlipViewport bool
}

type RenderAttachmentLoadOp C.VkAttachmentLoadOp

const (
	RenderAttachmentLoadOpLoad     RenderAttachmentLoadOp = vk.ATTACHMENT_LOAD_OP_LOAD
	RenderAttachmentLoadOpClear    RenderAttachmentLoadOp = vk.ATTACHMENT_LOAD_OP_CLEAR
	RenderAttachmentLoadOpDontCare RenderAttachmentLoadOp = vk.ATTACHMENT_LOAD_OP_DONT_CARE
	// requires vk1.4 or VK_EXT_load_store_op_none
	// load+store none means no access which means no sync required
	RenderAttachmentLoadOpNone RenderAttachmentLoadOp = vk.ATTACHMENT_LOAD_OP_NONE
)

type RenderAttachmentStoreOp C.VkAttachmentStoreOp

const (
	RenderAttachmentStoreOpStore    RenderAttachmentStoreOp = vk.ATTACHMENT_STORE_OP_STORE
	RenderAttachmentStoreOpDontCare RenderAttachmentStoreOp = vk.ATTACHMENT_STORE_OP_DONT_CARE
	RenderAttachmentStoreOpNone     RenderAttachmentStoreOp = vk.ATTACHMENT_STORE_OP_NONE
)

type BlendEquation struct {
	Src BlendFactor
	Dst BlendFactor
	Op  BlendOp
}

type RenderColorAttachmentBlendEquation struct {
	Color BlendEquation
	Alpha BlendEquation
}

func RenderColorAttachmentBlendAlpha() RenderColorAttachmentBlendEquation {
	return RenderColorAttachmentBlendEquation{
		Color: BlendEquation{
			Src: BLEND_FACTOR_SRC_ALPHA,
			Dst: BLEND_FACTOR_ONE_MINUS_SRC_ALPHA,
			Op:  BLEND_OP_ADD,
		},
		Alpha: BlendEquation{
			Src: BLEND_FACTOR_ONE,
			Dst: BLEND_FACTOR_ONE_MINUS_SRC_ALPHA,
			Op:  BLEND_OP_ADD,
		},
	}
}

func RenderColorAttachmentBlendPremultipliedAlpha() RenderColorAttachmentBlendEquation {
	return RenderColorAttachmentBlendEquation{
		Color: BlendEquation{
			Src: BLEND_FACTOR_ONE,
			Dst: BLEND_FACTOR_ONE_MINUS_SRC_ALPHA,
			Op:  BLEND_OP_ADD,
		},
		Alpha: BlendEquation{
			Src: BLEND_FACTOR_ONE,
			Dst: BLEND_FACTOR_ONE_MINUS_SRC_ALPHA,
			Op:  BLEND_OP_ADD,
		},
	}
}

type RenderColorBlendParameters struct {
	Enable         bool
	Equation       RenderColorAttachmentBlendEquation
	ComponentFlags ColorComponentFlags
}

type RenderColorAttachment struct {
	Image      ColorImage
	Layout     ImageLayout
	LoadOp     RenderAttachmentLoadOp
	StoreOp    RenderAttachmentStoreOp
	ClearValue ColorImageClearValue

	ColorBlend RenderColorBlendParameters
}

type RenderClearDepthValue struct {
	Depth float32
	_     uint32
	_     [2]uint32
}

func (c RenderClearDepthValue) vkClearValue() [16]byte {
	return *(*[unsafe.Sizeof(RenderClearDepthValue{})]byte)(unsafe.Pointer(&c))
}

type RenderClearStencilValue struct {
	_       float32
	Stencil uint32
	_       [2]uint32
}

func (c RenderClearStencilValue) vkClearValue() [16]byte {
	return *(*[unsafe.Sizeof(RenderClearStencilValue{})]byte)(unsafe.Pointer(&c))
}

type RenderDepthAttachment struct {
	Image      DepthStencilImage
	Layout     ImageLayout
	LoadOp     RenderAttachmentLoadOp
	StoreOp    RenderAttachmentStoreOp
	ClearValue RenderClearDepthValue
}

type RenderStencilAttachment struct {
	Image      DepthStencilImage
	Layout     ImageLayout
	LoadOp     RenderAttachmentLoadOp
	StoreOp    RenderAttachmentStoreOp
	ClearValue RenderClearStencilValue
}

type RenderAttachments struct {
	Color   []RenderColorAttachment
	Depth   RenderDepthAttachment
	Stencil RenderStencilAttachment
}

func (cb *GraphicsCommandBuffer) RenderPassBegin(name string, area gmath.Recti32, parameters RenderParameters, attachments RenderAttachments) {
	cb.noCopy.check()
	if cb.currentRenderPass != (renderPass{}) {
		abort("RenderPassBegin called when there's an active renderpass")
	}
	if (attachments.Depth.Image != nil) && (attachments.Stencil.Image != nil) && (attachments.Depth.Image != attachments.Stencil.Image) {
		abort("Depth and Stencil ImageViews must be the same if both are not nil")
	}

	cAttachments := make([]C.VkRenderingAttachmentInfo, len(attachments.Color))
	cColorBlendEnable := make([]C.VkBool32, len(attachments.Color))
	cColorBlendEquation := make([]C.VkColorBlendEquationEXT, len(attachments.Color))
	cColorComponentFlags := make([]C.VkColorComponentFlags, len(attachments.Color))
	vkColorFormats := make([]C.VkFormat, len(attachments.Color))

	if len(attachments.Color) == 0 {
		cb.currentRenderPass.id = "null,"
		cb.currentRenderPass.name = "null,"
	}
	for i, attachment := range attachments.Color {
		cAttachments[i] = C.VkRenderingAttachmentInfo{
			sType:       vk.STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO,
			imageView:   attachment.Image.vkImageView(),
			imageLayout: C.VkImageLayout(attachment.Layout),
			loadOp:      C.VkAttachmentLoadOp(attachment.LoadOp),
			storeOp:     C.VkAttachmentStoreOp(attachment.StoreOp),
		}
		if attachment.ColorBlend.Enable {
			cColorBlendEnable[i] = vk.TRUE
		}
		cColorBlendEquation[i] = C.VkColorBlendEquationEXT{
			srcColorBlendFactor: C.VkBlendFactor(attachment.ColorBlend.Equation.Color.Src),
			dstColorBlendFactor: C.VkBlendFactor(attachment.ColorBlend.Equation.Color.Dst),
			colorBlendOp:        C.VkBlendOp(attachment.ColorBlend.Equation.Color.Op),

			srcAlphaBlendFactor: C.VkBlendFactor(attachment.ColorBlend.Equation.Alpha.Src),
			dstAlphaBlendFactor: C.VkBlendFactor(attachment.ColorBlend.Equation.Alpha.Dst),
			alphaBlendOp:        C.VkBlendOp(attachment.ColorBlend.Equation.Alpha.Op),
		}
		cColorComponentFlags[i] = C.VkColorComponentFlags(attachment.ColorBlend.ComponentFlags)
		vkColorFormats[i] = attachments.Color[i].Image.vkFormat()
		cb.currentRenderPass.id += fmt.Sprintf("%s,", toHex(vkColorFormats[i]))
		cb.currentRenderPass.name += fmt.Sprintf("%s,", attachments.Color[i].Image.Format().String())
		if attachment.LoadOp == RenderAttachmentLoadOpClear && attachments.Color[i].ClearValue != nil {
			cAttachments[i].clearValue = attachments.Color[i].ClearValue.vkClearValue()
		}
	}

	{
		cInfo := C.vxr_vk_graphics_renderPassInfo{
			renderingInfo: C.VkRenderingInfo{
				sType: vk.STRUCTURE_TYPE_RENDERING_INFO,
				renderArea: C.VkRect2D{
					offset: C.VkOffset2D{C.int32_t(area.X), C.int32_t(area.Y)},
					extent: C.VkExtent2D{C.uint32_t(area.W), C.uint32_t(area.H)},
				},
				layerCount:           1,
				colorAttachmentCount: C.uint32_t(len(attachments.Color)),
				pColorAttachments:    unsafe.SliceData(cAttachments),
			},
			colorBlendInfo: C.vxr_vk_graphics_colorBlendInfo{
				enable:         unsafe.SliceData(cColorBlendEnable),
				equation:       unsafe.SliceData(cColorBlendEquation),
				componentFlags: unsafe.SliceData(cColorComponentFlags),
			},
		}
		if parameters.FlipViewport {
			cInfo.flipViewport = vk.TRUE
		}
		if attachments.Depth.Image != nil {
			depthAttachment := &C.VkRenderingAttachmentInfo{
				sType:       vk.STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO,
				imageView:   attachments.Depth.Image.vkImageView(),
				imageLayout: C.VkImageLayout(attachments.Depth.Layout),
				loadOp:      C.VkAttachmentLoadOp(attachments.Depth.LoadOp),
				storeOp:     C.VkAttachmentStoreOp(attachments.Depth.StoreOp),
				clearValue:  attachments.Depth.ClearValue.vkClearValue(),
			}
			defer runtime.KeepAlive(depthAttachment)
			cInfo.renderingInfo.pDepthAttachment = depthAttachment
			cb.currentRenderPass.id += fmt.Sprintf("%s,", toHex(attachments.Depth.Image.vkFormat()))
			cb.currentRenderPass.name += fmt.Sprintf("%s,", attachments.Depth.Image.Format().String())
		}
		if attachments.Stencil.Image != nil {
			stencilAttachment := &C.VkRenderingAttachmentInfo{
				sType:       vk.STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO,
				imageView:   attachments.Stencil.Image.vkImageView(),
				imageLayout: C.VkImageLayout(attachments.Stencil.Layout),
				loadOp:      C.VkAttachmentLoadOp(attachments.Stencil.LoadOp),
				storeOp:     C.VkAttachmentStoreOp(attachments.Stencil.StoreOp),
				clearValue:  attachments.Stencil.ClearValue.vkClearValue(),
			}
			defer runtime.KeepAlive(stencilAttachment)
			cInfo.renderingInfo.pStencilAttachment = stencilAttachment
			cb.currentRenderPass.id += fmt.Sprintf("%s,", toHex(attachments.Stencil.Image.vkFormat()))
			cb.currentRenderPass.name += fmt.Sprintf("%s,", attachments.Stencil.Image.Format().String())
		}

		C.vxr_vk_graphics_renderPassBegin(instance.cInstance, cb.vkCommandBuffer,
			C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))), cInfo)
		runtime.KeepAlive(name)
		runtime.KeepAlive(cAttachments)
		runtime.KeepAlive(cColorBlendEnable)
		runtime.KeepAlive(cColorBlendEquation)
		runtime.KeepAlive(cColorComponentFlags)
	}

	{
		cb.currentRenderPass.id = fmt.Sprintf("[fragment_output:[%s]]", strings.TrimSuffix(cb.currentRenderPass.id, ","))
		cb.currentRenderPass.name = fmt.Sprintf("[%s]", strings.TrimSuffix(cb.currentRenderPass.name, ","))
		cb.currentRenderPass.numColorAttachments = len(attachments.Color)
		cb.currentRenderPass.fragmentOutputPipeline = instance.graphics.pipelineCache.createOrRetrievePipeline(cb.currentRenderPass.id, func() C.VkPipeline {
			cInfo := C.vxr_vk_graphics_fragmentOutputPipelineCreateInfo{
				numColorAttachments:    C.uint32_t(len(attachments.Color)),
				colorAttachmentFormats: unsafe.SliceData(vkColorFormats),
			}
			if attachments.Depth.Image != nil {
				cInfo.depthFormat = attachments.Depth.Image.vkFormat()
			}
			if attachments.Stencil.Image != nil {
				cInfo.stencilFormat = attachments.Stencil.Image.vkFormat()
			}
			C.vxr_vk_graphics_createFragmentOutputPipeline(instance.cInstance,
				C.size_t(len(cb.currentRenderPass.name)), (*C.char)(unsafe.Pointer(unsafe.StringData(cb.currentRenderPass.name))),
				cInfo, &cb.currentRenderPass.fragmentOutputPipeline)
			runtime.KeepAlive(vkColorFormats)
			return cb.currentRenderPass.fragmentOutputPipeline
		})
		cb.currentRenderPass.id = genID(cb.currentRenderPass.fragmentOutputPipeline)
	}
}

func (cb *GraphicsCommandBuffer) RenderPassSetViewport(flip bool, viewport gmath.Recti32) {
	cb.noCopy.check()
	if cb.currentRenderPass == (renderPass{}) {
		abort("RenderPassSetViewport called outside a renderpass")
	}
	cViewport := C.VkViewport{
		x: C.float(viewport.X), y: C.float(viewport.Y),
		width: C.float(viewport.W), height: C.float(viewport.H),
		maxDepth: 1,
	}
	if flip {
		C.vxr_vk_graphics_renderPassSetViewport(instance.cInstance, cb.vkCommandBuffer, vk.TRUE, cViewport)
	} else {
		C.vxr_vk_graphics_renderPassSetViewport(instance.cInstance, cb.vkCommandBuffer, vk.FALSE, cViewport)
	}
}

func (cb *GraphicsCommandBuffer) RenderPassSetScissor(rect gmath.Recti32) {
	cb.noCopy.check()
	if cb.currentRenderPass == (renderPass{}) {
		abort("RenderPassSetScissor called outside a renderpass")
	}
	cRect := C.VkRect2D{
		offset: C.VkOffset2D{C.int32_t(rect.X), C.int32_t(rect.Y)},
		extent: C.VkExtent2D{C.uint32_t(rect.W), C.uint32_t(rect.H)},
	}
	C.vxr_vk_graphics_renderPassSetScissor(instance.cInstance, cb.vkCommandBuffer, cRect)
}

func (cb *GraphicsCommandBuffer) RenderPassSetViewportAndScissor(flip bool, viewport gmath.Recti32, rect gmath.Recti32) {
	cb.noCopy.check()
	if cb.currentRenderPass == (renderPass{}) {
		abort("RenderPassSetViewportAndScissor called outside a renderpass")
	}
	cViewport := C.VkViewport{
		x: C.float(viewport.X), y: C.float(viewport.Y),
		width: C.float(viewport.W), height: C.float(viewport.H),
		maxDepth: 1,
	}
	cRect := C.VkRect2D{
		offset: C.VkOffset2D{C.int32_t(rect.X), C.int32_t(rect.Y)},
		extent: C.VkExtent2D{C.uint32_t(rect.W), C.uint32_t(rect.H)},
	}
	if flip {
		C.vxr_vk_graphics_renderPassSetViewportAndScissor(instance.cInstance, cb.vkCommandBuffer, vk.TRUE, cViewport, cRect)
	} else {
		C.vxr_vk_graphics_renderPassSetViewportAndScissor(instance.cInstance, cb.vkCommandBuffer, vk.FALSE, cViewport, cRect)
	}
}

func (cb *GraphicsCommandBuffer) RenderPassSetColorBlendParameters(firstAttachment int, infos []RenderColorBlendParameters) {
	cb.noCopy.check()
	if cb.currentRenderPass == (renderPass{}) {
		abort("RenderPassSetColorBlendParameters called outside a renderpass")
	}
	if (firstAttachment + len(infos)) > cb.currentRenderPass.numColorAttachments {
		abort("RenderPassSetColorBlendParameters called with more RenderColorBlendParameters than there are color attachments after taking into account firstAttachment")
	}
	cColorBlendEnable := make([]C.VkBool32, len(infos))
	cColorBlendEquation := make([]C.VkColorBlendEquationEXT, len(infos))
	cColorComponentFlags := make([]C.VkColorComponentFlags, len(infos))

	for i, info := range infos {
		if info.Enable {
			cColorBlendEnable[i] = vk.TRUE
		}
		cColorBlendEquation[i] = C.VkColorBlendEquationEXT{
			srcColorBlendFactor: C.VkBlendFactor(info.Equation.Color.Src),
			dstColorBlendFactor: C.VkBlendFactor(info.Equation.Color.Dst),
			colorBlendOp:        C.VkBlendOp(info.Equation.Color.Op),

			srcAlphaBlendFactor: C.VkBlendFactor(info.Equation.Alpha.Src),
			dstAlphaBlendFactor: C.VkBlendFactor(info.Equation.Alpha.Dst),
			alphaBlendOp:        C.VkBlendOp(info.Equation.Alpha.Op),
		}
		cColorComponentFlags[i] = C.VkColorComponentFlags(info.ComponentFlags)
	}

	cInfo := C.vxr_vk_graphics_colorBlendInfo{
		enable:         unsafe.SliceData(cColorBlendEnable),
		equation:       unsafe.SliceData(cColorBlendEquation),
		componentFlags: unsafe.SliceData(cColorComponentFlags),
	}
	C.vxr_vk_graphics_renderPassSetColorBlend(instance.cInstance, cb.vkCommandBuffer, C.uint32_t(firstAttachment), C.uint32_t(len(infos)), cInfo)
}

type CullMode C.VkCullModeFlags

const (
	CullModeNone         CullMode = vk.CULL_MODE_NONE
	CullModeFront        CullMode = vk.CULL_MODE_FRONT_BIT
	CullModeBack         CullMode = vk.CULL_MODE_BACK_BIT
	CullModeFrontAndBack CullMode = vk.CULL_MODE_FRONT_AND_BACK
)

type FrontFace C.VkFrontFace

const (
	FrontFaceCounterClockwise FrontFace = vk.FRONT_FACE_COUNTER_CLOCKWISE
	FrontFaceClockwise        FrontFace = vk.FRONT_FACE_CLOCKWISE
)

type CompareOp C.VkCompareOp

const (
	CompareOpNever          CompareOp = vk.COMPARE_OP_NEVER
	CompareOpLess           CompareOp = vk.COMPARE_OP_LESS
	CompareOpEqual          CompareOp = vk.COMPARE_OP_EQUAL
	CompareOpLessOrEqual    CompareOp = vk.COMPARE_OP_LESS_OR_EQUAL
	CompareOpGreater        CompareOp = vk.COMPARE_OP_GREATER
	CompareOpNotEqual       CompareOp = vk.COMPARE_OP_NOT_EQUAL
	CompareOpGreaterOrEqual CompareOp = vk.COMPARE_OP_GREATER_OR_EQUAL
	CompareOpAlways         CompareOp = vk.COMPARE_OP_ALWAYS
)

type StencilOp C.VkStencilOp

const (
	StencilOpKeep              StencilOp = vk.STENCIL_OP_KEEP
	StencilOpZero              StencilOp = vk.STENCIL_OP_ZERO
	StencilOpReplace           StencilOp = vk.STENCIL_OP_REPLACE
	StencilOpIncrementAndClamp StencilOp = vk.STENCIL_OP_INCREMENT_AND_CLAMP
	StencilOpDecrementAndClamp StencilOp = vk.STENCIL_OP_DECREMENT_AND_CLAMP
	StencilOpInvert            StencilOp = vk.STENCIL_OP_INVERT
	StencilOpIncrementAndWrap  StencilOp = vk.STENCIL_OP_INCREMENT_AND_WRAP
	StencilOpDecrementAndWrap  StencilOp = vk.STENCIL_OP_DECREMENT_AND_WRAP
)

type StencilTestParameters struct {
	FailOp      StencilOp
	PassOp      StencilOp
	DepthFailOp StencilOp
	CompareOp   CompareOp
	CompareMask uint32
	WriteMask   uint32
	Reference   uint32
}

type DrawParameters struct {
	PushConstants  []byte
	DescriptorSets []*DescriptorSet

	CullMode  CullMode
	FrontFace FrontFace

	DepthTestEnable  bool
	DepthWriteEnable bool
	DepthCompareOp   CompareOp

	StencilTestEnable          bool
	StencilFrontFaceParameters StencilTestParameters
	StencilBackFaceParameters  StencilTestParameters
}

func (cb *GraphicsCommandBuffer) draw(p GraphicsPipelineLibrary, info DrawParameters, fn func(C.vxr_vk_graphics_drawParameters)) {
	cb.noCopy.check()

	if cb.currentRenderPass == (renderPass{}) {
		abort("Draw called outside a renderpass")
	}
	if err := p.validate(); err != nil {
		abort("Failed to validate GraphicsPipeline: %s", err)
	}
	if err := p.Layout.cmdValidate(info.PushConstants, info.DescriptorSets); err != nil {
		abort("Failed to validate DrawParameters: %s", err)
	}

	descriptorSets := make([]C.VkDescriptorSet, 0, len(info.DescriptorSets))
	defer runtime.KeepAlive(descriptorSets)
	for _, s := range info.DescriptorSets {
		s.noCopy.check()
		descriptorSets = append(descriptorSets, s.cDescriptorSet)
	}

	id := chainIDs(p.VertexInput.id, p.VertexShader.id, p.FragmentShader.id, cb.currentRenderPass.id)
	name := chainIDs(p.VertexInput.name, p.VertexShader.name, p.FragmentShader.name, cb.currentRenderPass.name)
	cParameters := C.vxr_vk_graphics_drawParameters{
		layout: p.Layout.vkPipelinelayout,
		pipeline: instance.graphics.pipelineCache.linkOrRetrieveExecutablePipeline(id, name, p.Layout.vkPipelinelayout,
			[]C.VkPipeline{p.VertexInput.vkPipeline, p.VertexShader.vkPipeline, p.FragmentShader.vkPipeline, cb.currentRenderPass.fragmentOutputPipeline}),
		topology: p.VertexInput.topology,

		cullMode:  C.VkCullModeFlags(info.CullMode),
		frontFace: C.VkFrontFace(info.FrontFace),

		depthCompareOp: C.VkCompareOp(info.DepthCompareOp),

		stencilTestFrontFace: C.VkStencilOpState{
			failOp:      C.VkStencilOp(info.StencilFrontFaceParameters.FailOp),
			passOp:      C.VkStencilOp(info.StencilFrontFaceParameters.PassOp),
			depthFailOp: C.VkStencilOp(info.StencilFrontFaceParameters.DepthFailOp),
			compareOp:   C.VkCompareOp(info.StencilFrontFaceParameters.CompareOp),
			compareMask: C.uint32_t(info.StencilFrontFaceParameters.CompareMask),
			writeMask:   C.uint32_t(info.StencilFrontFaceParameters.WriteMask),
			reference:   C.uint32_t(info.StencilFrontFaceParameters.Reference),
		},
		stencilTestBackFace: C.VkStencilOpState{
			failOp:      C.VkStencilOp(info.StencilBackFaceParameters.FailOp),
			passOp:      C.VkStencilOp(info.StencilBackFaceParameters.PassOp),
			depthFailOp: C.VkStencilOp(info.StencilBackFaceParameters.DepthFailOp),
			compareOp:   C.VkCompareOp(info.StencilBackFaceParameters.CompareOp),
			compareMask: C.uint32_t(info.StencilBackFaceParameters.CompareMask),
			writeMask:   C.uint32_t(info.StencilBackFaceParameters.WriteMask),
			reference:   C.uint32_t(info.StencilBackFaceParameters.Reference),
		},

		numDescriptorSets: C.uint32_t(len(descriptorSets)),
		descriptorSets:    unsafe.SliceData(descriptorSets),
	}

	if p.Layout.pushConstantRange.size > 0 {
		defer runtime.KeepAlive(info.PushConstants)
		cParameters.pushConstantRange = p.Layout.pushConstantRange
		cParameters.pushConstantData = unsafe.Pointer(unsafe.SliceData(info.PushConstants))
	}
	if info.DepthTestEnable {
		cParameters.depthTestEnable = vk.TRUE
	}
	if info.DepthWriteEnable {
		cParameters.depthWriteEnable = vk.TRUE
	}
	if info.StencilTestEnable {
		cParameters.stencilTestEnable = vk.TRUE
	}
	fn(cParameters)
}

type DrawInfo struct {
	DrawParameters DrawParameters
	VertexCount    uint32
	InstanceCount  uint32
}

func (cb *GraphicsCommandBuffer) Draw(p GraphicsPipelineLibrary, info DrawInfo) {
	cb.noCopy.check()
	cb.draw(p, info.DrawParameters, func(cParmameters C.vxr_vk_graphics_drawParameters) {
		cInfo := C.vxr_vk_graphics_drawInfo{
			parameters: cParmameters,

			vertexCount:   C.uint32_t(info.VertexCount),
			instanceCount: C.uint32_t(info.InstanceCount),
		}
		C.vxr_vk_graphics_draw(instance.cInstance, cb.vkCommandBuffer, cInfo)
	})
}

type DrawIndirectBufferInfo struct {
	Buffer    Buffer
	Offset    uint64
	DrawCount uint32
}

func (i DrawIndirectBufferInfo) cIndirectBufferInfo() C.vxr_vk_graphics_drawIndirectBufferInfo {
	if !i.Buffer.Usage().HasBits(BufferUsageIndirectBuffer) {
		abort("DrawIndirectBufferInfo.Buffer was not created with BufferUsageIndirectBuffer")
	}
	if (i.Buffer.Size() - i.Offset) < (uint64(i.DrawCount) * uint64(unsafe.Sizeof(C.VkDrawIndirectCommand{}))) {
		abort("DrawIndirectBufferInfo.Offset + (DrawIndirectBufferInfo.DrawCount * sizeof(VkDrawIndirectCommand)) [%d + (%d * %d)] overflows buffer [%d]",
			i.Offset, i.DrawCount, unsafe.Sizeof(C.VkDrawIndirectCommand{}), i.Buffer.Size())
	}
	return C.vxr_vk_graphics_drawIndirectBufferInfo{
		vkBuffer:  i.Buffer.vkBuffer(),
		offset:    C.VkDeviceSize(i.Offset),
		drawCount: C.uint32_t(i.DrawCount),
	}
}

type DrawIndirectInfo struct {
	DrawParameters DrawParameters
	IndirectBuffer DrawIndirectBufferInfo
}

func (cb *GraphicsCommandBuffer) DrawIndirect(p GraphicsPipelineLibrary, info DrawIndirectInfo) {
	cb.noCopy.check()

	cb.draw(p, info.DrawParameters, func(cParmameters C.vxr_vk_graphics_drawParameters) {
		cInfo := C.vxr_vk_graphics_drawIndirectInfo{
			parameters:     cParmameters,
			indirectBuffer: info.IndirectBuffer.cIndirectBufferInfo(),
		}
		C.vxr_vk_graphics_drawIndirect(instance.cInstance, cb.vkCommandBuffer, cInfo)
	})
}

type DrawIndexedBufferInfo struct {
	Buffer     Buffer
	Offset     uint64
	IndexType  IndexType
	IndexCount uint32
}

func (i DrawIndexedBufferInfo) cIndexBufferInfo() C.vxr_vk_graphics_indexBufferInfo {
	if !i.Buffer.Usage().HasBits(BufferUsageIndexBuffer) {
		abort("DrawIndexedBufferInfo.Buffer was not created with BufferUsageIndexBuffer")
	}
	var indexTypeSize C.VkDeviceSize
	switch i.IndexType {
	case INDEX_TYPE_UINT8:
		indexTypeSize = C.VkDeviceSize(unsafe.Sizeof(C.uint8_t(0)))
	case INDEX_TYPE_UINT16:
		indexTypeSize = C.VkDeviceSize(unsafe.Sizeof(C.uint16_t(0)))
	case INDEX_TYPE_UINT32:
		indexTypeSize = C.VkDeviceSize(unsafe.Sizeof(C.uint32_t(0)))
	default:
		abort("Unknown IndexType: %d", i.IndexType)
	}
	sz := C.VkDeviceSize(i.IndexCount) * indexTypeSize
	if sz > C.VkDeviceSize(i.Buffer.Size()-i.Offset) {
		abort("DrawIndexedBufferInfo.Offset + (DrawIndexedBufferInfo.IndexCount * sizeof(IndexType)) [%d + (%d * %d)] overflows buffer [%d]",
			i.Offset, i.IndexCount, indexTypeSize, i.Buffer.Size())
	}
	return C.vxr_vk_graphics_indexBufferInfo{
		vkBuffer:   i.Buffer.vkBuffer(),
		offset:     C.VkDeviceSize(i.Offset),
		size:       sz,
		indexType:  C.VkIndexType(i.IndexType),
		indexCount: C.uint32_t(i.IndexCount),
	}
}

type DrawIndexedInfo struct {
	DrawParameters DrawParameters
	IndexBuffer    DrawIndexedBufferInfo
	InstanceCount  uint32
}

func (cb *GraphicsCommandBuffer) DrawIndexed(p GraphicsPipelineLibrary, info DrawIndexedInfo) {
	cb.noCopy.check()
	cb.draw(p, info.DrawParameters, func(cParmameters C.vxr_vk_graphics_drawParameters) {
		cInfo := C.vxr_vk_graphics_drawIndexedInfo{
			parameters:    cParmameters,
			indexBuffer:   info.IndexBuffer.cIndexBufferInfo(),
			instanceCount: C.uint32_t(info.InstanceCount),
		}
		C.vxr_vk_graphics_drawIndexed(instance.cInstance, cb.vkCommandBuffer, cInfo)
	})
}

type DrawIndexedIndirectInfo struct {
	DrawParameters DrawParameters
	IndexBuffer    DrawIndexedBufferInfo
	IndirectBuffer DrawIndirectBufferInfo
}

func (cb *GraphicsCommandBuffer) DrawIndexedIndirect(p GraphicsPipelineLibrary, info DrawIndexedIndirectInfo) {
	cb.noCopy.check()
	cb.draw(p, info.DrawParameters, func(cParmameters C.vxr_vk_graphics_drawParameters) {
		cInfo := C.vxr_vk_graphics_drawIndexedIndirectInfo{
			parameters:     cParmameters,
			indexBuffer:    info.IndexBuffer.cIndexBufferInfo(),
			indirectBuffer: info.IndirectBuffer.cIndirectBufferInfo(),
		}
		C.vxr_vk_graphics_drawIndexedIndirect(instance.cInstance, cb.vkCommandBuffer, cInfo)
	})
}

func (cb *GraphicsCommandBuffer) RenderPassEnd() {
	cb.noCopy.check()
	if cb.currentRenderPass == (renderPass{}) {
		abort("RenderPassEnd called when there's no active renderpass")
	}
	C.vxr_vk_graphics_renderPassEnd(instance.cInstance, cb.vkCommandBuffer)
	cb.currentRenderPass = renderPass{}
}

func (cb *GraphicsCommandBuffer) Submit(waitSemaphores []SemaphoreWaitInfo, signalSemaphores []SemaphoreSignalInfo) {
	cb.noCopy.check()
	if cb.currentRenderPass != (renderPass{}) {
		abort("End called when there's an active renderpass")
	}

	waitSemaphoreInfos := make([]C.VkSemaphoreSubmitInfo, 0, len(waitSemaphores))
	signalSemaphoreInfos := make([]C.VkSemaphoreSubmitInfo, 0, len(signalSemaphores))

	for _, info := range waitSemaphores {
		waitSemaphoreInfos = append(waitSemaphoreInfos, info.Semaphore.vkWaitInfo(info.Stage))
	}
	for _, info := range signalSemaphores {
		signalSemaphoreInfos = append(signalSemaphoreInfos, info.Semaphore.vkSignalInfo(info.Stage))
	}

	C.vxr_vk_graphics_frame_commandBufferSubmit(
		instance.cInstance,
		cb.cFrame,
		cb.vkCommandBuffer,
		C.uint32_t(len(waitSemaphores)), unsafe.SliceData(waitSemaphoreInfos),
		C.uint32_t(len(signalSemaphores)), unsafe.SliceData(signalSemaphoreInfos),
	)
	runtime.KeepAlive(waitSemaphores)
	runtime.KeepAlive(signalSemaphores)
	cb.noCopy.close()
}
