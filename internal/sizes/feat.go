//go:build go1.22

package sizes

import (
	"go/types"
)

// ---- copied unexported functions from "go/types" ----

// isTypeParam reports whether t is a type parameter.
func isTypeParam(t types.Type) bool {
	_, ok := types.Unalias(t).(*types.TypeParam)
	return ok
}
