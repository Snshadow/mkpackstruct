package main

import (
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/Snshadow/mkpackstruct/testdata"
)

const (
	sampleStub = `// stub for testing go file generation

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
`
)

func TestMain(t *testing.M) {
	log.SetFlags(0)
	code := t.Run()

	_ = os.WriteFile("testdata/sample_gopack.go", []byte(sampleStub), 0644)

    os.Exit(code)
}

func TestPackStruct(t *testing.T) {
	pu32 := new(uint32)
    u64 := uint64(1<<64-2)

    before := testdata.TestStruct{
        Field4: [2]testdata.InnerStruct{
            {},
            {
                InnerField2: &pu32,
            },
        },
        Field7: 471,
        EmbedStruct: testdata.EmbedStruct{
            EmbedField2: 0x12345678,
        },
        Field10: &u64,
    }

    packedBuf := before.ToPackedByte()

    after, err := testdata.ToStruct[testdata.TestStruct](packedBuf)
    if err != nil {
        t.Fatalf("packed byte to struct failed: %v\n", err)
    }

    if !reflect.DeepEqual(before, after) {
        t.Fatalf("not equal %v != %v", before, after)
    }
}
