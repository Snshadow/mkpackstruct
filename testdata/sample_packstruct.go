// stub for testing go file generation

package testdata

import ()

func (s *TestStruct) ToPackedByte() []byte {
	panic("STUB")
}

func (s *Test2Struct) ToPackedByte() []byte {
	panic("STUB")
}

func (s *Test4Struct) ToPackedByte() []byte {
	panic("STUB")
}

func (s *TestBackTo2Struct) ToPackedByte() []byte {
	panic("STUB")
}

func (s *TestPack4NoPushStruct) ToPackedByte() []byte {
	panic("STUB")
}

func (s *TestBackTo1Struct) ToPackedByte() []byte {
	panic("STUB")
}

func (s *Test8Struct) ToPackedByte() []byte {
	panic("STUB")
}

func (s *Test16Struct) ToPackedByte() []byte {
	panic("STUB")
}

type PackedStruct interface {
	TestStruct | Test2Struct | Test4Struct | TestBackTo2Struct | TestPack4NoPushStruct | TestBackTo1Struct | Test8Struct | Test16Struct
}

func ToStruct[P PackedStruct](buf []byte) (P, error) {
	panic("STUB")
}
