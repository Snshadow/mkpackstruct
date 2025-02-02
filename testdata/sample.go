package testdata

import (
	"unsafe"
)

type NamedUint32 uint32
type NamedUint64 uint64

type NamedUint32_2 NamedUint32

type AliasUintptr = uintptr

type TestStruct struct {
	Field1 uint32
	Field2 [9]uint8
	Field3 NamedUint32
	Field4 [2]InnerStruct
	Field5 [5]NamedUint32_2
	Field6 complex128
	Field7 AliasUintptr
	Field8 unsafe.Pointer
	Field9 [3]float32 
	Field10 *uint64
	EmbedStruct
	Field11 [7]uint16
	NestedStruct struct {
		NestedField uint32
		NestedField2 [9]byte
		NestedField3 *uintptr
		NestedField4 uint64
	}
}

type InnerStruct struct {
	InnerField1 [7]uint8
	InnerField2 *uint32
	InnerField3 [2]*unsafe.Pointer
	InnerField4 uint64
	InnerField5 [4]uint16
	InnerField6 uintptr
}

type EmbedStruct struct {
	EmbedField1 [9]byte
	EmbedField2 NamedUint32_2
	EmbedField3 [1]uint16
	EmbedField4 *uintptr
}
