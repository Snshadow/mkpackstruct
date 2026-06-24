// sizes implements "go/types".Sizes with some change in code referenced from
// std package "go/types" which is licensed under BSD-3-Clause, which is located at
// https://cs.opensource.google/go/go/+/master:src/cmd/compile/internal/types2/sizes.go
//
// -- original disclaimer --
// 	Copyright 2013 The Go Authors. All rights reserved.
// 	Use of this source code is governed by a BSD-style
// 	license that can be found in the LICENSE file.
//

package sizes

import (
	"fmt"
	"go/types"
	"runtime"
	"unsafe"
)

const (
	DefaultPackAlign int64 = 1
)

type NamedPackFunc func(*types.TypeName) int64

// PackedSizes is similar to StdSize from "go/types", except struct member
// alignment is capped by the current pack alignment.
//
// *PackedSizes implements [types.Sizes].
type PackedSizes struct {
	wordSize  int64
	maxAlign  int64
	namedPack NamedPackFunc
}

func NewPackedSizes(wordSize int64) *PackedSizes {
	return NewPackedSizesWithNamedPack(wordSize, nil)
}

func NewPackedSizesWithNamedPack(wordSize int64, namedPack NamedPackFunc) *PackedSizes {
	if wordSize == 0 {
		wordSize = int64(unsafe.Sizeof(uintptr(0)))
	}

	return &PackedSizes{
		wordSize:  wordSize,
		maxAlign:  DefaultPackAlign,
		namedPack: namedPack,
	}
}

func (s *PackedSizes) WithMaxAlign(maxAlign int64) *PackedSizes {
	cp := *s
	cp.maxAlign = maxAlign
	return &cp
}

func (s *PackedSizes) Alignof(T types.Type) (result int64) {
	if named, ok := T.(*types.Named); ok {
		if st, ok := named.Underlying().(*types.Struct); ok {
			return s.WithMaxAlign(s.packAlignOfNamed(named.Obj())).structAlign(st)
		}
		return s.Alignof(named.Underlying())
	}

	switch t := T.Underlying().(type) {
	case *types.Array:
		return s.Alignof(t.Elem())
	case *types.Struct:
		return s.structAlign(t)
	}

	std := &types.StdSizes{WordSize: s.wordSize, MaxAlign: min(s.wordSize, 8)}
	if a := std.Alignof(T); a > 0 {
		return min(a, s.maxAlign)
	}
	return min(s.wordSize, s.maxAlign)
}

func (s *PackedSizes) Offsetsof(fields []*types.Var) []int64 {
	offsets := make([]int64, len(fields))
	var offs int64
	for i, f := range fields {
		if offs < 0 {
			// all remaining offsets are too large
			offsets[i] = -1
			continue
		}
		// offs >= 0
		a := min(s.Alignof(f.Type()), s.maxAlign)
		offs = align(offs, a) // possibly < 0 if align overflows
		offsets[i] = offs
		if d := s.Sizeof(f.Type()); d >= 0 && offs >= 0 {
			offs += d // ok to overflow to < 0
		} else {
			offs = -1 // f.typ or offs is too large
		}
	}
	return offsets
}

var basicSizes = [...]byte{
	types.Bool:       1,
	types.Int8:       1,
	types.Int16:      2,
	types.Int32:      4,
	types.Int64:      8,
	types.Uint8:      1,
	types.Uint16:     2,
	types.Uint32:     4,
	types.Uint64:     8,
	types.Float32:    4,
	types.Float64:    8,
	types.Complex64:  8,
	types.Complex128: 16,
}

func (s *PackedSizes) Sizeof(T types.Type) int64 {
	if named, ok := T.(*types.Named); ok {
		if _, ok := named.Underlying().(*types.Struct); ok {
			return s.WithMaxAlign(s.packAlignOfNamed(named.Obj())).Sizeof(named.Underlying())
		}
		return s.Sizeof(named.Underlying())
	}

	switch t := T.Underlying().(type) {
	case *types.Basic:
		assert(isTyped(T))
		k := t.Kind()
		if int(k) < len(basicSizes) {
			if s := basicSizes[k]; s > 0 {
				return int64(s)
			}
		}
		if k == types.String {
			return s.wordSize * 2
		}
	case *types.Array:
		n := t.Len()
		if n <= 0 {
			return 0
		}
		// n > 0
		esize := s.Sizeof(t.Elem())
		if esize < 0 {
			return -1 // element too large
		}
		if esize == 0 {
			return 0 // 0-size element
		}
		// esize > 0
		a := s.Alignof(t.Elem())
		ea := align(esize, a) // possibly < 0 if align overflows
		if ea < 0 {
			return -1
		}
		// ea >= 1
		n1 := n - 1 // n1 >= 0
		// Final size is ea*n1 + esize; and size must be <= maxInt64.
		const maxInt64 = 1<<63 - 1
		if n1 > 0 && ea > maxInt64/n1 {
			return -1 // ea*n1 overflows
		}
		return ea*n1 + esize // may still overflow to < 0 which is ok
	case *types.Slice:
		return s.wordSize * 3
	case *types.Struct:
		n := t.NumFields()
		if n == 0 {
			return 0
		}
		offsets := s.Offsetsof(getFields(t))
		offs := offsets[n-1]
		size := s.Sizeof(t.Field(n - 1).Type())
		if offs < 0 || size < 0 {
			return -1 // type too large
		}
		return align(offs+size, s.structAlign(t)) // may overflow to < 0 which is ok
	case *types.Interface:
		// type parameters lead to variable sizes/alignments;
		// stdSizes.Sizeof won't be called for them;
		assert(!isTypeParam(T))
		return s.wordSize * 2
	case *types.TypeParam, *types.Union:
		panic("unreachable")
	}
	return s.wordSize // catch-all
}

func (s *PackedSizes) structAlign(st *types.Struct) int64 {
	var max int64 = 1
	for i := 0; i < st.NumFields(); i++ {
		if a := min(s.Alignof(st.Field(i).Type()), s.maxAlign); a > max {
			max = a
		}
	}
	return max
}

func (s *PackedSizes) packAlignOfNamed(obj *types.TypeName) int64 {
	if s.namedPack == nil {
		return DefaultPackAlign
	}
	if a := s.namedPack(obj); a != 0 {
		return a
	}
	return DefaultPackAlign
}

// ---- copied unexported functions from "go/types" ----

// isTyped reports whether t is typed; i.e., not an untyped
// constant or boolean.
// safe to call from types that are not fully set up
func isTyped(t types.Type) bool {
	// Alias and named types cannot denote untyped types
	// so there's no need to call Unalias or under, below.
	b, _ := t.(*types.Basic)
	return b == nil || b.Info()&types.IsUntyped == 0
}

// align returns the smallest y >= x such that y % a == 0.
// a must be within 1 and 16 and it must be a power of 2.
// The result may be negative due to overflow.
func align(x, a int64) int64 {
	assert(x >= 0 && 1 <= a && a <= 16 && a&(a-1) == 0)
	return (x + a - 1) &^ (a - 1)
}

func assert(p bool) {
	if !p {
		msg := "assertion failed"
		// include information about the assertion location, due to panic recovery,
		// this location is otherwise buried in the middle of the panicking stack.
		if _, file, line, ok := runtime.Caller(1); ok {
			msg = fmt.Sprintf("%s:%d: %s", file, line, msg)
		}
		panic(msg)
	}
}

func getFields(st *types.Struct) []*types.Var {
	numField := st.NumFields()

	fields := make([]*types.Var, numField)

	for i := 0; i < numField; i++ {
		fields[i] = st.Field(i)
	}

	return fields
}
