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

#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <algorithm>
#include <new>	// IWYU pragma: keep

#include "std/stdlib.hpp"
#include "std/log.hpp"
#include "std/array.hpp"
#include "std/vector.hpp"
#include "std/string.hpp"
#include "std/utility.hpp"
#include "std/unit.hpp"

#include "vxr/vxr.h"  // IWYU pragma: associated
#include "vk/vk.hpp"
#include "vk/vklog.hpp"
#include "vk/vkfns.hpp"
#include "vk/device/device.hpp"
#include "vk/device/device_features_reflection.hpp"
#include "vk/device/selector/selector.hpp"

// NOLINTBEGIN(clang-analyzer-core.CallAndMessage)

inline static VkDeviceSize vramSize(VkPhysicalDevice device) {
	VkPhysicalDeviceMemoryProperties memProperties = {};
	VK_PROC(vkGetPhysicalDeviceMemoryProperties)(device, &memProperties);

	VkDeviceSize memSize = 0;

	for (uint32_t i = 0; i < memProperties.memoryTypeCount; i++) {
		auto type = memProperties.memoryTypes[i];
		auto heap = memProperties.memoryHeaps[type.heapIndex];
		if (vxr::std::cmpBitFlags(type.propertyFlags, VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT, VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT)) {
			memSize = vxr::std::max(heap.size, memSize);
		}
	}

	return memSize;
}

inline static void printDevices(const auto& devices) {
	vxr::std::stringbuilder builder;
	builder << "Detected Devices:";
	for (size_t i = 0; const auto& device : devices) {
		VkPhysicalDeviceDriverProperties driverProperties = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_DRIVER_PROPERTIES};
		VkPhysicalDeviceProperties2 properties = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_PROPERTIES_2, .pNext = &driverProperties};
		VK_PROC(vkGetPhysicalDeviceProperties2)(device.first, &properties);

		builder << "\n[" << i++ << "] ";

		switch (properties.properties.deviceType) {
			case VK_PHYSICAL_DEVICE_TYPE_OTHER:
				builder << "(Other) ";
				break;
			case VK_PHYSICAL_DEVICE_TYPE_INTEGRATED_GPU:
				builder << "(Integrated) ";
				break;
			case VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU:
				builder << "(Discrete) ";
				break;
			case VK_PHYSICAL_DEVICE_TYPE_VIRTUAL_GPU:
				builder << "(Virtual) ";
				break;
			case VK_PHYSICAL_DEVICE_TYPE_CPU:
				builder << "(Software) ";
				break;
			default:
				builder << "(UNKNOWN: " << properties.properties.deviceType << ") ";
				break;
		}

		builder << static_cast<const char*>(properties.properties.deviceName);

		builder << " UUID: ";
		static_assert(VK_UUID_SIZE == 16);
		for (size_t i = 0; i < 4; i++) {
			builder.writef("%02X", device.second[i]);
		}
		builder << "-";
		for (size_t i = 4; i < 6; i++) {
			builder.writef("%02X", device.second[i]);
		}
		builder << "-";
		for (size_t i = 6; i < 8; i++) {
			builder.writef("%02X", device.second[i]);
		}
		builder << "-";
		for (size_t i = 8; i < 10; i++) {
			builder.writef("%02X", device.second[i]);
		}
		builder << "-";
		for (size_t i = 10; i < 16; i++) {
			builder.writef("%02X", device.second[i]);
		}

		builder.writef(" VRAM: %.2f GiB", (double)vramSize(device.first) / (double)vxr::std::unit::memory::gibibyte);
		builder << " VK: " << VK_VERSION_MAJOR(properties.properties.apiVersion) << "."
				<< VK_VERSION_MINOR(properties.properties.apiVersion) << "." << VK_VERSION_PATCH(properties.properties.apiVersion)
				<< " Driver: " << static_cast<const char*>(driverProperties.driverName) << " "
				<< static_cast<const char*>(driverProperties.driverInfo);
	}
	vxr::std::iPrintf(builder.cStr());
}

