package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/Snshadow/mkpackstruct/testdata"
)

const (
	sampleStub = `// stub for testing go file generation

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
`
)

func TestMain(t *testing.M) {
	code := t.Run()

	_ = os.WriteFile("testdata/sample_packstruct.go", []byte(sampleStub), 0644)

	os.Exit(code)
}

type packedByteWriter interface {
	ToPackedByte() []byte
}

func checkRoundTrip[P testdata.PackedStruct](t *testing.T, before P, writer packedByteWriter, wantSize int) {
	t.Helper()

	packedBuf := writer.ToPackedByte()
	if wantSize != 0 && len(packedBuf) != wantSize {
		t.Fatalf("packed size = %d, want %d", len(packedBuf), wantSize)
	}

	after, err := testdata.ToStruct[P](packedBuf)
	if err != nil {
		t.Fatalf("packed byte to struct failed: %v", err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatalf("not equal %v != %v", before, after)
	}
}

func testStructValues() ([2]testdata.InnerStruct, testdata.EmbedStruct, *uint64) {
	pu32 := new(uint32)
	u64 := uint64(1<<64 - 2)

	return [2]testdata.InnerStruct{
			{},
			{InnerField2: &pu32},
		},
		testdata.EmbedStruct{
			EmbedField2: 0x12345678,
		},
		&u64
}

func TestPackedPack1Alignment(t *testing.T) {
	inner, embed, u64 := testStructValues()

	before := testdata.TestStruct{
		Field4:      inner,
		Field7:      471,
		EmbedStruct: embed,
		Field10:     u64,
	}
	checkRoundTrip(t, before, &before, 437)
}

func TestPackedPack2Alignment(t *testing.T) {
	inner, embed, u64 := testStructValues()

	before := testdata.Test2Struct{
		Field4:      inner,
		Field7:      471,
		EmbedStruct: embed,
		Field10:     u64,
	}
	checkRoundTrip(t, before, &before, 440)
}

func TestPackedPack4Alignment(t *testing.T) {
	inner, embed, u64 := testStructValues()

	before := testdata.Test4Struct{
		Field4:      inner,
		Field7:      471,
		EmbedStruct: embed,
		Field10:     u64,
	}
	checkRoundTrip(t, before, &before, 448)
}

func TestPackedPopBackToPack2Alignment(t *testing.T) {
	inner, embed, u64 := testStructValues()

	before := testdata.TestBackTo2Struct{
		Field4:      inner,
		Field7:      471,
		EmbedStruct: embed,
		Field10:     u64,
	}
	checkRoundTrip(t, before, &before, 440)
}

func TestPackedPlainPack(t *testing.T) {
	before := testdata.TestPack4NoPushStruct{
		Field1: 1,
		Field2: 2,
		Field3: 3,
	}
	checkRoundTrip(t, before, &before, 16)
}

func TestPackedPopRestore(t *testing.T) {
	before := testdata.TestBackTo1Struct{
		Field1: 1,
		Field2: 2,
		Field3: 3,
	}
	checkRoundTrip(t, before, &before, 10)
}

func TestPackedPack8Alignment(t *testing.T) {
	inner, embed, u64 := testStructValues()

	before := testdata.Test8Struct{
		Field4:      inner,
		Field7:      471,
		EmbedStruct: embed,
		Field10:     u64,
	}
	checkRoundTrip(t, before, &before, 456)
}

func TestPackedPack16Alignment(t *testing.T) {
	inner, embed, u64 := testStructValues()

	before := testdata.Test16Struct{
		Field4:      inner,
		Field7:      471,
		EmbedStruct: embed,
		Field10:     u64,
	}
	checkRoundTrip(t, before, &before, 456)
}
