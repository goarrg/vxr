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
	"goarrg.com/gmath"
	"goarrg.com/rhi/vxr/internal/vk"
)

type SurfaceInfo struct {
	Extent            gmath.Extent3i32
	Format            Format
	NumFramesInFlight int32
}

func CurrentSurfaceInfo() SurfaceInfo {
	var cInfo C.vxr_vk_surfaceInfo
	C.vxr_vk_graphics_getSurfaceInfo(instance.cInstance, &cInfo)

	if cInfo == (C.vxr_vk_surfaceInfo{}) {
		return SurfaceInfo{}
	}

	if instance.sleep {
		return SurfaceInfo{Format: Format(cInfo.format), NumFramesInFlight: min((int32)(cInfo.numImages), instance.config.maxFramesInFlight)}
	}

	return SurfaceInfo{
		Extent:            gmath.Extent3i32{X: int32(cInfo.extent.width), Y: int32(cInfo.extent.height), Z: 1},
		Format:            Format(cInfo.format),
		NumFramesInFlight: min((int32)(cInfo.numImages), instance.config.maxFramesInFlight),
	}
}

type Surface struct {
	noCopy   noCopy
	waited   bool
	cSurface C.vxr_vk_surface
}

var _ Image = (*Surface)(nil)

func (s *Surface) Extent() gmath.Extent3i32 {
	s.noCopy.check()
	return gmath.Extent3i32{
		X: int32(s.cSurface.info.extent.width),
		Y: int32(s.cSurface.info.extent.height),
		Z: 1,
	}
}

func (s *Surface) Format() Format {
	s.noCopy.check()
	return Format(s.cSurface.info.format)
}

func (s *Surface) usage() ImageUsageFlags {
	s.noCopy.check()
	return ImageUsageColorAttachment
}

func (s *Surface) vkFormat() C.VkFormat {
	s.noCopy.check()
	return s.cSurface.info.format
}

func (s *Surface) vkImageViewType() C.VkImageViewType {
	s.noCopy.check()
	return vk.IMAGE_VIEW_TYPE_2D
}

func (s *Surface) vkImage() C.VkImage {
	s.noCopy.check()
	return s.cSurface.vkImage
}

func (s *Surface) vkImageAspectFlags() C.VkImageAspectFlags {
	s.noCopy.check()
	return vk.IMAGE_ASPECT_COLOR_BIT
}

func (s *Surface) vkImageView() C.VkImageView {
	s.noCopy.check()
	return s.cSurface.vkImageView
}

func (s *Surface) vkSignalInfo(stage PipelineStage) C.VkSemaphoreSubmitInfo {
	s.noCopy.check()
	defer s.noCopy.close()
	return C.VkSemaphoreSubmitInfo{
		sType:     vk.STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
		semaphore: s.cSurface.releaseSemaphore,
		stageMask: C.VkPipelineStageFlags2(stage),
	}
}

func (s *Surface) vkWaitInfo(stage PipelineStage) C.VkSemaphoreSubmitInfo {
	s.noCopy.check()
	if s.waited {
		abort("Surface has already been waited on")
	}
	s.waited = true
	return C.VkSemaphoreSubmitInfo{
		sType:     vk.STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
		semaphore: s.cSurface.acquireSemaphore,
		stageMask: C.VkPipelineStageFlags2(stage),
	}
}
