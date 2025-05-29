//go:build !goarrg_vxr_disable_shadercompiler
// +build !goarrg_vxr_disable_shadercompiler

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
	#cgo CFLAGS: -fvisibility=hidden -Wno-dll-attribute-on-redeclaration

	#include <stdlib.h>

	#include "vxr/vxr.h"

	extern vxr_vk_shader_includeResult goShaderIncludeResolver(uintptr_t, char*, vxr_vk_shader_includeType, char*);
	extern void goShaderIncludeResultRelease(uintptr_t, vxr_vk_shader_includeResult);
*/
import "C"

import (
	"path"
	"runtime"
	"runtime/cgo"
	"slices"
	"unsafe"

	"goarrg.com/asset"
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

type ShaderCompilerOptions struct {
	API                 uint32
	Strip               bool
	OptimizePerformance bool
	OptimizeSize        bool
}

func InitShaderCompiler(options ShaderCompilerOptions) {
	if instance.cShaderCompiler != nil {
		C.vxr_vk_shader_destroyToolchain(instance.cShaderCompiler)
	}
	cOptions := C.vxr_vk_shader_toolchainOptions{api: C.uint32_t(options.API)}
	if options.Strip {
		cOptions.strip = vk.TRUE
	}
	if options.OptimizePerformance {
		cOptions.optimizePerformance = vk.TRUE
	}
	if options.OptimizeSize {
		cOptions.optimizeSize = vk.TRUE
	}
	instance.logger.IPrintf("vxr_vk_shader_initToolchain with %+v", options)
	C.vxr_vk_shader_initToolchain(cOptions, &instance.cShaderCompiler)
}

func DestroyShaderCompiler() {
	instance.logger.IPrintf("vxr_vk_shader_destroyToolchain")
	C.vxr_vk_shader_destroyToolchain(instance.cShaderCompiler)
	instance.cShaderCompiler = nil
}

type ShaderMacro struct {
	Name  string
	Value string
}

func CompileShader(fs *asset.FileSystem, name string, macros ...ShaderMacro) (*Shader, *ShaderLayout, *ShaderMetadata) {
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
	pinner := runtime.Pinner{}
	defer pinner.Unpin()
	cMacros := make([]C.vxr_vk_shader_macro, len(macros))
	pinner.Pin(unsafe.SliceData(cMacros))
	for i := range macros {
		m := unsafe.Pointer(unsafe.StringData(macros[i].Name))
		pinner.Pin(m)
		v := unsafe.Pointer(unsafe.StringData(macros[i].Value))
		pinner.Pin(v)

		cMacros[i] = C.vxr_vk_shader_macro{
			nameSize: C.size_t(len(macros[i].Name)),
			name:     (*C.char)(m),

			valueSize: C.size_t(len(macros[i].Value)),
			value:     (*C.char)(v),
		}
	}
	info := C.vxr_vk_shader_compileInfo{
		nameSize:    C.size_t(len(name)),
		name:        cName,
		contentSize: C.size_t(a.Size()),
		content:     C.uintptr_t(a.Uintptr()),

		numMacros: C.size_t(len(macros)),
		macros:    unsafe.SliceData(cMacros),

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
