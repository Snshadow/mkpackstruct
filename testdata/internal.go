package testdata

import (
	"unsafe"
)

type NamedUint32 uint32
type NamedUint64 uint64

type NamedUint32_2 NamedUint32

type AliasUintptr = uintptr

type InnerStruct struct {
	InnerField1 [7]uint8
	InnerField2 **uint32
	InnerField3 chan NamedUint32
	InnerField4 map[string]NamedUint32_2
	InnerField5 [2]RepeatedStruct
	InnerField6 uintptr
}

type EmbedStruct struct {
	EmbedField1 [9]byte
	EmbedField2 NamedUint32_2
	EmbedField3 [1]uint16
	EmbedField4 *uintptr
}

type RepeatedStruct struct {
	RepeatedField1 [9]uint8
	RepeatedField2 <-chan error
	RepeatedField3 []uint32
	RepeatedField4 *unsafe.Pointer
}
