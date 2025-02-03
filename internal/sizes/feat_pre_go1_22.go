//go:build !go1.22

package sizes

import (
	"go/types"
)

// ---- copied or modified unexported functions from "go/types" ----

// isTypeParam reports whether t is a type parameter.
func isTypeParam(t types.Type) bool {
	_, ok := t.(*types.TypeParam)
	return ok
}

// asNamed returns t as *Named if that is t's
// actual type. It returns nil otherwise.
func asNamed(t types.Type) *types.Named {
	n, _ := t.(*types.Named)
	return n
}
