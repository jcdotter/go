// Copyright 2023 james dotter.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://github.com/jcdotter/go/LICENSE
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package typ

import "unsafe"

func offset(p unsafe.Pointer, offset uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + offset)
}

func offseti(p unsafe.Pointer, offset int) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + uintptr(offset))
}

func prepScanDest(v Value, dest any) Value {
	d := ValueOf(dest)

	// dest must be a pointer
	if d.Kind() != POINTER {
		panic("typ: not a pointer")
	}
	d = d.Elem()

	// dest must have the capacity
	// to hold the value
	if d.Len() < v.Len() {
		if d.Kind() != SLICE {
			panic("typ: not enough space")
		}
		d.Slice().Extend(v.Len() - d.Len())
	}
	return d
}
