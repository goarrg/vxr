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

#include "vxr/vxr.h"  // IWYU pragma: associated

#include <stdint.h>
#include <stddef.h>

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/array.hpp"
#include "std/vector.hpp"
#include "std/string.hpp"

#include "vk/vk.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"
#include "vk/graphics/graphics.hpp"

extern "C" {
VXR_FN void vxr_vk_graphics_frame_commandBufferBegin(vxr_vk_instance instanceHandle, vxr_vk_graphics_frame frameHandle,
													 size_t nameSz, const char* name, VkCommandBuffer* cb) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	// auto* graphics = &instance->graphics;
	auto* frame = vxr::vk::graphics::frame::fromHandle(frameHandle);

	if (frame->freeCommandBuffers.size() > 0u) {
		*cb = frame->freeCommandBuffers.popFront();
	} else {
		VkCommandBufferAllocateInfo layoutCmdAllocateInfo = {};
		layoutCmdAllocateInfo.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
		layoutCmdAllocateInfo.commandPool = frame->vkCommandPool;
		layoutCmdAllocateInfo.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
		layoutCmdAllocateInfo.commandBufferCount = 1;

		const VkResult ret = VK_PROC_DEVICE(vkAllocateCommandBuffers)(instance->device.vkDevice, &layoutCmdAllocateInfo, cb);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to create graphics command buffer: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}
		frame->allocatedCommandBuffers += 1;
	}

	{
		VkCommandBufferBeginInfo beginInfo = {};
		beginInfo.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
		beginInfo.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;

		const VkResult ret = VK_PROC_DEVICE(vkBeginCommandBuffer)(*cb, &beginInfo);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to begin graphics command buffer: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}

		vxr::std::debugRun([=]() {
			vxr::std::stringbuilder builder;
			builder.write("graphics_cmd_buffer_").write(nameSz, name);
			vxr::vk::debugLabelBegin(*cb, builder.cStr());
		});
	}
}
VXR_FN void vxr_vk_graphics_frame_commandBufferSubmit(
	vxr_vk_instance instanceHandle, vxr_vk_graphics_frame frameHandle, VkCommandBuffer cb, uint32_t numWaitSemaphores,
	VkSemaphoreSubmitInfo* waitSemaphores, uint32_t numSignalSemaphores, VkSemaphoreSubmitInfo* signalSemaphores) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	// auto* graphics = &instance->graphics;
	auto* frame = vxr::vk::graphics::frame::fromHandle(frameHandle);

	vxr::std::debugRun([=]() { vxr::vk::debugLabelEnd(cb); });

	{
		const VkResult ret = VK_PROC_DEVICE(vkEndCommandBuffer)(cb);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to end command buffer: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}
	}

	{
		VkSubmitInfo2 submitInfo = {};
		submitInfo.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO_2;

		submitInfo.waitSemaphoreInfoCount = numWaitSemaphores;
		submitInfo.pWaitSemaphoreInfos = waitSemaphores;

		vxr::std::array commandbuffers = {
			VkCommandBufferSubmitInfo{
				.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_SUBMIT_INFO,
				.commandBuffer = cb,
			},
		};
		submitInfo.commandBufferInfoCount = commandbuffers.size();
		submitInfo.pCommandBufferInfos = commandbuffers.get();

		submitInfo.signalSemaphoreInfoCount = numSignalSemaphores;
		submitInfo.pSignalSemaphoreInfos = signalSemaphores;

		const VkResult ret = VK_PROC_DEVICE(vkQueueSubmit2)(instance->device.graphicsQueue.vkQueue, 1, &submitInfo, VK_NULL_HANDLE);
		if (ret != VK_SUCCESS) {
			vxr::std::ePrintf("Failed to submit frame: %s", vxr::vk::vkResultStr(ret).cStr());
			vxr::std::abort();
		}
	}

	frame->pendingCommandBuffers.pushBack(cb);
}
VXR_FN void vxr_vk_graphics_renderPassBegin(vxr_vk_instance, VkCommandBuffer cb, size_t nameSz, const char* name,
											vxr_vk_graphics_renderPassInfo info) {
	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder builder;
		builder.write(nameSz, name).write("_pass");
		vxr::vk::debugLabelBegin(cb, builder.cStr());
	});
	VK_PROC_DEVICE(vkCmdBeginRendering)(cb, &info.renderingInfo);

	const size_t numViewports = 1;
	vxr::std::vector<VkViewport> viewports(numViewports);
	vxr::std::vector<VkRect2D> scissors(numViewports);
	if (info.flipViewport == VK_TRUE) {
		for (size_t i = 0; i < numViewports; i++) {
			viewports[i] = VkViewport{
				.x = static_cast<float>(info.renderingInfo.renderArea.offset.x),
				.y = static_cast<float>(info.renderingInfo.renderArea.offset.y),
				.width = static_cast<float>(info.renderingInfo.renderArea.extent.width),
				.height = static_cast<float>(info.renderingInfo.renderArea.extent.height),
				.maxDepth = 1.0f,
			};
			scissors[i] = VkRect2D{
				.offset = info.renderingInfo.renderArea.offset,
				.extent = info.renderingInfo.renderArea.extent,
			};
		}
	} else {
		for (size_t i = 0; i < numViewports; i++) {
			viewports[i] = VkViewport{
				.x = static_cast<float>(info.renderingInfo.renderArea.offset.x),
				.y = static_cast<float>(info.renderingInfo.renderArea.offset.y + info.renderingInfo.renderArea.extent.height),
				.width = static_cast<float>(info.renderingInfo.renderArea.extent.width),
				.height = -static_cast<float>(info.renderingInfo.renderArea.extent.height),
				.maxDepth = 1.0f,
			};
			scissors[i] = VkRect2D{
				.offset = info.renderingInfo.renderArea.offset,
				.extent = info.renderingInfo.renderArea.extent,
			};
		}
	}
	VK_PROC_DEVICE(vkCmdSetViewportWithCount)(cb, numViewports, viewports.get());
	VK_PROC_DEVICE(vkCmdSetScissorWithCount)(cb, numViewports, scissors.get());

	if (info.renderingInfo.colorAttachmentCount > 0) {
		VK_PROC_DEVICE(vkCmdSetColorBlendEnableEXT)(cb, 0, info.renderingInfo.colorAttachmentCount, info.colorBlendEnable);
		VK_PROC_DEVICE(vkCmdSetColorBlendEquationEXT)(cb, 0, info.renderingInfo.colorAttachmentCount, info.colorBlendEquation);
		VK_PROC_DEVICE(vkCmdSetColorWriteMaskEXT)(cb, 0, info.renderingInfo.colorAttachmentCount, info.colorComponentFlags);
	}
}
VXR_FN void vxr_vk_graphics_renderPassSetViewport(vxr_vk_instance, VkCommandBuffer cb, VkBool32 flip, VkViewport viewport) {
	if (flip == VK_FALSE) {
		viewport = VkViewport{
			.x = viewport.x,
			.y = (viewport.y + viewport.height),
			.width = viewport.width,
			.height = -viewport.height,
			.maxDepth = 1.0f,
		};
	}
	VK_PROC_DEVICE(vkCmdSetViewportWithCount)(cb, 1, &viewport);
}
VXR_FN void vxr_vk_graphics_renderPassSetScissor(vxr_vk_instance, VkCommandBuffer cb, VkRect2D rect) {
	VK_PROC_DEVICE(vkCmdSetScissorWithCount)(cb, 1, &rect);
}
VXR_FN void vxr_vk_graphics_renderPassSetViewportAndScissor(vxr_vk_instance, VkCommandBuffer cb, VkBool32 flip,
															VkViewport viewport, VkRect2D rect) {
	if (flip == VK_FALSE) {
		viewport = VkViewport{
			.x = viewport.x,
			.y = (viewport.y + viewport.height),
			.width = viewport.width,
			.height = -viewport.height,
			.maxDepth = 1.0f,
		};
	}
	VK_PROC_DEVICE(vkCmdSetViewportWithCount)(cb, 1, &viewport);
	VK_PROC_DEVICE(vkCmdSetScissorWithCount)(cb, 1, &rect);
}
inline static VXR_FN void setupDraw(vxr_vk_graphics_drawParameters parameters, VkCommandBuffer cb) {
	VK_PROC_DEVICE(vkCmdBindPipeline)(cb, VK_PIPELINE_BIND_POINT_GRAPHICS, parameters.pipeline);

	VK_PROC_DEVICE(vkCmdSetPrimitiveTopology)(cb, parameters.topology);
	// VK_PROC_DEVICE(vkCmdSetVertexInputEXT)(cb, 0, nullptr, 0, nullptr);
	// VK_PROC_DEVICE(vkCmdBindVertexBuffers)(cb, 0, 0, nullptr, nullptr);

	VK_PROC_DEVICE(vkCmdSetCullMode)(cb, parameters.cullMode);
	VK_PROC_DEVICE(vkCmdSetFrontFace)(cb, parameters.frontFace);

	VK_PROC_DEVICE(vkCmdSetDepthTestEnable)(cb, parameters.depthTestEnable);
	VK_PROC_DEVICE(vkCmdSetDepthWriteEnable)(cb, parameters.depthWriteEnable);
	VK_PROC_DEVICE(vkCmdSetDepthCompareOp)(cb, parameters.depthCompareOp);

	VK_PROC_DEVICE(vkCmdSetStencilTestEnable)(cb, parameters.stencilTestEnable);
	if (parameters.stencilTestEnable == VK_TRUE) {
		if ((parameters.stencilTestFrontFace.failOp == parameters.stencilTestBackFace.failOp)
			&& (parameters.stencilTestFrontFace.passOp == parameters.stencilTestBackFace.passOp)
			&& (parameters.stencilTestFrontFace.depthFailOp == parameters.stencilTestBackFace.depthFailOp)
			&& (parameters.stencilTestFrontFace.compareOp == parameters.stencilTestBackFace.compareOp)) {
			VK_PROC_DEVICE(vkCmdSetStencilOp)(
				cb, VK_STENCIL_FACE_FRONT_AND_BACK, parameters.stencilTestFrontFace.failOp,
				parameters.stencilTestFrontFace.passOp, parameters.stencilTestFrontFace.depthFailOp,
				parameters.stencilTestFrontFace.compareOp);
		} else {
			VK_PROC_DEVICE(vkCmdSetStencilOp)(
				cb, VK_STENCIL_FACE_FRONT_BIT, parameters.stencilTestFrontFace.failOp, parameters.stencilTestFrontFace.passOp,
				parameters.stencilTestFrontFace.depthFailOp, parameters.stencilTestFrontFace.compareOp);
			VK_PROC_DEVICE(vkCmdSetStencilOp)(
				cb, VK_STENCIL_FACE_BACK_BIT, parameters.stencilTestBackFace.failOp, parameters.stencilTestBackFace.passOp,
				parameters.stencilTestBackFace.depthFailOp, parameters.stencilTestBackFace.compareOp);
		}
		if (parameters.stencilTestFrontFace.compareMask == parameters.stencilTestBackFace.compareMask) {
			VK_PROC_DEVICE(vkCmdSetStencilCompareMask)(
				cb, VK_STENCIL_FACE_FRONT_AND_BACK, parameters.stencilTestFrontFace.compareMask);
		} else {
			VK_PROC_DEVICE(vkCmdSetStencilCompareMask)(cb, VK_STENCIL_FACE_FRONT_BIT, parameters.stencilTestFrontFace.compareMask);
			VK_PROC_DEVICE(vkCmdSetStencilCompareMask)(cb, VK_STENCIL_FACE_BACK_BIT, parameters.stencilTestBackFace.compareMask);
		}
		if (parameters.stencilTestFrontFace.writeMask == parameters.stencilTestBackFace.writeMask) {
			VK_PROC_DEVICE(vkCmdSetStencilWriteMask)(cb, VK_STENCIL_FACE_FRONT_AND_BACK, parameters.stencilTestFrontFace.writeMask);
		} else {
			VK_PROC_DEVICE(vkCmdSetStencilWriteMask)(cb, VK_STENCIL_FACE_FRONT_BIT, parameters.stencilTestFrontFace.writeMask);
			VK_PROC_DEVICE(vkCmdSetStencilWriteMask)(cb, VK_STENCIL_FACE_BACK_BIT, parameters.stencilTestBackFace.writeMask);
		}
		if (parameters.stencilTestFrontFace.reference == parameters.stencilTestBackFace.reference) {
			VK_PROC_DEVICE(vkCmdSetStencilReference)(cb, VK_STENCIL_FACE_FRONT_AND_BACK, parameters.stencilTestFrontFace.reference);
		} else {
			VK_PROC_DEVICE(vkCmdSetStencilReference)(cb, VK_STENCIL_FACE_FRONT_BIT, parameters.stencilTestFrontFace.reference);
			VK_PROC_DEVICE(vkCmdSetStencilReference)(cb, VK_STENCIL_FACE_BACK_BIT, parameters.stencilTestBackFace.reference);
		}
	}

	if (parameters.pushConstantRange.size > 0) {
		VK_PROC_DEVICE(vkCmdPushConstants)
		(cb, parameters.layout, parameters.pushConstantRange.stageFlags, parameters.pushConstantRange.offset,
		 parameters.pushConstantRange.size, parameters.pushConstantData);
	}
	if (parameters.numDescriptorSets > 0) {
		VK_PROC_DEVICE(vkCmdBindDescriptorSets)
		(cb, VK_PIPELINE_BIND_POINT_GRAPHICS, parameters.layout, 0, parameters.numDescriptorSets,
		 parameters.descriptorSets, 0, nullptr);
	}
}
VXR_FN void vxr_vk_graphics_draw(vxr_vk_instance, VkCommandBuffer cb, vxr_vk_graphics_drawInfo info) {
	setupDraw(info.parameters, cb);
	VK_PROC_DEVICE(vkCmdDraw)(cb, info.vertexCount, info.instanceCount, 0, 0);
}
VXR_FN void vxr_vk_graphics_drawIndirect(vxr_vk_instance, VkCommandBuffer cb, vxr_vk_graphics_drawIndirectInfo info) {
	setupDraw(info.parameters, cb);
	VK_PROC_DEVICE(vkCmdDrawIndirect)(cb, info.indirectBuffer.vkBuffer, info.indirectBuffer.offset,
									  info.indirectBuffer.drawCount, sizeof(VkDrawIndirectCommand));
}
VXR_FN void vxr_vk_graphics_drawIndexed(vxr_vk_instance instanceHandle, VkCommandBuffer cb, vxr_vk_graphics_drawIndexedInfo info) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	setupDraw(info.parameters, cb);
	instance->device.fnTable.bindIndexBuffer(cb, info.indexBuffer);
	VK_PROC_DEVICE(vkCmdDrawIndexed)(cb, info.indexBuffer.indexCount, info.instanceCount, 0, 0, 0);
}
VXR_FN void vxr_vk_graphics_drawIndexedIndirect(vxr_vk_instance instanceHandle, VkCommandBuffer cb, vxr_vk_graphics_drawIndexedIndirectInfo info) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	setupDraw(info.parameters, cb);
	instance->device.fnTable.bindIndexBuffer(cb, info.indexBuffer);
	VK_PROC_DEVICE(vkCmdDrawIndexedIndirect)(cb, info.indirectBuffer.vkBuffer, info.indirectBuffer.offset,
											 info.indirectBuffer.drawCount, sizeof(VkDrawIndexedIndirectCommand));
}
VXR_FN void vxr_vk_graphics_renderPassEnd(vxr_vk_instance, VkCommandBuffer cb) {
	VK_PROC_DEVICE(vkCmdEndRendering)(cb);
	vxr::vk::debugLabelEnd(cb);
}
}
