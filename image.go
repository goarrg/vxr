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
	"strings"
	"unsafe"

	"goarrg.com/gmath"
	"goarrg.com/rhi/vxr/internal/vk"
)

type ImageViewType C.VkImageViewType

const (
	ImageViewType1D        ImageViewType = vk.IMAGE_VIEW_TYPE_1D
	ImageViewType1DArray   ImageViewType = vk.IMAGE_VIEW_TYPE_1D_ARRAY
	ImageViewType2D        ImageViewType = vk.IMAGE_VIEW_TYPE_2D
	ImageViewType2DArray   ImageViewType = vk.IMAGE_VIEW_TYPE_2D_ARRAY
	ImageViewType3D        ImageViewType = vk.IMAGE_VIEW_TYPE_3D
	ImageViewTypeCube      ImageViewType = vk.IMAGE_VIEW_TYPE_CUBE
	ImageViewTypeCubeArray ImageViewType = vk.IMAGE_VIEW_TYPE_CUBE_ARRAY
)

func (t ImageViewType) String() string {
	switch t {
	case ImageViewType1D:
		return "1D"
	case ImageViewType1DArray:
		return "1DArray"
	case ImageViewType2D:
		return "2D"
	case ImageViewType2DArray:
		return "2DArray"
	case ImageViewType3D:
		return "3D"
	case ImageViewTypeCube:
		return "Cube"
	case ImageViewTypeCubeArray:
		return "CubeArray"
	}
	abort("Unknown image type: %d", t)
	return ""
}

type ImageUsageFlags C.VkImageUsageFlags

const (
	ImageUsageTransferSrc            ImageUsageFlags = vk.IMAGE_USAGE_TRANSFER_SRC_BIT
	ImageUsageTransferDst            ImageUsageFlags = vk.IMAGE_USAGE_TRANSFER_DST_BIT
	ImageUsageSampled                ImageUsageFlags = vk.IMAGE_USAGE_SAMPLED_BIT
	ImageUsageStorage                ImageUsageFlags = vk.IMAGE_USAGE_STORAGE_BIT
	ImageUsageColorAttachment        ImageUsageFlags = vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT
	ImageUsageDepthStencilAttachment ImageUsageFlags = vk.IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT
	ImageUsageTransientAttachment    ImageUsageFlags = vk.IMAGE_USAGE_TRANSIENT_ATTACHMENT_BIT
)

func (u ImageUsageFlags) HasBits(want ImageUsageFlags) bool {
	return (u & want) == want
}

func (u ImageUsageFlags) FormatFeatureFlags() FormatFeatureFlags {
	var flags FormatFeatureFlags
	if u.HasBits(ImageUsageTransferSrc) {
		flags |= FORMAT_FEATURE_TRANSFER_SRC
	}
	if u.HasBits(ImageUsageTransferDst) {
		flags |= FORMAT_FEATURE_TRANSFER_DST
	}
	if u.HasBits(ImageUsageSampled) {
		flags |= FORMAT_FEATURE_SAMPLED_IMAGE
	}
	if u.HasBits(ImageUsageStorage) {
		flags |= FORMAT_FEATURE_STORAGE_IMAGE
	}
	if u.HasBits(ImageUsageColorAttachment) {
		flags |= FORMAT_FEATURE_COLOR_ATTACHMENT
	}
	if u.HasBits(ImageUsageDepthStencilAttachment) {
		flags |= FORMAT_FEATURE_DEPTH_STENCIL_ATTACHMENT
	}
	return flags
}

func (u ImageUsageFlags) String() string {
	str := ""
	if u.HasBits(ImageUsageTransferSrc) {
		str += "TransferSrc|"
	}
	if u.HasBits(ImageUsageTransferDst) {
		str += "TransferDst|"
	}
	if u.HasBits(ImageUsageSampled) {
		str += "Sampled|"
	}
	if u.HasBits(ImageUsageStorage) {
		str += "Storage|"
	}
	if u.HasBits(ImageUsageColorAttachment) {
		str += "ColorAttachment|"
	}
	if u.HasBits(ImageUsageDepthStencilAttachment) {
		str += "DepthStencilAttachment|"
	}
	if u.HasBits(ImageUsageTransientAttachment) {
		str += "TransientAttachment|"
	}
	return strings.TrimSuffix(str, "|")
}

