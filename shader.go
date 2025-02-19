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

	#include <stdlib.h>

	#include "vxr/vxr.h"

	extern vxr_vk_shader_includeResult goShaderIncludeResolver(uintptr_t, char*, vxr_vk_shader_includeType, char*);
	extern void goShaderIncludeResultRelease(uintptr_t, vxr_vk_shader_includeResult);
*/
import "C"

import (
	"path"
	"runtime/cgo"
	"slices"
	"strings"
	"unsafe"

	"goarrg.com/asset"
	"goarrg.com/debug"
	"goarrg.com/rhi/vxr/internal/vk"
)

type shaderCompileState struct {
	fs    *asset.FileSystem
	files []*asset.File
}

func (s *shaderCompileState) destroy() {
	for _, f := range s.files {
		f.Close()
	}
}

//export goShaderIncludeResolver
func goShaderIncludeResolver(data C.uintptr_t, cTarget *C.char, t C.vxr_vk_shader_includeType, cRequester *C.char) C.vxr_vk_shader_includeResult {
	var target string

	switch t {
	case C.vxr_vk_shader_includeType_relative:
		target = path.Join(path.Dir(C.GoString(cRequester)), C.GoString(cTarget))
	case C.vxr_vk_shader_includeType_system:
		target = C.GoString(cTarget)
	}

	cTarget = C.CString(target)
	s := cgo.Handle(data).Value().(*shaderCompileState)
	f, err := s.fs.Open(target)
	if err != nil {
		abort("%s", err)
	}
	a := f.(*asset.File)
	s.files = append(s.files, a)
	return C.vxr_vk_shader_includeResult{
		nameSize:    C.size_t(len(target)),
		name:        cTarget,
		contentSize: C.size_t(a.Size()),
		content:     C.uintptr_t(a.Uintptr()),
	}
}

//export goShaderIncludeResultRelease
func goShaderIncludeResultRelease(data C.uintptr_t, result C.vxr_vk_shader_includeResult) {
	C.free(unsafe.Pointer(result.name))
}

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
type ShaderMetadata struct {
	SpecConstants []struct {
		Name    string
		Default uint32
	}
	DescriptorSetBindings map[string]ShaderBindingMetadata
}