inline static auto getDeviceUUID(VkPhysicalDevice target, uint16_t index) {
	vxr::std::array<uint8_t, VK_UUID_SIZE> uuid;
	{
		VkPhysicalDeviceVulkan11Properties device11Properties = {
			.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_1_PROPERTIES,
		};
		VkPhysicalDeviceProperties2 deviceProperties2 = {
			.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_PROPERTIES_2,
			.pNext = &device11Properties,
		};
		VK_PROC(vkGetPhysicalDeviceProperties2)(target, &deviceProperties2);

		static constexpr vxr::std::array<uint8_t, 6> test;
		static_assert((VK_UUID_SIZE - test.size()) == 10);
		// drivers will have either all 0s, front half 0s or second half 0s, a valid and useable uuid will not have those regions completely 0ed
		if (vxr::std::inRange(device11Properties.deviceUUID[6] >> 4, 1, 8) &&
			memcmp(device11Properties.deviceUUID, test.get(), test.size()) != 0 &&
			memcmp(&device11Properties.deviceUUID[10], test.get(), test.size()) != 0) {
			memcpy(uuid.get(), device11Properties.deviceUUID, VK_UUID_SIZE);
		} else {
			auto properties = deviceProperties2.properties;
			// byte 6 contains UUID version, version 8 means do whatever you want, byte 8 contains variant, F is an
			// invalid value we use that as we are not following any known variant
			// NOLINTNEXTLINE(modernize-avoid-c-arrays)
			static constexpr uint8_t uuidBaseBits[VK_UUID_SIZE] = {0, 0, 0, 0, 0, 0, 0x80, 0, 0xF0};
			memcpy(uuid.get(), uuidBaseBits, VK_UUID_SIZE);
			memcpy(uuid.get(), &properties.vendorID, sizeof(properties.vendorID));
			// deviceID is not unique enough on multi-gpu systems, throw in the index inot the uuid too
			memcpy(uuid.get() + 4, &index, sizeof(index));
			memcpy(uuid.get() + 10, &properties.deviceID, sizeof(properties.deviceID));
		}
	}
	return uuid;
}
inline static auto getDevices(VkPhysicalDevice preferredDevice, vxr::vk::instance* instance) {
	uint32_t numDevices = 0;
	VkResult ret = VK_PROC(vkEnumeratePhysicalDevices)(instance->vkInstance, &numDevices, nullptr);
	if (ret != VK_SUCCESS) {
		vxr::std::abortPopup(
			vxr::std::sourceLocation::current(), "Failed to get list of GPU devices: %s", vxr::vk::vkResultStr(ret).cStr());
	}
	if (numDevices == 0) {
		vxr::std::abortPopup(vxr::std::sourceLocation::current(), "Failed to get list of GPU devices: List is empty");
	}
	vxr::std::vector<VkPhysicalDevice> devices(numDevices);
	ret = VK_PROC(vkEnumeratePhysicalDevices)(instance->vkInstance, &numDevices, devices.get());
	if (ret != VK_SUCCESS) {
		vxr::std::abortPopup(
			vxr::std::sourceLocation::current(), "Failed to get list of GPU devices: %s", vxr::vk::vkResultStr(ret).cStr());
	}
	vxr::std::vector<vxr::std::pair<VkPhysicalDevice, vxr::std::array<uint8_t, VK_UUID_SIZE>>> list(numDevices);
	// no sane person is going to overflow uint16_t
	for (uint16_t i = 0; i < uint16_t(numDevices); i++) {
		list[i] = vxr::std::pair(devices[i], getDeviceUUID(devices[i], i));
	}
	if (numDevices > 1) {
		bool found = false;
		for (size_t i = 0; i < devices.size(); i++) {
			if (devices[i] == preferredDevice) {
				vxr::std::iPrintf("Putting preferred VkPhysicalDevice to top of list");
				vxr::std::swap(list[0], list[i]);
				found = true;
				break;
			}
		}
		std::ranges::stable_sort(list.begin() + size_t(found), list.end(), [](const auto& deviceA, const auto& deviceB) {
			VkPhysicalDeviceProperties propertiesA = {};
			VK_PROC(vkGetPhysicalDeviceProperties)(deviceA.first, &propertiesA);

			VkPhysicalDeviceProperties propertiesB = {};
			VK_PROC(vkGetPhysicalDeviceProperties)(deviceB.first, &propertiesB);

			if (propertiesA.deviceType == VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU) {
				if (propertiesB.deviceType != VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU) {
					return true;
				}
			} else if (propertiesB.deviceType == VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU) {
				return false;
			}

			if (propertiesA.apiVersion > propertiesB.apiVersion) {
				return true;
			}

			return vramSize(deviceA.first) > vramSize(deviceB.first);
		});
	}
	printDevices(list);
	return vxr::std::move(list);
}