type ImageLayout C.VkImageLayout

const (
	ImageLayoutUndefined         ImageLayout = vk.IMAGE_LAYOUT_UNDEFINED
	ImageLayoutGeneral           ImageLayout = vk.IMAGE_LAYOUT_GENERAL
	ImageLayoutTransferSrc       ImageLayout = vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL
	ImageLayoutTransferDst       ImageLayout = vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
	ImageLayoutReadOnlyOptimal   ImageLayout = vk.IMAGE_LAYOUT_READ_ONLY_OPTIMAL
	ImageLayoutAttachmentOptimal ImageLayout = vk.IMAGE_LAYOUT_ATTACHMENT_OPTIMAL
	ImageLayoutPresent           ImageLayout = vk.IMAGE_LAYOUT_PRESENT_SRC_KHR
)

type Sampler struct {
	noCopy   noCopy
	cSampler C.VkSampler
}

var _ interface {
	DescriptorInfo
	Destroyer
} = (*Sampler)(nil)

func (s *Sampler) isDescriptorInfo() {}

func (s *Sampler) Destroy() {
	s.noCopy.check()
	C.vxr_vk_destroySampler(instance.cInstance, s.cSampler)
	s.noCopy.close()
}

type SamplerFilter C.VkFilter

const (
	SamplerFilterNearest  SamplerFilter = vk.FILTER_NEAREST
	SamplerFilterLinear   SamplerFilter = vk.FILTER_LINEAR
	SamplerFilterCubicExt SamplerFilter = vk.FILTER_CUBIC_EXT
)

type SamplerMipMapMode C.VkSamplerMipmapMode

const (
	SamplerMipMapModeNearest SamplerMipMapMode = vk.SAMPLER_MIPMAP_MODE_NEAREST
	SamplerMipMapModeLinear  SamplerMipMapMode = vk.SAMPLER_MIPMAP_MODE_LINEAR
)

type SamplerAddressMode C.VkSamplerAddressMode

const (
	SamplerAddressModeRepeat              SamplerAddressMode = vk.SAMPLER_ADDRESS_MODE_REPEAT
	SamplerAddressModeMirroredRepeat      SamplerAddressMode = vk.SAMPLER_ADDRESS_MODE_MIRRORED_REPEAT
	SamplerAddressModeClampToEdge         SamplerAddressMode = vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE
	SamplerAddressModeClampToBorder       SamplerAddressMode = vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_BORDER
	SamplerAddressModeMirroredClampToEdge SamplerAddressMode = vk.SAMPLER_ADDRESS_MODE_MIRROR_CLAMP_TO_EDGE
)

type SamplerCreateInfo struct {
	MagFilter  SamplerFilter
	MinFilter  SamplerFilter
	MipMapMode SamplerMipMapMode
	BorderMode SamplerAddressMode
	Anisotropy float32
	// UnNormalizedCoordinates bool
}

func NewSampler(name string, info SamplerCreateInfo) *Sampler {
	sampler := &Sampler{}
	sampler.noCopy.init()

	cInfo := C.vxr_vk_samplerCreateInfo{
		magFilter:  C.VkFilter(info.MagFilter),
		minFilter:  C.VkFilter(info.MinFilter),
		mipmapMode: C.VkSamplerMipmapMode(info.MipMapMode),
		borderMode: C.VkSamplerAddressMode(info.BorderMode),
		anisotropy: C.float(info.Anisotropy),
	}

	/*
		if info.UnNormalizedCoordinates {
			cInfo.unnormalizedCoordinates = vk.TRUE
		}
	*/

	C.vxr_vk_createSampler(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
		cInfo, &sampler.cSampler)

	return sampler
}

