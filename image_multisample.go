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
	"math/bits"
	"strings"
	"unsafe"

	"goarrg.com/gmath"
	"goarrg.com/rhi/vxr/internal/vk"
)

type SampleCountFlags C.VkSampleCountFlags

const (
	SampleCount1  SampleCountFlags = vk.SAMPLE_COUNT_1_BIT
	SampleCount2  SampleCountFlags = vk.SAMPLE_COUNT_2_BIT
	SampleCount4  SampleCountFlags = vk.SAMPLE_COUNT_4_BIT
	SampleCount8  SampleCountFlags = vk.SAMPLE_COUNT_8_BIT
	SampleCount16 SampleCountFlags = vk.SAMPLE_COUNT_16_BIT
	SampleCount32 SampleCountFlags = vk.SAMPLE_COUNT_32_BIT
	SampleCount64 SampleCountFlags = vk.SAMPLE_COUNT_64_BIT
)

func (s SampleCountFlags) HasBits(want SampleCountFlags) bool {
	return (s & want) == want
}

func (s SampleCountFlags) String() string {
	str := ""
	if s.HasBits(SampleCount1) {
		str += "1|"
	}
	if s.HasBits(SampleCount2) {
		str += "2|"
	}
	if s.HasBits(SampleCount4) {
		str += "4|"
	}
	if s.HasBits(SampleCount8) {
		str += "8|"
	}
	if s.HasBits(SampleCount16) {
		str += "16|"
	}
	if s.HasBits(SampleCount32) {
		str += "32|"
	}
	if s.HasBits(SampleCount64) {
		str += "64|"
	}
	return strings.TrimSuffix(str, "|")
}

type imageMultiSampled struct {
	noCopy     noCopy
	usageFlags ImageUsageFlags
	samples    SampleCountFlags
	extent     gmath.Extent3i32

	cImage     C.vxr_vk_image
	cImageView C.VkImageView
}

func (img *imageMultiSampled) Destroy() {
	img.noCopy.check()
	C.vxr_vk_destroyImage(instance.cInstance, img.cImage)
	C.vxr_vk_destroyImageView(instance.cInstance, img.cImageView)
	img.noCopy.close()
}

func (img *imageMultiSampled) usage() ImageUsageFlags {
	img.noCopy.check()
	return img.usageFlags
}

func (img *imageMultiSampled) vkImage() C.VkImage {
	img.noCopy.check()
	return img.cImage.vkImage
}

func (img *imageMultiSampled) vkImageViewType() C.VkImageViewType {
	img.noCopy.check()
	return vk.IMAGE_VIEW_TYPE_2D
}

func (img *imageMultiSampled) vkImageView() C.VkImageView {
	img.noCopy.check()
	return img.cImageView
}

type DeviceColorImageMultiSampled struct {
	imageMultiSampled
	format Format
}

var _ interface {
	ColorImage
	Destroyer
} = (*DeviceColorImageMultiSampled)(nil)

func (img *DeviceColorImageMultiSampled) Destroy() {
	if img == nil {
		return
	}
	img.imageMultiSampled.Destroy()
}

func (img *DeviceColorImageMultiSampled) Extent() gmath.Extent3i32 {
	if img == nil {
		return gmath.Extent3i32{}
	}
	img.noCopy.check()
	return img.extent
}

func (img *DeviceColorImageMultiSampled) Format() Format {
	img.noCopy.check()
	return img.format
}

func (img *DeviceColorImageMultiSampled) sampleCount() SampleCountFlags {
	if img == nil {
		return 0
	}
	img.noCopy.check()
	return img.samples
}

func (img *DeviceColorImageMultiSampled) vkFormat() C.VkFormat {
	img.noCopy.check()
	return C.VkFormat(img.format)
}

func (img *DeviceColorImageMultiSampled) Aspect() ImageAspectFlags {
	return vk.IMAGE_ASPECT_COLOR_BIT
}

type DeviceDepthStencilImageMultiSampled struct {
	imageMultiSampled
	format DepthStencilFormat
	aspect ImageAspectFlags
}

var _ interface {
	DepthStencilImage
	Destroyer
} = (*DeviceDepthStencilImageMultiSampled)(nil)

func (img *DeviceDepthStencilImageMultiSampled) Destroy() {
	if img == nil {
		return
	}
	img.imageMultiSampled.Destroy()
}

func (img *DeviceDepthStencilImageMultiSampled) Extent() gmath.Extent3i32 {
	if img == nil {
		return gmath.Extent3i32{}
	}
	img.noCopy.check()
	return img.extent
}

func (img *DeviceDepthStencilImageMultiSampled) Format() DepthStencilFormat {
	img.noCopy.check()
	return img.format
}

func (img *DeviceDepthStencilImageMultiSampled) sampleCount() SampleCountFlags {
	if img == nil {
		return 0
	}
	img.noCopy.check()
	return img.samples
}

func (img *DeviceDepthStencilImageMultiSampled) vkFormat() C.VkFormat {
	img.noCopy.check()
	return C.VkFormat(img.format)
}

func (img *DeviceDepthStencilImageMultiSampled) Aspect() ImageAspectFlags {
	img.noCopy.check()
	return img.aspect
}

type ImageMultiSampledCreateInfo struct {
	ViewFlags ImageViewCreateFlags
	Usage     ImageUsageFlags
	Samples   SampleCountFlags
	Extent    gmath.Extent2i32
}

