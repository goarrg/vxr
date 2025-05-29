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

	"goarrg.com/debug"
	"goarrg.com/rhi/vxr/internal/vk"
)

type ShaderStage C.VkShaderStageFlagBits

const (
	ShaderStageVertex   ShaderStage = vk.SHADER_STAGE_VERTEX_BIT
	ShaderStageFragment ShaderStage = vk.SHADER_STAGE_FRAGMENT_BIT
	ShaderStageGraphics ShaderStage = vk.SHADER_STAGE_ALL_GRAPHICS
	ShaderStageCompute  ShaderStage = vk.SHADER_STAGE_COMPUTE_BIT
	ShaderStageAll      ShaderStage = vk.SHADER_STAGE_ALL
)

func (s ShaderStage) String() string {
	str := ""

	if hasBits(s, ShaderStageVertex) {
		str += "Vertex|"
	}
	if hasBits(s, ShaderStageFragment) {
		str += "Fragment|"
	}
	if hasBits(s, ShaderStageCompute) {
		str += "Compute|"
	}

	return strings.TrimSuffix(str, "|")
}

type ShaderConstant struct {
	Value          uint32
	IsSpecConstant bool
}
type ShaderEntryPointLayout interface {
	EntryPointName() string
	ShaderStage() ShaderStage
	isShaderEntryPointLayout()
}
type ShaderEntryPointComputeLayout struct {
	Name      string
	LocalSize [3]ShaderConstant
}

var _ ShaderEntryPointLayout = ShaderEntryPointComputeLayout{}

func (l ShaderEntryPointComputeLayout) EntryPointName() string  { return l.Name }
func (ShaderEntryPointComputeLayout) ShaderStage() ShaderStage  { return ShaderStageCompute }
func (ShaderEntryPointComputeLayout) isShaderEntryPointLayout() {}

type ShaderEntryPointVertexLayout struct{ Name string }

var _ ShaderEntryPointLayout = ShaderEntryPointVertexLayout{}

func (l ShaderEntryPointVertexLayout) EntryPointName() string  { return l.Name }
func (ShaderEntryPointVertexLayout) ShaderStage() ShaderStage  { return ShaderStageVertex }
func (ShaderEntryPointVertexLayout) isShaderEntryPointLayout() {}

type ShaderEntryPointFragmentLayout struct {
	Name                      string
	NumRenderColorAttachments uint32
}

var _ ShaderEntryPointLayout = ShaderEntryPointFragmentLayout{}

func (l ShaderEntryPointFragmentLayout) EntryPointName() string  { return l.Name }
func (ShaderEntryPointFragmentLayout) ShaderStage() ShaderStage  { return ShaderStageFragment }
func (ShaderEntryPointFragmentLayout) isShaderEntryPointLayout() {}

type ShaderEntryPointUnknownLayout struct {
	Name  string
	Stage ShaderStage
}

var _ ShaderEntryPointLayout = ShaderEntryPointUnknownLayout{}

func (l ShaderEntryPointUnknownLayout) EntryPointName() string   { return l.Name }
func (l ShaderEntryPointUnknownLayout) ShaderStage() ShaderStage { return l.Stage }
func (ShaderEntryPointUnknownLayout) isShaderEntryPointLayout()  {}

type ShaderBindingInfo struct {
	DescriptorType DescriptorType
	Set, Binding   int
}

func (i ShaderBindingInfo) Info() ShaderBindingInfo {
	return i
}

type ShaderBindingMetadata interface {
	Info() ShaderBindingInfo
	ValidateDescriptor(info DescriptorInfo) error
	isShaderBindingMetadata()
}

type ShaderBindingTypeBufferMetadata struct {
	ShaderBindingInfo
	Size               uint64
	RuntimeArrayStride uint64
}

var _ ShaderBindingMetadata = (*ShaderBindingTypeBufferMetadata)(nil)

func (b ShaderBindingTypeBufferMetadata) ValidateDescriptor(info DescriptorInfo) error {
	verifySize := func(size uint64) error {
		if b.RuntimeArrayStride > 0 {
			if size < b.Size {
				return debug.Errorf("Buffer size [%d] does not match shader definition [%d]", size, b.Size)
			}
			if ((size - b.Size) % b.RuntimeArrayStride) != 0 {
				return debug.Errorf("Buffer size [%d] does not match shader definition [%d] with runtime array stride [%d]", size, b.Size, b.RuntimeArrayStride)
			}
		} else if b.Size != size {
			return debug.Errorf("Buffer size [%d] does not match shader definition [%d]", size, b.Size)
		}
		return nil
	}
	d, ok := info.(DescriptorBufferInfo)
	if !ok {
		return debug.Errorf("Trying to validate unknown descriptor info: %#v", info)
	}
	switch b.DescriptorType {
	case vk.DESCRIPTOR_TYPE_UNIFORM_BUFFER:
		if d.Buffer.Usage().HasBits(BufferUsageUniformBuffer) {
			return verifySize(d.Buffer.Size())
		}
	case vk.DESCRIPTOR_TYPE_STORAGE_BUFFER:
		if d.Buffer.Usage().HasBits(BufferUsageStorageBuffer) {
			return verifySize(d.Buffer.Size())
		}
	default:
		return debug.Errorf("Trying to validate buffer as invalid/unimplemented descriptor type: %s",
			b.DescriptorType.String())
	}
	return debug.Errorf("Failed trying to validate buffer as [%s], buffer wasn't created with the proper usage flags, have flags: %s",
		b.DescriptorType.String(), d.Buffer.Usage().String())
}