type Image interface {
	usage() ImageUsageFlags

	vkFormat() C.VkFormat
	vkImage() C.VkImage
	vkImageViewType() C.VkImageViewType
	vkImageView() C.VkImageView
	vkImageAspectFlags() C.VkImageAspectFlags
}

type ColorImage interface {
	Image
	Format() Format
}

type ColorImageClearValue interface {
	isColorClearValue()
	vkClearValue() [16]byte
}

type ColorImageClearValueFloat struct {
	R, G, B, A float32
}

var _ ColorImageClearValue = ColorImageClearValueFloat{}

func (c ColorImageClearValueFloat) isColorClearValue() {
}

func (c ColorImageClearValueFloat) vkClearValue() [16]byte {
	return *(*[unsafe.Sizeof(ColorImageClearValueFloat{})]byte)(unsafe.Pointer(&c))
}

type ColorImageClearValueInt32 struct {
	R, G, B, A int32
}

var _ ColorImageClearValue = ColorImageClearValueInt32{}

func (c ColorImageClearValueInt32) isColorClearValue() {
}

func (c ColorImageClearValueInt32) vkClearValue() [16]byte {
	return *(*[unsafe.Sizeof(ColorImageClearValueInt32{})]byte)(unsafe.Pointer(&c))
}

type ColorImageClearValueUint32 struct {
	R, G, B, A uint32
}

var _ ColorImageClearValue = ColorImageClearValueUint32{}

func (c ColorImageClearValueUint32) isColorClearValue() {
}

func (c ColorImageClearValueUint32) vkClearValue() [16]byte {
	return *(*[unsafe.Sizeof(ColorImageClearValueUint32{})]byte)(unsafe.Pointer(&c))
}

type ColorImageClearValueUint64 struct {
	R, G uint64
}

var _ ColorImageClearValue = ColorImageClearValueUint64{}

func (c ColorImageClearValueUint64) isColorClearValue() {
}

func (c ColorImageClearValueUint64) vkClearValue() [16]byte {
	return *(*[unsafe.Sizeof(ColorImageClearValueUint64{})]byte)(unsafe.Pointer(&c))
}

type DepthStencilImage interface {
	Image
	Format() DepthStencilFormat
}

type image struct {
	noCopy     noCopy
	usageFlags ImageUsageFlags
	extent     gmath.Extent3i32

	cImage C.vxr_vk_image

	cImageViewType C.VkImageViewType
	cImageView     C.VkImageView
}

func (img *image) Extent() gmath.Extent3i32 {
	img.noCopy.check()
	return img.extent
}

func (img *image) Destroy() {
	img.noCopy.check()
	C.vxr_vk_destroyImage(instance.cInstance, img.cImage)
	C.vxr_vk_destroyImageView(instance.cInstance, img.cImageView)
	img.noCopy.close()
}

func (img *image) usage() ImageUsageFlags {
	img.noCopy.check()
	return img.usageFlags
}

func (img *image) vkImage() C.VkImage {
	img.noCopy.check()
	return img.cImage.vkImage
}

func (img *image) vkImageViewType() C.VkImageViewType {
	img.noCopy.check()
	return img.cImageViewType
}

func (img *image) vkImageView() C.VkImageView {
	img.noCopy.check()
	return img.cImageView
}

type DeviceColorImage struct {
	image
	format Format
}

var _ interface {
	ColorImage
	Destroyer
} = (*DeviceColorImage)(nil)

func (img *DeviceColorImage) Format() Format {
	img.noCopy.check()
	return img.format
}

func (img *DeviceColorImage) BufferSize() uint64 {
	img.noCopy.check()
	textelsPerBlock := uint64(img.format.BlockExtent().Volume())
	textels := uint64(img.extent.Volume())
	return (textels / textelsPerBlock) * uint64(img.format.BlockSize())
}

func (img *DeviceColorImage) vkFormat() C.VkFormat {
	img.noCopy.check()
	return C.VkFormat(img.format)
}

func (img *DeviceColorImage) vkImageAspectFlags() C.VkImageAspectFlags {
	return vk.IMAGE_ASPECT_COLOR_BIT
}

