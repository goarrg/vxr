//go:build ignore
// +build ignore

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

/*
	#define VXR_GENERATOR
	#include "libvxr/include/vxr/vxr.h"
*/
import "C"

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	vxr "goarrg.com/rhi/vxr/make"
	"goarrg.com/toolchain"
	"goarrg.com/toolchain/cgodep"
	"goarrg.com/toolchain/golang"
	"golang.org/x/exp/maps"
	"golang.org/x/tools/go/packages"
)

type extension struct {
	name        string
	kind        string
	promoted    string
	provisional bool
	depends     []string
	structs     []string
}

type format struct {
	blockSize   uint
	blockExtent [3]uint
	components  string
}

func main() {
	vxr.InstallGeneratorDeps()

	// vulkan_core.h
	types := parseHeader()
	{
		genInternalConst(types)
	}

	// vk.xml
	extensions, formats := parseXML()
	{
		// TODO: genConst does not filter out provisional extensions
		genConst(types, formats)
		genStruct(types, extensions)
		genExtensions(types, extensions)
		genFeatureReflection(types, extensions)
	}
}

func parseHeader() map[string][]string {
	header := filepath.Join(cgodep.InstallDir("vulkan-headers", toolchain.Target{}, toolchain.BuildRelease), "include", "vulkan", "vulkan_core.h")
	fIn, err := os.Open(header)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(fIn)
	lastType := ""
	types := map[string][]string{}

	scanEnum := func() {
		for scanner.Scan() {
			line := strings.TrimSpace(strings.TrimRight(scanner.Text(), " ,;\n"))
			if strings.Contains(line, "=") {
				types[lastType] = append(types[lastType], line)
			}
			if strings.Contains(line, "MAX_ENUM") {
				return
			}
		}
	}
	scanFlags := func(t string) {
		for scanner.Scan() {
			line := strings.TrimSpace(strings.TrimRight(scanner.Text(), " ,;\n"))
			if strings.Contains(line, "#") || strings.Contains(line, "//") {
				continue
			}
			if !strings.Contains(line, t) {
				return
			}
			if strings.Contains(line, "=") {
				types[lastType] = append(types[lastType], line)
			}
		}
	}
	scanFeatureStruct := func() {
		if lastType == "VkPhysicalDeviceFeatures2" {
			for scanner.Scan(); !strings.Contains(scanner.Text(), "}"); scanner.Scan() {
			}
			return
		}
		if lastType != "VkPhysicalDeviceFeatures" {
			scanner.Scan()
			if !strings.HasPrefix(strings.TrimSpace(scanner.Text()), "VkStructureType") {
				panic(fmt.Sprintf("%s: %s", lastType, scanner.Text()))
			}
			types[lastType] = append(types[lastType], "VkStructureType sType")
			scanner.Scan()
			if !strings.HasPrefix(strings.TrimSpace(scanner.Text()), "void*") {
				panic(scanner.Text())
			}
			types[lastType] = append(types[lastType], "void* pNext")
		}
		for scanner.Scan() {
			str := strings.TrimSuffix(strings.TrimSpace(scanner.Text()), ";")
			if strings.ContainsAny(str, "}") {
				return
			}
			if !strings.HasPrefix(str, "VkBool32") {
				panic(str)
			}
			types[lastType] = append(types[lastType], str)
		}
	}
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), " ,;{\n"))
		{
			t, ok := strings.CutPrefix(line, "typedef enum ")
			if ok {
				lastType = strings.ReplaceAll(t, "FlagBits", "Flags")
				scanEnum()
				continue
			}
		}
		{
			t, ok := strings.CutPrefix(line, "typedef VkFlags64 ")
			if ok && strings.Contains(t, "FlagBits") {
				lastType = strings.ReplaceAll(t, "FlagBits", "Flags")
				scanFlags(t)
				continue
			}
		}
		{
			isAlias := strings.HasPrefix(line, "typedef VkPhysicalDevice")
			isPhysicalDevice := strings.HasPrefix(line, "typedef struct VkPhysicalDevice")
			isFeatures := strings.Contains(line, "Features")
			if isAlias && isFeatures {
				fields := strings.Fields(line)
				blacklist := []string{
					"VkPhysicalDeviceFeatures2KHR",
					"VkPhysicalDeviceFloat16Int8FeaturesKHR",
					"VkPhysicalDeviceShaderDrawParameterFeatures",
					"VkPhysicalDeviceVariablePointerFeatures",
					"VkPhysicalDeviceVariablePointerFeaturesKHR",
				}
				if _, skip := slices.BinarySearch(blacklist, fields[2]); !skip {
					types[fields[2]] = types[fields[1]]
				}
				continue
			} else if isPhysicalDevice && isFeatures {
				lastType = strings.Fields(line)[2]
				scanFeatureStruct()
				continue
			}
		}
	}

	return types
}

