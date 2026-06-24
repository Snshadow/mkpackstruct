package gengo

import (
	"strings"
	"testing"

	"github.com/Snshadow/mkpackstruct/parsestruct"
)

func TestWritePackedBytesEmitsPadding(t *testing.T) {
	st := &parsestruct.StructInfo{
		StructName: "Pack2",
		StructSize: 8,
		Fields: []*parsestruct.FieldInfo{
			{Name: "A", Offset: 0, Size: 1, Type: "byte"},
			{Name: "B", Offset: 2, Size: 4, Type: "uint32"},
			{Name: "C", Offset: 6, Size: 1, Type: "byte"},
		},
	}

	got := writePackedBytes(st, "s.", 0, nil)
	if strings.Count(got, "b.Write(make([]byte, 1))") != 2 {
		t.Fatalf("expected internal and tail padding, got:\n%s", got)
	}
	if !strings.Contains(got, "unsafe.Pointer(&s.B)), 4)") {
		t.Fatalf("expected B field write, got:\n%s", got)
	}
}
