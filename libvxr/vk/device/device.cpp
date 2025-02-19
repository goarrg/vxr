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

#include "std/log.hpp"
#include "std/array.hpp"
#include "std/utility.hpp"

#include "vk/vk.hpp"
#include "vk/device/device.hpp"
#include "vk/device/selector/selector.hpp"

#include "vxr/vxr.h"		   // IWYU pragma: associated
#include "device_vkfns.hpp"	   // IWYU pragma: associated
#include "device_vma.hpp"	   // IWYU pragma: associated
#include "device_queues.hpp"   // IWYU pragma: associated
#include "device_fntable.hpp"  // IWYU pragma: associated

extern "C" {
VXR_FN void vxr_vk_device_init(vxr_vk_instance instanceHandle, vxr_vk_device_selector selectorHandle) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	auto* selector = vxr::vk::device::selector::selector::fromHandle(selectorHandle);
	selector->findAndCreateDevice(instance);

	vxr::std::iPrintf("Setting up device");
	static constexpr vxr::std::array deviceSetups = {
		vxr::std::pair("setupVKFNs", setupVKFNs),
		vxr::std::pair("setupVMA", setupVMA),
		vxr::std::pair("setupQueues", setupQueues),
		vxr::std::pair("setupFNTable", setupFNTable),
	};
	for (auto setup : deviceSetups) {
		vxr::std::vPrintf(setup.first);
		setup.second(instance);
	}
	vxr::std::iPrintf("Device setup complete");
}
VXR_FN void vxr_vk_device_destroy(vxr_vk_instance instanceHandle) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	static constexpr vxr::std::array deviceDestructors = {
		vxr::std::pair("destroyVMA", destroyVMA),
	};
	for (auto destroy : deviceDestructors) {
		vxr::std::vPrintf(destroy.first);
		destroy.second(instance);
	}
	VK_PROC_DEVICE(vkDestroyDevice)(instance->device.vkDevice, nullptr);
}
VXR_FN void vxr_vk_device_getProperties(vxr_vk_instance instanceHandle, vxr_vk_device_properties* properties) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	*properties = instance->device.properties;
}
}
