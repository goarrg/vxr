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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unsafe"

	"goarrg.com/asset"
	"goarrg.com/debug"
	"goarrg.com/rhi/vxr"

	"golang.org/x/tools/go/packages"
)

var flags flag.FlagSet

type vkapi uint32

func (api *vkapi) UnmarshalText(data []byte) error {
	parts := strings.Split(string(data), ".")
	if len(parts) != 2 {
		return debug.Errorf("API string not in the format \"X.Y\"")
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return debug.ErrorWrapf(err, "Invalid api string")
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return debug.ErrorWrapf(err, "Invalid api string")
	}
	*api = vkapi((((uint32)(major)) << 22) | (((uint32)(minor)) << 12))
	return nil
}

func (api vkapi) MarshalText() (text []byte, err error) {
	return fmt.Appendf(nil, "%d.%d", ((api >> 22) & 0x7F), ((api >> 12) & 0x3FF)), err
}

type macros []vxr.ShaderMacro

func (m *macros) UnmarshalText(data []byte) error {
	str := string(data)
	i := strings.Index(str, "=")
	switch {
	case i == 0:
		return debug.Errorf("Macro not in the format \"macro=value\"")
	case i < 0:
		*m = append(*m, vxr.ShaderMacro{
			Name: str,
		})
	default:
		*m = append(*m, vxr.ShaderMacro{
			Name:  str[:i],
			Value: str[i+1:],
		})
	}
	return nil
}

func (m macros) MarshalText() (text []byte, err error) {
	str := ""
	for _, i := range m {
		str += fmt.Sprintf("%s=%s\n", i.Name, i.Value)
	}
	return ([]byte)(strings.TrimSuffix(str, "\n")), nil
}

type generator uint32

const (
	generatorJSON generator = iota
	generatorGO
)

func (g *generator) UnmarshalText(data []byte) error {
	switch string(data) {
	case "json":
		*g = generatorJSON
	case "go":
		*g = generatorGO
	default:
		return debug.Errorf("Invalid value: %q", data)
	}
	return nil
}

func (g generator) MarshalText() (text []byte, err error) {
	switch g {
	case generatorJSON:
		return ([]byte)("json"), nil
	case generatorGO:
		return ([]byte)("go"), nil
	default:
		return nil, debug.Errorf("Invalid value: %d", g)
	}
}

func main() {
	debug.SetLevel(debug.LogLevelWarn)

	flags.Usage = help
	flags.Init("", flag.ExitOnError)

	v := flags.Bool("v", false, "Verbose - Print high level tasks")
	vv := flags.Bool("vv", false, "Very Verbose - Print everything")

	dir := flags.String("dir", ".", "Sets the directory for the purposes of <file> and #include<...> resolution.\n"+
		"vxrc does not support multiple search paths.")
	outDir := flags.String("out-dir", ".", "Sets the output directory.")

	api := vkapi(0)
	flags.TextVar(&api, "target-api", vkapi(vxr.MinAPI), "Sets the target vulkan version in the format \"X.Y\".")
	strip := flags.Bool("strip", false, "Strips debug and non-semantic information.")
	opt := flags.Bool("O", false, "Optimize for performance with spirv-opt.")
	optS := flags.Bool("Os", false, "Optimize for size with spirv-opt.")

	defines := macros{}
	flags.TextVar(&defines, "D", macros{}, "Define macro in the format \"macro=value\".")

	g := generator(0)
	flags.TextVar(&g, "generator", generatorJSON, "Sets the generator to use when outputting metadata.\n"+
		"Valid values are \"json\" and \"go\".")

	separateSPV := flags.Bool("separate-spirv", false, "Output spirv as a separate .spv file.")
	skipMetadata := flags.Bool("skip-metadata", false, "Skips outputting vxr.ShaderMetadata.")

	err := flags.Parse(os.Args[1:])
	if err != nil {
		panic(err)
	}

	if *v {
		debug.SetLevel(debug.LogLevelInfo)
	} else if *vv {
		debug.SetLevel(debug.LogLevelVerbose)
	}

	args := flags.Args()
	if len(args) == 0 {
		debug.EPrintf("No input file provided.")
		help()
		os.Exit(2)
	} else if len(args) > 1 {
		debug.EPrintf("vxrc can only compile one file at a time.")
		help()
		os.Exit(2)
	}

	vxr.InitShaderCompiler(vxr.ShaderCompilerOptions{
		API:                 uint32(api),
		Strip:               *strip,
		OptimizePerformance: *opt,
		OptimizeSize:        *optS,
	})
	defer vxr.DestroyShaderCompiler()

	name := args[0]
	debug.IPrintf("Compiling shader")
	spv, layout, meta := vxr.CompileShader(asset.DirFS(*dir), name, defines...)

	outName := filepath.Base(name)
	err = os.MkdirAll(*outDir, 0o755)
	if err != nil {
		panic(err)
	}

	if *separateSPV {
		spvFile := filepath.Join(*outDir, outName+".spv")
		debug.IPrintf("Writing SPIRV to: %q", spvFile)
		err := os.WriteFile(spvFile,
			unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(spv.SPIRV))), int(unsafe.Sizeof(spv.SPIRV[0]))*len(spv.SPIRV)), 0o655)
		if err != nil {
			panic(err)
		}
	}

	switch g {
	case generatorJSON:
		genJson(*outDir, outName, *separateSPV, *skipMetadata, spv, layout, meta)
	case generatorGO:
		genGo(*outDir, outName, *separateSPV, *skipMetadata, spv, layout, meta)
	}
}