func (b ShaderBindingTypeBufferMetadata) isShaderBindingMetadata() {
}

type ShaderBindingTypeSamplerMetadata struct{ ShaderBindingInfo }

var _ ShaderBindingMetadata = (*ShaderBindingTypeSamplerMetadata)(nil)

func (b ShaderBindingTypeSamplerMetadata) ValidateDescriptor(info DescriptorInfo) error {
	_, ok := info.(*Sampler)
	if !ok {
		return debug.Errorf("Trying to validate unknown descriptor info: %#v", info)
	}
	switch b.DescriptorType {
	case vk.DESCRIPTOR_TYPE_SAMPLER:
		return nil
	default:
		return debug.Errorf("Trying to validate sampler as invalid/unimplemented descriptor type: %s",
			b.DescriptorType.String())
	}
}

func (s ShaderBindingTypeSamplerMetadata) isShaderBindingMetadata() {
}

type ShaderBindingTypeImageMetadata struct {
	ShaderBindingInfo
	ViewType ImageViewType
}

var _ ShaderBindingMetadata = (*ShaderBindingTypeImageMetadata)(nil)

func (i ShaderBindingTypeImageMetadata) ValidateDescriptor(info DescriptorInfo) error {
	switch d := info.(type) {
	case DescriptorImageInfo:
		if i.ViewType != ImageViewType(d.Image.vkImageViewType()) {
			return debug.Errorf("Failed Trying to validate [%s] Image, non matching ImageViewType type: %s",
				ImageViewType(d.Image.vkImageViewType()).String(), i.ViewType.String())
		}
		switch i.DescriptorType {
		case vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE:
			if d.Image.usage().HasBits(ImageUsageSampled) {
				return nil
			}
		case vk.DESCRIPTOR_TYPE_STORAGE_IMAGE:
			if d.Image.usage().HasBits(ImageUsageStorage) {
				return nil
			}
		default:
			return debug.Errorf("Trying to validate Image as invalid/unimplemented descriptor type: %s", i.DescriptorType.String())
		}
		return debug.Errorf("Failed trying to validate Image as [%s], Image wasn't created with the proper usage flags, have flags: %s",
			i.DescriptorType.String(), d.Image.usage().String())
	case DescriptorCombinedImageSamplerInfo:
		switch i.DescriptorType {
		case vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
			if d.Image.usage().HasBits(ImageUsageSampled) {
				return nil
			}
		default:
			return debug.Errorf("Trying to validate CombinedImageSampler as invalid/unimplemented descriptor type: %s",
				i.DescriptorType.String())
		}
		return debug.Errorf("Failed trying to validate CombinedImageSampler as [%s], Image wasn't created with the proper usage flags, have flags: %s",
			i.DescriptorType.String(), d.Image.usage().String())
	default:
		return debug.Errorf("Trying to validate unknown descriptor info: %#v", info)
	}
}

func (i ShaderBindingTypeImageMetadata) isShaderBindingMetadata() {
}

type Shader struct {
	ID    string
	SPIRV []uint32
}
type ShaderLayout struct {
	EntryPoints   map[string]ShaderEntryPointLayout
	PushConstants struct {
		Offset uint32
		Size   uint32
	}
	DescriptorSetLayouts [][]struct {
		DescriptorType  DescriptorType
		DescriptorCount ShaderConstant
	}
}

func (l *ShaderLayout) Validate() error {
	if (l.PushConstants.Offset + l.PushConstants.Size) > instance.deviceProperties.Limits.PerPipeline.MaxPushConstantsSize {
		return debug.Errorf("Shader's push constants Offset [%d] + Size [%d] is greater than Properties.Limits.PerPipeline.MaxPushConstantsSize [%d]",
			l.PushConstants.Offset, l.PushConstants.Size, instance.deviceProperties.Limits.PerPipeline.MaxPushConstantsSize)
	}

	return nil
}

type ShaderMetadata struct {
	SpecConstants []struct {
		Name    string
		Default uint32
	}
	DescriptorSetBindings map[string]ShaderBindingMetadata
}