func parseXML() ([]extension, map[string]format) {
	docs := filepath.Join(cgodep.InstallDir("vulkan-docs", toolchain.Target{}, toolchain.BuildRelease), "xml", "vk.xml")
	fIn, err := os.Open(docs)
	if err != nil {
		panic(err)
	}

	extensions := []extension{}
	formats := map[string]format{}

	{
		decoder := xml.NewDecoder(fIn)
		findNextElement := func() (xml.StartElement, error) {
			for {
				t, err := decoder.Token()
				if err != nil {
					return xml.StartElement{}, err
				}
				if start, ok := t.(xml.StartElement); ok {
					return start, nil
				}
				if _, ok := t.(xml.EndElement); ok {
					return xml.StartElement{}, io.EOF
				}
			}
		}
		findElementEnd := func() (xml.EndElement, error) {
			started := false
			for {
				t, err := decoder.Token()
				if err != nil {
					return xml.EndElement{}, err
				}
				if _, ok := t.(xml.StartElement); ok {
					started = true
				}
				if end, ok := t.(xml.EndElement); ok {
					if started {
						started = false
						continue
					}
					return end, nil
				}
			}
		}
		findAttribute := func(name string, attrs []xml.Attr) xml.Attr {
			for _, a := range attrs {
				if a.Name.Local == name {
					return a
				}
			}
			return xml.Attr{}
		}

		findExtensions := func() {
			for {
				start, err := findNextElement()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					panic(err)
				}
				{
					rootName := findAttribute("name", start.Attr)
					rootKind := findAttribute("type", start.Attr)
					rootDepends := findAttribute("depends", start.Attr)
					rootSupported := findAttribute("supported", start.Attr)
					rootPromoted := findAttribute("promotedto", start.Attr)
					rootDeprecated := findAttribute("deprecatedby", start.Attr)
					rootProvisional := findAttribute("provisional", start.Attr)

					if !slices.Contains(strings.Split(rootSupported.Value, ","), "vulkan") {
						decoder.Skip()
						continue
					}

					e := extension{
						name:     rootName.Value,
						kind:     rootKind.Value,
						promoted: rootPromoted.Value,
						depends:  strings.Split(strings.NewReplacer("(", "", ")", "", ",", "+").Replace(rootDepends.Value), "+"),
					}
					if rootProvisional.Value == "true" {
						e.provisional = true
					}
					if rootDeprecated.Value != "" {
						if e.promoted != "" {
							panic(fmt.Sprintf("unexpected promotion and deprecation for: %s", rootName.Value))
						}
						e.promoted = rootDeprecated.Value
					}
					for {
						node, err := findNextElement()
						if err != nil {
							if errors.Is(err, io.EOF) {
								break
							}
							panic(err)
						}
						if node.Name.Local == "require" {
							for {
								t, err := findNextElement()
								if err != nil {
									if errors.Is(err, io.EOF) {
										break
									}
									panic(err)
								}
								_, err = findElementEnd()
								if err != nil {
									panic(err)
								}

								if t.Name.Local == "type" {
									typename := findAttribute("name", t.Attr)
									if typename.Name.Local == "name" {
										e.structs = append(e.structs, typename.Value)
									}
								}
							}
						} else {
							_, err = findElementEnd()
							if err != nil {
								panic(err)
							}
						}
					}
					extensions = append(extensions, e)
				}
			}
		}
		findFormats := func() {
			for {
				start, err := findNextElement()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					panic(err)
				}
				{
					name := findAttribute("name", start.Attr)
					blockSize := findAttribute("blockSize", start.Attr)
					blockExtent := findAttribute("blockExtent", start.Attr)

					bs, err := strconv.ParseUint(blockSize.Value, 10, 0)
					if err != nil {
						panic(err)
					}

					be := [3]uint{}
					if blockExtent.Value != "" {
						for i, e := range strings.Split(blockExtent.Value, ",") {
							v, err := strconv.ParseUint(e, 10, 0)
							if err != nil {
								panic(err)
							}
							be[i] = uint(v)
						}
					}

					f := format{
						blockSize:   uint(bs),
						blockExtent: be,
					}

					for {
						component, err := findNextElement()
						if err != nil {
							if errors.Is(err, io.EOF) {
								break
							}
							panic(err)
						}
						_, err = findElementEnd()
						if err != nil {
							panic(err)
						}

						if component.Name.Local == "component" {
							name := findAttribute("name", component.Attr)
							switch name.Value {
							case "R", "G", "B", "A":
								f.components += "COLOR_COMPONENT_" + name.Value + " | "
							case "D", "S":
							default:
								panic(fmt.Sprintf("Unexpected component: %v", name))
							}
						}
					}

					f.components = strings.TrimSuffix(f.components, " | ")
					formats[name.Value] = f
				}
			}
		}

		registry, err := findNextElement()
		if err != nil {
			panic(err)
		}
		if registry.Name.Local != "registry" {
			panic(fmt.Sprintf("unknown xml format %s", registry.Name.Local))
		}
		for {
			next, err := findNextElement()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				panic(err)
			}
			switch next.Name.Local {
			case "extensions":
				findExtensions()

			case "formats":
				findFormats()

			default:
				decoder.Skip()
			}
		}

		if len(extensions) == 0 || len(formats) == 0 {
			panic("failed to parse vk.xml")
		}
	}
	return extensions, formats
}