type DeviceDepthImage struct {
	image
	format DepthStencilFormat
}

var _ interface {
	DepthStencilImage
	Destroyer
} = (*DeviceDepthImage)(nil)

func (img *DeviceDepthImage) Format() DepthStencilFormat {
	img.noCopy.check()
	return img.format
}

func (img *DeviceDepthImage) vkFormat() C.VkFormat {
	img.noCopy.check()
	return C.VkFormat(img.format)
}

func (img *DeviceDepthImage) vkImageAspectFlags() C.VkImageAspectFlags {
	return vk.IMAGE_ASPECT_DEPTH_BIT
}

type ImageCreateInfo struct {
	Usage          ImageUsageFlags
	Flags          ImageCreateFlags
	ViewFlags      ImageViewCreateFlags
	Extent         gmath.Extent3i32
	NumMipLevels   int32
	NumArrayLayers int32
}

func newImage(name string, format C.VkFormat, aspect C.VkImageAspectFlags, info ImageCreateInfo) image {
	var vkImageType C.VkImageType
	var vkImage C.vxr_vk_image
	var vkImageViewType C.VkImageViewType
	var vkImageView C.VkImageView

	if min(min(info.Extent.X, info.Extent.Y), info.Extent.Z) < 1 {
		abort("Trying to create image with ImageCreateInfo.Extent [%+v], all values must be >= 1", info.Extent)
	}
	if min(info.NumMipLevels, info.NumArrayLayers) < 1 {
		abort("Trying to create image with ImageCreateInfo.NumMipLevels [%d] and ImageCreateInfo.NumArrayLayers [%d], both values must be >= 1",
			info.NumMipLevels, info.NumArrayLayers)
	}
	if info.NumArrayLayers > instance.deviceProperties.Limits.PerDesctiptor.MaxImageArrayLayers {
		abort("Trying to create image with ImageCreateInfo.NumArrayLayers [%d] which is larger than DeviceProperties.Limits.PerDesctiptor.MaxImageArrayLayers [%d]",
			info.NumArrayLayers, instance.deviceProperties.Limits.PerDesctiptor.MaxImageArrayLayers)
	}

	if info.Extent.Z > 1 {
		vkImageType = vk.IMAGE_TYPE_3D
	} else if info.Extent.Y > 1 {
		vkImageType = vk.IMAGE_TYPE_2D
	}

	switch vkImageType {
	case vk.IMAGE_TYPE_1D:
		if info.Flags.HasBits(IMAGE_CREATE_CUBE_COMPATIBLE) {
			abort("Trying to create a 1D image with ImageCreateInfo.Flags containing IMAGE_CREATE_CUBE_COMPATIBLE which is only valid for 2D images")
		}
		if info.Extent.X > instance.deviceProperties.Limits.PerDesctiptor.MaxImageDimension1D {
			abort("Trying to create 1D image with ImageCreateInfo.Extent [%+v] which is larger than DeviceProperties.Limits.PerDesctiptor.MaxImageDimension1D [%d]",
				info.Extent, instance.deviceProperties.Limits.PerDesctiptor.MaxImageDimension1D)
		}
	case vk.IMAGE_TYPE_2D:
		if info.Flags.HasBits(IMAGE_CREATE_CUBE_COMPATIBLE) {
			if max(info.Extent.X, info.Extent.Y) > instance.deviceProperties.Limits.PerDesctiptor.MaxImageDimensionCube {
				abort("Trying to create cube image with ImageCreateInfo.Extent [%+v] which is larger than DeviceProperties.Limits.PerDesctiptor.MaxImageDimensionCube [%d]",
					info.Extent, instance.deviceProperties.Limits.PerDesctiptor.MaxImageDimensionCube)
			}
			if info.Extent.X != info.Extent.Y {
				abort("Trying to create cube image with ImageCreateInfo.Extent [%+v], width and height are required to be equal",
					info.Extent)
			}
			if (info.NumArrayLayers % 6) != 0 {
				abort("Trying to create cube image with ImageCreateInfo.NumArrayLayers [%d] which is not the required multiple of 6",
					info.NumArrayLayers)
			}
		} else {
			if max(info.Extent.X, info.Extent.Y) > instance.deviceProperties.Limits.PerDesctiptor.MaxImageDimension2D {
				abort("Trying to create 2D image with ImageCreateInfo.Extent [%+v] which is larger than DeviceProperties.Limits.PerDesctiptor.MaxImageDimension2D [%d]",
					info.Extent, instance.deviceProperties.Limits.PerDesctiptor.MaxImageDimension2D)
			}
		}
	case vk.IMAGE_TYPE_3D:
		if info.Flags.HasBits(IMAGE_CREATE_CUBE_COMPATIBLE) {
			abort("Trying to create a 3D image with ImageCreateInfo.Flags containing IMAGE_CREATE_CUBE_COMPATIBLE which is only valid for 2D images")
		}
		if max(max(info.Extent.X, info.Extent.Y), info.Extent.Z) > instance.deviceProperties.Limits.PerDesctiptor.MaxImageDimension3D {
			abort("Trying to create 3D image with ImageCreateInfo.Extent [%+v] which is larger than DeviceProperties.Limits.PerDesctiptor.MaxImageDimension3D [%d]",
				info.Extent, instance.deviceProperties.Limits.PerDesctiptor.MaxImageDimension3D)
		}
		if info.NumArrayLayers != 1 {
			abort("Trying to create 3D image with ImageCreateInfo.NumArrayLayers [%d], a value != 1 is only valid for 1D and 2D images",
				info.NumArrayLayers)
		}
	default:
		abort("Unimplemented ImageType [%d]", vkImageType)
	}

	if info.Flags.HasBits(IMAGE_CREATE_CUBE_COMPATIBLE) {
		if info.NumArrayLayers > 6 {
			vkImageViewType = vk.IMAGE_VIEW_TYPE_CUBE_ARRAY
		} else {
			vkImageViewType = vk.IMAGE_VIEW_TYPE_CUBE
		}
	} else if info.NumArrayLayers > 1 {
		switch vkImageType {
		case vk.IMAGE_TYPE_1D:
			vkImageViewType = vk.IMAGE_VIEW_TYPE_1D_ARRAY
		case vk.IMAGE_TYPE_2D:
			vkImageViewType = vk.IMAGE_VIEW_TYPE_2D_ARRAY
		}
	} else {
		vkImageViewType = C.VkImageViewType(vkImageType)
	}

	C.vxr_vk_createImage(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
		C.vxr_vk_imageCreateInfo{
			flags:  C.VkImageCreateFlags(info.Flags),
			_type:  vkImageType,
			format: format,
			usage:  C.VkImageUsageFlags(info.Usage),
			extent: C.VkExtent3D{
				width:  C.uint32_t(info.Extent.X),
				height: C.uint32_t(info.Extent.Y),
				depth:  C.uint32_t(info.Extent.Z),
			},
			mipLevels:   C.uint32_t(info.NumMipLevels),
			arrayLayers: C.uint32_t(info.NumArrayLayers),
		}, &vkImage,
	)
	C.vxr_vk_createImageView(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
		C.vxr_vk_imageViewCreateInfo{
			flags:   C.VkImageViewCreateFlags(info.ViewFlags),
			vkImage: vkImage.vkImage,
			_type:   vkImageViewType,
			format:  format,
			_range: C.VkImageSubresourceRange{
				aspectMask:   aspect,
				baseMipLevel: C.uint32_t(0), levelCount: C.uint32_t(vk.REMAINING_MIP_LEVELS),
				baseArrayLayer: C.uint32_t(0), layerCount: C.uint32_t(vk.REMAINING_ARRAY_LAYERS),
			},
		}, &vkImageView)

	return image{
		usageFlags: info.Usage,
		extent:     info.Extent,

		cImage: vkImage,

		cImageViewType: vkImageViewType,
		cImageView:     vkImageView,
	}
}

