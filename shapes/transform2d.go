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

package shapes

/*
#include "polygonmode.h"
*/
import "C"

import (
	"math"

	"goarrg.com/gmath"
)

type Pivot uint32

const (
	PivotTopLeft Pivot = iota
	PivotTopRight
	PivotBottomRight
	PivotBottomLeft
	PivotCenter
)

func (p *Pivot) vector() gmath.Vector2f32 {
	switch *p {
	case PivotTopLeft:
		return gmath.Vector2f32{X: -1, Y: -1}
	case PivotTopRight:
		return gmath.Vector2f32{X: 1, Y: -1}
	case PivotBottomRight:
		return gmath.Vector2f32{X: 1, Y: 1}
	case PivotBottomLeft:
		return gmath.Vector2f32{X: -1, Y: 1}
	case PivotCenter:
		return gmath.Vector2f32{}
	default:
		abort("Unknown Pivot: %d", *p)
		return gmath.Vector2f32{}
	}
}

func (p *Pivot) findPoint(m0, m1 gmath.Vector2f32, verts []gmath.Vector2f32) gmath.Vector2f32 {
	pM := gmath.Vector2f32{X: m0.Dot(verts[0]), Y: m1.Dot(verts[0])}
	switch *p {
	case PivotTopLeft:
		for _, v := range verts[1:] {
			pM = pM.Min(gmath.Vector2f32{X: m0.Dot(v), Y: m1.Dot(v)})
		}
	case PivotTopRight:
		for _, v := range verts[1:] {
			pM = gmath.Vector2f32{X: max(pM.X, m0.Dot(v)), Y: min(pM.Y, m1.Dot(v))}
		}
	case PivotBottomRight:
		for _, v := range verts[1:] {
			pM = pM.Max(gmath.Vector2f32{X: m0.Dot(v), Y: m1.Dot(v)})
		}
	case PivotBottomLeft:
		for _, v := range verts[1:] {
			pM = gmath.Vector2f32{X: min(pM.X, m0.Dot(v)), Y: max(pM.Y, m1.Dot(v))}
		}
	case PivotCenter:
		return gmath.Vector2f32{}
	default:
		abort("Unknown Pivot: %d", *p)
	}
	return pM.Abs().Scale(p.vector())
}

type TransformOrder uint32

const (
	// TransformTRS will create a model matrix by effectively doing
	// translation * rotation * scale
	TransformTRS TransformOrder = iota
	// TransformTSR will create a model matrix by effectively doing
	// translation * scale * rotation
	TransformTSR
)

type Transform2D struct {
	Pos            gmath.Point2f32
	Rot            float32
	Size           gmath.Vector2f32
	TransformOrder TransformOrder
	/*
	 TranslationPivot sets where in the object is Pos at.
	 Pivot locations are determined after rotating and scaling
	 the object, so top left always means the top left on the screen.
	*/
	TranslationPivot Pivot
}

func (t *Transform2D) modelMatrix(mode, triangles uint32) [2][3]float32 {
	var m0, m1 gmath.Vector2f32
	var p gmath.Point2f32

	switch t.TransformOrder {
	case TransformTRS:
		m0 = gmath.Vector2f32{
			X: float32(math.Cos(float64(t.Rot))), Y: -float32(math.Sin(float64(t.Rot))),
		}.Scale(t.Size)
		m1 = gmath.Vector2f32{
			X: float32(math.Sin(float64(t.Rot))), Y: float32(math.Cos(float64(t.Rot))),
		}.Scale(t.Size)
	case TransformTSR:
		m0 = gmath.Vector2f32{
			X: float32(math.Cos(float64(t.Rot))), Y: -float32(math.Sin(float64(t.Rot))),
		}.Scale(gmath.Vector2f32{X: t.Size.X, Y: t.Size.X})
		m1 = gmath.Vector2f32{
			X: float32(math.Sin(float64(t.Rot))), Y: float32(math.Cos(float64(t.Rot))),
		}.Scale(gmath.Vector2f32{X: t.Size.Y, Y: t.Size.Y})
	default:
		abort("invalid TransformOrder: %d", t.TransformOrder)
	}

	if t.TranslationPivot == PivotCenter {
		p = t.Pos
	} else {
		pivot := t.TranslationPivot.vector()
		switch mode {
		case C.POLYGON_MODE_REGULAR_CONCAVE:
			switch triangles {
			case 1:
				verts := []gmath.Vector2f32{
					{X: 0.0, Y: -0.5},
					{X: 0.5, Y: 0.5},
					{X: -0.5, Y: 0.5},
				}
				p = t.Pos.Subtract(t.TranslationPivot.findPoint(m0, m1, verts))

			case 2:
				p = gmath.Point2f32{
					X: t.Pos.X - (m0.Abs().Dot(pivot) * 0.5),
					Y: t.Pos.Y - (m1.Abs().Dot(pivot) * 0.5),
				}

			case 3:
				verts := []gmath.Vector2f32{
					{X: 0.0, Y: -0.5},
					{X: 0.43301, Y: 0.25},
					{X: -0.43301, Y: 0.25},
				}
				p = t.Pos.Subtract(t.TranslationPivot.findPoint(m0, m1, verts))

			case 4:
				p = gmath.Point2f32{
					X: t.Pos.X - (m0.Abs().Dot(pivot) * math.Sqrt2 * 0.25),
					Y: t.Pos.Y - (m1.Abs().Dot(pivot) * math.Sqrt2 * 0.25),
				}

			default:
				p = gmath.Point2f32{
					X: t.Pos.X - (float32(math.Sqrt(float64(m0.Scale(m0).Dot(gmath.Vector2f32{X: 0.25, Y: 0.25})))) * pivot.X),
					Y: t.Pos.Y - (float32(math.Sqrt(float64(m1.Scale(m1).Dot(gmath.Vector2f32{X: 0.25, Y: 0.25})))) * pivot.Y),
				}
			}

		case C.POLYGON_MODE_REGULAR_STAR:
			pivot := t.TranslationPivot.vector()
			p = gmath.Point2f32{
				X: t.Pos.X - (float32(math.Sqrt(float64(m0.Scale(m0).Dot(gmath.Vector2f32{X: 0.25, Y: 0.25})))) * pivot.X),
				Y: t.Pos.Y - (float32(math.Sqrt(float64(m1.Scale(m1).Dot(gmath.Vector2f32{X: 0.25, Y: 0.25})))) * pivot.Y),
			}
		}
	}

	return [2][3]float32{
		{m0.X, m0.Y, p.X},
		{m1.X, m1.Y, p.Y},
	}
}