func genInternalConst(types map[string][]string) {
	p := golang.CallersPackage(packages.NeedModule | packages.NeedName)
	fOut, err := os.Create(filepath.Join(p.Module.Dir, strings.TrimPrefix(p.PkgPath, p.Module.Path), "internal", "vk", "const.go"))
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(fOut, "// go run vk_gen.go\n")
	fmt.Fprintf(fOut, "// Code generated by the command above; DO NOT EDIT.\n\n")
	fmt.Fprintf(fOut, "package vk\n")
	fmt.Fprintf(fOut, `
const(
	WHOLE_SIZE = 1<<64 - 1
	REMAINING_MIP_LEVELS = 1<<32 - 1
	REMAINING_ARRAY_LAYERS = 1<<32 - 1
)
`)
	fmt.Fprintf(fOut, `
// type Bool32 C.VkBool32
const(
	FALSE = 0
	TRUE = 1
)
`)

	typeNames := maps.Keys(types)
	slices.Sort(typeNames)

	for _, k := range typeNames {
		flags := types[k]
		if len(flags) == 0 {
			panic(k)
		}
		if !strings.Contains(flags[0], "=") {
			continue
		}

		fmt.Fprintf(fOut, "\n// type %[1]s C.Vk%[1]s\n", strings.TrimPrefix(k, "Vk"))
		fmt.Fprintf(fOut, "const(\n")

		values := map[string]struct{}{}
		for _, f := range flags {
			parts := strings.Split(f, " = ")
			id := parts[0][strings.Index(f, "VK_")+3:]
			value := parts[1]
			if strings.HasPrefix(value, "VK_") {
				value = strings.TrimPrefix(value, "VK_")
			}
			if strings.HasPrefix(value, "0") {
				value = strings.TrimSuffix(value, "ULL")
			}
			if (id != "") && (value != "") {
				values[value] = struct{}{}
				fmt.Fprintf(fOut, "%s = %s\n", id, value)
			}
		}

		fmt.Fprintf(fOut, ")\n")
	}
}

func genFeatureReflection(types map[string][]string, extensions []extension) {
	p := golang.CallersPackage(packages.NeedModule | packages.NeedName)
	fOut, err := os.Create(filepath.Join(p.Module.Dir, strings.TrimPrefix(p.PkgPath, p.Module.Path),
		"libvxr", "vk", "device", "device_features_reflection.inc"))
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(fOut, "// go run vk_gen.go\n")
	fmt.Fprintf(fOut, "// Code generated by the command above; DO NOT EDIT.\n\n")

	writeField := func(k string, field []string) {
		t := field[0]
		switch t {
		case "VkStructureType":
			t = "vkStructureType"
		case "void*":
			t = "voidPtr"
		case "VkBool32":
			t = "vkBool32"
		}
		fmt.Fprintf(fOut, "\tstructField{type::%[2]s, offsetof(%[1]s, %[3]s), \"%[3]s\"},\n", k, t, field[1])
	}

	{
		structFields := types["VkPhysicalDeviceFeatures"]
		fmt.Fprintf(fOut, "static constexpr internal::structTypeImpl<%d> typeVkPhysicalDeviceFeatures {\n", len(structFields))
		fmt.Fprintf(fOut, "\t\"VkPhysicalDeviceFeatures\", sizeof(VkPhysicalDeviceFeatures),\n")
		for _, line := range structFields {
			field := strings.Fields(line)
			writeField("VkPhysicalDeviceFeatures", field)
		}
		fmt.Fprintf(fOut, "};\n")
	}

	typeNames := maps.Keys(types)
	slices.Sort(typeNames)

	sTypes := map[string]string{}
	provisional := map[string]bool{}
	structs := []string{}

	{
		for _, e := range extensions {
			if !e.provisional {
				continue
			}
			for _, s := range e.structs {
				isPhysicalDevice := strings.HasPrefix(s, "VkPhysicalDevice")
				isFeatures := strings.Contains(s, "Features")
				if !(isPhysicalDevice && isFeatures) {
					continue
				}
				provisional[s] = true
			}
		}
	}

	// map struct name to stype
	for _, t := range types["VkStructureType"] {
		isPhysicalDevice := strings.HasPrefix(t, "VK_STRUCTURE_TYPE_PHYSICAL_DEVICE")
		isFeatures := strings.Contains(t, "FEATURES")
		if !(isPhysicalDevice && isFeatures) {
			continue
		}
		fields := strings.Fields(t)
		if _, err := strconv.Atoi(fields[2]); err == nil {
			structType := "VK" + strings.ReplaceAll(strings.TrimPrefix(fields[0], "VK_STRUCTURE_TYPE"), "_", "")
			if structType == "VKPHYSICALDEVICEFEATURES2" {
				continue
			}
			sTypes[structType] = fields[0]
		}
	}

	// get struct information
	for _, k := range typeNames {
		isPhysicalDevice := strings.HasPrefix(k, "VkPhysicalDevice")
		isFeatures := strings.Contains(k, "Features")
		if !(isPhysicalDevice && isFeatures) {
			continue
		}
		if stype, ok := sTypes[strings.ToUpper(k)]; ok && !provisional[k] {
			structs = append(structs, k)
			structFields := types[k]
			fmt.Fprintf(fOut, "static constexpr internal::structChainTypeImpl<%d> type%s {\n", len(structFields), k)
			fmt.Fprintf(fOut, "\t%[1]s, \"%[2]s\", sizeof(%[2]s),\n", stype, k)
			for _, line := range structFields {
				field := strings.Fields(line)
				writeField(k, field)
			}
			fmt.Fprintf(fOut, "};\n")
		}
	}

	slices.Sort(structs)

	// TypeOf

	fmt.Fprintf(fOut, "[[nodiscard]] static inline const structType* typeOf(VkStructureType sType) noexcept {\n")
	fmt.Fprintf(fOut, "\tswitch(sType){\n")

	for _, t := range structs {
		if sType, ok := sTypes[strings.ToUpper(t)]; ok {
			fmt.Fprintf(fOut, "\t\tcase %s:\n", sType)
			fmt.Fprintf(fOut, "\t\t\treturn &type%s;\n", t)
		}
	}

	fmt.Fprintf(fOut, "\t};\n")
	fmt.Fprintf(fOut, "\treturn nullptr;\n")
	fmt.Fprintf(fOut, "}\n")

	fmt.Fprintf(fOut, "[[nodiscard]] static inline const structType* typeOf(const vkStructureChain* ptr) noexcept {\n")
	fmt.Fprintf(fOut, "\treturn typeOf(ptr->sType);\n")
	fmt.Fprintf(fOut, "}\n")

	fmt.Fprintf(fOut, "[[nodiscard]] static inline const structType* typeOf(const VkPhysicalDeviceFeatures*) noexcept {\n")
	fmt.Fprintf(fOut, "\treturn &typeVkPhysicalDeviceFeatures;\n")
	fmt.Fprintf(fOut, "}\n")

	// ValueOf

	fmt.Fprintf(fOut, "[[nodiscard]] static inline vxr::std::smartPtr<structValue> valueOf(vkStructureChain* ptr) noexcept {\n")
	fmt.Fprintf(fOut, "\tswitch(ptr->sType){\n")

	for _, t := range structs {
		if sType, ok := sTypes[strings.ToUpper(t)]; ok {
			fmt.Fprintf(fOut, "\t\tcase %s:\n", sType)
			fmt.Fprintf(fOut, "\t\t\treturn new (::std::nothrow) internal::structValueImpl<type%[1]s.numField()>{&type%[1]s, ptr};\n", t)
		}
	}

	fmt.Fprintf(fOut, "\t};\n")
	fmt.Fprintf(fOut, "\treturn nullptr;\n")
	fmt.Fprintf(fOut, "}\n")

	fmt.Fprintf(fOut, "[[nodiscard]] static inline vxr::std::smartPtr<structValue> valueOf(VkPhysicalDeviceFeatures* ptr) noexcept {\n")
	fmt.Fprintf(fOut, "\treturn new (::std::nothrow) internal::structValueImpl<typeVkPhysicalDeviceFeatures.numField()>{&typeVkPhysicalDeviceFeatures, ptr};\n")
	fmt.Fprintf(fOut, "}\n")
}

