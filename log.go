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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unsafe"

	"goarrg.com/rhi/vxr/internal/vk"
)

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

	// format message
	if (cMessageType & (vk.DEBUG_UTILS_MESSAGE_TYPE_VALIDATION_BIT_EXT | vk.DEBUG_UTILS_MESSAGE_TYPE_PERFORMANCE_BIT_EXT)) != 0 {
		sb := bytes.Buffer{}
		sb.Grow(len(message))

		sc := bufio.NewScanner(bytes.NewReader([]byte(message)))
		sc.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, io.EOF
			}

			{
				var i int
				{
					i = bytes.Index(data, []byte("("))
					off := i + 1
					for len(data) > off && data[i+1] == ')' {
						k := bytes.Index(data[off:], []byte("("))
						if k == -1 {
							break
						}
						i = off + k
						off += k + 1
					}
					if i > 0 && data[i+1] == ')' {
						i = -1
					}
				}
				if i == 0 {
					j := bytes.Index(data, []byte(")"))
					off := j + 1
					for j > 0 && len(data) > off && data[j-1] == '(' {
						k := bytes.Index(data[off:], []byte(")"))
						if k == -1 {
							break
						}
						j = off + k
						off += k + 1
					}
					if j > 0 {
						i = j + 1
					} else {
						return 0, nil, nil
					}
				} else {
					{
						j := bytes.IndexAny(data, ":;")
						off := j + 2
						for len(data) > off && !unicode.IsSpace(rune(data[j+1])) {
							k := bytes.Index(data[off:], []byte(":"))
							if k == -1 {
								break
							}
							j = off + k
							off += k + 1
						}
						if !atEOF && len(data) <= j+1 {
							return 0, nil, nil
						}
						if j > 0 && (len(data) <= j+1 || unicode.IsSpace(rune(data[j+1]))) {
							if i < 0 {
								i = j + 1
							} else {
								i = min(i, j+1)
							}
						}
					}
					{
						j := bytes.Index(data, []byte("."))
						off := j + 1
						for len(data) > off && !unicode.IsSpace(rune(data[j+1])) {
							k := bytes.Index(data[off:], []byte("."))
							if k == -1 || k > i {
								break
							}
							j = off + k
							off += k + 1
						}
						if !atEOF && len(data) <= j+1 {
							return 0, nil, nil
						}
						if j > 0 && (len(data) <= j+1 || unicode.IsSpace(rune(data[j+1]))) {
							if i < 0 {
								i = j + 1
							} else {
								i = min(i, j+1)
							}
						}
					}
				}

				if i > 0 {
					return i, data[:i], nil
				}
			}

			if atEOF {
				return len(data), data, bufio.ErrFinalToken
			}

			return 0, nil, nil
		})

		lastLine := ""
		for sc.Scan() {
			l := strings.TrimSpace(sc.Text())
			switch {
			case l[0] == '(':
				{
					text := l[1 : len(l)-1]
					if len(lastLine)+len(text) > 160 {
						fields := strings.Split(text, ", ")
						sb.Truncate(sb.Len() - 1)

						if len(fields) > 1 {
							sb.WriteString(":\n(\n")
							for _, f := range fields {
								if strings.Contains(f, "|") && len(f) > 80 {
									fields2 := strings.Split(f, "|")
									sb.WriteString("    " + fields2[0] + "\n")
									align := strings.Repeat(" ", max(strings.LastIndex(fields2[0], " ")+1, 4)+4)
									for _, f2 := range fields2[1:] {
										sb.WriteString(align + f2 + "\n")
									}
								} else {
									sb.WriteString("    " + f + "\n")
								}
							}
							sb.WriteString(")\n")
						} else {
							sb.WriteString(":\n(")
							sb.WriteString(text)
							sb.WriteString(")\n")
						}
					} else {
						sb.Truncate(sb.Len() - 1)
						sb.WriteString(": (")
						sb.WriteString(text)
						sb.WriteString(")\n")
					}
				}

			case l[len(l)-1] == ':', l[len(l)-1] == ';':
				{
					l = strings.TrimLeftFunc(l, func(r rune) bool { return r == '.' || unicode.IsSpace(r) })
					l = strings.ReplaceAll(l, ". ", ".\n")
					if l[len(l)-1] == ':' {
						sb.WriteString("\n")
					}
					sb.WriteString(l)
					sb.WriteString("\n")
				}

			default:
				{
					l = strings.TrimLeftFunc(l, func(r rune) bool { return r == '.' || unicode.IsSpace(r) })
					sb.WriteString(l)
					sb.WriteString("\n")
				}
			}
			lastLine = l
		}
		message = strings.TrimSpace(sb.String())
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
