// stub for testing go file generation

package testdata

import ()

func (s TestStruct) ToPackedByte() []byte {
	panic("STUB")
}

type PackedStruct interface {
	TestStruct
}

func ToStruct[P PackedStruct](buf []byte) (P, error) {
	panic("STUB")
}