func genConst(types map[string][]string, formats map[string]format) {
	p := golang.CallersPackage(packages.NeedModule | packages.NeedName)
	fOut, err := os.Create(filepath.Join(p.Module.Dir, strings.TrimPrefix(p.PkgPath, p.Module.Path), "zvk_const.go"))
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(fOut, "// go run vk_gen.go\n")
	fmt.Fprintf(fOut, "// Code generated by the command above; DO NOT EDIT.\n\n")
	fmt.Fprintf(fOut, "package vxr\n")
	fmt.Fprintf(fOut, `
/*
	#cgo pkg-config: vxr

	#include "vxr/vxr.h"
*/
import "C"
import (
	"strings"

	"goarrg.com/gmath"
)
`)

	// VkResult
	{
		{
			process := func(line string) (string, string) {
				parts := strings.Split(line, " = ")
				identifier := parts[0][strings.Index(line, "VK_")+3:]
				value := parts[1]
				if strings.HasPrefix(value, "VK_") {
					return "", ""
				}
				if strings.HasPrefix(value, "0") {
					value = strings.TrimSuffix(value, "ULL")
				}
				return identifier, value
			}

			fmt.Fprintf(fOut, "\nfunc vkResultStr(v C.VkResult) string {\n")
			fmt.Fprintf(fOut, "\tswitch v {\n")

			values := map[string]struct{}{}
			for _, f := range types["VkResult"] {
				id, value := process(f)
				if !strings.Contains(id, "MAX_ENUM") {
					if _, seen := values[value]; (id != "") && (value != "") && (!seen) {
						values[value] = struct{}{}
						fmt.Fprintf(fOut, "\tcase %s:\n\t\treturn %q\n", value, id)
					}
				}
			}

			fmt.Fprintf(fOut, "\t}\n\tabort(\"Unknown VkResult: %%d\", v)\n\treturn \"\"\n")
			fmt.Fprintf(fOut, "}\n")
		}
	}

	// flags
	{
		processFlagType := func(cT, gT string, isBitflag bool, process func(string) (string, string, bool), toString func(string) string) {
			identifiers := []string{}

			if len(types[cT]) == 0 {
				panic(fmt.Sprintf("Empty flags for type: %s", cT))
			}

			{
				fmt.Fprintf(fOut, "\ntype %s C.%s\n", gT, cT)
				fmt.Fprintf(fOut, "const(\n")

				values := map[string]string{}
				for _, f := range types[cT] {
					id, value, alias := process(f)
					if (id != "") && (value != "") {
						if _, seen := values[value]; !seen {
							if !alias {
								identifiers = append(identifiers, id)
								values[value] = id
							}
							fmt.Fprintf(fOut, "%s %s = %s\n", id, gT, value)
						} else {
							fmt.Fprintf(fOut, "%s %s = %s\n", id, gT, values[value])
						}
					}
				}

				fmt.Fprintf(fOut, ")\n")
			}

			if isBitflag {
				{
					fmt.Fprintf(fOut, "\nfunc (v %[1]s) HasBits(want %[1]s) bool {\n", gT)
					fmt.Fprintf(fOut, "\treturn (v & want) == want\n")
					fmt.Fprintf(fOut, "}\n")
				}

				{
					fmt.Fprintf(fOut, "\nfunc (v %s) String() string {\n", gT)
					fmt.Fprintf(fOut, "\tstr := \"\"\n")
					for _, i := range identifiers {
						if !strings.Contains(i, "MAX_ENUM") && !strings.Contains(i, "MASK") {
							fmt.Fprintf(fOut, "\tif v.HasBits(%s) {\n\t\tstr += \"%s|\"\n\t}\n", i, toString(i))
						}
					}
					fmt.Fprintf(fOut, "\treturn strings.TrimSuffix(str, \"|\")\n")
					fmt.Fprintf(fOut, "}\n")
				}
			} else {
				{
					fmt.Fprintf(fOut, "\nfunc (v %s) String() string {\n", gT)
					fmt.Fprintf(fOut, "\tswitch v {\n")
					for _, i := range identifiers {
						if !strings.Contains(i, "MAX_ENUM") && !strings.Contains(i, "MASK") {
							fmt.Fprintf(fOut, "\tcase %s: \n\t\treturn \"%s\"\n", i, toString(i))
						}
					}
					fmt.Fprintf(fOut, "\t}\n\tabort(\"Unknown %s: %%d\", v)\n\treturn \"\"\n", gT)
					fmt.Fprintf(fOut, "}\n")
				}
			}
		}

		processFlagType("VkFormatFeatureFlags2", "FormatFeatureFlags", true,
			func(line string) (string, string, bool) {
				parts := strings.Split(line, " = ")
				identifier := strings.ReplaceAll(parts[0][strings.Index(line, "VK_")+3:], "FORMAT_FEATURE_2", "FORMAT_FEATURE")
				value := parts[1]
				alias := false
				if strings.HasPrefix(value, "VK_") {
					value = strings.ReplaceAll(value[strings.Index(value, "VK_")+3:], "_BIT", "")
					alias = true
				}
				if strings.HasPrefix(value, "0") {
					value = strings.TrimSuffix(value, "ULL")
				}
				return strings.ReplaceAll(identifier, "_BIT", ""), value, alias
			},
			func(id string) string {
				return strings.TrimPrefix(id, "FORMAT_FEATURE_")
			},
		)

		processFlagType("VkImageCreateFlags", "ImageCreateFlags", true,
			func(line string) (string, string, bool) {
				parts := strings.Split(line, " = ")
				identifier := parts[0][strings.Index(line, "VK_")+3:]
				value := parts[1]
				alias := false
				if strings.HasPrefix(value, "VK_") {
					value = strings.ReplaceAll(value[strings.Index(value, "VK_")+3:], "_BIT", "")
					alias = true
				}
				if strings.HasPrefix(value, "0") {
					value = strings.TrimSuffix(value, "ULL")
				}
				return strings.ReplaceAll(identifier, "_BIT", ""), value, alias
			},
			func(id string) string {
				return strings.TrimPrefix(id, "IMAGE_CREATE_")
			},
		)

		processFlagType("VkImageViewCreateFlags", "ImageViewCreateFlags", true,
			func(line string) (string, string, bool) {
				parts := strings.Split(line, " = ")
				identifier := parts[0][strings.Index(line, "VK_")+3:]
				value := parts[1]
				alias := false
				if strings.HasPrefix(value, "VK_") {
					value = strings.ReplaceAll(value[strings.Index(value, "VK_")+3:], "_BIT", "")
					alias = true
				}
				if strings.HasPrefix(value, "0") {
					value = strings.TrimSuffix(value, "ULL")
				}
				return strings.ReplaceAll(identifier, "_BIT", ""), value, alias
			},
			func(id string) string {
				return strings.TrimPrefix(id, "IMAGE_VIEW_CREATE_")
			},
		)

		processFlagType("VkColorComponentFlags", "ColorComponentFlags", true,
			func(line string) (string, string, bool) {
				parts := strings.Split(line, " = ")
				identifier := parts[0][strings.Index(line, "VK_")+3:]
				value := parts[1]
				alias := false
				if strings.HasPrefix(value, "VK_") {
					value = strings.ReplaceAll(value[strings.Index(value, "VK_")+3:], "_BIT", "")
					alias = true
				}
				if strings.HasPrefix(value, "0") {
					value = strings.TrimSuffix(value, "ULL")
				}
				return strings.ReplaceAll(identifier, "_BIT", ""), value, alias
			},
			func(id string) string {
				switch strings.TrimPrefix(id, "COLOR_COMPONENT_") {
				case "R":
					return "Red"
				case "G":
					return "Green"
				case "B":
					return "Blue"
				case "A":
					return "Alpha"
				default:
					panic("unknown component: " + id)
				}
			},
		)

		processFlagType("VkBlendFactor", "BlendFactor", false,
			func(line string) (string, string, bool) {
				parts := strings.Split(line, " = ")
				identifier := parts[0][strings.Index(line, "VK_")+3:]
				value := parts[1]
				alias := false
				if strings.HasPrefix(value, "VK_") {
					value = value[strings.Index(value, "VK_")+3:]
					alias = true
				}
				if strings.HasPrefix(value, "0") {
					value = strings.TrimSuffix(value, "ULL")
				}
				return identifier, value, alias
			},
			func(id string) string {
				return strings.TrimPrefix(id, "BLEND_FACTOR_")
			},
		)

		processFlagType("VkBlendOp", "BlendOp", false,
			func(line string) (string, string, bool) {
				parts := strings.Split(line, " = ")
				identifier := parts[0][strings.Index(line, "VK_")+3:]
				value := parts[1]
				alias := false
				if strings.HasPrefix(value, "VK_") {
					value = value[strings.Index(value, "VK_")+3:]
					alias = true
				}
				if strings.HasPrefix(value, "0") {
					value = strings.TrimSuffix(value, "ULL")
				}
				return identifier, value, alias
			},
			func(id string) string {
				return strings.TrimPrefix(id, "BLEND_OP_")
			},
		)

		processFlagType("VkIndexType", "IndexType", false,
			func(line string) (string, string, bool) {
				parts := strings.Split(line, " = ")
				identifier := parts[0][strings.Index(line, "VK_")+3:]
				value := parts[1]
				alias := false
				if strings.HasPrefix(value, "VK_") {
					value = value[strings.Index(value, "VK_")+3:]
					alias = true
				}
				if strings.HasPrefix(value, "0") {
					value = strings.TrimSuffix(value, "ULL")
				}
				return identifier, value, alias
			},
			func(id string) string {
				return strings.TrimPrefix(id, "INDEX_TYPE_")
			},
		)
	}

	// VkFormat
	{
		identifiers := []string{}

		{
			process := func(line string) (string, string, bool) {
				parts := strings.Split(line, " = ")
				identifier := parts[0][strings.Index(line, "VK_")+3:]
				value := parts[1]
				alias := false

				if strings.Contains(identifier, "_D") || strings.Contains(identifier, "_S8") {
					identifier = "DEPTH_STENCIL_" + identifier
					if strings.HasPrefix(value, "VK_") {
						value = "DEPTH_STENCIL_" + value[strings.Index(value, "VK_")+3:]
						alias = true
					}
				} else if strings.Contains(identifier, "PLANE_") {
					return "", "", false
				} else if identifier == "FORMAT_UNDEFINED" {
					return "", "", false
				} else if strings.HasPrefix(value, "VK_") {
					value = value[strings.Index(value, "VK_")+3:]
					alias = true
				}
				if strings.HasPrefix(value, "0") {
					value = strings.TrimSuffix(value, "ULL")
				}
				return identifier, value, alias
			}

			fmt.Fprintf(fOut, "\ntype Format C.VkFormat\n")
			fmt.Fprintf(fOut, "type DepthStencilFormat C.VkFormat\n")
			fmt.Fprintf(fOut, "const(\n")
			for _, f := range types["VkFormat"] {
				id, value, alias := process(f)
				if (id != "") && (value != "") {
					if !alias {
						identifiers = append(identifiers, id)
					}
					if strings.HasPrefix(id, "DEPTH_STENCIL_FORMAT_") {
						fmt.Fprintf(fOut, "%s DepthStencilFormat = %s\n", id, value)
					} else {
						fmt.Fprintf(fOut, "%s Format = %s\n", id, value)
					}
				}
			}
			fmt.Fprintf(fOut, ")\n")
		}

		{
			fmt.Fprintf(fOut, "\nfunc (v Format) HasFeatures(want FormatFeatureFlags) bool {\n")
			fmt.Fprintf(fOut, "\treturn instance.formatProperties.colorFeatures(v).HasBits(want) \n")
			fmt.Fprintf(fOut, "}\n")
		}

		{
			fmt.Fprintf(fOut, "\nfunc (v Format) String() string {\n")
			fmt.Fprintf(fOut, "\tswitch v {\n")
			fmt.Fprintf(fOut, "\tcase 0:\n\t\treturn \"UNDEFINED\"\n")
			for _, i := range identifiers {
				if !strings.HasPrefix(i, "DEPTH_STENCIL_FORMAT_") && !strings.Contains(i, "MAX_ENUM") {
					fmt.Fprintf(fOut, "\tcase %s:\n\t\treturn %q\n", i, strings.TrimPrefix(i, "FORMAT_"))
				}
			}
			fmt.Fprintf(fOut, "\t}\n\tabort(\"Unknown format: %%d\", v)\n\treturn \"\"\n")
			fmt.Fprintf(fOut, "}\n")
		}

		{
			fmt.Fprintf(fOut, "\nfunc (v Format) BlockSize() int32 {\n")
			fmt.Fprintf(fOut, "\tswitch v {\n")
			for _, i := range identifiers {
				if !strings.HasPrefix(i, "DEPTH_STENCIL_FORMAT_") && !strings.Contains(i, "MAX_ENUM") {
					fmt.Fprintf(fOut, "\tcase %s:\n\t\treturn %d\n", i, formats["VK_"+i].blockSize)
				}
			}
			fmt.Fprintf(fOut, "\t}\n\tabort(\"Unknown format: %%d\", v)\n\treturn 0\n")
			fmt.Fprintf(fOut, "}\n")
		}

		{
			fmt.Fprintf(fOut, "\nfunc (v Format) ColorComponentFlags() ColorComponentFlags {\n")
			fmt.Fprintf(fOut, "\tswitch v {\n")
			for _, i := range identifiers {
				if !strings.HasPrefix(i, "DEPTH_STENCIL_FORMAT_") && !strings.Contains(i, "MAX_ENUM") {
					fmt.Fprintf(fOut, "\tcase %s:\n\t\treturn %s\n", i, formats["VK_"+i].components)
				}
			}
			fmt.Fprintf(fOut, "\t}\n\tabort(\"Unknown format: %%d\", v)\n\treturn 0\n")
			fmt.Fprintf(fOut, "}\n")
		}

		{
			fmt.Fprintf(fOut, "\nfunc (v Format) BlockExtent() gmath.Extent3i32 {\n")
			fmt.Fprintf(fOut, "\tswitch v {\n")
			for _, i := range identifiers {
				if !strings.HasPrefix(i, "DEPTH_STENCIL_FORMAT_") {
					e := formats["VK_"+i].blockExtent
					if e != [3]uint{} {
						fmt.Fprintf(fOut, "\tcase %s:\n\t\treturn gmath.Extent3i32{X: %d, Y: %d, Z: %d}\n", i, e[0], e[1], e[2])
					}
				}
			}
			fmt.Fprintf(fOut, "\tdefault:\n\t\treturn gmath.Extent3i32{X: 1, Y: 1, Z: 1}\n")
			fmt.Fprintf(fOut, "\t}\n")
			fmt.Fprintf(fOut, "}\n")
		}

		{
			fmt.Fprintf(fOut, "\nfunc (v DepthStencilFormat) HasFeatures(want FormatFeatureFlags) bool {\n")
			fmt.Fprintf(fOut, "\treturn instance.formatProperties.depthFeatures(v).HasBits(want) \n")
			fmt.Fprintf(fOut, "}\n")
		}

		{
			fmt.Fprintf(fOut, "\nfunc (v DepthStencilFormat) String() string {\n")
			fmt.Fprintf(fOut, "\tswitch v {\n")
			fmt.Fprintf(fOut, "\tcase 0:\n\t\treturn \"UNDEFINED\"\n")
			for _, i := range identifiers {
				if strings.HasPrefix(i, "DEPTH_STENCIL_FORMAT_") {
					fmt.Fprintf(fOut, "\tcase %s:\n\t\treturn %q\n", i, strings.TrimPrefix(i, "DEPTH_STENCIL_FORMAT_"))
				}
			}
			fmt.Fprintf(fOut, "\t}\n\tabort(\"Unknown depth stencil format: %%d\", v)\n\treturn \"\"\n")
			fmt.Fprintf(fOut, "}\n")
		}
	}
}

