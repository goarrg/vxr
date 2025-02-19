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

import (
	"io"
	"os"
	"path/filepath"

	"goarrg.com/debug"
	"goarrg.com/toolchain"
	"goarrg.com/toolchain/cgodep"
	"goarrg.com/toolchain/cmake"
	"goarrg.com/toolchain/golang"
)

const (
	spirvCrossBuild = spirvCrossVersion + "-goarrg0"
)

func installSPIRVCross(t toolchain.Target) error {
	installDir := cgodep.InstallDir("spirv-cross", t, toolchain.BuildRelease)
	if cgodep.ReadVersion(installDir) == spirvCrossBuild {
		return cgodep.SetActiveBuild("spirv-cross", t, toolchain.BuildRelease)
	}
	if err := os.RemoveAll(installDir); err != nil {
		return err
	}

	data, err := cgodep.Get("https://github.com/KhronosGroup/SPIRV-Cross/archive/refs/heads/"+spirvCrossVersion+".tar.gz", "spirv-cross.tar.gz", func(target io.ReadSeeker) error {
		return cgodep.VerifySHA256(target, spirvCrossSHA256)
	})
	if err != nil {
		return debug.ErrorWrapf(err, "Failed to download spirv-cross")
	}

	srcDir, err := os.MkdirTemp("", "goarrg-spirv-cross")
	if err != nil {
		return debug.ErrorWrapf(err, "Failed to make temp dir: %q", srcDir)
	}
	defer os.RemoveAll(srcDir)

	debug.VPrintf("Extracting spirv-cross")

	if err := extractTARGZ(data, srcDir); err != nil {
		return debug.ErrorWrapf(err, "Failed to extract spirv-cross")
	}

	buildDir, err := os.MkdirTemp("", "goarrg-spirv-cross-build")
	if err != nil {
		return debug.ErrorWrapf(err, "Failed to make temp dir: %q", buildDir)
	}

	defer os.RemoveAll(buildDir)

	args := map[string]string{
		"CMAKE_SKIP_INSTALL_RPATH": "1", "CMAKE_SKIP_RPATH": "1",
		"SPIRV_CROSS_CLI": "0", "SPIRV_CROSS_ENABLE_TESTS": "0",
		"SPIRV_CROSS_ENABLE_CPP": "1", "SPIRV_CROSS_ENABLE_C_API": "1",
		"SPIRV_CROSS_ENABLE_HLSL": "0", "SPIRV_CROSS_ENABLE_MSL": "0",
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
	ldflags := []string{
		"-L" + filepath.Join(installDir, "lib"), "-lspirv-cross-c", "-lspirv-cross-cpp",
		"-lspirv-cross-glsl",
		"-lspirv-cross-util", "-lspirv-cross-core", "-lspirv-cross-reflect",
	}
	return cgodep.WriteMetaFile("spirv-cross", t, toolchain.BuildRelease, cgodep.Meta{
		Version: spirvCrossBuild, Flags: cgodep.Flags{
			CFlags:        []string{"-I" + filepath.Join(installDir, "include")},
			LDFlags:       ldflags,
			StaticLDFlags: ldflags,
		},
	})
}
