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

#include "vk/vk.hpp"
#include "vk/device/device.hpp"

extern "C" {
VXR_FN void vxr_vk_compute_dispatch(vxr_vk_instance, VkCommandBuffer cb, vxr_vk_compute_dispatchInfo info) {
	if (info.pushConstantRange.size > 0) {
		VK_PROC_DEVICE(vkCmdPushConstants)
		(cb, info.layout, info.pushConstantRange.stageFlags, info.pushConstantRange.offset, info.pushConstantRange.size,
		 info.pushConstantData);
	}
	if (info.numDescriptorSets > 0) {
		VK_PROC_DEVICE(vkCmdBindDescriptorSets)
		(cb, VK_PIPELINE_BIND_POINT_COMPUTE, info.layout, 0, info.numDescriptorSets, info.descriptorSets, 0, nullptr);
	}
	VK_PROC_DEVICE(vkCmdBindPipeline)(cb, VK_PIPELINE_BIND_POINT_COMPUTE, info.pipeline);
	VK_PROC_DEVICE(vkCmdDispatch)(cb, info.groupCount.width, info.groupCount.height, info.groupCount.depth);
}
VXR_FN void vxr_vk_compute_dispatchIndirect(vxr_vk_instance, VkCommandBuffer cb, vxr_vk_compute_dispatchIndirectInfo info) {
	if (info.pushConstantRange.size > 0) {
		VK_PROC_DEVICE(vkCmdPushConstants)
		(cb, info.layout, info.pushConstantRange.stageFlags, info.pushConstantRange.offset, info.pushConstantRange.size,
		 info.pushConstantData);
	}

	VK_PROC_DEVICE(vkCmdBindDescriptorSets)
	(cb, VK_PIPELINE_BIND_POINT_COMPUTE, info.layout, 0, info.numDescriptorSets, info.descriptorSets, 0, nullptr);
	VK_PROC_DEVICE(vkCmdBindPipeline)(cb, VK_PIPELINE_BIND_POINT_COMPUTE, info.pipeline);

	VK_PROC_DEVICE(vkCmdDispatchIndirect)(cb, info.buffer, info.offset);
}
}