func genStruct(types map[string][]string, extensions []extension) {
	// parse vk.xml
	extensionNameMap := map[string]extension{}
	structExtensionMap := map[string]extension{}
	{
		for _, e := range extensions {
			for _, s := range e.structs {
				isPhysicalDevice := strings.HasPrefix(s, "VkPhysicalDevice")
				isFeatures := strings.Contains(s, "Features")
				if !(isPhysicalDevice && isFeatures) {
					continue
				}
				structExtensionMap[s] = e
			}
			extensionNameMap[e.name] = e
		}
	}

	// parse vulkan_core.h
	sTypes := map[string]string{}
	structs := []string{}
	{
		// map struct name to stype
		for _, t := range types["VkStructureType"] {
			isPhysicalDevice := strings.HasPrefix(t, "VK_STRUCTURE_TYPE_PHYSICAL_DEVICE")
			isFeatures := strings.Contains(t, "FEATURES")
			if !(isPhysicalDevice && isFeatures) {
				continue
			}
			fields := strings.Fields(t)
			structType := "VK" + strings.ReplaceAll(strings.TrimPrefix(fields[0], "VK_STRUCTURE_TYPE"), "_", "")
			if structType == "VKPHYSICALDEVICEFEATURES2" {
				continue
			}
			sTypes[structType] = fields[0]
		}
		// list of structs
		for k := range types {
			isPhysicalDevice := strings.HasPrefix(k, "VkPhysicalDevice")
			isFeatures := strings.Contains(k, "Features")
			if !(isPhysicalDevice && isFeatures) {
				continue
			}
			if k == "VkPhysicalDeviceFeatures2" {
				continue
			}
			{
				e := structExtensionMap[k]
				if e.provisional {
					continue
				}
				for e.promoted != "" && !strings.HasPrefix(e.promoted, "VK_VERSION_") {
					e = extensionNameMap[e.promoted]
				}
				if strings.HasPrefix(e.promoted, "VK_VERSION_") {
					versionPair := strings.Split(strings.TrimPrefix(e.promoted, "VK_VERSION_"), "_")
					major, err := strconv.ParseUint(versionPair[0], 10, 0)
					if err != nil {
						panic(err)
					}
					minor, err := strconv.ParseUint(versionPair[1], 10, 0)
					if err != nil {
						panic(err)
					}
					if uint64(C.VXR_VK_MIN_API) >= (major<<22)|(minor<<12) {
						continue
					}
				}
			}
			structs = append(structs, k)
		}
		slices.Sort(structs)
	}

	{
		p := golang.CallersPackage(packages.NeedModule | packages.NeedName)
		fOut, err := os.Create(filepath.Join(p.Module.Dir, strings.TrimPrefix(p.PkgPath, p.Module.Path),
			"zvk_struct.go"))
		if err != nil {
			panic(err)
		}

		fmt.Fprintf(fOut, "// go run vk_gen.go\n")
		fmt.Fprintf(fOut, "// Code generated by the command above; DO NOT EDIT.\n\n")
		fmt.Fprintf(fOut, "package vxr\n")
		fmt.Fprintf(fOut, `
/*
	#cgo pkg-config: vxr

	#include "vxr/vxr.h"
*/
import "C"
import "goarrg.com/rhi/vxr/internal/vk"
`)

		{
			fmt.Fprintf(fOut, "\ntype VkFeatureStruct interface {\n")
			fmt.Fprintf(fOut, "\tsType() C.VkStructureType\n")
			fmt.Fprintf(fOut, "\tenabledList() []C.size_t\n")
			fmt.Fprintf(fOut, "\textension() string\n")
			fmt.Fprintf(fOut, "}\n")
		}

		{
			for _, s := range structs {
				typename := strings.ReplaceAll(s, "Ycbcr", "YCbCr")
				fmt.Fprintf(fOut, "type %s struct {\n", typename)

				offset := 0
				features := []string{}
				for i, line := range types[s] {
					field := strings.Fields(line)
					if field[0] == "VkBool32" {
						member := field[1]
						if strings.HasPrefix(member, "ycbcr") {
							member = "YCbCr" + strings.TrimPrefix(member, "ycbcr")
						} else {
							member = strings.ToUpper(string(member[0])) + string(member[1:])
						}
						member = strings.ReplaceAll(member, "Ycbcr", "YCbCr")
						member = strings.ReplaceAll(member, "plane", "Plane")
						features = append(features, member)
						fmt.Fprintf(fOut, "\t%s bool\n", member)
					} else {
						offset = i + 1
					}
				}
				fmt.Fprintf(fOut, "}\n")

				fmt.Fprintf(fOut, "func (%s) extension() string {\n", typename)
				fmt.Fprintf(fOut, "\treturn %q\n", structExtensionMap[s].name)
				fmt.Fprintf(fOut, "}\n")

				fmt.Fprintf(fOut, "func (%s) sType() C.VkStructureType {\n", typename)
				if sType, ok := sTypes[strings.ToUpper(s)]; ok {
					fmt.Fprintf(fOut, "\treturn vk.%s\n", strings.TrimPrefix(sType, "VK_"))
				} else {
					if s != "VkPhysicalDeviceFeatures" {
						panic(s)
					}
					fmt.Fprintf(fOut, "\treturn vk.STRUCTURE_TYPE_PHYSICAL_DEVICE_FEATURES_2\n")
				}
				fmt.Fprintf(fOut, "}\n")

				fmt.Fprintf(fOut, "func (s %s) enabledList() []C.size_t {\n", typename)
				fmt.Fprintf(fOut, "\tlist := make([]C.size_t, 0, %d)\n", len(features))
				for i, f := range features {
					fmt.Fprintf(fOut, "\tif s.%s {\n", f)
					fmt.Fprintf(fOut, "\t\tlist = append(list, %d)\n", i+offset)
					fmt.Fprintf(fOut, "\t}\n")
				}
				fmt.Fprintf(fOut, "\treturn list\n")
				fmt.Fprintf(fOut, "}\n")
			}
		}
	}
}

