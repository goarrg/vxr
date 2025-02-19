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

#include <stddef.h>

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/memory.hpp"
#include "std/vector.hpp"

#include "vxr/vxr.h"
#include "vk/vk.hpp"
#include "vk/vkfns.hpp"
#include "vk/device/device_features_reflection.hpp"
#include "vk/device/selector/selector.hpp"

// NOLINTNEXTLINE(readability-function-cognitive-complexity)
bool vxr::vk::device::selector::selector::findFeatures(vxr::vk::instance* instance) {
	if (requiredFeatureChain.allocations.size() != optionalFeatureChain.allocations.size()) {
		vxr::std::ePrintf("Size mismatch between required and optional feature chains: %d != %d",
						  requiredFeatureChain.allocations.size(), optionalFeatureChain.allocations.size());
		vxr::std::abort();
	}

	vxr::vk::device::selector::featureChain haveFeatureChain;
	for (size_t i = 0; i < requiredFeatureChain.allocations.size(); i++) {
		if (requiredFeatureChain.allocations[i]->sType != optionalFeatureChain.allocations[i]->sType) {
			vxr::std::ePrintf("Required and optional feature chains must be in the same order: sType %d != %d",
							  requiredFeatureChain.allocations[i]->sType, optionalFeatureChain.allocations[i]->sType);
			vxr::std::abort();
		}
		haveFeatureChain.append(requiredFeatureChain.allocations[i]->sType);
	}
	VK_PROC(vkGetPhysicalDeviceFeatures2)(instance->device.vkPhysicalDevice, &haveFeatureChain.start);

	bool ok = true;

	auto* required = reinterpret_cast<vxr::vk::device::reflect::vkStructureChain*>(&requiredFeatureChain.start);
	auto* optional = reinterpret_cast<vxr::vk::device::reflect::vkStructureChain*>(&optionalFeatureChain.start);
	auto* enabled = reinterpret_cast<vxr::vk::device::reflect::vkStructureChain*>(&enabledFeatureChain.start);
	auto* have = reinterpret_cast<vxr::vk::device::reflect::vkStructureChain*>(&haveFeatureChain.start);

	auto rV = vxr::vk::device::reflect::valueOf(&requiredFeatureChain.start.features);
	auto oV = vxr::vk::device::reflect::valueOf(&optionalFeatureChain.start.features);
	auto eV = vxr::vk::device::reflect::valueOf(&enabledFeatureChain.start.features);
	auto hV = vxr::vk::device::reflect::valueOf(&haveFeatureChain.start.features);

	enabledFeatureChain.reset();
	auto enableFeature = [&](VkStructureType sType, size_t fieldIndex) {
		if (enabled == nullptr) {
			enabledFeatureChain.append(sType, 1, &fieldIndex);
			enabled = (enabledFeatureChain.allocations.end() - 1)->get();
			eV = vxr::vk::device::reflect::valueOf(enabled);
		} else {
			auto eF = eV->field(fieldIndex);
			auto* e = static_cast<VkBool32*>(eF.ptr);
			*e = VK_TRUE;
		}
	};

	while (have != nullptr) {
		for (size_t fieldIndex = 0; fieldIndex < hV->numField(); fieldIndex++) {
			auto rF = rV->field(fieldIndex);
			auto oF = oV->field(fieldIndex);
			auto hF = hV->field(fieldIndex);
			switch (hF.type.id) {
				case vxr::vk::device::reflect::type::vkBool32: {
					auto r = *static_cast<VkBool32*>(rF.ptr);
					auto o = *static_cast<VkBool32*>(oF.ptr);
					auto h = *static_cast<VkBool32*>(hF.ptr);

					if (r == VK_TRUE) {
						if (h == VK_TRUE) {
							vxr::std::vPrintf("Found required feature %s.%s", hV->type->name, hF.name);
							enableFeature(have->sType, fieldIndex);
						} else {
							vxr::std::iPrintf("Missing required feature %s.%s", hV->type->name, hF.name);
							ok = false;
						}
					} else if (o == VK_TRUE) {
						if (h == VK_TRUE) {
							vxr::std::vPrintf("Found optional feature %s.%s", hV->type->name, hF.name);
							enableFeature(have->sType, fieldIndex);
						} else {
							vxr::std::iPrintf("Missing optional feature %s.%s", hV->type->name, hF.name);
						}
					}
				} break;
				default:
					break;
			}
		}
		required = required->pNext;
		optional = optional->pNext;
		have = have->pNext;
		if (have != nullptr) {
			rV = vxr::vk::device::reflect::valueOf(required);
			oV = vxr::vk::device::reflect::valueOf(optional);
			hV = vxr::vk::device::reflect::valueOf(have);
		}
		if (enabled != nullptr) {
			enabled = enabled->pNext;
		}
	}

	return ok;
}
