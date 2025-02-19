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

//go:generate go run ./const_gen.go

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"goarrg.com/debug"
	"goarrg.com/toolchain"
	"goarrg.com/toolchain/cgodep"
	"goarrg.com/toolchain/golang"
)

func InstallGeneratorDeps() {
	if err := installVkHeaders(); err != nil {
		panic(debug.ErrorWrapf(err, "Failed to install vulkan-headers"))
	}
	if err := installVkDocs(); err != nil {
		panic(debug.ErrorWrapf(err, "Failed to install vulkan-docs"))
	}
}

func Install(t toolchain.Target, b toolchain.Build) {
	if err := installShaderc(t); err != nil {
		panic(debug.ErrorWrapf(err, "Failed to install shaderc"))
	}
	if err := installSPIRVCross(t); err != nil {
		panic(debug.ErrorWrapf(err, "Failed to install spirv-cross"))
	}
	if err := installVXR(t, b); err != nil {
		panic(debug.ErrorWrapf(err, "Failed to install vxr"))
	}
}

func scanDirModTime(dir string, ignore []string) time.Time {
	latestMod := time.Unix(0, 0)

	err := filepath.Walk(dir, func(path string, fs fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return err
		}
		{
			rel := strings.TrimPrefix(path, dir+string(filepath.Separator))
			for _, i := range ignore {
				if rel == i {
					if fs.IsDir() {
						return filepath.SkipDir
					}
					return err
				}
			}
		}
		mod := fs.ModTime()
		if mod.After(latestMod) {
			latestMod = mod
		}
		return err
	})
	if err != nil {
		panic(err)
	}

	return latestMod
}

func processSrc(t toolchain.Target, srcDir string, p func(string, []string) error) ([]string, []string, error) {
	includeDir := filepath.Join(srcDir, "include")

	deps := []string{"vulkan-headers", "shaderc", "spirv-cross"}
	cFlags, err := cgodep.Resolve(t, cgodep.ResolveCFlags, deps...)
	if err != nil {
		return nil, nil, err
	}
	ldFlags, err := cgodep.Resolve(t, cgodep.ResolveLDFlags, deps...)
	if err != nil {
		return nil, nil, err
	}

	flags := append(cFlags, "-I"+srcDir, "-I"+includeDir, "-Werror=vla", "-Wall", "-Wextra", "-Wpedantic",
		"-Wno-unknown-pragmas", "-Wno-missing-field-initializers",
	)

	return cFlags, ldFlags, filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		switch filepath.Ext(path) {
		case ".h", ".c":
			args := append([]string(nil), flags...)
			args = append(args, strings.Split(toolchain.EnvGet("CGO_CFLAGS"), " ")...)
			args = append(args,
				"-std=c17",
			)
			return p(path, args)

		case ".hpp", ".cpp":
			args := append([]string(nil), flags...)
			args = append(args, strings.Split(toolchain.EnvGet("CGO_CXXFLAGS"), " ")...)
			args = append(args,
				"-std=c++23",
				//"-ffile-prefix-map=" + srcDir + "=libvxr",
			)
			return p(path, args)

		default:
		}

		return nil
	})
}

