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

package managed

import (
	"goarrg.com/rhi/vxr"
	"goarrg.com/rhi/vxr/internal/container"
	"goarrg.com/rhi/vxr/internal/util"
)

type descriptorArray[DescriptorType comparable] struct {
	noCopy             util.NoCopy
	set                *vxr.DescriptorSet
	binding            int
	index              int
	freeStack          container.Stack[int]
	managedDescriptors map[DescriptorType]int
}

func (d *descriptorArray[DescriptorType]) push(key DescriptorType, info vxr.DescriptorInfo) int {
	d.noCopy.Check()
	if i, found := d.managedDescriptors[key]; found {
		return i
	}
	var i int
	if d.freeStack.Empty() {
		if d.index >= d.set.MaxDescriptorCount(d.binding) {
			abort("Trying to push descriptor into a full set")
		}
		i = d.index
		d.index++
	} else {
		i = d.freeStack.Pop()
	}
	d.managedDescriptors[key] = i
	d.set.Bind(d.binding, i, info)
	return i
}

/*
Pop marks the descriptor index containing target as unused, which will become available for reuse
the next time f.Index() has the same value.
*/
func (d *descriptorArray[DescriptorType]) Pop(f *vxr.Frame, target DescriptorType) {
	d.noCopy.Check()
	i, found := d.managedDescriptors[target]
	if !found {
		return
	}
	f.QueueDestory(destroyFunc{
		func() {
			delete(d.managedDescriptors, target)
			d.freeStack.Push(i)
		},
	})
}

/*
DescriptorArrayBuffer manages inserting and removing Buffers from a descriptor array,
it is the user's responsibility to handle sync.
*/
type DescriptorArrayBuffer struct {
	descriptorArray[vxr.Buffer]
}

func NewDescriptorArrayBuffer(set *vxr.DescriptorSet, binding int) *DescriptorArrayBuffer {
	ret := DescriptorArrayBuffer{
		descriptorArray: descriptorArray[vxr.Buffer]{
			set:                set,
			binding:            binding,
			managedDescriptors: map[vxr.Buffer]int{},
		},
	}
	ret.noCopy.Init()
	return &ret
}

func (d *DescriptorArrayBuffer) Push(info vxr.DescriptorBufferInfo) int {
	return d.push(info.Buffer, info)
}

/*
DescriptorArrayImage manages inserting and removing Images from a descriptor array,
it is the user's responsibility to handle sync and layout changes
*/
type DescriptorArrayImage struct {
	descriptorArray[vxr.Image]
}

func NewDescriptorArrayImage(set *vxr.DescriptorSet, binding int) *DescriptorArrayImage {
	ret := DescriptorArrayImage{
		descriptorArray: descriptorArray[vxr.Image]{
			set:                set,
			binding:            binding,
			managedDescriptors: map[vxr.Image]int{},
		},
	}
	ret.noCopy.Init()
	return &ret
}

func (d *DescriptorArrayImage) Push(info vxr.DescriptorImageInfo) int {
	return d.push(info.Image, info)
}

/*
DescriptorArrayCombinedImageSampler manages inserting and removing CombinedImageSamplers from a descriptor array,
it is the user's responsibility to handle sync and layout changes
*/
type DescriptorArrayCombinedImageSampler struct {
	descriptorArray[vxr.Image]
}

func NewDescriptorArrayCombinedImageSampler(set *vxr.DescriptorSet, binding int) *DescriptorArrayCombinedImageSampler {
	ret := DescriptorArrayCombinedImageSampler{
		descriptorArray: descriptorArray[vxr.Image]{
			set:                set,
			binding:            binding,
			managedDescriptors: map[vxr.Image]int{},
		},
	}
	ret.noCopy.Init()
	return &ret
}

func (d *DescriptorArrayCombinedImageSampler) Push(info vxr.DescriptorCombinedImageSamplerInfo) int {
	return d.push(info.Image, info)
}
