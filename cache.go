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
	"slices"
	"unsafe"

	"goarrg.com/rhi/vxr/internal/container"
	"goarrg.com/rhi/vxr/internal/vk"
	"golang.org/x/exp/maps"
)

type descriptorSetLayoutCache struct {
	cache map[string]C.VkDescriptorSetLayout
}

func (c *descriptorSetLayoutCache) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	{
		err := mapRunFuncSorted(c.cache, func(k string, v C.VkDescriptorSetLayout) error {
			buff.WriteString(fmt.Sprintf("%q: %q,", k, toHex(v)))
			return nil
		})
		if err == nil {
			buff.Truncate(buff.Len() - 1)
		}
	}

	buff.WriteString("}")
	return buff.Bytes(), nil
}

func (c *descriptorSetLayoutCache) createOrRetrieveDescriptorSetLayout(descriptorSet *descriptorSetLayout, descriptorBindings []C.VkDescriptorSetLayoutBinding) {
	var ok bool
	descriptorSet.cDescriptorSetLayout, ok = c.cache[descriptorSet.id]
	if !ok {
		C.vxr_vk_shader_createDescriptorSetLayout(instance.cInstance, C.size_t(len(descriptorSet.name)), (*C.char)(unsafe.Pointer(unsafe.StringData(descriptorSet.name))),
			C.uint32_t(len(descriptorBindings)), unsafe.SliceData(descriptorBindings), &descriptorSet.cDescriptorSetLayout,
		)
		runtime.KeepAlive(descriptorBindings)
		c.cache[descriptorSet.id] = descriptorSet.cDescriptorSetLayout
	}
}

type pipelineLayoutCache struct {
	cache map[string]C.VkPipelineLayout
}

func (c *pipelineLayoutCache) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	{
		buff.WriteString("\"cache\": {")
		keys := maps.Keys(c.cache)
		slices.Sort(keys)
		if len(keys) > 0 {
			for _, k := range keys {
				buff.WriteString(fmt.Sprintf("%q: %q,", k, toHex(c.cache[k])))
			}
			buff.Truncate(buff.Len() - 1)
		}
		buff.WriteString("}")
	}

	buff.WriteString("}")
	return buff.Bytes(), nil
}

func (c *pipelineLayoutCache) createOrRetrievePipelineLayout(layout *PipelineLayout) {
	var ok bool
	layout.vkPipelinelayout, ok = c.cache[layout.id]
	if !ok {
		descriptorSetLayouts := []C.VkDescriptorSetLayout{}
		for _, set := range layout.descriptorSetLayouts {
			descriptorSetLayouts = append(descriptorSetLayouts,
				set.cDescriptorSetLayout,
			)
		}
		cInfo := C.vxr_vk_shader_pipelineLayoutCreateInfo{
			numDescriptorSetLayouts: C.uint32_t(len(descriptorSetLayouts)),
			descriptorSetLayouts:    unsafe.SliceData(descriptorSetLayouts),
		}
		if layout.pushConstantRange.size > 0 {
			pushConstantRanges := []C.VkPushConstantRange{layout.pushConstantRange}
			defer runtime.KeepAlive(pushConstantRanges)
			cInfo.numPushConstantRanges = 1
			cInfo.pushConstantRanges = unsafe.SliceData(pushConstantRanges)
		}
		C.vxr_vk_shader_createPipelineLayout(instance.cInstance, C.size_t(len(layout.name)), (*C.char)(unsafe.Pointer(unsafe.StringData(layout.name))),
			cInfo, &layout.vkPipelinelayout)
		runtime.KeepAlive(layout.id)
		runtime.KeepAlive(descriptorSetLayouts)
		c.cache[layout.id] = layout.vkPipelinelayout
	}
}

type descriptorPoolBank struct {
	name             string
	vkDescriptorPool C.VkDescriptorPool
	len              int32
	cap              int32
	freeSets         container.Stack[C.VkDescriptorSet]
}

func (b *descriptorPoolBank) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	{
		buff.WriteString(fmt.Sprintf("\"vkDescriptorPool\": %q,", toHex(b.vkDescriptorPool)))
		buff.WriteString(fmt.Sprintf("\"len\": %d,", b.len))
		buff.WriteString(fmt.Sprintf("\"cap\": %d,", b.cap))
	}

	{
		buff.WriteString("\"freeSets\": [")
		sets := b.freeSets.Data()
		if len(sets) > 0 {
			for _, s := range sets {
				buff.WriteString(fmt.Sprintf("%q,", toHex(s)))
			}
			buff.Truncate(buff.Len() - 1)
		}
		buff.WriteString("]")
	}

	buff.WriteString("}")
	return buff.Bytes(), nil
}

