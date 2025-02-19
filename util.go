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

	#include "vxr/vxr.h"
*/
import "C"

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unsafe"

	"goarrg.com/debug"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
)

func toHex(v any) string {
	switch t := v.(type) {
	case C.VkDescriptorType, C.VkFormat, C.VkShaderStageFlagBits,
		C.uint32_t, C.uint64_t:
		return fmt.Sprintf("0x%02X", v)
	case C.VkDescriptorSetLayout:
		return fmt.Sprintf("0x%X", uintptr(unsafe.Pointer(t)))
	case C.VkDescriptorPool:
		return fmt.Sprintf("0x%X", uintptr(unsafe.Pointer(t)))
	case C.VkDescriptorSet:
		return fmt.Sprintf("0x%X", uintptr(unsafe.Pointer(t)))
	case C.VkPipelineLayout:
		return fmt.Sprintf("0x%X", uintptr(unsafe.Pointer(t)))
	case C.VkPipeline:
		return fmt.Sprintf("0x%X", uintptr(unsafe.Pointer(t)))
	case uint64, uintptr:
		return fmt.Sprintf("0x%016X", t)
	}
	abort("Unknown/Unhandled type: %T", v)
	return ""
}

func genID(items ...any) string {
	sb := strings.Builder{}
	for _, i := range items {
		switch t := i.(type) {
		case string:
			sb.WriteString(t)
		case fmt.Stringer:
			sb.WriteString(t.String())
		default:
			sb.WriteString(toHex(i))
		}
		sb.WriteRune(',')
	}
	return "[" + sb.String()[:sb.Len()-1] + "]"
}

func chainIDs(ids ...string) string {
	sb := strings.Builder{}
	for _, id := range ids {
		sb.WriteString(id)
	}
	return strings.Clone(sb.String())
}

func jsonString(target any) string {
	bytes, err := json.Marshal(target)
	if err != nil {
		abort("%s", err)
	}
	return strings.TrimSpace(string(bytes))
}

func prettyString(target json.Marshaler) string {
	bytes, err := json.MarshalIndent(target, "", "    ")
	if err != nil {
		abort("%s", err)
	}
	return strings.TrimSpace(string(bytes))
}

func hasBits[N constraints.Unsigned](t, want N) bool {
	return (t & want) == want
}

func mapRunFuncSorted[M ~map[K]V, K cmp.Ordered, V any](m M, f func(K, V) error) error {
	keys := maps.Keys(m)

	if len(keys) == 0 {
		return debug.Errorf("Empty map")
	}

	slices.Sort(keys)

	for _, k := range keys {
		err := f(k, m[k])
		if err != nil {
			return err
		}
	}

	return nil
}

func mapRunFuncStringSorted[M ~map[K]V, K interface {
	comparable
	fmt.Stringer
}, V any](m M, f func(K, V) error) error {
	var sKeys []string
	skMap := map[string]K{}

	{
		keys := maps.Keys(m)

		if len(keys) == 0 {
			return debug.Errorf("Empty map")
		}

		sKeys = make([]string, len(keys))
		for i, k := range keys {
			sKeys[i] = k.String()
			skMap[sKeys[i]] = k
		}
		slices.Sort(sKeys)
	}

	for _, sk := range sKeys {
		k := skMap[sk]
		err := f(k, m[k])
		if err != nil {
			return err
		}
	}

	return nil
}

func growSlice[S ~[]E, E any](s S, n int) S {
	if n -= cap(s); n > 0 {
		s = append(s[:cap(s)], make([]E, n)...)
	}

	return s
}
