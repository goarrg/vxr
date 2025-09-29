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
	"strings"

	"goarrg.com/debug"
	"goarrg.com/rhi/vxr/internal/vk"
)

type PipelineStage C.VkPipelineStageFlags2

const (
	PipelineStageNone PipelineStage = vk.PIPELINE_STAGE_2_NONE
	// VK_PIPELINE_STAGE_2_DRAW_INDIRECT_BIT is not exclusive to draw calls so drop the draw prefix
	PipelineStageIndirect       PipelineStage = vk.PIPELINE_STAGE_2_DRAW_INDIRECT_BIT
	PipelineStageCompute        PipelineStage = vk.PIPELINE_STAGE_2_COMPUTE_SHADER_BIT
	PipelineStageVertexInput    PipelineStage = vk.PIPELINE_STAGE_2_VERTEX_INPUT_BIT
	PipelineStageVertexShader   PipelineStage = vk.PIPELINE_STAGE_2_VERTEX_SHADER_BIT
	PipelineStageFragmentShader PipelineStage = vk.PIPELINE_STAGE_2_FRAGMENT_SHADER_BIT
	// these are likely not useful as rarely could you use one without the other
	// PipelineStageEarlyFragmentTests     PipelineStage = vk.PIPELINE_STAGE_2_EARLY_FRAGMENT_TESTS_BIT
	// PipelineStageLateFragmentTests      PipelineStage = vk.PIPELINE_STAGE_2_LATE_FRAGMENT_TESTS_BIT
	PipelineStageFragmentTests PipelineStage = vk.PIPELINE_STAGE_2_EARLY_FRAGMENT_TESTS_BIT | vk.PIPELINE_STAGE_2_LATE_FRAGMENT_TESTS_BIT
	// VK_PIPELINE_STAGE_2_COLOR_ATTACHMENT_OUTPUT_BIT actually also includes depth resolve so drop the color label
	// might as well also add late fragment so it applies to any attachment write
	PipelineStageRenderAttachmentWrite PipelineStage = vk.PIPELINE_STAGE_2_COLOR_ATTACHMENT_OUTPUT_BIT | vk.PIPELINE_STAGE_2_LATE_FRAGMENT_TESTS_BIT
	PipelineStageGraphics              PipelineStage = vk.PIPELINE_STAGE_2_ALL_GRAPHICS_BIT
	PipelineStageTransfer              PipelineStage = vk.PIPELINE_STAGE_2_ALL_TRANSFER_BIT
	PipelineStageAll                   PipelineStage = vk.PIPELINE_STAGE_2_ALL_COMMANDS_BIT
)

type PipelineLayout struct {
	id               string
	name             string
	vkPipelinelayout C.VkPipelineLayout

	pushConstantRange    C.VkPushConstantRange
	descriptorSetLayouts []descriptorSetLayout
}

type PipelineLayoutCreateInfo struct {
	ShaderLayout  *ShaderLayout
	ShaderStage   ShaderStage
	SpecConstants []uint32
}