func help() {
	fmt.Fprintf(os.Stderr, "vxrc is a cli wrapper over vxr.CompileShader to generate shader metadata offline.\n"+
		"\nShader reflection works best with non-optimized spirv files, so the old way would involve at least 3 different programs and libs.\n"+
		"vxr.CompileShader and thus this cli was created to streamline the process.\n"+
		"\nvxrc only supports glsl comp,vert,frag shaders as those are the pipelines vxr currently supports.\n"+
		"Shader stage is determined by the #pragma shader_stage(...) line in the file.\n"+
		"\n")
	args := ""
	flags.VisitAll(func(f *flag.Flag) {
		n, u := flag.UnquoteUsage(f)
		if f.DefValue != "" {
			u += "\n\nDefaults to \"" + f.DefValue + "\"."
		}
		args += "\t-" + f.Name + " " + n + "\n\t\t" + strings.ReplaceAll(strings.TrimSpace(u), "\n", "\n\t\t") + "\n"
	})
	fmt.Fprintf(os.Stderr, "Usage:\n\t%s [arguments] <file>\n\nArguments:\n%s", filepath.Base(os.Args[0]), args)
}

func genJson(dir, name string, separateSPV, skipMetadata bool, spv *vxr.Shader, layout *vxr.ShaderLayout, meta *vxr.ShaderMetadata) {
	{
		m := map[string]any{
			"Layout": layout,
		}
		if !separateSPV {
			m["Shader"] = spv
		}
		if !skipMetadata {
			m["Metadata"] = meta
		}

		j, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}

		jsonFile := filepath.Join(dir, name+".json")
		debug.IPrintf("Writing metadata to: %q", jsonFile)
		err = os.WriteFile(jsonFile, j, 0o655)
		if err != nil {
			panic(err)
		}
	}
}

func genGo(dir, name string, separateSPV, skipMetadata bool, spv *vxr.Shader, layout *vxr.ShaderLayout, meta *vxr.ShaderMetadata) {
	filename := filepath.Join(dir, "zvxrc_"+name+".go")
	debug.IPrintf("Writing metadata to: %q", filename)
	fOut, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer fOut.Close()

	{
		args := ""
		for _, arg := range os.Args[1:] {
			args += arg + " "
		}
		fmt.Fprintf(fOut, "// go run goarrg.com/rhi/vxr/cmd/vxrc %s\n", args)
		fmt.Fprintf(fOut, "// Code generated by the command above; DO NOT EDIT.\n\n")
	}

	{
		p, err := packages.Load(&packages.Config{Mode: packages.NeedName}, dir)
		if err != nil {
			panic(debug.ErrorWrapf(err, "Failed to load package at %q", dir))
		}
		if len(p) == 0 {
			fmt.Fprintf(fOut, "package %s\n\n", filepath.Base(dir))
		} else if p[0].Name != "" {
			fmt.Fprintf(fOut, "package %s\n\n", filepath.Base(p[0].Name))
		} else {
			fmt.Fprintf(fOut, "package %s\n\n", filepath.Base(p[0].PkgPath))
		}

		fmt.Fprintf(fOut, "import(\n")
		fmt.Fprintf(fOut, "\t\"goarrg.com/rhi/vxr\"\n")
		fmt.Fprintf(fOut, ")\n\n")
	}

	{
		type returnValue struct {
			key   string
			value any
		}
		vars := []returnValue{}
		if !separateSPV {
			vars = append(vars, returnValue{
				key:   "spv",
				value: spv,
			})
		}
		vars = append(vars, returnValue{
			key:   "layout",
			value: layout,
		})
		if !skipMetadata {
			vars = append(vars, returnValue{
				key:   "meta",
				value: meta,
			})
		}

		{
			fnName := filepath.ToSlash(spv.ID)
			sb := strings.Builder{}
			sb.Grow(len(fnName))
			for _, r := range fnName {
				if unicode.IsDigit(r) || unicode.IsLetter(r) {
					sb.WriteRune(r)
				}
				if r == '/' || r == '.' {
					sb.WriteRune('_')
				}
			}
			fnReturns := ""
			for _, v := range vars {
				fnReturns += fmt.Sprintf("%s %T, ", v.key, v.value)
			}
			fmt.Fprintf(fOut, "func vxrcLoad_%s() (%s) {\n", sb.String(), strings.TrimSuffix(fnReturns, ", "))
		}

		for _, v := range vars {
			fmt.Fprintf(fOut, "\t%s = %#v\n", v.key, v.value)
		}

		fmt.Fprintf(fOut, "\treturn\n")
		fmt.Fprintf(fOut, "}\n")
	}
}
