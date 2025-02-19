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
#include <new>

#include "std/stdlib.hpp"
#include "std/log.hpp"

#include "vk/vk.hpp"
#include "vk/vkfns.hpp"
#include "vk/vklog.hpp"
#include "vk/device/device.hpp"

// Vulkan dependencies version check, update and run ./make/const_gen.go
static_assert(VK_HEADER_VERSION_COMPLETE >= VXR_VK_MAX_API);

extern "C" {
VXR_FN void vxr_vk_init(uintptr_t vkInstance, uintptr_t vkProcAddr,
						PFN_vkDebugUtilsMessengerCallbackEXT vkMessengerCallback, vxr_vk_instance* instanceHandle) {
	auto* instance = new (::std::nothrow) vxr::vk::instance();
	instance->vkInstance = reinterpret_cast<VkInstance>(vkInstance);  // NOLINT(performance-no-int-to-ptr)
	(*instanceHandle) = instance->handle();

	vxr::std::vPrintf("vkInitFns");
	// NOLINTNEXTLINE(performance-no-int-to-ptr)
	vxr::vk::initFns(instance, reinterpret_cast<PFN_vkGetInstanceProcAddr>(vkProcAddr));
	vxr::std::vPrintf("vkInitMessenger");
	vxr::vk::initMessenger(instance, vkMessengerCallback);
}
VXR_FN void vxr_vk_waitIdle(vxr_vk_instance instanceHandle) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	const VkResult ret = VK_PROC_DEVICE(vkDeviceWaitIdle)(instance->device.vkDevice);
	if (ret != VK_SUCCESS) {
		vxr::std::ePrintf("vkDeviceWaitIdle: %s", vxr::vk::vkResultStr(ret).cStr());
		vxr::std::abort();
	}
}
VXR_FN void vxr_vk_destroy(vxr_vk_instance instanceHandle) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);
	vxr::std::vPrintf("vkDestroyMessenger");
	vxr::vk::destroyMessenger(instance);
	delete instance;
}
}
