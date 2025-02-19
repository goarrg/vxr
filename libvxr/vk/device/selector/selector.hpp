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

#include <stddef.h>
#include <stdint.h>

#include "std/string.hpp"
#include "std/vector.hpp"
#include "std/utility.hpp"
#include "std/memory.hpp"
#include "std/stdlib.hpp"
#include "std/log.hpp"

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/device/device_features_reflection.hpp"

namespace vxr::vk {
struct instance;
namespace device::selector {
struct featureChain {
	vxr::std::vector<vxr::std::smartPtr<vxr::vk::device::reflect::vkStructureChain>> allocations;
	VkPhysicalDeviceFeatures2 start{.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_FEATURES_2};

	void reset() {
		this->start = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_FEATURES_2};
		this->allocations.resize(0);
	}

	void append(VkStructureType sType) {
		if (sType != VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_FEATURES_2) {
			bool found = false;
			for (auto& s : this->allocations) {
				if (s->sType == sType) {
					found = true;
					break;
				}
			}
			if (!found) {
				auto alloc = vxr::vk::device::reflect::typeOf(sType)->allocate();
				if (this->allocations.size() == 0) {
					this->start.pNext = alloc.get();
				} else {
					this->allocations[this->allocations.size() - 1]->pNext = alloc.get();
				}
				this->allocations.pushBack(vxr::std::move(alloc));
			}
		}
	}
	void append(VkStructureType sType, size_t numFeatures, size_t* features) {
		vxr::std::smartPtr<vxr::vk::device::reflect::structValue> v;

		if (sType == VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_FEATURES_2) {
			v = vxr::vk::device::reflect::valueOf(&this->start.features);
		} else {
			for (auto& s : this->allocations) {
				if (s->sType == sType) {
					v = vxr::vk::device::reflect::valueOf(s.get());
					break;
				}
			}
			if (v.get() == nullptr) {
				auto alloc = vxr::vk::device::reflect::typeOf(sType)->allocate();
				v = vxr::vk::device::reflect::valueOf(alloc.get());
				if (this->allocations.size() == 0) {
					this->start.pNext = alloc.get();
				} else {
					this->allocations[this->allocations.size() - 1]->pNext = alloc.get();
				}
				this->allocations.pushBack(vxr::std::move(alloc));
			}
		}

		for (size_t i = 0; i < numFeatures; i++) {
			auto f = v->field(features[i]);
			if (f.type.id != vxr::vk::device::reflect::type::vkBool32) {
				vxr::std::ePrintf("Tryting to set %s.%s which is not a feature toggle", v->type->name, f.name);
				vxr::std::abort();
			}
			*static_cast<VkBool32*>(f.ptr) = VK_TRUE;
		}
	}
};
class selector {
   private:
	VkPhysicalDevice preferredDevice;
	uint32_t requiredAPI;
	VkSurfaceKHR targetSurface;

	vxr::std::vector<vxr::std::string<char>> requiredExtensions;
	vxr::std::vector<vxr::std::string<char>> optionalExtensions;
	vxr::std::vector<vxr::std::string<char>> enabledExtensions;

	featureChain requiredFeatureChain;
	featureChain optionalFeatureChain;
	featureChain enabledFeatureChain;
	vxr::std::string<char> enabledFeatureString;

	vxr::std::vector<vxr::std::pair<VkFormat, VkFormatFeatureFlags2>> requiredFormatFeatures;

	vxr::std::vector<VkDeviceQueueCreateInfo> queueCreateInfos;
	vxr::std::vector<vxr::std::vector<float>> queuePriorities;

	bool findProperties(::vxr::vk::instance*);
	bool findExtensions(::vxr::vk::instance*);
	bool findFeatures(::vxr::vk::instance*);
	bool findFormats(::vxr::vk::instance*);
	bool findQueues(::vxr::vk::instance*);
	bool checkDevice(vxr::vk::instance*);

	// NOLINTBEGIN(readability-identifier-naming)
	friend VXR_FN void ::vxr_vk_device_selector_appendRequiredExtension(vxr_vk_device_selector, size_t, const char*);
	friend VXR_FN void ::vxr_vk_device_selector_appendOptionalExtension(vxr_vk_device_selector, size_t, const char*);
	friend VXR_FN void ::vxr_vk_device_selector_initFeatureChain(vxr_vk_device_selector, size_t, VkStructureType*);
	friend VXR_FN void ::vxr_vk_device_selector_appendRequiredFeature(vxr_vk_device_selector, VkStructureType, size_t, size_t*);
	friend VXR_FN void ::vxr_vk_device_selector_appendOptionalFeature(vxr_vk_device_selector, VkStructureType, size_t, size_t*);
	friend VXR_FN void ::vxr_vk_device_selector_appendRequiredFormatFeature(vxr_vk_device_selector, VkFormat, VkFormatFeatureFlags2);
	friend VXR_FN void ::vxr_vk_device_selector_getEnabledExtensions(vxr_vk_device_selector, size_t*, const char**);
	friend VXR_FN void ::vxr_vk_device_selector_getEnabledFeatures(vxr_vk_device_selector, const char**);
	// NOLINTEND(readability-identifier-naming)

   public:
	selector(VkPhysicalDevice preferredDevice, uint32_t api, VkSurfaceKHR targetSurface)
		: preferredDevice(preferredDevice), requiredAPI(api), targetSurface(targetSurface) {}

	[[nodiscard]] vxr_vk_device_selector handle() noexcept { return reinterpret_cast<vxr_vk_device_selector>(this); }
	[[nodiscard]] static selector* fromHandle(vxr_vk_device_selector handle) noexcept {
		return reinterpret_cast<selector*>(handle);
	}

	void findAndCreateDevice(vxr::vk::instance*);
};
}  // namespace device::selector
}  // namespace vxr::vk
