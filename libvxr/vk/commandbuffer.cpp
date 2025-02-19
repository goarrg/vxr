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

#include <stddef.h>
#include <stdint.h>

#include "std/stdlib.hpp"
#include "std/string.hpp"

#include "vk/vk.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"

extern "C" {
VXR_FN void vxr_vk_commandBuffer_beginNamedRegion(vxr_vk_instance, VkCommandBuffer cb, size_t nameSz, const char* name) {
	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder builder;
		builder.write(nameSz, name).write("_pass");
		vxr::vk::debugLabelBegin(cb, builder.cStr());
	});
}
VXR_FN void vxr_vk_commandBuffer_endNamedRegion(vxr_vk_instance, VkCommandBuffer cb) {
	vxr::vk::debugLabelEnd(cb);
}
VXR_FN void vxr_vk_commandBuffer_barrier(vxr_vk_instance, VkCommandBuffer cb, VkDependencyInfo info) {
	VK_PROC_DEVICE(vkCmdPipelineBarrier2)(cb, &info);
}
VXR_FN void vxr_vk_commandBuffer_fillBuffer(vxr_vk_instance, VkCommandBuffer cb, VkBuffer buffer, VkDeviceSize offset,
											VkDeviceSize size, uint32_t value) {
	VK_PROC_DEVICE(vkCmdFillBuffer)(cb, buffer, offset, size, value);
}
VXR_FN void vxr_vk_commandBuffer_updateBuffer(vxr_vk_instance, VkCommandBuffer cb, VkBuffer buffer, VkDeviceSize offset,
											  VkDeviceSize size, void* data) {
	VK_PROC_DEVICE(vkCmdUpdateBuffer)(cb, buffer, offset, size, data);
}
VXR_FN void vxr_vk_commandBuffer_clearColorImage(vxr_vk_instance, VkCommandBuffer cb, VkImage img, VkImageLayout layout,
												 VkClearColorValue value, uint32_t numRanges, VkImageSubresourceRange* ranges) {
	VK_PROC_DEVICE(vkCmdClearColorImage)(cb, img, layout, &value, numRanges, ranges);
}
VXR_FN void vxr_vk_commandBuffer_copyBuffer(vxr_vk_instance, VkCommandBuffer cb, VkBuffer bIn, VkBuffer bOut,
											uint32_t regionCount, VkBufferCopy* regions) {
	VK_PROC_DEVICE(vkCmdCopyBuffer)(cb, bIn, bOut, regionCount, regions);
}
VXR_FN void vxr_vk_commandBuffer_copyBufferToImage(vxr_vk_instance, VkCommandBuffer cb, VkBuffer buffer, VkImage image,
												   VkImageLayout layout, uint32_t regionCount, VkBufferImageCopy* regions) {
	VK_PROC_DEVICE(vkCmdCopyBufferToImage)(cb, buffer, image, layout, regionCount, regions);
}
}