namespace vxr::vk::device::selector {
bool selector::checkDevice(vxr::vk::instance* instance) {
	static constexpr vxr::std::array deviceChecks = {
		vxr::std::pair("findProperties", &selector::findProperties),
		vxr::std::pair("findExtensions", &selector::findExtensions),
		vxr::std::pair("findFeatures", &selector::findFeatures),
		vxr::std::pair("findFormats", &selector::findFormats),
		vxr::std::pair("findQueues", &selector::findQueues),
	};
	bool checksOK = true;
	for (auto check : deviceChecks) {
		vxr::std::vPrintf("%s", check.first);
		if (!(this->*(check.second))(instance)) {
			vxr::std::vPrintf("%s: Fail", check.first);
			checksOK = false;
			break;
		}
		vxr::std::vPrintf("%s: Pass", check.first);
	}
	return checksOK;
}
void selector::findAndCreateDevice(vxr::vk::instance* instance) {
	auto devices = getDevices(this->preferredDevice, instance);
	for (size_t i = 0; const auto& device : devices) {
		vxr::std::iPrintf("Trying Device: [%d]", i++);
		instance->device.vkPhysicalDevice = device.first;

		vxr::std::iPrintf("Running device checks");
		if (!checkDevice(instance)) {
			vxr::std::iPrintf("Device checks failed");
			continue;
		}
		vxr::std::iPrintf("Device checks passed");
		memcpy(instance->device.properties.uuid, device.second.get(), VK_UUID_SIZE);

		{
			for (size_t i = 0; i < queueCreateInfos.size(); i++) {
				queueCreateInfos[i].pQueuePriorities = queuePriorities[i].get();
			}

			vxr::std::vector<const char*> extensions(enabledExtensions);
			const VkDeviceCreateInfo createInfo = {
				.sType = VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO,
				.pNext = &enabledFeatureChain.start,

				.queueCreateInfoCount = static_cast<uint32_t>(queueCreateInfos.size()),
				.pQueueCreateInfos = queueCreateInfos.get(),

				.enabledExtensionCount = static_cast<uint32_t>(extensions.size()),
				.ppEnabledExtensionNames = extensions.get(),
			};
			const VkResult ret = VK_PROC(vkCreateDevice)(
				instance->device.vkPhysicalDevice, &createInfo, nullptr, &instance->device.vkDevice);
			if (ret != VK_SUCCESS) {
				vxr::std::abortPopup(
					vxr::std::sourceLocation::current(), "Failed to initialize device: %s", vkResultStr(ret).cStr());
				vxr::std::iPrintf("Incompatible Device");
				continue;
			}

			vxr::std::iPrintf("Device Created");
		}
		return;
	}

	vxr::std::abortPopup(vxr::std::sourceLocation::current(),
						 "No compatible vulkan devices found.\n"
						 "Ensure your GPU and drivers meet the minimum requirements to run this software.");
}
}  // namespace vxr::vk::device::selector

