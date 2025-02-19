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
	"runtime"
	"unsafe"

	"goarrg.com/rhi/vxr/internal/vk"
)

type SemaphoreWaiter interface {
	vkWaitInfo(stage PipelineStage) C.VkSemaphoreSubmitInfo
}

type SemaphoreSignaler interface {
	vkSignalInfo(stage PipelineStage) C.VkSemaphoreSubmitInfo
}

type SemaphoreWaitInfo struct {
	Semaphore SemaphoreWaiter
	Stage     PipelineStage
}

type SemaphoreSignalInfo struct {
	Semaphore SemaphoreSignaler
	Stage     PipelineStage
}

type binarySemaphore struct {
	noCopy      noCopy
	vkSemaphore C.VkSemaphore
}

var (
	_ SemaphoreWaiter   = (*binarySemaphore)(nil)
	_ SemaphoreSignaler = (*binarySemaphore)(nil)
)

func (s *binarySemaphore) vkSignalInfo(stage PipelineStage) C.VkSemaphoreSubmitInfo {
	s.noCopy.check()
	return C.VkSemaphoreSubmitInfo{
		sType:     vk.STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
		semaphore: s.vkSemaphore,
		stageMask: C.VkPipelineStageFlags2(stage),
	}
}

func (s *binarySemaphore) vkWaitInfo(stage PipelineStage) C.VkSemaphoreSubmitInfo {
	s.noCopy.check()
	return C.VkSemaphoreSubmitInfo{
		sType:     vk.STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
		semaphore: s.vkSemaphore,
		stageMask: C.VkPipelineStageFlags2(stage),
	}
}

type TimelineSemaphore struct {
	noCopy        noCopy
	vkSemaphore   C.VkSemaphore
	gpuPending    C.uint64_t
	cpuPending    C.uint64_t
	pendingSignal C.uint64_t
	value         C.uint64_t
}

var _ interface {
	SemaphoreWaiter
	SemaphoreSignaler
	Destroyer
} = (*TimelineSemaphore)(nil)

func NewTimelineSemaphore(name string) *TimelineSemaphore {
	s := TimelineSemaphore{}
	s.noCopy.init()
	C.vxr_vk_createSemaphore(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
		vk.SEMAPHORE_TYPE_TIMELINE, &s.vkSemaphore)
	runtime.KeepAlive(name)
	return &s
}

func (s *TimelineSemaphore) Destroy() {
	s.noCopy.check()
	s.Wait()
	C.vxr_vk_destroySemaphore(instance.cInstance, s.vkSemaphore)
	s.noCopy.close()
}

func (s *TimelineSemaphore) Value() uint64 {
	s.noCopy.check()
	s.value = C.vxr_vk_getSemaphoreValue(instance.cInstance, s.vkSemaphore)
	return uint64(s.value)
}

func (s *TimelineSemaphore) vkSignalInfo(stage PipelineStage) C.VkSemaphoreSubmitInfo {
	s.noCopy.check()
	s.pendingSignal += 1
	s.gpuPending = s.pendingSignal
	return C.VkSemaphoreSubmitInfo{
		sType:     vk.STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
		semaphore: s.vkSemaphore,
		value:     s.gpuPending,
		stageMask: C.VkPipelineStageFlags2(stage),
	}
}

func (s *TimelineSemaphore) sendSignal(signal C.uint64_t) {
	s.noCopy.check()
	if s.value >= s.cpuPending {
		abort("No pending CPU signal promise")
	}
	C.vxr_vk_signalSemaphore(instance.cInstance, s.vkSemaphore, signal)
	s.value = signal
}

type TimelineSemaphorePromise struct {
	noCopy    noCopy
	semaphore *TimelineSemaphore
	value     C.uint64_t
}

func (s *TimelineSemaphore) Promise() *TimelineSemaphorePromise {
	s.noCopy.check()
	s.pendingSignal += 1
	s.cpuPending = s.pendingSignal
	p := TimelineSemaphorePromise{semaphore: s, value: s.pendingSignal}
	p.noCopy.init()
	return &p
}

func (p *TimelineSemaphorePromise) Signal() {
	p.noCopy.check()
	// this ensures we signal in order
	p.semaphore.waitForSignal(p.value - 1)
	p.semaphore.sendSignal(p.value)
	p.noCopy.close()
}

func (p *TimelineSemaphorePromise) Value() uint64 {
	p.noCopy.check()
	return uint64(p.value)
}

func (s *TimelineSemaphore) vkWaitInfo(stage PipelineStage) C.VkSemaphoreSubmitInfo {
	s.noCopy.check()
	return C.VkSemaphoreSubmitInfo{
		sType:     vk.STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
		semaphore: s.vkSemaphore,
		value:     s.pendingSignal,
		stageMask: C.VkPipelineStageFlags2(stage),
	}
}

func (s *TimelineSemaphore) waitForSignal(signal C.uint64_t) {
	if s.value >= signal {
		return
	}
	s.noCopy.check()
	C.vxr_vk_waitSemaphore(instance.cInstance, s.vkSemaphore, signal)
	s.value = signal
}

func (s *TimelineSemaphore) Wait() {
	s.waitForSignal(s.pendingSignal)
}

type TimelineSemaphoreWaiter struct {
	noCopy    noCopy
	semaphore *TimelineSemaphore
	value     C.uint64_t
}

var _ SemaphoreWaiter = (*TimelineSemaphoreWaiter)(nil)

func (s *TimelineSemaphore) WaiterForPendingValue() *TimelineSemaphoreWaiter {
	s.noCopy.check()
	f := TimelineSemaphoreWaiter{semaphore: s, value: s.pendingSignal}
	f.noCopy.init()
	return &f
}

func (s *TimelineSemaphore) WaiterForCurrentValue() *TimelineSemaphoreWaiter {
	s.noCopy.check()
	s.value = C.vxr_vk_getSemaphoreValue(instance.cInstance, s.vkSemaphore)
	f := TimelineSemaphoreWaiter{semaphore: s, value: s.value}
	f.noCopy.init()
	return &f
}

func (w *TimelineSemaphoreWaiter) vkWaitInfo(stage PipelineStage) C.VkSemaphoreSubmitInfo {
	w.noCopy.check()
	w.semaphore.noCopy.check()
	return C.VkSemaphoreSubmitInfo{
		sType:     vk.STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
		semaphore: w.semaphore.vkSemaphore,
		value:     w.value,
		stageMask: C.VkPipelineStageFlags2(stage),
	}
}

func (w *TimelineSemaphoreWaiter) Poll() bool {
	w.noCopy.check()
	status := C.vxr_vk_getSemaphoreValue(instance.cInstance, w.semaphore.vkSemaphore)
	return status >= w.value
}

func (w *TimelineSemaphoreWaiter) Wait() {
	w.noCopy.check()
	w.semaphore.waitForSignal(w.value)
}

func (w *TimelineSemaphoreWaiter) Value() uint64 {
	w.noCopy.check()
	return uint64(w.value)
}