func genExtensions(_ map[string][]string, extensions []extension) {
	p := golang.CallersPackage(packages.NeedModule | packages.NeedName)
	fOut, err := os.Create(filepath.Join(p.Module.Dir, strings.TrimPrefix(p.PkgPath, p.Module.Path),
		"zvk_extension.go"))
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(fOut, "// go run vk_gen.go\n")
	fmt.Fprintf(fOut, "// Code generated by the command above; DO NOT EDIT.\n\n")
	fmt.Fprintf(fOut, "package vxr\n")
	fmt.Fprintf(fOut, "\nimport \"slices\"\n")

	// process depends
	{
		extensionMap := map[string][]string{}
		for _, e := range extensions {
			if e.kind == "device" && !e.provisional {
				extensionMap[e.name] = e.depends
			}
		}
		for k, e := range extensionMap {
			filteredDepends := []string{}
			for _, d := range e {
				if _, ok := extensionMap[d]; ok {
					filteredDepends = append(filteredDepends, d)
				}
			}
			extensionMap[k] = filteredDepends
		}

		fmt.Fprintf(fOut, "\nfunc getExtensionDependencies(name string) []string {\n")
		fmt.Fprintf(fOut, "\tswitch (name) {\n")
		{
			keys := maps.Keys(extensionMap)
			slices.Sort(keys)
			for _, e := range keys {
				if len(extensionMap[e]) > 0 {
					fmt.Fprintf(fOut, "\tcase %q:\n", e)
					fmt.Fprintf(fOut, "\t\treturn %#v\n", extensionMap[e])
				}
			}
		}
		fmt.Fprintf(fOut, "\t}\n")
		fmt.Fprintf(fOut, "\treturn nil\n")
		fmt.Fprintf(fOut, "}\n")
	}

	// process promotion
	{
		extensionMap := map[string]uint64{}
		for _, e := range extensions {
			if e.kind == "device" && !e.provisional {
				if e.promoted != "" && strings.HasPrefix(e.promoted, "VK_VERSION_") {
					versionPair := strings.Split(strings.TrimPrefix(e.promoted, "VK_VERSION_"), "_")
					major, err := strconv.ParseUint(versionPair[0], 10, 0)
					if err != nil {
						panic(err)
					}
					minor, err := strconv.ParseUint(versionPair[1], 10, 0)
					if err != nil {
						panic(err)
					}
					extensionMap[e.name] = (major << 22) | (minor << 12)
				}
			}
		}

		fmt.Fprintf(fOut, "\nfunc filterCorePromotedExtensions(version uint32, extensions []string) []string {\n")
		fmt.Fprintf(fOut, "\tslices.Sort(extensions)\n")
		fmt.Fprintf(fOut, "\textensions = slices.DeleteFunc(extensions, func(e string) bool {\n")
		fmt.Fprintf(fOut, "\t\tswitch e {\n")
		{
			keys := maps.Keys(extensionMap)
			slices.Sort(keys)
			for _, e := range keys {
				fmt.Fprintf(fOut, "\t\t\tcase %q:\n", e)
				fmt.Fprintf(fOut, "\t\t\t\treturn version >= %d\n", extensionMap[e])
			}
		}
		fmt.Fprintf(fOut, "\t\t}\n")
		fmt.Fprintf(fOut, "\t\treturn false\n")
		fmt.Fprintf(fOut, "\t})\n")
		fmt.Fprintf(fOut, "\treturn slices.Clip(extensions)\n")
		fmt.Fprintf(fOut, "}\n")
	}
}