func newImageMultiSampled(name string, format C.VkFormat, aspect C.VkImageAspectFlags, info ImageMultiSampledCreateInfo) imageMultiSampled {
	var vkImage C.vxr_vk_image
	var vkImageView C.VkImageView

	if min(info.Extent.X, info.Extent.Y) < 1 {
		abort("Trying to create multisampled image with ImageCreateInfo.Extent [%+v], all values must be >= 1", info.Extent)
	}
	if max(info.Extent.X, info.Extent.Y) > instance.deviceProperties.Limits.PerDesctiptor.MaxImageDimension2D {
		abort("Trying to create multisampled image with ImageCreateInfo.Extent [%+v] which is larger than DeviceProperties.Limits.PerDesctiptor.MaxImageDimension2D [%d]",
			info.Extent, instance.deviceProperties.Limits.PerDesctiptor.MaxImageDimension2D)
	}
	if info.Samples < SampleCount2 || bits.OnesCount(uint(info.Samples)) > 1 {
		abort("Trying to create multisampled image with ImageCreateInfo.Samples [%#b], only one bit can be set and must be at least 2", info.Samples)
	}

	C.vxr_vk_createImageMultiSampled(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
		C.vxr_vk_imageMultiSampledCreateInfo{
			flags:   C.VkImageCreateFlags(0),
			format:  format,
			samples: C.VkSampleCountFlagBits(info.Samples),
			usage:   C.VkImageUsageFlags(info.Usage),
			extent: C.VkExtent2D{
				width:  C.uint32_t(info.Extent.X),
				height: C.uint32_t(info.Extent.Y),
			},
		}, &vkImage,
	)
	C.vxr_vk_createImageView(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
		C.vxr_vk_imageViewCreateInfo{
			flags:   C.VkImageViewCreateFlags(info.ViewFlags),
			vkImage: vkImage.vkImage,
			_type:   vk.IMAGE_VIEW_TYPE_2D,
			format:  format,
			_range: C.VkImageSubresourceRange{
				aspectMask:   aspect,
				baseMipLevel: C.uint32_t(0), levelCount: C.uint32_t(vk.REMAINING_MIP_LEVELS),
				baseArrayLayer: C.uint32_t(0), layerCount: C.uint32_t(vk.REMAINING_ARRAY_LAYERS),
			},
		}, &vkImageView)

	return imageMultiSampled{
		usageFlags: info.Usage,
		samples:    info.Samples,
		extent: gmath.Extent3i32{
			X: info.Extent.X,
			Y: info.Extent.Y,
			Z: 1,
		},

		cImage: vkImage,

		cImageView: vkImageView,
	}
}

func NewColorImageMultiSampled(name string, format Format, info ImageMultiSampledCreateInfo) *DeviceColorImageMultiSampled {
	if !format.HasFeatures(info.Usage.FormatFeatureFlags()) {
		abort("Format [%s] does not have all the required feature flags [%s] for usage [%s]",
			format.String(), info.Usage.FormatFeatureFlags().String(), info.Usage.String())
	}
	{
		blockExtent := format.BlockExtent()
		if ((info.Extent.X % blockExtent.X) != 0) ||
			((info.Extent.Y % blockExtent.Y) != 0) ||
			((1 % blockExtent.Z) != 0) {
			abort("Format [%s] requires ImageCreateInfo.Extent %+v to be a multiple of Format.BlockExtent() %+v",
				format.String(), info.Extent, blockExtent)
		}
	}
	instance.logger.VPrintf("Creating color image with format [%s] and info: %+v", format.String(), info)
	name = "color_" + name
	img := &DeviceColorImageMultiSampled{format: format}
	img.imageMultiSampled = newImageMultiSampled(name, C.VkFormat(format),
		C.VkImageAspectFlags(img.Aspect()), info)
	img.noCopy.init()
	return img
}

func NewDepthStencilImageMultiSampled(name string, format DepthStencilFormat, aspect ImageAspectFlags, info ImageMultiSampledCreateInfo) *DeviceDepthStencilImageMultiSampled {
	if aspect == 0 {
		aspect = format.ImageAspectFlags()
	}
	if !format.ImageAspectFlags().HasBits(aspect) {
		abort("DepthStencilFormat [%s] does not have aspect [%s]",
			format.String(), aspect.String())
	}
	if !format.HasFeatures(info.Usage.FormatFeatureFlags()) {
		abort("DepthStencilFormat [%s] does not have all the required feature flags [%s] for usage [%s]",
			format.String(), info.Usage.FormatFeatureFlags().String(), info.Usage.String())
	}
	instance.logger.VPrintf("Creating depth image with format [%s] and info: %+v", format.String(), info)
	if aspect.HasBits(vk.IMAGE_ASPECT_STENCIL_BIT) {
		name = "stencil_" + name
	}
	if aspect.HasBits(vk.IMAGE_ASPECT_DEPTH_BIT) {
		name = "depth_" + name
	}
	img := &DeviceDepthStencilImageMultiSampled{
		imageMultiSampled: newImageMultiSampled(name, C.VkFormat(format), C.VkImageAspectFlags(aspect), info),
		format:            format, aspect: aspect,
	}
	img.noCopy.init()
	return img
}
