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
#include "std/log.hpp"
#include "std/array.hpp"
#include "std/time.hpp"
#include "std/string.hpp"

#include "vk/vk.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"

extern "C" {
VXR_FN void vxr_vk_createSemaphore(vxr_vk_instance instanceHandle, size_t nameSz, const char* name,
								   VkSemaphoreType type, VkSemaphore* semaphore) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	VkSemaphoreTypeCreateInfo semaphoreTypeInfo = {};
	semaphoreTypeInfo.sType = VK_STRUCTURE_TYPE_SEMAPHORE_TYPE_CREATE_INFO;
	semaphoreTypeInfo.semaphoreType = type;

	VkSemaphoreCreateInfo semaphoreInfo = {};
	semaphoreInfo.sType = VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO;
	semaphoreInfo.pNext = &semaphoreTypeInfo;

	const VkResult ret = VK_PROC_DEVICE(vkCreateSemaphore)(instance->device.vkDevice, &semaphoreInfo, nullptr, semaphore);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed to create semaphore: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}

	vxr::std::debugRun([=]() {
		vxr::std::stringbuilder builder;

		switch (type) {
			case VK_SEMAPHORE_TYPE_BINARY:
				builder.write("semaphore_binary_");
				break;

			case VK_SEMAPHORE_TYPE_TIMELINE:
				builder.write("semaphore_timeline_");
				break;

			default:
				vxr::std::ePrintf("Failed to create semaphore: invalid type %d", type);
				vxr::std::abort();
				break;
		}

		builder.write(nameSz, name);
		vxr::vk::debugLabel(instance->device.vkDevice, *semaphore, builder.cStr());
	});
}
VXR_FN void vxr_vk_signalSemaphore(vxr_vk_instance instanceHandle, VkSemaphore semaphore, uint64_t value) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	VkSemaphoreSignalInfo signalInfo = {};
	signalInfo.sType = VK_STRUCTURE_TYPE_SEMAPHORE_SIGNAL_INFO;
	signalInfo.semaphore = semaphore;
	signalInfo.value = value;

	const VkResult ret = VK_PROC_DEVICE(vkSignalSemaphore)(instance->device.vkDevice, &signalInfo);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed waiting on semaphore: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}
}
VXR_FN void vxr_vk_waitSemaphore(vxr_vk_instance instanceHandle, VkSemaphore semaphore, uint64_t value) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	VkSemaphoreWaitInfo waitInfo = {};
	waitInfo.sType = VK_STRUCTURE_TYPE_SEMAPHORE_WAIT_INFO;

	vxr::std::array semaphores = {
		semaphore,
	};
	vxr::std::array<uint64_t, semaphores.size()> values = {
		value,
	};
	waitInfo.semaphoreCount = semaphores.size();
	waitInfo.pSemaphores = semaphores.get();
	waitInfo.pValues = values.get();

	const VkResult ret = VK_PROC_DEVICE(vkWaitSemaphores)(instance->device.vkDevice, &waitInfo, vxr::std::time::second);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed waiting on semaphore: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}
}
VXR_FN uint64_t vxr_vk_getSemaphoreValue(vxr_vk_instance instanceHandle, VkSemaphore semaphore) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	uint64_t value = 0;
	const VkResult ret = VK_PROC_DEVICE(vkGetSemaphoreCounterValue)(instance->device.vkDevice, semaphore, &value);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("Failed getting semaphore value: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}
	return value;
}
VXR_FN void vxr_vk_destroySemaphore(vxr_vk_instance instanceHandle, VkSemaphore semaphore) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	VK_PROC_DEVICE(vkDestroySemaphore)(instance->device.vkDevice, semaphore, nullptr);
}
}