func NewColorImageWithFormat(name string, format Format, info ImageCreateInfo) *DeviceColorImage {
	if !format.HasFeatures(info.Usage.FormatFeatureFlags()) {
		abort("Format [%s] does not have all the required feature flags [%s] for usage [%s]",
			format.String(), info.Usage.FormatFeatureFlags().String(), info.Usage.String())
	}
	{
		blockExtent := format.BlockExtent()
		if ((info.Extent.X % blockExtent.X) != 0) ||
			((info.Extent.Y % blockExtent.Y) != 0) ||
			((info.Extent.Z % blockExtent.Z) != 0) {
			abort("Format [%s] requires ImageCreateInfo.Extent %+v to be a multiple of Format.BlockExtent() %+v",
				format.String(), info.Extent, blockExtent)
		}
	}
	instance.logger.VPrintf("Creating color image with format [%s] and info: %+v", format.String(), info)
	name = "color_" + name
	img := &DeviceColorImage{format: format}
	img.image = newImage(name, C.VkFormat(format), img.vkImageAspectFlags(), info)
	img.noCopy.init()
	return img
}

func NewDepthImageWithFormat(name string, format DepthStencilFormat, info ImageCreateInfo) *DeviceDepthImage {
	if !format.HasFeatures(info.Usage.FormatFeatureFlags()) {
		abort("DepthStencilFormat [%s] does not have all the required feature flags [%s] for usage [%s]",
			format.String(), info.Usage.FormatFeatureFlags().String(), info.Usage.String())
	}
	instance.logger.VPrintf("Creating depth image with format [%s] and info: %+v", format.String(), info)
	name = "depth_" + name
	img := &DeviceDepthImage{format: format}
	img.image = newImage(name, C.VkFormat(format), img.vkImageAspectFlags(), info)
	img.noCopy.init()
	return img
}