func (b *descriptorPoolBank) canAllocate() bool {
	return (!b.freeSets.Empty()) || (b.len < b.cap)
}

func (b *descriptorPoolBank) createOrRetrieveDescriptorSet(layout descriptorSetLayout) *DescriptorSet {
	descriptorSet := DescriptorSet{
		descriptorSetLayout: layout,
		bank:                b,
	}
	descriptorSet.noCopy.init()

	if !b.freeSets.Empty() {
		descriptorSet.cDescriptorSet = b.freeSets.Pop()
	} else {
		descriptorSetLayout := layout.cDescriptorSetLayout
		info := C.VkDescriptorSetAllocateInfo{
			sType:              vk.STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
			descriptorPool:     b.vkDescriptorPool,
			descriptorSetCount: 1,
			pSetLayouts:        &descriptorSetLayout,
		}
		name := fmt.Sprintf("%s_%s_set_%d", layout.name, b.name, b.len)
		C.vxr_vk_shader_createDescriptorSet(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
			info, &descriptorSet.cDescriptorSet)
		b.len++
	}
	return &descriptorSet
}

func (b *descriptorPoolBank) releaseDescriptorSet(set *DescriptorSet) {
	b.freeSets.Push(set.cDescriptorSet)
}

type descriptorPool struct {
	banks []*descriptorPoolBank
}

func (p *descriptorPool) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("[")

	if len(p.banks) > 0 {
		for _, b := range p.banks {
			buff.WriteString(jsonString(b))
			buff.WriteString(",")
		}
		buff.Truncate(buff.Len() - 1)
	}

	buff.WriteString("]")
	return buff.Bytes(), nil
}

func (p *descriptorPool) createOrRetrieveBank(layout descriptorSetLayout) *descriptorPoolBank {
	for _, b := range p.banks {
		if b.canAllocate() {
			return b
		}
	}

	bank := &descriptorPoolBank{name: fmt.Sprintf("bank_%d", len(p.banks)), cap: instance.config.descriptorPoolBankSize}
	poolSizes := make([]C.VkDescriptorPoolSize, 0, len(layout.bindings))
	for _, b := range layout.bindings {
		if b.descriptorCount > 0 {
			poolSizes = append(poolSizes, C.VkDescriptorPoolSize{
				_type:           b.descriptorType,
				descriptorCount: b.descriptorCount * C.uint32_t(instance.config.descriptorPoolBankSize),
			})
		}
	}
	info := C.VkDescriptorPoolCreateInfo{
		sType: vk.STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
		// flags: vk.DESCRIPTOR_POOL_CREATE_FREE_DESCRIPTOR_SET_BIT,

		maxSets:       C.uint32_t(instance.config.descriptorPoolBankSize),
		poolSizeCount: C.uint32_t(len(poolSizes)),
		pPoolSizes:    unsafe.SliceData(poolSizes),
	}
	name := fmt.Sprintf("%s_%s_maxSets_%d", layout.name, bank.name, info.maxSets)
	C.vxr_vk_shader_createDescriptorPool(instance.cInstance, C.size_t(len(name)), (*C.char)(unsafe.Pointer(unsafe.StringData(name))),
		info, &bank.vkDescriptorPool)
	runtime.KeepAlive(poolSizes)
	p.banks = append(p.banks, bank)
	return bank
}

func (p *descriptorPool) createOrRetrieveDescriptorSet(layout descriptorSetLayout) *DescriptorSet {
	b := p.createOrRetrieveBank(layout)
	return b.createOrRetrieveDescriptorSet(layout)
}

func (p *descriptorPool) destroy() {
	for _, b := range p.banks {
		C.vxr_vk_shader_destroyDescriptorPool(instance.cInstance, b.vkDescriptorPool)
	}
}

type descriptorSetCache struct {
	descriptorPools map[string]*descriptorPool
}

func (a *descriptorSetCache) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	{
		buff.WriteString("\"descriptorPools\": {")
		{
			err := mapRunFuncSorted(a.descriptorPools, func(k string, v *descriptorPool) error {
				buff.WriteString(fmt.Sprintf("%q: %s,", k, jsonString(v)))
				return nil
			})
			if err == nil {
				buff.Truncate(buff.Len() - 1)
			}
		}
		buff.WriteString("}")
	}

	buff.WriteString("}")
	return buff.Bytes(), nil
}

func (a *descriptorSetCache) createOrRetrieveDescriptorSet(layout descriptorSetLayout) *DescriptorSet {
	pool, ok := a.descriptorPools[layout.id]
	if !ok {
		pool = &descriptorPool{}
		a.descriptorPools[layout.id] = pool
	}
	return pool.createOrRetrieveDescriptorSet(layout)
}
