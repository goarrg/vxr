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

#pragma once

#include <stddef.h>
#include <stdint.h>

#ifdef NDEBUG
#define VXR_DEBUG 0
#else
#define VXR_DEBUG 1
#endif

#define VK_MAKE_API_VERSION(variant, major, minor, patch) \
	((((uint32_t)(variant)) << 29U) | (((uint32_t)(major)) << 22U) | (((uint32_t)(minor)) << 12U) | ((uint32_t)(patch)))

#define VXR_VK_MIN_API VK_MAKE_API_VERSION(0, 1, 3, 0)
#define VXR_VK_MAX_API VK_MAKE_API_VERSION(0, 1, 4, 0)

#define VXR_FN __attribute__((nothrow))
#define VXR_HANDLE(object) typedef struct object##_t* object

#ifndef VXR_GENERATOR
#ifdef __cplusplus
extern "C" {
#endif

// prevents including vulkan.h and therefore windows.h
// this block must be before anything else that includes vulkan
#define VULKAN_H_  // NOLINT(readability-identifier-naming)
#define VK_NO_PROTOTYPES
// #define VK_USE_64_BIT_PTR_DEFINES 0
#include <vulkan/vulkan_core.h>	 // IWYU pragma: export

typedef void (*vxr_loggerCallback)(size_t, char*);

VXR_HANDLE(vxr_vk_instance);
VXR_HANDLE(vxr_vk_device_selector);
VXR_HANDLE(vxr_vk_device_allocation);
VXR_HANDLE(vxr_vk_buffer);

VXR_HANDLE(vxr_vk_shader_toolchain);
VXR_HANDLE(vxr_vk_shader_compileResult);
VXR_HANDLE(vxr_vk_shader_reflectResult);

VXR_HANDLE(vxr_vk_graphics_frame);

typedef struct {
	float minPointSize;
	float maxPointSize;

	float minLineWidth;
	float maxLineWidth;

	struct {
		uint64_t maxAllocationSize;
		uint32_t maxMemoryAllocationCount;
		uint32_t maxSamplerAllocationCount;
	} global;

	struct {
		int32_t maxImageDimension1D;
		int32_t maxImageDimension2D;
		int32_t maxImageDimension3D;
		int32_t maxImageDimensionCube;
		int32_t maxImageArrayLayers;
		float maxSamplerAnisotropy;
		uint32_t maxUBOSize;
		uint32_t maxSBOSize;
	} perDescriptor;

	struct {
		uint32_t maxSamplerCount;
		uint32_t maxSampledImageCount;
		uint32_t maxCombinedImageSamplerCount;
		uint32_t maxStorageImageCount;

		uint32_t maxUBOCount;
		uint32_t maxSBOCount;
		uint32_t maxResourceCount;
	} perStage;

	struct {
		uint32_t maxSamplerCount;
		uint32_t maxSampledImageCount;
		uint32_t maxCombinedImageSamplerCount;
		uint32_t maxStorageImageCount;

		uint32_t maxUBOCount;
		uint32_t maxSBOCount;

		uint32_t maxBoundDescriptorSets;
		uint32_t maxPushConstantsSize;
	} perPipeline;

	struct {
		VkExtent3D maxDispatchSize;
		VkExtent3D maxLocalSize;
		uint32_t minSubgroupSize;
		uint32_t maxSubgroupSize;
		struct {
			uint32_t maxInvocations;
			uint32_t maxSubgroupCount;
		} workgroup;
	} compute;
} vxr_vk_device_limits;

typedef struct {
	uint8_t uuid[VK_UUID_SIZE];
	uint32_t vendorID;
	uint32_t deviceID;
	uint32_t driverVersion;
	uint32_t api;

	struct {
		uint32_t subgroupSize;
	} compute;

	vxr_vk_device_limits limits;
} vxr_vk_device_properties;

typedef struct {
	VkDeviceSize size;
	VkBufferUsageFlags usage;
} vxr_vk_bufferCreateInfo;

typedef struct {
	vxr_vk_device_allocation allocation;
	VkBuffer vkBuffer;
	void* ptr;
} vxr_vk_hostBuffer;

typedef struct {
	vxr_vk_device_allocation allocation;
	VkBuffer vkBuffer;
} vxr_vk_deviceBuffer;

typedef struct {
	VkImageCreateFlags flags;
	VkImageType type;
	VkFormat format;
	VkExtent3D extent;
	uint32_t mipLevels;
	uint32_t arrayLayers;
	VkImageUsageFlags usage;
} vxr_vk_imageCreateInfo;

typedef struct {
	VkImageViewCreateFlags flags;
	VkImage vkImage;
	VkImageViewType type;
	VkFormat format;
	VkImageSubresourceRange range;
} vxr_vk_imageViewCreateInfo;

typedef struct {
	VkFilter magFilter;
	VkFilter minFilter;
	VkSamplerMipmapMode mipmapMode;
	VkSamplerAddressMode borderMode;
	float anisotropy;
	VkBool32 unnormalizedCoordinates;
} vxr_vk_samplerCreateInfo;

typedef struct {
	vxr_vk_device_allocation allocation;
	VkImage vkImage;
} vxr_vk_image;

typedef struct {
	VkFormat format;
	VkExtent2D extent;
	uint32_t numImages;
} vxr_vk_surfaceInfo;

typedef struct {
	vxr_vk_surfaceInfo info;
	VkImage vkImage;
	VkImageView vkImageView;
	VkSemaphore acquireSemaphore;
	VkSemaphore releaseSemaphore;
} vxr_vk_surface;

typedef struct {
	uint32_t api;
	VkBool32 strip;
	VkBool32 optimizePerformance;
	VkBool32 optimizeSize;
} vxr_vk_shader_toolchainOptions;

typedef enum {
	vxr_vk_shader_includeType_relative,
	vxr_vk_shader_includeType_system,
} vxr_vk_shader_includeType;

typedef struct {
	size_t nameSize;
	const char* name;
	size_t contentSize;
	uintptr_t content;
	uintptr_t userdata;
} vxr_vk_shader_includeResult;

typedef vxr_vk_shader_includeResult (*vxr_vk_shaderIncludeResolver)(uintptr_t, char*, vxr_vk_shader_includeType, char*);
typedef void (*vxr_vk_shaderIncludeResultReleaser)(uintptr_t, vxr_vk_shader_includeResult);

typedef struct {
	size_t nameSize;
	const char* name;
	size_t valueSize;
	const char* value;
} vxr_vk_shader_macro;

typedef struct {
	size_t nameSize;
	const char* name;
	size_t contentSize;
	uintptr_t content;

	size_t numMacros;
	vxr_vk_shader_macro* macros;

	vxr_vk_shaderIncludeResolver includeResolver;
	vxr_vk_shaderIncludeResultReleaser resultReleaser;
	uintptr_t userdata;
} vxr_vk_shader_compileInfo;

typedef struct {
	size_t nameSize;
	const char* name;
	VkShaderStageFlagBits stage;
} vxr_vk_shader_entryPoint;

typedef struct {
	size_t len;
	const uint32_t* data;
} vxr_vk_shader_spirv;

typedef struct {
	const char* name;
	uint32_t value;
} vxr_vk_shader_reflectResult_specConstant;

typedef struct {
	uint32_t value;
	VkBool32 isSpecConstant;
} vxr_vk_shader_reflectResult_constant;

typedef struct {
	VkDescriptorType type;
	vxr_vk_shader_reflectResult_constant count;
	uint32_t numAliases;
} vxr_vk_shader_reflectResult_descriptorSetBinding;

typedef struct {
	const char* name;
	VkDeviceSize size;
	VkDeviceSize runtimeArrayStride;
} vxr_vk_shader_reflectResult_bufferMetadata;

typedef struct {
	const char* name;
	VkImageViewType viewType;
} vxr_vk_shader_reflectResult_imageMetadata;

typedef struct {
	const char* name;
} vxr_vk_shader_reflectResult_samplerMetadata;

typedef struct {
	uint32_t numPushConstantRanges;
	VkPushConstantRange* pushConstantRanges;

	uint32_t numDescriptorSetLayouts;
	const VkDescriptorSetLayout* descriptorSetLayouts;
} vxr_vk_shader_pipelineLayoutCreateInfo;

typedef struct {
	VkPipelineShaderStageCreateFlags stageFlags;

	VkPipelineLayout layout;
	size_t entryPointSize;
	const char* entryPoint;
	vxr_vk_shader_spirv spirv;

	uint32_t requiredSubgroupSize;

	uint32_t numSpecConstants;
	const uint32_t* specConstants;
} vxr_vk_compute_shaderPipelineCreateInfo;
typedef struct {
	VkPipelineLayout layout;
	VkPipeline pipeline;

	VkPushConstantRange pushConstantRange;
	void* pushConstantData;

	uint32_t numDescriptorSets;
	VkDescriptorSet* descriptorSets;

	VkExtent3D groupCount;
} vxr_vk_compute_dispatchInfo;
typedef struct {
	VkPipelineLayout layout;
	VkPipeline pipeline;

	VkPushConstantRange pushConstantRange;
	void* pushConstantData;

	uint32_t numDescriptorSets;
	VkDescriptorSet* descriptorSets;

	VkBuffer buffer;
	VkDeviceSize offset;
} vxr_vk_compute_dispatchIndirectInfo;

typedef struct {
	VkPipelineLayout layout;
	size_t entryPointSize;
	const char* entryPoint;
	VkShaderStageFlagBits stage;
	vxr_vk_shader_spirv spirv;

	uint32_t numSpecConstants;
	const uint32_t* specConstants;
} vxr_vk_graphics_shaderPipelineCreateInfo;

typedef struct {
	uint32_t numColorAttachments;
	const VkFormat* colorAttachmentFormats;
	VkFormat depthFormat;
	VkFormat stencilFormat;
} vxr_vk_graphics_fragmentOutputPipelineCreateInfo;

typedef struct {
	VkPipelineLayout layout;
	VkPipeline pipeline;
	VkPrimitiveTopology topology;

	VkCullModeFlags cullMode;
	VkFrontFace frontFace;

	VkBool32 depthTestEnable;
	VkBool32 depthWriteEnable;
	VkCompareOp depthCompareOp;

	VkBool32 stencilTestEnable;
	VkStencilOpState stencilTestFrontFace;
	VkStencilOpState stencilTestBackFace;

	VkPushConstantRange pushConstantRange;
	void* pushConstantData;

	uint32_t numDescriptorSets;
	VkDescriptorSet* descriptorSets;
} vxr_vk_graphics_drawParameters;
typedef struct {
	vxr_vk_graphics_drawParameters parameters;

	uint32_t vertexCount;
	uint32_t instanceCount;
} vxr_vk_graphics_drawInfo;
typedef struct {
	VkBuffer vkBuffer;
	VkDeviceSize offset;
	uint32_t drawCount;
} vxr_vk_graphics_drawIndirectBufferInfo;
typedef struct {
	vxr_vk_graphics_drawParameters parameters;
	vxr_vk_graphics_drawIndirectBufferInfo indirectBuffer;
} vxr_vk_graphics_drawIndirectInfo;

typedef struct {
	VkBuffer vkBuffer;
	VkDeviceSize offset;
	VkDeviceSize size;
	VkIndexType indexType;
	uint32_t indexCount;
} vxr_vk_graphics_indexBufferInfo;
typedef struct {
	vxr_vk_graphics_drawParameters parameters;
	vxr_vk_graphics_indexBufferInfo indexBuffer;

	uint32_t instanceCount;
} vxr_vk_graphics_drawIndexedInfo;
typedef struct {
	vxr_vk_graphics_drawParameters parameters;
	vxr_vk_graphics_indexBufferInfo indexBuffer;
	vxr_vk_graphics_drawIndirectBufferInfo indirectBuffer;
} vxr_vk_graphics_drawIndexedIndirectInfo;

extern VXR_FN void vxr_stdlib_init(vxr_loggerCallback, vxr_loggerCallback, vxr_loggerCallback, vxr_loggerCallback,
								   vxr_loggerCallback, vxr_loggerCallback);

extern VXR_FN void vxr_vk_init(uintptr_t, uintptr_t, PFN_vkDebugUtilsMessengerCallbackEXT, vxr_vk_instance*);
extern VXR_FN void vxr_vk_destroy(vxr_vk_instance);

extern VXR_FN VkResult vxr_vk_device_vkPhysicalDeviceFromUUID(vxr_vk_instance, uint8_t (*)[VK_UUID_SIZE], uintptr_t*);

extern VXR_FN void vxr_vk_device_createSelector(uintptr_t, uint32_t, uint64_t, vxr_vk_device_selector*);
extern VXR_FN void vxr_vk_device_destroySelector(vxr_vk_device_selector);
extern VXR_FN void vxr_vk_device_selector_appendRequiredExtension(vxr_vk_device_selector, size_t, const char*);
extern VXR_FN void vxr_vk_device_selector_appendOptionalExtension(vxr_vk_device_selector, size_t, const char*);
extern VXR_FN void vxr_vk_device_selector_initFeatureChain(vxr_vk_device_selector, size_t, VkStructureType*);
extern VXR_FN void vxr_vk_device_selector_appendRequiredFeature(vxr_vk_device_selector, VkStructureType, size_t, size_t*);
extern VXR_FN void vxr_vk_device_selector_appendOptionalFeature(vxr_vk_device_selector, VkStructureType, size_t, size_t*);
extern VXR_FN void vxr_vk_device_selector_appendRequiredFormatFeature(vxr_vk_device_selector, VkFormat, VkFormatFeatureFlags2);
extern VXR_FN void vxr_vk_device_selector_getEnabledExtensions(vxr_vk_device_selector, size_t*, const char**);
extern VXR_FN void vxr_vk_device_selector_getEnabledFeatures(vxr_vk_device_selector, const char**);

extern VXR_FN void vxr_vk_device_init(vxr_vk_instance, vxr_vk_device_selector);
extern VXR_FN void vxr_vk_device_destroy(vxr_vk_instance);
extern VXR_FN void vxr_vk_device_getProperties(vxr_vk_instance, vxr_vk_device_properties*);

extern VXR_FN void vxr_vk_waitIdle(vxr_vk_instance);

extern VXR_FN void vxr_vk_commandBuffer_beginNamedRegion(vxr_vk_instance, VkCommandBuffer, size_t, const char*);
extern VXR_FN void vxr_vk_commandBuffer_endNamedRegion(vxr_vk_instance, VkCommandBuffer);
extern VXR_FN void vxr_vk_commandBuffer_barrier(vxr_vk_instance, VkCommandBuffer, VkDependencyInfo);
extern VXR_FN void vxr_vk_commandBuffer_fillBuffer(vxr_vk_instance, VkCommandBuffer, VkBuffer, VkDeviceSize, VkDeviceSize, uint32_t);
extern VXR_FN void vxr_vk_commandBuffer_updateBuffer(vxr_vk_instance, VkCommandBuffer, VkBuffer, VkDeviceSize, VkDeviceSize, void*);
extern VXR_FN void vxr_vk_commandBuffer_clearColorImage(vxr_vk_instance, VkCommandBuffer, VkImage, VkImageLayout,
														VkClearColorValue, uint32_t, VkImageSubresourceRange*);
extern VXR_FN void vxr_vk_commandBuffer_copyBuffer(vxr_vk_instance, VkCommandBuffer, VkBuffer, VkBuffer, uint32_t, VkBufferCopy*);
extern VXR_FN void vxr_vk_commandBuffer_copyBufferToImage(vxr_vk_instance, VkCommandBuffer, VkBuffer, VkImage,
														  VkImageLayout, uint32_t, VkBufferImageCopy*);

extern VXR_FN void vxr_vk_createSemaphore(vxr_vk_instance, size_t, const char*, VkSemaphoreType, VkSemaphore*);
extern VXR_FN void vxr_vk_signalSemaphore(vxr_vk_instance, VkSemaphore, uint64_t);
extern VXR_FN void vxr_vk_waitSemaphore(vxr_vk_instance, VkSemaphore, uint64_t);
extern VXR_FN uint64_t vxr_vk_getSemaphoreValue(vxr_vk_instance, VkSemaphore);
extern VXR_FN void vxr_vk_destroySemaphore(vxr_vk_instance, VkSemaphore);

extern VXR_FN void vxr_vk_createHostBuffer(vxr_vk_instance, size_t, const char*, vxr_vk_bufferCreateInfo, vxr_vk_hostBuffer*);
extern VXR_FN void vxr_vk_destroyHostBuffer(vxr_vk_instance, vxr_vk_hostBuffer);
extern VXR_FN void vxr_vk_hostBuffer_write(vxr_vk_instance, vxr_vk_hostBuffer, size_t, size_t, void*);
extern VXR_FN void vxr_vk_hostBuffer_read(vxr_vk_instance, vxr_vk_hostBuffer, size_t, size_t, void*);
extern VXR_FN void vxr_vk_createDeviceBuffer(vxr_vk_instance, size_t, const char*, vxr_vk_bufferCreateInfo, vxr_vk_deviceBuffer*);
extern VXR_FN void vxr_vk_destroyDeviceBuffer(vxr_vk_instance, vxr_vk_deviceBuffer);

extern VXR_FN void vxr_vk_getFormatProperties(vxr_vk_instance, VkFormat, VkFormatProperties3*);
extern VXR_FN void vxr_vk_createImage(vxr_vk_instance, size_t, const char*, vxr_vk_imageCreateInfo, vxr_vk_image*);
extern VXR_FN void vxr_vk_destroyImage(vxr_vk_instance, vxr_vk_image);

extern VXR_FN void vxr_vk_createImageView(vxr_vk_instance, size_t, const char*, vxr_vk_imageViewCreateInfo, VkImageView*);
extern VXR_FN void vxr_vk_destroyImageView(vxr_vk_instance, VkImageView);

extern VXR_FN void vxr_vk_createSampler(vxr_vk_instance, size_t, const char*, vxr_vk_samplerCreateInfo, VkSampler*);
extern VXR_FN void vxr_vk_destroySampler(vxr_vk_instance, VkSampler);

extern VXR_FN void vxr_vk_shader_initToolchain(vxr_vk_shader_toolchainOptions, vxr_vk_shader_toolchain*);
extern VXR_FN void vxr_vk_shader_destroyToolchain(vxr_vk_shader_toolchain);

extern VXR_FN void vxr_vk_shader_compile(vxr_vk_shader_toolchain, vxr_vk_shader_compileInfo,
										 vxr_vk_shader_compileResult*, vxr_vk_shader_reflectResult*);
extern VXR_FN void vxr_vk_shader_destroyCompileResult(vxr_vk_shader_compileResult);

extern VXR_FN void vxr_vk_shader_compileResult_getSPIRV(vxr_vk_shader_compileResult, vxr_vk_shader_spirv*);

extern VXR_FN void vxr_vk_shader_reflect(vxr_vk_shader_toolchain, vxr_vk_shader_spirv, vxr_vk_shader_reflectResult*);
extern VXR_FN void vxr_vk_shader_destroyReflectResult(vxr_vk_shader_reflectResult);

extern VXR_FN void vxr_vk_shader_reflectResult_getEntryPoints(vxr_vk_shader_reflectResult, size_t*, vxr_vk_shader_entryPoint*);
extern VXR_FN void vxr_vk_shader_reflectResult_getSpecConstants(vxr_vk_shader_reflectResult, uint32_t*, vxr_vk_shader_reflectResult_specConstant*);
extern VXR_FN void vxr_vk_shader_reflectResult_getLocalSize(vxr_vk_shader_reflectResult, vxr_vk_shader_reflectResult_constant (*)[3]);
extern VXR_FN void vxr_vk_shader_reflectResult_getNumOutputs(vxr_vk_shader_reflectResult, size_t, uint32_t*);
extern VXR_FN void vxr_vk_shader_reflectResult_getPushConstantRange(vxr_vk_shader_reflectResult, VkPushConstantRange*);
extern VXR_FN void vxr_vk_shader_reflectResult_getDescriptorSetSizes(vxr_vk_shader_reflectResult, uint32_t*, uint32_t*);
extern VXR_FN void vxr_vk_shader_reflectResult_getDescriptorSetBinding(vxr_vk_shader_reflectResult, uint32_t, uint32_t,
																	   vxr_vk_shader_reflectResult_descriptorSetBinding*);
extern VXR_FN void vxr_vk_shader_reflectResult_getBufferMetadata(vxr_vk_shader_reflectResult, uint32_t, uint32_t,
																 uint32_t, vxr_vk_shader_reflectResult_bufferMetadata*);
extern VXR_FN void vxr_vk_shader_reflectResult_getSamplerMetadata(vxr_vk_shader_reflectResult, uint32_t, uint32_t,
																  uint32_t, vxr_vk_shader_reflectResult_samplerMetadata*);
extern VXR_FN void vxr_vk_shader_reflectResult_getImageMetadata(vxr_vk_shader_reflectResult, uint32_t, uint32_t,
																uint32_t, vxr_vk_shader_reflectResult_imageMetadata*);

extern VXR_FN void vxr_vk_shader_createDescriptorSetLayout(vxr_vk_instance, size_t, const char*, uint32_t,
														   VkDescriptorSetLayoutBinding*, VkDescriptorSetLayout*);
extern VXR_FN void vxr_vk_shader_destroyDescriptorSetLayout(vxr_vk_instance, VkDescriptorSetLayout);

extern VXR_FN void vxr_vk_shader_createDescriptorPool(vxr_vk_instance, size_t, const char*, VkDescriptorPoolCreateInfo, VkDescriptorPool*);
extern VXR_FN void vxr_vk_shader_destroyDescriptorPool(vxr_vk_instance, VkDescriptorPool);

extern VXR_FN void vxr_vk_shader_createDescriptorSet(vxr_vk_instance, size_t, const char*, VkDescriptorSetAllocateInfo, VkDescriptorSet*);
extern VXR_FN void vxr_vk_shader_updateDescriptorSet(vxr_vk_instance, VkWriteDescriptorSet);
extern VXR_FN void vxr_vk_shader_destroyDescriptorSet(vxr_vk_instance, VkDescriptorPool, VkDescriptorSet);

extern VXR_FN void vxr_vk_shader_createPipelineLayout(vxr_vk_instance, size_t, const char*,
													  vxr_vk_shader_pipelineLayoutCreateInfo, VkPipelineLayout*);
extern VXR_FN void vxr_vk_shader_destroyPipelineLayout(vxr_vk_instance, VkPipelineLayout);

extern VXR_FN void vxr_vk_shader_destroyPipeline(vxr_vk_instance, VkPipeline);

extern VXR_FN void vxr_vk_compute_createShaderPipeline(vxr_vk_instance, size_t, const char*,
													   vxr_vk_compute_shaderPipelineCreateInfo, VkPipeline*);
extern VXR_FN void vxr_vk_compute_dispatch(vxr_vk_instance, VkCommandBuffer, vxr_vk_compute_dispatchInfo);
extern VXR_FN void vxr_vk_compute_dispatchIndirect(vxr_vk_instance, VkCommandBuffer, vxr_vk_compute_dispatchIndirectInfo);

extern VXR_FN VkResult vxr_vk_graphics_init(vxr_vk_instance, uint64_t, uint32_t);
extern VXR_FN void vxr_vk_graphics_destroy(vxr_vk_instance);

extern VXR_FN void vxr_vk_graphics_getSurfaceInfo(vxr_vk_instance, vxr_vk_surfaceInfo*);

extern VXR_FN void vxr_vk_graphics_createVertexInputPipeline(vxr_vk_instance, size_t, const char*, VkPrimitiveTopology, VkBool32, VkPipeline*);
extern VXR_FN void vxr_vk_graphics_createShaderPipeline(vxr_vk_instance, size_t, const char*,
														vxr_vk_graphics_shaderPipelineCreateInfo, VkPipeline*);
extern VXR_FN void vxr_vk_graphics_createFragmentOutputPipeline(vxr_vk_instance, size_t, const char*,
																vxr_vk_graphics_fragmentOutputPipelineCreateInfo, VkPipeline*);
extern VXR_FN VkBool32 vxr_vk_graphics_linkPipelines(vxr_vk_instance, size_t, const char*, VkPipelineLayout, uint32_t,
													 VkPipeline*, VkPipeline*);
extern VXR_FN void vxr_vk_graphics_linkOptimizePipelines(vxr_vk_instance, size_t, const char*, VkPipelineLayout,
														 uint32_t, VkPipeline*, VkPipeline*);

extern VXR_FN void vxr_vk_graphics_createFrame(vxr_vk_instance, size_t, const char*, vxr_vk_graphics_frame*);
extern VXR_FN void vxr_vk_graphics_destroyFrame(vxr_vk_graphics_frame);

extern VXR_FN void vxr_vk_graphics_frame_begin(vxr_vk_instance, size_t, const char*, vxr_vk_graphics_frame);
extern VXR_FN void vxr_vk_graphics_frame_cancel(vxr_vk_instance, vxr_vk_graphics_frame);
extern VXR_FN void vxr_vk_graphics_frame_end(vxr_vk_instance, vxr_vk_graphics_frame);
extern VXR_FN VkResult vxr_vk_graphics_frame_acquireSurface(vxr_vk_instance, vxr_vk_graphics_frame, vxr_vk_surface*);
extern VXR_FN VkResult vxr_vk_graphics_frame_submit(vxr_vk_instance, vxr_vk_graphics_frame);
extern VXR_FN void vxr_vk_graphics_frame_wait(vxr_vk_instance, vxr_vk_graphics_frame);

extern VXR_FN void vxr_vk_graphics_frame_createHostScratchBuffer(vxr_vk_instance, vxr_vk_graphics_frame, size_t,
																 const char*, vxr_vk_bufferCreateInfo, vxr_vk_hostBuffer*);

extern VXR_FN void vxr_vk_graphics_frame_commandBufferBegin(vxr_vk_instance, vxr_vk_graphics_frame, size_t, const char*, VkCommandBuffer*);
extern VXR_FN void vxr_vk_graphics_frame_commandBufferSubmit(vxr_vk_instance, vxr_vk_graphics_frame, VkCommandBuffer, uint32_t,
															 VkSemaphoreSubmitInfo*, uint32_t, VkSemaphoreSubmitInfo*);

extern VXR_FN void vxr_vk_graphics_renderPassBegin(vxr_vk_instance, VkCommandBuffer, size_t, const char*, VkRenderingInfo,
												   VkBool32*, VkColorBlendEquationEXT*, VkColorComponentFlags*);
extern VXR_FN void vxr_vk_graphics_draw(vxr_vk_instance, VkCommandBuffer, vxr_vk_graphics_drawInfo);
extern VXR_FN void vxr_vk_graphics_drawIndirect(vxr_vk_instance, VkCommandBuffer, vxr_vk_graphics_drawIndirectInfo);
extern VXR_FN void vxr_vk_graphics_drawIndexed(vxr_vk_instance, VkCommandBuffer, vxr_vk_graphics_drawIndexedInfo);
extern VXR_FN void vxr_vk_graphics_drawIndexedIndirect(vxr_vk_instance, VkCommandBuffer, vxr_vk_graphics_drawIndexedIndirectInfo);
extern VXR_FN void vxr_vk_graphics_renderPassEnd(vxr_vk_instance, VkCommandBuffer);

#ifdef __cplusplus
}
#endif
#endif
