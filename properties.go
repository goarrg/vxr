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
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"goarrg.com/debug"
	"goarrg.com/gmath"
	"goarrg.com/rhi/vxr/internal/vk"
)

type UUID [16]byte

func (uuid *UUID) String() string {
	return fmt.Sprintf("%08X-%04X-%04X-%04X-%012X", uuid[:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func (uuid *UUID) UnmarshalText(data []byte) error {
	if len(data) != 36 || data[8] != '-' || data[13] != '-' || data[18] != '-' || data[23] != '-' {
		return debug.Errorf("Invalid UUID format")
	}
	var filteredData []byte
	for i, b := range data {
		switch i {
		case 8, 13, 18, 23:
			continue
		}
		filteredData = append(filteredData, b)
	}
	_, err := hex.Decode(uuid[:], filteredData)
	return err
}

type VendorID uint32

const (
	VendorAMD    VendorID = 0x1002
	VendorNVIDIA VendorID = 0x10de
	VendorIntel  VendorID = 0x8086
)

func (id VendorID) String() string {
	switch id {
	case VendorAMD:
		return "AMD"
	case VendorNVIDIA:
		return "NVIDIA"
	case VendorIntel:
		return "Intel"
	default:
		return fmt.Sprintf("Unknown: 0x%04X", uint32(id))
	}
}

type (
	Limits struct {
		PointSize gmath.Bounds[float32]
		LineWidth gmath.Bounds[float32]
		Global    struct {
			MaxAllocationSize         uint64
			MaxMemoryAllocationCount  uint32
			MaxSamplerAllocationCount uint32
		}
		PerDesctiptor struct {
			MaxImageDimension1D   int32
			MaxImageDimension2D   int32
			MaxImageDimension3D   int32
			MaxImageDimensionCube int32
			MaxImageArrayLayers   int32
			MaxSamplerAnisotropy  float32
			MaxUBOSize            uint32
			MaxSBOSize            uint32
		}
		PerStage struct {
			MaxSamplerCount              uint32
			MaxSampledImageCount         uint32
			MaxCombinedImageSamplerCount uint32
			MaxStorageImageCount         uint32

			MaxUBOCount      uint32
			MaxSBOCount      uint32
			MaxResourceCount uint32
		}
		PerPipeline struct {
			MaxSamplerCount              uint32
			MaxSampledImageCount         uint32
			MaxCombinedImageSamplerCount uint32
			MaxStorageImageCount         uint32

			MaxUBOCount uint32
			MaxSBOCount uint32

			MaxBoundDescriptorSets uint32
			MaxPushConstantsSize   uint32
		}
		Compute struct {
			MaxDispatchSize gmath.Extent3u32
			MaxLocalSize    gmath.Extent3u32
			SubgroupSize    gmath.Bounds[uint32]
			Workgroup       struct {
				MaxInvocations   uint32
				MaxSubgroupCount uint32
			}
		}
	}
	Properties struct {
		UUID          UUID
		VendorID      VendorID
		DeviceID      uint32
		DriverVersion uint32
		API           uint32
		Compute       struct {
			SubgroupSize uint32
		}
		Limits            Limits
		EnabledExtensions []string
		EnabledFeatures   map[string]map[string]bool
	}
)

func (p *Properties) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	buff.WriteString(fmt.Sprintf("\"UUID\": %q,", p.UUID.String()))
	buff.WriteString(fmt.Sprintf("\"VendorID\": %q,", p.VendorID.String()))
	buff.WriteString(fmt.Sprintf("\"DeviceID\": %d,", p.DeviceID))
	buff.WriteString(fmt.Sprintf("\"DriverVersion\": %d,", p.DriverVersion))
	buff.WriteString(fmt.Sprintf("\"API\": %q,", vkAPI2String(p.API)))
	buff.WriteString(fmt.Sprintf("\"Compute\": %s,", jsonString(p.Compute)))
	buff.WriteString(fmt.Sprintf("\"Limits\": %s,", jsonString(p.Limits)))
	buff.WriteString(fmt.Sprintf("\"EnabledExtensions\": %s,", jsonString(p.EnabledExtensions)))
	buff.WriteString(fmt.Sprintf("\"EnabledFeatures\": %s,", jsonString(p.EnabledFeatures)))

	buff.Truncate(buff.Len() - 1)
	buff.WriteString("}")
	return buff.Bytes(), nil
}

type colorFormatProperties struct {
	optimalTilingFeatures map[Format]FormatFeatureFlags
}

type depthFormatProperties struct {
	optimalTilingFeatures map[DepthStencilFormat]FormatFeatureFlags
}
type formatProperties struct {
	color colorFormatProperties
	depth depthFormatProperties
}

func (p *formatProperties) MarshalJSON() ([]byte, error) {
	buff := bytes.Buffer{}
	buff.WriteString("{")

	{
		buff.WriteString("\"optimalTilingFeatures\": {")
		hasEntries := false
		// color
		{
			err := mapRunFuncStringSorted(p.color.optimalTilingFeatures, func(k Format, v FormatFeatureFlags) error {
				buff.WriteString(fmt.Sprintf("%q: [", k.String()))
				features := strings.Split(v.String(), "|")
				if len(features) > 0 {
					for _, f := range features {
						buff.WriteString(fmt.Sprintf("%q,", f))
					}
					buff.Truncate(buff.Len() - 1)
				}
				buff.WriteString("],")
				return nil
			})
			if err == nil {
				hasEntries = true
			}
		}
		// depth
		{
			err := mapRunFuncStringSorted(p.depth.optimalTilingFeatures, func(k DepthStencilFormat, v FormatFeatureFlags) error {
				buff.WriteString(fmt.Sprintf("%q: [", k.String()))
				features := strings.Split(v.String(), "|")
				if len(features) > 0 {
					for _, f := range features {
						buff.WriteString(fmt.Sprintf("%q,", f))
					}
					buff.Truncate(buff.Len() - 1)
				}
				buff.WriteString("],")
				return nil
			})
			if err == nil {
				hasEntries = true
			}
		}

		if hasEntries {
			buff.Truncate(buff.Len() - 1)
		}
		buff.WriteString("}")
	}

	buff.WriteString("}")
	return buff.Bytes(), nil
}

func (p *formatProperties) colorFeatures(f Format) FormatFeatureFlags {
	haveFeatures, ok := p.color.optimalTilingFeatures[f]
	if !ok {
		formatProperties := C.VkFormatProperties3{
			sType: vk.STRUCTURE_TYPE_FORMAT_PROPERTIES_3,
		}
		C.vxr_vk_getFormatProperties(instance.cInstance, C.VkFormat(f), &formatProperties)
		haveFeatures = FormatFeatureFlags(formatProperties.optimalTilingFeatures)
		p.color.optimalTilingFeatures[f] = haveFeatures
		instance.logger.VPrintf("Format [%s] has features: %s", f.String(), haveFeatures.String())
	}
	return haveFeatures
}

func (p *formatProperties) depthFeatures(f DepthStencilFormat) FormatFeatureFlags {
	haveFeatures, ok := p.depth.optimalTilingFeatures[f]
	if !ok {
		formatProperties := C.VkFormatProperties3{
			sType: vk.STRUCTURE_TYPE_FORMAT_PROPERTIES_3,
		}
		C.vxr_vk_getFormatProperties(instance.cInstance, C.VkFormat(f), &formatProperties)
		haveFeatures = FormatFeatureFlags(formatProperties.optimalTilingFeatures)
		p.depth.optimalTilingFeatures[f] = haveFeatures
		instance.logger.VPrintf("DepthStencilFormat [%s] has features: %s", f.String(), haveFeatures.String())
	}
	return haveFeatures
}