func installVXR(t toolchain.Target, b toolchain.Build) error {
	module := golang.CallersModule()
	srcDir := filepath.Join(module.Dir, "libvxr")
	includeDir := filepath.Join(srcDir, "include")
	srcVersion := scanDirModTime(srcDir, []string{".cache", ".clang-tidy", ".clangd", "compile_commands.json"})
	makeVersion := scanDirModTime(filepath.Join(module.Dir, "make"), nil)

	version := strconv.FormatInt(srcVersion.Unix(), 16) + "-" + strconv.FormatInt(makeVersion.Unix(), 16)
	installDir := cgodep.InstallDir("vxr", t, b)
	installedVersion := cgodep.ReadVersion(installDir)
	if installedVersion == version {
		return cgodep.SetActiveBuild("vxr", t, b)
	}

	if err := os.RemoveAll(installDir); err != nil {
		return err
	}

	type cmd struct {
		Directory string   `json:"directory"`
		Arguments []string `json:"arguments"`
		File      string   `json:"file"`
	}
	var cmds []cmd
	var cFlags, ldFlags []string

	{
		objs := []string{}
		buildDir, err := os.MkdirTemp("", "vxr")
		if err != nil {
			return err
		}
		defer os.RemoveAll(buildDir)
		cFlags, ldFlags, err = processSrc(t, srcDir, func(path string, args []string) error {
			path = strings.TrimPrefix(path, srcDir+string(filepath.Separator))
			switch filepath.Ext(path) {
			case ".c":
				objs = append(objs, path+".o")
				cmds = append(cmds, cmd{Directory: srcDir, Arguments: append([]string{os.Getenv("CC")}, args...), File: path})
			case ".cpp":
				objs = append(objs, path+".o")
				cmds = append(cmds, cmd{Directory: srcDir, Arguments: append([]string{os.Getenv("CXX")}, args...), File: path})
			}
			return nil
		})
		if err != nil {
			return err
		}

		if fOut, err := os.Create(filepath.Join(srcDir, "compile_commands.json")); err == nil {
			enc := json.NewEncoder(fOut)
			err = enc.Encode(cmds)
			if err != nil {
				return err
			}
		} else if !strings.Contains(srcDir, filepath.Join("pkg", "mod")) {
			// do not warn if obtained through "go get" as the dir is read only
			debug.WPrintf("Failed to write compile_commands.json")
		}

		wg := sync.WaitGroup{}
		for i, c := range cmds {
			obj := filepath.Join(buildDir, objs[i])
			if err := os.MkdirAll(filepath.Dir(obj), 0o755); err != nil {
				return err
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				if out, err := toolchain.RunCombinedOutput(c.Arguments[0], append(c.Arguments[1:], "-o", obj, "-c", filepath.Join(c.Directory, c.File))...); err != nil {
					debug.EPrintf("%s", out)
					os.Exit(1)
				}
			}()
		}
		wg.Wait()
		if err := os.MkdirAll(filepath.Join(installDir, "lib"), 0o755); err != nil {
			return err
		}
		args := []string{"rcs", filepath.Join(installDir, "lib", "libvxr.a")}
		if err := toolchain.RunDir(buildDir, os.Getenv("AR"), append(args, objs...)...); err != nil {
			return err
		}
	}

	golang.SetShouldCleanCache()
	return cgodep.WriteMetaFile("vxr", t, b, cgodep.Meta{
		Version: version,
		Flags: cgodep.Flags{
			CFlags:        append([]string{"-I" + includeDir}, cFlags...),
			LDFlags:       append([]string{"-L" + filepath.Join(installDir, "lib"), "-lvxr"}, ldFlags...),
			StaticLDFlags: append([]string{"-L" + filepath.Join(installDir, "lib"), "-lvxr"}, ldFlags...),
		},
	})
}

func Lint() error {
	module := golang.CallersModule()
	srcDir := filepath.Join(module.Dir, "libvxr")
	wg := sync.WaitGroup{}
	_, _, err := processSrc(toolchain.Target{OS: os.Getenv("GOOS"), Arch: os.Getenv("GOARCH")}, srcDir, func(path string, args []string) error {
		wg.Add(1)
		go func() {
			defer wg.Done()
			path = strings.TrimPrefix(path, srcDir+string(filepath.Separator))
			if strings.HasSuffix(path, "vk_mem_alloc.h") {
				return
			}
			if out, err := toolchain.RunDirCombinedOutput(srcDir, "clang-tidy", append([]string{"-warnings-as-errors=*", path, "--"}, args...)...); err != nil {
				debug.EPrintf("%s", out)
				os.Exit(1)
			}
		}()
		return nil
	})
	wg.Wait()
	return err
}
