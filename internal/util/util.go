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

package util

import (
	"unsafe"

	"goarrg.com"
	"goarrg.com/debug"
)

type platform struct{}

func (platform) Abort()                           { panic("Fatal Error") }
func (platform) AbortPopup(f string, args ...any) { panic("Fatal Error") }

var instance = struct {
	platform goarrg.PlatformInterface
	logger   *debug.Logger
}{
	platform: platform{},
	logger:   debug.NewLogger("vxr", "internal", "util"),
}

func abort(fmt string, args ...any) {
	instance.logger.EPrintf(fmt, args...)
	instance.platform.Abort()
}

func Init(platform goarrg.PlatformInterface) {
	instance.platform = platform
}

type HostWriter interface {
	HostWrite(offset uintptr, data []byte)
}

func HostWrite[T comparable](target HostWriter, offset uintptr, data T) uintptr {
	target.HostWrite(offset,
		unsafe.Slice((*byte)(unsafe.Pointer(&data)), unsafe.Sizeof(data)),
	)
	return unsafe.Sizeof(data)
}

func HostWriteSlice[T comparable](target HostWriter, offset uintptr, data []T) uintptr {
	target.HostWrite(offset,
		unsafe.Slice(
			(*byte)(unsafe.Pointer(unsafe.SliceData(data))), uint64(unsafe.Sizeof(data[0]))*uint64(len(data)),
		),
	)
	return unsafe.Sizeof(data[0]) * uintptr(len(data))
}
