package testdata

import (
	"unsafe"
)

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

//mkpackstruct:pack(push, 2)
type Test2Struct struct {
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

//mkpackstruct:pack(push, 4)
type Test4Struct struct {
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
//mkpackstruct:pack(pop)

type TestBackTo2Struct struct {
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
//mkpackstruct:pack(pop)

//mkpackstruct:pack(push, 8)
type Test8Struct struct {
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
//mkpackstruct:pack(pop)

//mkpackstruct:pack(push, 16)
type Test16Struct struct {
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
//mkpackstruct:pack(pop)