func CompileShader(fs *asset.FileSystem, name string) (*Shader, *ShaderLayout, *ShaderMetadata) {
	instance.logger.VPrintf("Compiling shader: %q", name)

	f, err := fs.Open(name)
	if err != nil {
		abort("%s", err)
	}
	a := f.(*asset.File)
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	s := shaderCompileState{
		fs:    fs,
		files: []*asset.File{a},
	}
	defer s.destroy()
	h := cgo.NewHandle(&s)
	defer h.Delete()

	var cResult C.vxr_vk_shader_compileResult
	var cReflection C.vxr_vk_shader_reflectResult
	info := C.vxr_vk_shader_compileInfo{
		nameSize:    C.size_t(len(name)),
		name:        cName,
		contentSize: C.size_t(a.Size()),
		content:     C.uintptr_t(a.Uintptr()),

		includeResolver: C.vxr_vk_shaderIncludeResolver(C.goShaderIncludeResolver),
		resultReleaser:  C.vxr_vk_shaderIncludeResultReleaser(C.goShaderIncludeResultRelease),
		userdata:        C.uintptr_t(h),
	}
	C.vxr_vk_shader_compile(instance.cShaderCompiler, info, &cResult, &cReflection)
	defer C.vxr_vk_shader_destroyCompileResult(cResult)

	shader := Shader{
		ID: name,
	}
	layout := ShaderLayout{
		EntryPoints: map[string]ShaderEntryPointLayout{},
	}
	reflection := ShaderMetadata{
		DescriptorSetBindings: map[string]ShaderBindingMetadata{},
	}

	{
		var src C.vxr_vk_shader_spirv
		C.vxr_vk_shader_compileResult_getSPIRV(cResult, &src)
		shader.SPIRV = slices.Clone(unsafe.Slice((*uint32)(unsafe.Pointer(src.data)), src.len))
	}

	{
		var numEntryPoints C.size_t
		C.vxr_vk_shader_reflectResult_getEntryPoints(cReflection, &numEntryPoints, nil)
		entryPoints := make([]C.vxr_vk_shader_entryPoint, numEntryPoints)
		C.vxr_vk_shader_reflectResult_getEntryPoints(cReflection, &numEntryPoints, unsafe.SliceData(entryPoints))

		for i, e := range entryPoints {
			entryPointName := C.GoString(e.name)
			s := ShaderStage(e.stage)

			switch s {
			case ShaderStageCompute:
				var cLocalSize [3]C.vxr_vk_shader_reflectResult_constant
				C.vxr_vk_shader_reflectResult_getLocalSize(cReflection, &cLocalSize)
				layout.EntryPoints[entryPointName] = ShaderEntryPointComputeLayout{
					Name: entryPointName,
					LocalSize: [3]ShaderConstant{
						{
							uint32(cLocalSize[0].value),
							cLocalSize[0].isSpecConstant == vk.TRUE,
						},
						{
							uint32(cLocalSize[1].value),
							cLocalSize[1].isSpecConstant == vk.TRUE,
						},
						{
							uint32(cLocalSize[2].value),
							cLocalSize[2].isSpecConstant == vk.TRUE,
						},
					},
				}
			case ShaderStageVertex:
				layout.EntryPoints[entryPointName] = ShaderEntryPointVertexLayout{Name: entryPointName}
			case ShaderStageFragment:
				var cNumOutput C.uint32_t
				C.vxr_vk_shader_reflectResult_getNumOutputs(cReflection, C.size_t(i), &cNumOutput)
				layout.EntryPoints[entryPointName] = ShaderEntryPointFragmentLayout{
					Name:                      entryPointName,
					NumRenderColorAttachments: uint32(cNumOutput),
				}
			default:
				layout.EntryPoints[entryPointName] = ShaderEntryPointUnknownLayout{
					Name:  entryPointName,
					Stage: s,
				}
			}
		}
	}

	{
		type specConstant = struct {
			Name    string
			Default uint32
		}
		var numSpecConstants C.uint32_t
		C.vxr_vk_shader_reflectResult_getSpecConstants(cReflection, &numSpecConstants, nil)
		specConstants := make([]C.vxr_vk_shader_reflectResult_specConstant, numSpecConstants)
		C.vxr_vk_shader_reflectResult_getSpecConstants(cReflection, &numSpecConstants, unsafe.SliceData(specConstants))

		for _, c := range specConstants {
			reflection.SpecConstants = append(reflection.SpecConstants, specConstant{
				Name:    C.GoString(c.name),
				Default: uint32(c.value),
			})
		}
	}

	{
		var cRange C.VkPushConstantRange
		C.vxr_vk_shader_reflectResult_getPushConstantRange(cReflection, &cRange)
		layout.PushConstants.Offset = uint32(cRange.offset)
		layout.PushConstants.Size = uint32(cRange.size)
	}

	{
		type descriptorSetBindingLayout = struct {
			DescriptorType  DescriptorType
			DescriptorCount ShaderConstant
		}
		var numDescriptorSets C.uint32_t
		C.vxr_vk_shader_reflectResult_getDescriptorSetSizes(cReflection, &numDescriptorSets, nil)
		descriptorSetSizes := make([]C.uint32_t, numDescriptorSets)
		C.vxr_vk_shader_reflectResult_getDescriptorSetSizes(cReflection, &numDescriptorSets, unsafe.SliceData(descriptorSetSizes))
		layout.DescriptorSetLayouts = make([][]descriptorSetBindingLayout, numDescriptorSets)

		for set := C.uint32_t(0); set < numDescriptorSets; set++ {
			layout.DescriptorSetLayouts[set] = make([]descriptorSetBindingLayout, descriptorSetSizes[set])
			for binding := C.uint32_t(0); binding < descriptorSetSizes[set]; binding++ {
				var info C.vxr_vk_shader_reflectResult_descriptorSetBinding
				C.vxr_vk_shader_reflectResult_getDescriptorSetBinding(cReflection, set, binding, &info)
				layout.DescriptorSetLayouts[set][binding] = descriptorSetBindingLayout{
					DescriptorType: DescriptorType(info._type),
					DescriptorCount: ShaderConstant{
						Value:          uint32(info.count.value),
						IsSpecConstant: info.count.isSpecConstant == vk.TRUE,
					},
				}
				bindingInfo := ShaderBindingInfo{
					DescriptorType: DescriptorType(info._type),
					Set:            int(set), Binding: int(binding),
				}
				for alias := C.uint32_t(0); alias < info.numAliases; alias++ {
					switch info._type {
					case vk.DESCRIPTOR_TYPE_UNIFORM_BUFFER, vk.DESCRIPTOR_TYPE_STORAGE_BUFFER:
						{
							var metadata C.vxr_vk_shader_reflectResult_bufferMetadata
							C.vxr_vk_shader_reflectResult_getBufferMetadata(cReflection, set, binding, alias, &metadata)
							reflection.DescriptorSetBindings[C.GoString(metadata.name)] = ShaderBindingTypeBufferMetadata{
								ShaderBindingInfo:  bindingInfo,
								Size:               uint64(metadata.size),
								RuntimeArrayStride: uint64(metadata.runtimeArrayStride),
							}
						}

					case vk.DESCRIPTOR_TYPE_SAMPLER:
						{
							var metadata C.vxr_vk_shader_reflectResult_samplerMetadata
							C.vxr_vk_shader_reflectResult_getSamplerMetadata(cReflection, set, binding, alias, &metadata)
							reflection.DescriptorSetBindings[C.GoString(metadata.name)] = ShaderBindingTypeSamplerMetadata{
								ShaderBindingInfo: bindingInfo,
							}
						}

					case vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER, vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE, vk.DESCRIPTOR_TYPE_STORAGE_IMAGE:
						{
							var metadata C.vxr_vk_shader_reflectResult_imageMetadata
							C.vxr_vk_shader_reflectResult_getImageMetadata(cReflection, set, binding, alias, &metadata)
							reflection.DescriptorSetBindings[C.GoString(metadata.name)] = ShaderBindingTypeImageMetadata{
								ShaderBindingInfo: bindingInfo,
								ViewType:          ImageViewType(metadata.viewType),
							}
						}

					case vk.DESCRIPTOR_TYPE_MAX_ENUM:
						if info.count.value != 0 || info.count.isSpecConstant == vk.TRUE {
							abort("Unknown DescriptorType at set [%d] binding [%d]", set, binding)
						}

					default:
						abort("Descriptor type: %s is unimplemented", DescriptorType(info._type))
					}
				}
			}
		}
	}

	return &shader, &layout, &reflection
}

func (l *ShaderLayout) Validate() error {
	if (l.PushConstants.Offset + l.PushConstants.Size) > instance.deviceProperties.Limits.PerPipeline.MaxPushConstantsSize {
		return debug.Errorf("Shader's push constants Offset [%d] + Size [%d] is greater than Properties.Limits.PerPipeline.MaxPushConstantsSize [%d]",
			l.PushConstants.Offset, l.PushConstants.Size, instance.deviceProperties.Limits.PerPipeline.MaxPushConstantsSize)
	}

	return nil
}
