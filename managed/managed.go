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

package managed

import (
	"goarrg.com/debug"
)

var instance = struct {
	logger *debug.Logger
}{
	logger: debug.NewLogger("vxr", "managed"),
}

func abort(fmt string, args ...any) {
	instance.logger.EPrintf(fmt, args...)
	panic("Fatal Error")
}

type destroyFunc struct {
	f func()
}

func (d destroyFunc) Destroy() {
	d.f()
}
