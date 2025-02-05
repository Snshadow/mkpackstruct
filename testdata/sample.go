package testdata

import (
	"unsafe"
)

type NamedUint32 uint32
type NamedUint64 uint64

type NamedUint32_2 NamedUint32

type AliasUintptr = uintptr

type TestStruct struct {
	Field1  uint32
	Field2  [9]uint8
	Field3  NamedUint32
	Field4  [2]InnerStruct
	Field5  [5]NamedUint32_2
	Field6  complex128
	Field7  AliasUintptr
	Field8  unsafe.Pointer
	Field9  [3]float32
	Field10 *uint64
	EmbedStruct
	Field11      [7]uint16
	NestedStruct struct {
		NestedField1 [1]byte
		NestedField2 *uintptr
	}
}

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