func newDepthImageWithBits(name string, bits int, info ImageCreateInfo) *DeviceDepthImage {
	var formats []DepthStencilFormat

	switch bits {
	case 16:
		formats = []DepthStencilFormat{DEPTH_STENCIL_FORMAT_D16_UNORM, DEPTH_STENCIL_FORMAT_D16_UNORM_S8_UINT}
	case 24:
		formats = []DepthStencilFormat{DEPTH_STENCIL_FORMAT_X8_D24_UNORM_PACK32, DEPTH_STENCIL_FORMAT_D24_UNORM_S8_UINT}
	case 32:
		formats = []DepthStencilFormat{DEPTH_STENCIL_FORMAT_D32_SFLOAT, DEPTH_STENCIL_FORMAT_D32_SFLOAT_S8_UINT}
	default:
		abort("Invalid bit count for depth image: %d", bits)
	}

	for _, f := range formats {
		if f.HasFeatures(info.Usage.FormatFeatureFlags()) {
			return NewDepthImageWithFormat(name, f, info)
		}
	}

	return nil
}

func NewDepthImageWithAtLestBits(name string, bits int, info ImageCreateInfo) *DeviceDepthImage {
	if (!gmath.InRange(bits, 16, 32)) || ((bits % 8) != 0) {
		abort("Invalid bit count for depth image: %d", bits)
	}

	for queryBits := bits; queryBits <= 32; queryBits += 8 {
		img := newDepthImageWithBits(name, queryBits, info)
		if img != nil {
			return img
		}
	}

	abort("No depth format with at least %d bits that matches usage: %s", bits, info.Usage.String())
	return nil
}

func NewDepthImageWithAtMostBits(name string, bits int, info ImageCreateInfo) *DeviceDepthImage {
	if (!gmath.InRange(bits, 16, 32)) || ((bits % 8) != 0) {
		abort("Invalid bit count for depth image: %d", bits)
	}

	for queryBits := bits; queryBits >= 16; queryBits -= 8 {
		img := newDepthImageWithBits(name, queryBits, info)
		if img != nil {
			return img
		}
	}

	abort("No depth format with at most %d bits that matches usage: %s", bits, info.Usage.String())
	return nil
}
