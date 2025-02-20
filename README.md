# VXR

VXR is a Vulkan only graphics abstraction library closer to a Render Hardware Interface but not quite.
The goals are to reduce the boilerplate and verboseness of Vulkan and create an API targeted towards indie game development.
What this means is that doing things the most efficient way is less important than
doing things good enough but with a nicer API.

Implementing all of Vulkan is a non goal but we do plan on having API to retrieve
Vulkan objects to let users do whatever they want.

## Design choices

- No binary semaphore or fences
    - Timeline semaphores basically supersede these, the only exception is vkAcquireNextImageKHR and vkQueuePresentKHR which we have special cased.
- No manual vkDescriptorSetLayout creation
    - We use shader reflection to determine the layout and if descriptorCount is > 1 we pass VK_DESCRIPTOR_BINDING_UPDATE_UNUSED_WHILE_PENDING_BIT | VK_DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT.
- Graphics Pipeline Library
    - We use VK_EXT_graphics_pipeline_library to allow more dynamic pipeline creation while keeping the benefits of a vkPipeline such as driver optimizations which VK_EXT_shader_object may not have access to.
- Dynamic Rendering
    - We use VK_KHR_dynamic_rendering as it removes bookkeeping of vkRenderPass and vkFramebuffer objects and simplifies the API.
- VK_EXT_surface_maintenance1/VK_EXT_swapchain_maintenance1
    - The spec has a long standing issue of surface/swapchain resizing being technically undefined behavior, we require these extensions so we do not have to deal with it.

# API Stability

We consider VXR to be in "request for comments" mode, this means that the API can and will change
but hopefully breaking changes will be few and far between.

# Supported Platforms

VXR is known to work on Windows 10 and Ubuntu 24.04.
However MSVC is not supported as Go doesn't support it so Windows builds must use 
[mingw-w64](https://github.com/mstorsjo/llvm-mingw).

We do not currently have plans for Android/Apple.

## Hardware Requirements

VXR is designed for Vulkan 1.3+ devices with the entire list of features and extensions used
in [config.go](https://github.com/goarrg/vxr/blob/main/config.go#:~:text=createDeviceSelector)

# Setup

VXR is `go get` able however it is not trivially `go build` able. What this means
is that there are extra steps required before doing a `go build`.
VXR follows the make program pattern goARRG uses which involves creating a `/cmd/make`
folder and writing a go program.
An example of that that looks like can be found in the examples repository.
https://github.com/goarrg/examples/tree/main/vxrhelloworld/cmd/make

The vxrhelloworld folder is also self contained.

To create a new project from it:
- Copy paste the folder somewhere
- Open the terminal and cd to it
- Run `go mod init example.com/repository-name`
- Followed by a `go mod tidy`
- After which you can `go run ./cmd/make` to build
    - Assuming you already installed Go, GCC/Clang and all the other things listed on the main repo.

# TODO
- Async Compute/Transfer API
- API to retrieve Vulkan handles for advanced usage
- Testing system and infrastructure
- Figure out a better map key for caches
- Multiview? 
