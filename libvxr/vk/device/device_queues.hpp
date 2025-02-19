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

#pragma once

#ifndef __cplusplus
#error C++ only header
#endif

#include "std/array.hpp"
#include "std/utility.hpp"

#include "vk/vk.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"

inline static void setupQueues(vxr::vk::instance* instance) {
	const vxr::std::array queues{
		vxr::std::pair("compute", &instance->device.computeQueue),
		vxr::std::pair("graphics", &instance->device.graphicsQueue),
		vxr::std::pair("transfer", &instance->device.transferQueue),
	};
	for (auto queue : queues) {
		VK_PROC_DEVICE(vkGetDeviceQueue)
		(instance->device.vkDevice, queue.second->family, queue.second->index, &queue.second->vkQueue);
		vxr::vk::debugLabel(instance->device.vkDevice, queue.second->vkQueue, "queue_%s", queue.first);
	}
}