extern "C" {
VXR_FN void vxr_vk_device_vkPhysicalDeviceFromUUID(vxr_vk_instance instanceHandle, uint8_t (*wantUUID)[VK_UUID_SIZE], uintptr_t* physicalDevice) {
	auto* instance = vxr::vk::instance::fromHandle(instanceHandle);

	uint32_t numDevices = 0;
	VkResult ret = VK_PROC(vkEnumeratePhysicalDevices)(instance->vkInstance, &numDevices, nullptr);
	if (ret != VK_SUCCESS) {
		vxr::std::abortPopup(
			vxr::std::sourceLocation::current(), "Failed to get list of GPU devices: %s", vxr::vk::vkResultStr(ret).cStr());
	}
	if (numDevices == 0) {
		vxr::std::abortPopup(vxr::std::sourceLocation::current(), "Failed to get list of GPU devices: List is empty");
	}
	vxr::std::vector<VkPhysicalDevice> devices(numDevices);
	ret = VK_PROC(vkEnumeratePhysicalDevices)(instance->vkInstance, &numDevices, devices.get());
	if (ret != VK_SUCCESS) {
		vxr::std::abortPopup(
			vxr::std::sourceLocation::current(), "Failed to get list of GPU devices: %s", vxr::vk::vkResultStr(ret).cStr());
	}
	for (uint16_t i = 0; i < uint16_t(numDevices); i++) {
		auto uuid = getDeviceUUID(devices[i], i);
		if (memcmp(*wantUUID, uuid.get(), VK_UUID_SIZE) == 0) {
			*physicalDevice = reinterpret_cast<uintptr_t>(devices[i]);
			return;
		}
	}
	*physicalDevice = 0;
}
VXR_FN void vxr_vk_device_createSelector(uintptr_t preferredDevice, uint32_t api, uint64_t targetSurface, vxr_vk_device_selector* selectorHandle) {
	// NOLINTBEGIN(performance-no-int-to-ptr)
	auto* selector = new (::std::nothrow) vxr::vk::device::selector::selector(
		reinterpret_cast<VkPhysicalDevice>(preferredDevice), api, reinterpret_cast<VkSurfaceKHR>(targetSurface));
	*selectorHandle = selector->handle();
	// NOLINTEND(performance-no-int-to-ptr)
}
VXR_FN void vxr_vk_device_destroySelector(vxr_vk_device_selector selectorHandle) {
	auto* selector = vxr::vk::device::selector::selector::fromHandle(selectorHandle);
	delete selector;
}
VXR_FN void vxr_vk_device_selector_appendRequiredExtension(vxr_vk_device_selector selectorHandle, size_t sz, const char* extension) {
	auto* selector = vxr::vk::device::selector::selector::fromHandle(selectorHandle);
	for (auto& s : selector->optionalExtensions) {
		if (s == extension) {
			vxr::std::ePrintf("Extension: %s cannot be both required and optional", s.cStr());
			vxr::std::abort();
		}
	}
	for (auto& s : selector->requiredExtensions) {
		if (s == extension) {
			return;
		}
	}
	selector->requiredExtensions.pushBack(vxr::std::string<char>(sz, extension));
}
VXR_FN void vxr_vk_device_selector_appendOptionalExtension(vxr_vk_device_selector selectorHandle, size_t sz, const char* extension) {
	auto* selector = vxr::vk::device::selector::selector::fromHandle(selectorHandle);
	for (auto& s : selector->requiredExtensions) {
		if (s == extension) {
			vxr::std::ePrintf("Extension: %s cannot be both required and optional", s.cStr());
			vxr::std::abort();
		}
	}
	for (auto& s : selector->optionalExtensions) {
		if (s == extension) {
			return;
		}
	}
	selector->optionalExtensions.pushBack(vxr::std::string<char>(sz, extension));
}
VXR_FN void vxr_vk_device_selector_initFeatureChain(vxr_vk_device_selector selectorHandle, size_t numStructs, VkStructureType* structs) {
	auto* selector = vxr::vk::device::selector::selector::fromHandle(selectorHandle);
	selector->requiredFeatureChain.reset();
	selector->optionalFeatureChain.reset();

	for (size_t i = 0; i < numStructs; i++) {
		selector->requiredFeatureChain.append(structs[i]);
		selector->optionalFeatureChain.append(structs[i]);
	}
}
VXR_FN void vxr_vk_device_selector_appendRequiredFeature(vxr_vk_device_selector selectorHandle, VkStructureType sType,
														 size_t numFeatures, size_t* features) {
	auto* selector = vxr::vk::device::selector::selector::fromHandle(selectorHandle);
	selector->requiredFeatureChain.append(sType, numFeatures, features);
}
VXR_FN void vxr_vk_device_selector_appendOptionalFeature(vxr_vk_device_selector selectorHandle, VkStructureType sType,
														 size_t numFeatures, size_t* features) {
	auto* selector = vxr::vk::device::selector::selector::fromHandle(selectorHandle);
	selector->optionalFeatureChain.append(sType, numFeatures, features);
}
VXR_FN void vxr_vk_device_selector_appendRequiredFormatFeature(vxr_vk_device_selector selectorHandle, VkFormat format, VkFormatFeatureFlags2 feature) {
	auto* selector = vxr::vk::device::selector::selector::fromHandle(selectorHandle);
	selector->requiredFormatFeatures.pushBack(vxr::std::pair(format, feature));
}
VXR_FN void vxr_vk_device_selector_getEnabledExtensions(vxr_vk_device_selector selectorHandle, size_t* sz, const char** out) {
	auto* selector = vxr::vk::device::selector::selector::fromHandle(selectorHandle);

	if (out != nullptr) {
		for (size_t i = 0; i < vxr::std::min(*sz, selector->enabledExtensions.size()); i++) {
			out[i] = selector->enabledExtensions[i].cStr();
		}
	} else {
		*sz = selector->enabledExtensions.size();
	}
}
VXR_FN void vxr_vk_device_selector_getEnabledFeatures(vxr_vk_device_selector selectorHandle, const char** out) {
	auto* selector = vxr::vk::device::selector::selector::fromHandle(selectorHandle);
	auto sb = vxr::std::stringbuilder();
	sb.write("{");

	{
		auto s = vxr::vk::device::reflect::valueOf(&selector->enabledFeatureChain.start.features);
		auto* next = reinterpret_cast<vxr::vk::device::reflect::vkStructureChain*>(&selector->enabledFeatureChain.start);
		while (next != nullptr) {
			sb.writef("\"%s\":{", s->type->name);
			bool hasFeatures = false;
			for (auto& field : *s) {
				switch (field.type.id) {
					case vxr::vk::device::reflect::type::vkBool32: {
						if (*static_cast<VkBool32*>(field.ptr) == VK_TRUE) {
							sb.writef("\"%s\": true,", field.name);
							hasFeatures = true;
						}
					} break;
					default:
						break;
				}
			}
			next = next->pNext;
			if (next != nullptr) {
				s = vxr::vk::device::reflect::valueOf(next);
			}
			if (hasFeatures) {
				sb.backspace();
			}
			sb.write("},");
		}
		sb.backspace();
	}

	sb.write("}");
	selector->enabledFeatureString = vxr::std::move(sb.str());
	*out = selector->enabledFeatureString.cStr();
}
}

// NOLINTEND(clang-analyzer-core.CallAndMessage)
