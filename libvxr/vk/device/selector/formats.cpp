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

#include "std/utility.hpp"
#include "std/log.hpp"

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/vkfns.hpp"
#include "vk/device/selector/selector.hpp"

// NOLINTNEXTLINE(readability-convert-member-functions-to-static)
bool vxr::vk::device::selector::selector::findFormats(vxr::vk::instance* instance) {
	bool formatsOK = true;

	for (auto formatFeature : requiredFormatFeatures) {
		VkFormatProperties3 formatProperties3 = {
			.sType = VK_STRUCTURE_TYPE_FORMAT_PROPERTIES_3,
		};
		VkFormatProperties2 formatProperties2 = {
			.sType = VK_STRUCTURE_TYPE_FORMAT_PROPERTIES_2,
			.pNext = &formatProperties3,
		};
		VK_PROC(vkGetPhysicalDeviceFormatProperties2)
		(instance->device.vkPhysicalDevice, formatFeature.first, &formatProperties2);
		if (vxr::std::cmpBitFlagsContains(formatProperties3.optimalTilingFeatures, formatFeature.second)) {
			vxr::std::vPrintf("Found required features: 0x%X for format: %d", formatFeature.second, formatFeature.first);
		} else {
			vxr::std::iPrintf("Missing required features for format: %d, have: 0x%X want 0x%X", formatFeature.first,
							  formatProperties3.optimalTilingFeatures, formatFeature.second);
			formatsOK = false;
		}
	}

	return formatsOK;
}
