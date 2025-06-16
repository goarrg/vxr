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

import "goarrg.com/debug"

type noCopy struct {
	addr *noCopy
}

func (n *noCopy) init() {
	if n.addr != nil {
		abort("init called on non zero value")
	}
	n.addr = n
}

func (n *noCopy) check() {
	if n.addr != n {
		abort("Illegal copy by value or use of zero/dead value: \n%s", debug.StackTrace(0))
	}
}

func (n *noCopy) close() {
	n.addr = nil
}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
