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

package container

import "slices"

type Stack[E any] struct {
	data []E
}

func (s *Stack[E]) Data() []E {
	return slices.Clone(s.data)
}

func (s *Stack[E]) Resize(l int) {
	if len(s.data) < l {
		s.data = slices.Grow(s.data, l)
	} else {
		s.data = s.data[:l]
	}
}

func (s *Stack[E]) Empty() bool {
	return len(s.data) == 0
}

func (s *Stack[E]) Push(e E) {
	s.data = append(s.data, e)
}

func (s *Stack[E]) Pop() E {
	e := s.data[len(s.data)-1]
	s.data = s.data[:len(s.data)-1]
	return e
}
