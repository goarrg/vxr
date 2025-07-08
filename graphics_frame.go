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
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"

	"goarrg.com/rhi/vxr/internal/util"
	"goarrg.com/rhi/vxr/internal/vk"
)

type frame struct {
	cFrame     C.vxr_vk_graphics_frame
	waiter     *TimelineSemaphoreWaiter
	destroyers []Destroyer
}

func (f *frame) wait() {
	if f.waiter != nil {
		f.waiter.Wait()
		f.waiter = nil
	} else {
		f.waitSurface()
	}
	for _, d := range f.destroyers {
		d.Destroy()
	}
	f.destroyers = f.destroyers[:0]
}

func (f *frame) waitSurface() {
	C.vxr_vk_graphics_frame_wait(instance.cInstance, f.cFrame)
}

func (f *frame) destroy() {
	f.wait()
	C.vxr_vk_graphics_destroyFrame(f.cFrame)
}

type Frame struct {
	noCopy     util.NoCopy
	surface    *Surface
	frame      *frame
	name       string
	cancelable bool
}

func FrameBegin() *Frame {
	if instance.graphics.frameStarted {
		abort("FrameBegin called when there's an active frame")
	}
	f := &instance.graphics.framesInFlight[instance.graphics.frameIndex]
	f.wait()
	ret := Frame{frame: f, name: fmt.Sprintf("frame_%d", instance.graphics.frameIndex), cancelable: true}
	ret.noCopy.Init()
	C.vxr_vk_graphics_frame_begin(instance.cInstance, C.size_t(len(ret.name)), (*C.char)(unsafe.Pointer(unsafe.StringData(ret.name))),
		ret.frame.cFrame)
	instance.graphics.frameStarted = true
	return &ret
}

func (f *Frame) Index() int {
	f.noCopy.Check()
	return instance.graphics.frameIndex
}

func (f *Frame) Surface() *Surface {
	f.noCopy.Check()
	if instance.sleep {
		return nil
	}
	if f.surface != nil {
		return f.surface
	}
	f.frame.waitSurface()
	var surface Surface
	surface.noCopy.Init()
	switch ret := C.vxr_vk_graphics_frame_acquireSurface(instance.cInstance, f.frame.cFrame, &surface.cSurface); ret {
	case vk.SUCCESS:
	case vk.SUBOPTIMAL_KHR:
	case vk.ERROR_OUT_OF_DATE_KHR:
		instance.sleep = true
		return nil
	default:
		abort("Failed to acquire surface: %s", vkResultStr(ret))
	}
	f.surface = &surface
	f.cancelable = false
	return f.surface
}

type HostScratchBuffer struct {
	noCopy     util.NoCopy
	bufferSize uint64
	usageFlags BufferUsageFlags
	cBuffer    C.vxr_vk_hostBuffer
}

var _ Buffer = (*HostScratchBuffer)(nil)

func (f *Frame) NewHostScratchBuffer(name string, size uint64, usage BufferUsageFlags) *HostScratchBuffer {
	f.noCopy.Check()
	b := HostScratchBuffer{bufferSize: size, usageFlags: usage}
	b.noCopy.Init()
	info := C.vxr_vk_bufferCreateInfo{
		size:  C.VkDeviceSize(size),
		usage: C.VkBufferUsageFlags(usage),
	}
	name = fmt.Sprintf("%s_%s", f.name, name)
	C.vxr_vk_graphics_frame_createHostScratchBuffer(instance.cInstance, f.frame.cFrame,
		C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))), info, &b.cBuffer)
	runtime.KeepAlive(name)
	f.cancelable = false
	return &b
}

func (b *HostScratchBuffer) HostWrite(offset uintptr, data []byte) {
	b.noCopy.Check()
	if (uint64(len(data)) + uint64(offset)) > b.bufferSize {
		abort("HostWrite(%d, len(data): %d) will overflow buffer of size %d", offset, len(data), b.bufferSize)
	}

	C.vxr_vk_hostBuffer_write(instance.cInstance, b.cBuffer, C.size_t(offset), C.size_t(len(data)), unsafe.Pointer(unsafe.SliceData(data)))
	runtime.KeepAlive(data)
}

func (b *HostScratchBuffer) Usage() BufferUsageFlags {
	b.noCopy.Check()
	return b.usageFlags
}

func (b *HostScratchBuffer) Size() uint64 {
	b.noCopy.Check()
	return b.bufferSize
}

func (b *HostScratchBuffer) vkBuffer() C.VkBuffer {
	b.noCopy.Check()
	return b.cBuffer.vkBuffer
}

func (f *Frame) NewSingleUseCommandBuffer(name string) *GraphicsCommandBuffer {
	f.noCopy.Check()
	cb := GraphicsCommandBuffer{cFrame: f.frame.cFrame}
	cb.noCopy.Init()
	name = fmt.Sprintf("%s_%s", f.name, name)
	C.vxr_vk_graphics_frame_commandBufferBegin(instance.cInstance, f.frame.cFrame,
		C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))), &cb.vkCommandBuffer)
	runtime.KeepAlive(name)
	f.cancelable = false
	return &cb
}

func (f *Frame) Cancel() {
	f.noCopy.Check()
	if !f.cancelable {
		abort("Cannot cancel frame with acquired surface or after calling any of the New* functions")
	}
	C.vxr_vk_graphics_frame_end(instance.cInstance, f.frame.cFrame)
	f.frame.waiter = nil
	instance.graphics.frameStarted = false
	f.noCopy.Close()
}

/*
QueueDestory is a convenience function to avoid having to store destroyers until the end of the frame,
it is eq to passing the destroyers to any of the End functions.
*/
func (f *Frame) QueueDestory(destroyers ...Destroyer) {
	f.noCopy.Check()
	f.frame.destroyers = append(f.frame.destroyers, destroyers...)
}

func (f *Frame) EndWithWaiter(waiter *TimelineSemaphoreWaiter, destroyers ...Destroyer) {
	f.noCopy.Check()
	if f.surface != nil {
		if ret := C.vxr_vk_graphics_frame_submit(instance.cInstance, f.frame.cFrame); ret != vk.SUCCESS {
			instance.sleep = true
		}
	} else if waiter == nil {
		abort("Cannot end frame without an acquired surface and without a TimelineSemaphoreWaiter")
	}
	C.vxr_vk_graphics_frame_end(instance.cInstance, f.frame.cFrame)
loop:
	for {
		select {
		case j := <-instance.graphics.destroyerChan:
			f.frame.destroyers = append(f.frame.destroyers, j)
		default:
			break loop
		}
	}
	f.frame.waiter = waiter
	f.frame.destroyers = append(f.frame.destroyers, destroyers...)
	instance.graphics.frameIndex = (instance.graphics.frameIndex + 1) % len(instance.graphics.framesInFlight)
	instance.graphics.frameStarted = false
	f.noCopy.Close()
}

func (f *Frame) End(destroyers ...Destroyer) {
	f.EndWithWaiter(nil, destroyers...)
}
