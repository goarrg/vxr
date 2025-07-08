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
	"bytes"
	"fmt"
	"runtime"
	"unsafe"

	"goarrg.com/rhi/vxr/internal/util"
	"goarrg.com/rhi/vxr/internal/vk"
)

type DescriptorType C.VkDescriptorType

const (
	DescriptorTypeUniformBuffer        DescriptorType = vk.DESCRIPTOR_TYPE_UNIFORM_BUFFER
	DescriptorTypeUniformTexelBuffer   DescriptorType = vk.DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER
	DescriptorTypeStorageBuffer        DescriptorType = vk.DESCRIPTOR_TYPE_STORAGE_BUFFER
	DescriptorTypeStorageTexelBuffer   DescriptorType = vk.DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER
	DescriptorTypeStorageImage         DescriptorType = vk.DESCRIPTOR_TYPE_STORAGE_IMAGE
	DescriptorTypeCombinedImageSampler DescriptorType = vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER
	DescriptorTypeSampledImage         DescriptorType = vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE
	DescriptorTypeSampler              DescriptorType = vk.DESCRIPTOR_TYPE_SAMPLER
)

func (t DescriptorType) String() string {
	switch t {
	case DescriptorTypeUniformBuffer:
		return "UniformBuffer"
	case DescriptorTypeUniformTexelBuffer:
		return "UniformTexelBuffer"

	case DescriptorTypeStorageBuffer:
		return "StorageBuffer"
	case DescriptorTypeStorageTexelBuffer:
		return "StorageTexelBuffer"

	case DescriptorTypeStorageImage:
		return "StorageImage"

	case DescriptorTypeCombinedImageSampler:
		return "CombinedImageSampler"

	case DescriptorTypeSampledImage:
		return "SampledImage"
	case DescriptorTypeSampler:
		return "Sampler"

	default:
		abort("Unknown DescriptorType: %d", t)
	}

	return ""
}

type DescriptorInfo interface {
	isDescriptorInfo()
}

type DescriptorBufferInfo struct {
	Buffer Buffer
	Offset uint64
}

func (d DescriptorBufferInfo) isDescriptorInfo() {}

func (d DescriptorBufferInfo) vkDescriptorBufferInfo() C.VkDescriptorBufferInfo {
	return C.VkDescriptorBufferInfo{
		buffer: d.Buffer.vkBuffer(),
		offset: C.VkDeviceSize(d.Offset),
		_range: vk.WHOLE_SIZE,
	}
}

type DescriptorImageInfo struct {
	Image  Image
	Layout ImageLayout
}

func (d DescriptorImageInfo) isDescriptorInfo() {}

func (d DescriptorImageInfo) vkDescriptorImageInfo() C.VkDescriptorImageInfo {
	return C.VkDescriptorImageInfo{
		imageView:   d.Image.vkImageView(),
		imageLayout: C.VkImageLayout(d.Layout),
	}
}

type DescriptorCombinedImageSamplerInfo struct {
	Sampler *Sampler
	Image   Image
	Layout  ImageLayout
}

func (d DescriptorCombinedImageSamplerInfo) isDescriptorInfo() {
}

func (d DescriptorCombinedImageSamplerInfo) vkDescriptorImageInfo() C.VkDescriptorImageInfo {
	d.Sampler.noCopy.Check()
	return C.VkDescriptorImageInfo{
		sampler:     d.Sampler.cSampler,
		imageView:   d.Image.vkImageView(),
		imageLayout: C.VkImageLayout(d.Layout),
	}
}

type DescriptorSet struct {
	noCopy              util.NoCopy
	descriptorSetLayout descriptorSetLayout
	cDescriptorSet      C.VkDescriptorSet
	bank                *descriptorPoolBank
}