func NewPipelineLayout(infos ...PipelineLayoutCreateInfo) *PipelineLayout {
	var layout PipelineLayout

	for i, stageInfo := range infos {
		{
			if err := stageInfo.ShaderLayout.Validate(); err != nil {
				abort("ShaderLayout [%d] is invalid: %v\n%#v", i, err, stageInfo.ShaderLayout)
			}
		}

		{
			if stageInfo.ShaderLayout.PushConstants.Size > 0 {
				newRange := C.VkPushConstantRange{
					offset: C.uint32_t(stageInfo.ShaderLayout.PushConstants.Offset),
					size:   C.uint32_t(stageInfo.ShaderLayout.PushConstants.Size),
				}
				if layout.pushConstantRange.stageFlags == 0 {
					newRange.stageFlags = C.VkShaderStageFlags(stageInfo.ShaderStage)
					layout.pushConstantRange = newRange
				} else if newRange.stageFlags = layout.pushConstantRange.stageFlags; newRange == layout.pushConstantRange {
					layout.pushConstantRange.stageFlags |= C.VkShaderStageFlags(stageInfo.ShaderStage)
				} else {
					abort("Failed creating PipelineLayout: push constants must be either empty or consistent between all stages with non empty push constants")
				}
			}
		}

		{
			layout.descriptorSetLayouts = growSlice(layout.descriptorSetLayouts, len(stageInfo.ShaderLayout.DescriptorSetLayouts))
			for set := C.uint32_t(0); set < C.uint32_t(len(stageInfo.ShaderLayout.DescriptorSetLayouts)); set++ {
				layout.descriptorSetLayouts[set].bindings = growSlice(layout.descriptorSetLayouts[set].bindings,
					len(stageInfo.ShaderLayout.DescriptorSetLayouts[set]))
				for binding := C.uint32_t(0); binding < C.uint32_t(len(stageInfo.ShaderLayout.DescriptorSetLayouts[set])); binding++ {
					shaderBindingInfo := stageInfo.ShaderLayout.DescriptorSetLayouts[set][binding]
					currentBindingInfo := layout.descriptorSetLayouts[set].bindings[binding]
					currentBindingInfo.shaderStage |= C.VkShaderStageFlags(stageInfo.ShaderStage)
					newBindingInfo := descriptorSetBinding{
						shaderStage:     currentBindingInfo.shaderStage,
						descriptorType:  C.VkDescriptorType(shaderBindingInfo.DescriptorType),
						descriptorCount: C.uint32_t(shaderBindingInfo.DescriptorCount.Value),
					}
					if shaderBindingInfo.DescriptorCount.IsSpecConstant {
						newBindingInfo.descriptorCount = C.uint32_t(stageInfo.SpecConstants[shaderBindingInfo.DescriptorCount.Value])
					}
					if newBindingInfo.descriptorCount == 0 {
						continue
					}
					if currentBindingInfo.descriptorCount > 0 && currentBindingInfo != newBindingInfo {
						abort("Failed to create PipelineLayout: set[%d] binding[%d] have inconsistent metadata: %#v and %#v",
							set, binding, currentBindingInfo, newBindingInfo,
						)
					}
					layout.descriptorSetLayouts[set].bindings[binding] = newBindingInfo
				}
			}
		}
	}

	{
		if layout.pushConstantRange.stageFlags == 0 {
			layout.id = genID(layout.pushConstantRange.stageFlags, layout.pushConstantRange.offset, layout.pushConstantRange.size)
			layout.name = "[\"\",0,0]"
		} else {
			layout.id = genID(layout.pushConstantRange.stageFlags, layout.pushConstantRange.offset, layout.pushConstantRange.size)
			layout.name = fmt.Sprintf("[%s,%d,%d]",
				ShaderStage(layout.pushConstantRange.stageFlags).String(),
				layout.pushConstantRange.offset,
				layout.pushConstantRange.size,
			)
		}
		for i := range layout.descriptorSetLayouts {
			set := &layout.descriptorSetLayouts[i]
			if len(set.bindings) > 0 {
				cDescriptorSetBindings := make([]C.VkDescriptorSetLayoutBinding, 0, len(set.bindings))
				for j, binding := range set.bindings {
					if binding.descriptorCount == 0 {
						set.id += fmt.Sprintf("null,")
						set.name += fmt.Sprintf("null,")
					} else {
						set.id += fmt.Sprintf("%s:%s:%d,",
							toHex(binding.shaderStage), toHex(binding.descriptorType), binding.descriptorCount)
						set.name += fmt.Sprintf("%s:%s:%d,",
							ShaderStage(binding.shaderStage).String(), DescriptorType(binding.descriptorType).String(), binding.descriptorCount)
						cDescriptorSetBindings = append(cDescriptorSetBindings, C.VkDescriptorSetLayoutBinding{
							binding:         C.uint32_t(j),
							descriptorType:  binding.descriptorType,
							descriptorCount: binding.descriptorCount,
							stageFlags:      binding.shaderStage,
						})
					}
				}
				set.id = fmt.Sprintf("[%s]", strings.TrimSuffix(set.id, ","))
				set.name = fmt.Sprintf("[%s]", strings.TrimSuffix(set.name, ","))
				layout.id += fmt.Sprintf("%s", set.id)
				layout.name += fmt.Sprintf("%s", set.name)
				instance.descriptorSetLayoutCache.createOrRetrieveDescriptorSetLayout(set, cDescriptorSetBindings)
			} else {
				set.id = "[null]"
				set.name = "[null]"
				layout.id += fmt.Sprintf("[null]")
				layout.name += fmt.Sprintf("[null]")
				instance.descriptorSetLayoutCache.createOrRetrieveDescriptorSetLayout(set, nil)
			}
		}
		instance.pipelineLayoutCache.createOrRetrievePipelineLayout(&layout)
	}

	return &layout
}

func (l *PipelineLayout) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	buff.WriteString(fmt.Sprintf("\"id\": %q,", l.id))
	buff.WriteString(fmt.Sprintf("\"name\": %q,", l.name))
	buff.WriteString(fmt.Sprintf("\"vkPipelinelayout\": %q,", toHex(l.vkPipelinelayout)))

	buff.WriteString("\"pushConstantRange\": {")
	buff.WriteString(fmt.Sprintf("\"stage\": %q,", ShaderStage(l.pushConstantRange.stageFlags).String()))
	buff.WriteString(fmt.Sprintf("\"offset\": %d,", l.pushConstantRange.offset))
	buff.WriteString(fmt.Sprintf("\"size\": %d", l.pushConstantRange.size))
	buff.WriteString("},")

	buff.WriteString("\"descriptorSetLayout\": [")
	if len(l.descriptorSetLayouts) > 0 {
		for _, layout := range l.descriptorSetLayouts {
			buff.WriteString(fmt.Sprintf("%s,", jsonString(&layout)))
		}
		buff.Truncate(buff.Len() - 1)
	}
	buff.WriteString("]")

	buff.WriteString("}")
	return buff.Bytes(), nil
}

func (l *PipelineLayout) NewDescriptorSet(set int) *DescriptorSet {
	return instance.descriptorSetCache.createOrRetrieveDescriptorSet(l.descriptorSetLayouts[set])
}

func (l *PipelineLayout) cmdValidate(pushConstants []byte, descriptorSets []*DescriptorSet) error {
	if len(pushConstants) != int(l.pushConstantRange.size) {
		return debug.Errorf("Pushconstants size mismatch between given data and pipeline layout: expecting: %d bytes given: %d bytes",
			l.pushConstantRange.size, len(pushConstants))
	}
	if len(descriptorSets) != len(l.descriptorSetLayouts) {
		return debug.Errorf("DescriptorSet count mismatch between given sets and pipeline layout: expecting %d sets given %d sets",
			len(l.descriptorSetLayouts), len(descriptorSets))
	}
	return nil
}
