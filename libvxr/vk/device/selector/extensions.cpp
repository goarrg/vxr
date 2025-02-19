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

#include <stdint.h>

#include "std/vector.hpp"
#include "std/log.hpp"

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/vkfns.hpp"
#include "vk/device/selector/selector.hpp"

bool vxr::vk::device::selector::selector::findExtensions(vxr::vk::instance* instance) {
	uint32_t sz;
	VK_PROC(vkEnumerateDeviceExtensionProperties)(instance->device.vkPhysicalDevice, nullptr, &sz, nullptr);
	vxr::std::vector<VkExtensionProperties> properties(sz);
	VK_PROC(vkEnumerateDeviceExtensionProperties)(instance->device.vkPhysicalDevice, nullptr, &sz, properties.get());
	enabledExtensions.resize(0);

	bool ok = true;
	for (const auto& require : requiredExtensions) {
		bool found = false;
		for (auto have : properties) {
			if (require == have.extensionName) {
				found = true;
				vxr::std::vPrintf("Found required extension: %s", require.cStr());
				break;
			}
		}

		if (found) {
			enabledExtensions.pushBack(require);
		} else {
			vxr::std::iPrintf("Failed to find required extension: %s", require.cStr());
			ok = false;
		}
	}
	for (const auto& optional : optionalExtensions) {
		bool found = false;
		for (auto have : properties) {
			if (optional == have.extensionName) {
				found = true;
				vxr::std::vPrintf("Found optional extension: %s", optional.cStr());
				break;
			}
		}

		if (found) {
			enabledExtensions.pushBack(optional);
		} else {
			vxr::std::iPrintf("Failed to find optional extension: %s", optional.cStr());
		}
	}

	return ok;
}