func (s *DescriptorSet) Bind(bindingIndex, descriptorIndex int, descriptors ...DescriptorInfo) {
	s.noCopy.Check()
	if bindingIndex >= len(s.descriptorSetLayout.bindings) {
		abort("Trying to bind to descriptor index %d while layout's max is %d", bindingIndex, len(s.descriptorSetLayout.bindings)-1)
	}
	binding := s.descriptorSetLayout.bindings[bindingIndex]
	if len(descriptors) > int(binding.descriptorCount) {
		abort("Trying to bind %d descriptors while layout's max is %d", len(descriptors), binding.descriptorCount)
	}
	writeDescriptorSet := C.VkWriteDescriptorSet{
		sType:           vk.STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
		dstSet:          s.cDescriptorSet,
		dstBinding:      C.uint32_t(bindingIndex),
		dstArrayElement: C.uint32_t(descriptorIndex),
		descriptorCount: C.uint32_t(len(descriptors)),
		descriptorType:  binding.descriptorType,
	}
	switch descriptors[0].(type) {
	case DescriptorBufferInfo:
		s := make([]C.VkDescriptorBufferInfo, 0, len(descriptors))
		for _, d := range descriptors {
			s = append(s, d.(DescriptorBufferInfo).vkDescriptorBufferInfo())
		}
		defer runtime.KeepAlive(s)
		writeDescriptorSet.pBufferInfo = unsafe.SliceData(s)
	case *Sampler:
		s := make([]C.VkDescriptorImageInfo, 0, len(descriptors))
		for _, d := range descriptors {
			s = append(s, C.VkDescriptorImageInfo{sampler: d.(*Sampler).cSampler})
		}
		defer runtime.KeepAlive(s)
		writeDescriptorSet.pImageInfo = unsafe.SliceData(s)
	case DescriptorImageInfo:
		s := make([]C.VkDescriptorImageInfo, 0, len(descriptors))
		for _, d := range descriptors {
			s = append(s, d.(DescriptorImageInfo).vkDescriptorImageInfo())
		}
		defer runtime.KeepAlive(s)
		writeDescriptorSet.pImageInfo = unsafe.SliceData(s)
	case DescriptorCombinedImageSamplerInfo:
		s := make([]C.VkDescriptorImageInfo, 0, len(descriptors))
		for _, d := range descriptors {
			s = append(s, d.(DescriptorCombinedImageSamplerInfo).vkDescriptorImageInfo())
		}
		defer runtime.KeepAlive(s)
		writeDescriptorSet.pImageInfo = unsafe.SliceData(s)
	default:
		abort("Trying to bind unknown descriptor type: %#v", binding)
	}
	C.vxr_vk_shader_updateDescriptorSet(instance.cInstance, writeDescriptorSet)
}

func (s *DescriptorSet) Destroy() {
	if s == nil {
		return
	}
	s.noCopy.Check()
	s.bank.releaseDescriptorSet(s)
	s.noCopy.Close()
}

type descriptorSetBinding struct {
	shaderStage     C.VkShaderStageFlags
	descriptorType  C.VkDescriptorType
	descriptorCount C.uint32_t
}

func (b descriptorSetBinding) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	buff.WriteString(fmt.Sprintf("\"shaderStage\": %q,", ShaderStage(b.shaderStage).String()))
	buff.WriteString(fmt.Sprintf("\"descriptorType\": %q,", DescriptorType(b.descriptorType).String()))
	buff.WriteString(fmt.Sprintf("\"descriptorCount\": %d", b.descriptorCount))

	buff.WriteString("}")
	return buff.Bytes(), nil
}

type descriptorSetLayout struct {
	id                   string
	name                 string
	cDescriptorSetLayout C.VkDescriptorSetLayout
	bindings             []descriptorSetBinding
}

func (s *descriptorSetLayout) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	buff.WriteString(fmt.Sprintf("\"id\": %q,", s.id))
	buff.WriteString(fmt.Sprintf("\"name\": %q,", s.name))
	buff.WriteString(fmt.Sprintf("\"cDescriptorSetLayout\": %q,", toHex(s.cDescriptorSetLayout)))

	buff.WriteString("\"bindings\": [")
	if len(s.bindings) > 0 {
		for _, binding := range s.bindings {
			buff.WriteString(fmt.Sprintf("%s,", jsonString(binding)))
		}
		buff.Truncate(buff.Len() - 1)
	}
	buff.WriteString("]")

	buff.WriteString("}")
	return buff.Bytes(), nil
}
