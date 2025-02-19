/*
Copyright 2022 The goARRG Authors.

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

import (
	"io"
	"os"
	"path/filepath"
	"runtime"

	"goarrg.com/debug"
	"goarrg.com/toolchain"
	"goarrg.com/toolchain/cgodep"
	"goarrg.com/toolchain/cmake"
	"goarrg.com/toolchain/golang"
)

const (
	shadercBuild = "shaderc-" + shadercVersion + "-glslang-" + glslangVersion + "-spirv-headers-" + spirvHeadersVersion + "-spirv-tools-" + spirvToolsVersion + "-goarrg0"
)

func installShaderc(t toolchain.Target) error {
	installDir := cgodep.InstallDir("shaderc", t, toolchain.BuildRelease)
	if cgodep.ReadVersion(installDir) == shadercBuild {
		return cgodep.SetActiveBuild("shaderc", t, toolchain.BuildRelease)
	}
	if err := os.RemoveAll(installDir); err != nil {
		return err
	}

	data, err := cgodep.Get("https://github.com/google/shaderc/archive/refs/tags/"+shadercVersion+".tar.gz", "shaderc.tar.gz", func(target io.ReadSeeker) error {
		return cgodep.VerifySHA256(target, shadercSHA256)
	})
	if err != nil {
		return debug.ErrorWrapf(err, "Failed to download shaderc")
	}

	srcDir, err := os.MkdirTemp("", "goarrg-shaderc")
	if err != nil {
		return debug.ErrorWrapf(err, "Failed to make temp dir: %q", srcDir)
	}
	defer os.RemoveAll(srcDir)

	debug.VPrintf("Extracting shaderc")

	if err := extractTARGZ(data, srcDir); err != nil {
		return debug.ErrorWrapf(err, "Failed to extract shaderc")
	}

	{
		data, err := cgodep.Get("https://github.com/KhronosGroup/glslang/archive/refs/heads/"+glslangVersion+".tar.gz", "glslang.tar.gz", func(target io.ReadSeeker) error {
			return cgodep.VerifySHA256(target, glslangSHA256)
		})
		if err != nil {
			return debug.ErrorWrapf(err, "Failed to download glslang")
		}
		debug.VPrintf("Extracting glslang")
		if err := extractTARGZ(data, filepath.Join(srcDir, "third_party", "glslang")); err != nil {
			return debug.ErrorWrapf(err, "Failed to extract glslang")
		}
	}
	{
		data, err := cgodep.Get("https://github.com/KhronosGroup/SPIRV-Headers/archive/refs/heads/"+spirvHeadersVersion+".tar.gz", "spirv-headers.tar.gz", func(target io.ReadSeeker) error {
			return cgodep.VerifySHA256(target, spirvHeadersSHA256)
		})
		if err != nil {
			return debug.ErrorWrapf(err, "Failed to download spirv-headers")
		}
		debug.VPrintf("Extracting spirv-headers")
		if err := extractTARGZ(data, filepath.Join(srcDir, "third_party", "spirv-headers")); err != nil {
			return debug.ErrorWrapf(err, "Failed to extract spirv-headers")
		}
	}
	{
		data, err := cgodep.Get("https://github.com/KhronosGroup/SPIRV-Tools/archive/refs/heads/"+spirvToolsVersion+".tar.gz", "spirv-tools.tar.gz", func(target io.ReadSeeker) error {
			return cgodep.VerifySHA256(target, spirvToolsSHA256)
		})
		if err != nil {
			return debug.ErrorWrapf(err, "Failed to download spirv-tools")
		}
		debug.VPrintf("Extracting spirv-tools")
		if err := extractTARGZ(data, filepath.Join(srcDir, "third_party", "spirv-tools")); err != nil {
			return debug.ErrorWrapf(err, "Failed to extract spirv-tools")
		}
	}

	buildDir, err := os.MkdirTemp("", "goarrg-shaderc-build")
	if err != nil {
		return debug.ErrorWrapf(err, "Failed to make temp dir: %q", buildDir)
	}

	defer os.RemoveAll(buildDir)

	args := map[string]string{
		"CMAKE_SKIP_INSTALL_RPATH": "1", "CMAKE_SKIP_RPATH": "1",
		"BUILD_SHARED_LIBS": "0", "BUILD_TESTING": "0",
		"ENABLE_CTEST": "0", "ENABLE_GLSLANG_BINARIES": "0",
		"SHADERC_SKIP_EXAMPLES": "1", "SHADERC_SKIP_TESTS": "1",
		"SPIRV_SKIP_EXECUTABLES": "1", "SPIRV_SKIP_TESTS": "1",
	}
	if runtime.GOOS == "windows" {
		args["CMAKE_TOOLCHAIN_FILE"] = filepath.Join(srcDir, "cmake", "linux-mingw-toolchain.cmake")
	}

	if err := cmake.Configure(t, toolchain.BuildRelease, srcDir, buildDir, installDir, args); err != nil {
		return err
	}
	if err := cmake.Build(buildDir); err != nil {
		return err
	}
	if err := cmake.Install(buildDir); err != nil {
		return err
	}

	golang.SetShouldCleanCache()
	return cgodep.WriteMetaFile("shaderc", t, toolchain.BuildRelease, cgodep.Meta{
		Version: shadercBuild,
		Flags: cgodep.Flags{
			CFlags:        []string{"-I" + filepath.Join(installDir, "include")},
			LDFlags:       []string{"-L" + filepath.Join(installDir, "lib"), "-lshaderc_combined"},
			StaticLDFlags: []string{"-L" + filepath.Join(installDir, "lib"), "-lshaderc_combined"},
		},
	})
}
