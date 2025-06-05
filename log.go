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

package vxr

/*
	#cgo pkg-config: vxr
	#cgo CFLAGS: -fvisibility=hidden -Wno-dll-attribute-on-redeclaration

	#include "vxr/vxr.h"

	extern void goAbort(size_t, char*);
	extern void goAbortPopup(size_t, char*);

	extern void goLogV(size_t, char*);
	extern void goLogI(size_t, char*);
	extern void goLogW(size_t, char*);
	extern void goLogE(size_t, char*);

	extern VkBool32 goVkLog(VkDebugUtilsMessageSeverityFlagBitsEXT, VkDebugUtilsMessageTypeFlagsEXT, VkDebugUtilsMessengerCallbackDataEXT*, void*);
*/
import "C"

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"goarrg.com/rhi/vxr/internal/vk"
)

func init() {
	C.vxr_stdlib_init(cGoAbort, cGoAbortPopup, cGoLogV, cGoLogI, cGoLogW, cGoLogE)
}

func abort(fmt string, args ...any) {
	instance.logger.EPrintf(fmt, args...)
	instance.platform.Abort()
}

func abortPopup(fmt string, args ...any) {
	instance.logger.EPrintf("[popup] "+fmt, args...)
	instance.platform.AbortPopup(fmt, args...)
}

var (
	cGoAbort      = C.vxr_loggerCallback(C.goAbort)
	cGoAbortPopup = C.vxr_loggerCallback(C.goAbortPopup)

	cGoLogV = C.vxr_loggerCallback(C.goLogV)
	cGoLogI = C.vxr_loggerCallback(C.goLogI)
	cGoLogW = C.vxr_loggerCallback(C.goLogW)
	cGoLogE = C.vxr_loggerCallback(C.goLogE)

	cGoVkLog = C.PFN_vkDebugUtilsMessengerCallbackEXT(C.goVkLog)
)

//export goAbort
func goAbort(cSz C.size_t, cStr *C.char) {
	msg := unsafe.String(((*byte)(unsafe.Pointer(cStr))), cSz)
	abort("%s", msg)
}

//export goAbortPopup
func goAbortPopup(cSz C.size_t, cStr *C.char) {
	msg := unsafe.String(((*byte)(unsafe.Pointer(cStr))), cSz)
	abortPopup("%s", msg)
}

//export goLogV
func goLogV(cSz C.size_t, cStr *C.char) {
	instance.logger.VPrintf("%s", unsafe.String(((*byte)(unsafe.Pointer(cStr))), cSz))
}

//export goLogI
func goLogI(cSz C.size_t, cStr *C.char) {
	instance.logger.IPrintf("%s", unsafe.String(((*byte)(unsafe.Pointer(cStr))), cSz))
}

//export goLogW
func goLogW(cSz C.size_t, cStr *C.char) {
	instance.logger.WPrintf("%s", unsafe.String(((*byte)(unsafe.Pointer(cStr))), cSz))
}

//export goLogE
func goLogE(cSz C.size_t, cStr *C.char) {
	instance.logger.EPrintf("%s", unsafe.String(((*byte)(unsafe.Pointer(cStr))), cSz))
}

//export goVkLog
func goVkLog(cMessageSeverity C.VkDebugUtilsMessageSeverityFlagBitsEXT,
	cMessageType C.VkDebugUtilsMessageTypeFlagsEXT,
	cCallbackData *C.VkDebugUtilsMessengerCallbackDataEXT,
	cUserData unsafe.Pointer,
) C.VkBool32 {
	message := strings.TrimSpace(C.GoString(cCallbackData.pMessage))
	messageId := C.GoString(cCallbackData.pMessageIdName)
	format := ""

	{
		messageIdBlacklist := map[C.int32_t]struct{}{
			-840639837: {}, // BestPractices-AllocateMemory-SetPriority
			948173112:  {}, // BestPractices-Pipeline-NoRendering
		}
		if _, blacklisted := messageIdBlacklist[cCallbackData.messageIdNumber]; blacklisted {
			return vk.FALSE
		}
	}

	if cMessageType&vk.DEBUG_UTILS_MESSAGE_TYPE_GENERAL_BIT_EXT == vk.DEBUG_UTILS_MESSAGE_TYPE_GENERAL_BIT_EXT {
		format += "[VkGen] "
	}
	if cMessageType&vk.DEBUG_UTILS_MESSAGE_TYPE_VALIDATION_BIT_EXT == vk.DEBUG_UTILS_MESSAGE_TYPE_VALIDATION_BIT_EXT {
		format += "[VkVal] "
	}
	if cMessageType&vk.DEBUG_UTILS_MESSAGE_TYPE_PERFORMANCE_BIT_EXT == vk.DEBUG_UTILS_MESSAGE_TYPE_PERFORMANCE_BIT_EXT {
		format += "[VkPer] "
	}

	if cCallbackData.pMessageIdName != nil {
		format += fmt.Sprintf("[%s: %d] ", messageId, cCallbackData.messageIdNumber)
	} else {
		format += fmt.Sprintf("[MessageId: %d] ", cCallbackData.messageIdNumber)
	}
	if cCallbackData.queueLabelCount > 0 {
		cLabels := unsafe.Slice(cCallbackData.pQueueLabels, cCallbackData.queueLabelCount)
		for i := 0; i < int(cCallbackData.queueLabelCount); i++ {
			format += "[Queue: " + C.GoString(cLabels[i].pLabelName) + "] "
		}
	}
	if cCallbackData.cmdBufLabelCount > 0 {
		cLabels := unsafe.Slice(cCallbackData.pCmdBufLabels, cCallbackData.cmdBufLabelCount)
		for i := 0; i < int(cCallbackData.cmdBufLabelCount); i++ {
			format += "[CB: " + C.GoString(cLabels[i].pLabelName) + "] "
		}
	}
	if cCallbackData.objectCount > 0 {
		cLabels := unsafe.Slice(cCallbackData.pObjects, cCallbackData.objectCount)
		for i := 0; i < int(cCallbackData.objectCount); i++ {
			if cLabels[i].pObjectName != nil {
				format += "[Obj: " + C.GoString(cLabels[i].pObjectName) + " 0x" + strings.ToUpper(strconv.FormatUint(uint64(cLabels[i].objectHandle), 16)) + "] "
			} else {
				format += "[Obj: 0x" + strings.ToUpper(strconv.FormatUint(uint64(cLabels[i].objectHandle), 16)) + "] "
			}
		}
	}

	switch {
	case cMessageSeverity >= vk.DEBUG_UTILS_MESSAGE_SEVERITY_ERROR_BIT_EXT:
		instance.logger.EPrintf("%s\n%s", format, message)
	case cMessageSeverity >= vk.DEBUG_UTILS_MESSAGE_SEVERITY_WARNING_BIT_EXT:
		instance.logger.WPrintf("%s\n%s", format, message)
		/*
			case cMessageSeverity >= vk.DEBUG_UTILS_MESSAGE_SEVERITY_INFO_BIT_EXT:
				instance.logger.IPrintf("%s\n%s", format, message)
		*/
	case cMessageSeverity >= vk.DEBUG_UTILS_MESSAGE_SEVERITY_VERBOSE_BIT_EXT:
		instance.logger.VPrintf("%s\n%s", format, message)
	}

	return vk.FALSE
}
