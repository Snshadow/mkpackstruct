// originally from https://cs.opensource.google/go/go/+/release-branch.go1.21:src/cmd/compile/internal/types2/predicates.go
// licensed under the BSD-3-Clause license
//
// -- original disclaimer --
// 	Copyright 2012 The Go Authors. All rights reserved.
// 	Use of this source code is governed by a BSD-style
// 	license that can be found in the LICENSE file.
//

//go:build !go1.22

package sizes

import (
	"go/types"
)

// ---- copied unexported function(s) from "go/types" ----

// isTypeParam reports whether t is a type parameter.
func isTypeParam(t types.Type) bool {
	_, ok := t.(*types.TypeParam)
	return ok
}
