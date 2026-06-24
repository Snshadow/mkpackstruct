package parsestruct

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSource(t *testing.T, src string) string {
	t.Helper()

	dir := t.TempDir()
	filename := filepath.Join(dir, "sample.go")
	if err := os.WriteFile(filename, []byte(src), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return filename
}

func structByName(t *testing.T, info GoPackInfo, name string) *StructInfo {
	t.Helper()

	for _, st := range info.StructInfo {
		if st.StructName == name {
			return st
		}
	}
	t.Fatalf("struct %s not found", name)
	return nil
}

func TestPackDirectiveStackAndLayout(t *testing.T) {
	filename := writeSource(t, `package sample

type Default struct {
	A byte
	B uint32
}

//mkpackstruct:pack(push, 2)
type Pack2 struct {
	A byte
	B uint32
	C byte
}

//mkpackstruct:pack(push, 4)
type Pack4 struct {
	A byte
	B uint64
	C byte
}

//mkpackstruct:pack(pop)
type BackToPack2 struct {
	A byte
	B uint32
}

//mkpackstruct:pack()
type Reset struct {
	A byte
	B uint32
}

//mkpackstruct:pack(pop)
type BackToDefault struct {
	A byte
	B uint32
}
`)

	info, err := GetPackInfo(filename, 8, filename)
	if err != nil {
		t.Fatalf("GetPackInfo: %v", err)
	}

	defaultSt := structByName(t, info, "Default")
	if defaultSt.PackAlign != 1 || defaultSt.StructSize != 5 || defaultSt.Fields[1].Offset != 1 {
		t.Fatalf("Default layout = pack %d size %d B offset %d", defaultSt.PackAlign, defaultSt.StructSize, defaultSt.Fields[1].Offset)
	}

	pack2 := structByName(t, info, "Pack2")
	if pack2.PackAlign != 2 || pack2.StructSize != 8 || pack2.Fields[1].Offset != 2 || pack2.Fields[2].Offset != 6 {
		t.Fatalf("Pack2 layout = pack %d size %d offsets [%d %d]", pack2.PackAlign, pack2.StructSize, pack2.Fields[1].Offset, pack2.Fields[2].Offset)
	}

	pack4 := structByName(t, info, "Pack4")
	if pack4.PackAlign != 4 || pack4.StructSize != 16 || pack4.Fields[1].Offset != 4 || pack4.Fields[2].Offset != 12 {
		t.Fatalf("Pack4 layout = pack %d size %d offsets [%d %d]", pack4.PackAlign, pack4.StructSize, pack4.Fields[1].Offset, pack4.Fields[2].Offset)
	}

	backToPack2 := structByName(t, info, "BackToPack2")
	if backToPack2.PackAlign != 2 || backToPack2.StructSize != 6 || backToPack2.Fields[1].Offset != 2 {
		t.Fatalf("BackToPack2 layout = pack %d size %d B offset %d", backToPack2.PackAlign, backToPack2.StructSize, backToPack2.Fields[1].Offset)
	}

	reset := structByName(t, info, "Reset")
	if reset.PackAlign != 1 || reset.StructSize != 5 || reset.Fields[1].Offset != 1 {
		t.Fatalf("Reset layout = pack %d size %d B offset %d", reset.PackAlign, reset.StructSize, reset.Fields[1].Offset)
	}

	backToDefault := structByName(t, info, "BackToDefault")
	if backToDefault.PackAlign != 1 || backToDefault.StructSize != 5 || backToDefault.Fields[1].Offset != 1 {
		t.Fatalf("BackToDefault layout = pack %d size %d B offset %d", backToDefault.PackAlign, backToDefault.StructSize, backToDefault.Fields[1].Offset)
	}
}

func TestPackDirectiveErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "invalid alignment",
			src: `package sample
//mkpackstruct:pack(3)
type Bad struct{ A byte }
`,
			want: "unsupported pack alignment 3",
		},
		{
			name: "malformed directive",
			src: `package sample
//mkpackstruct:pack(push, 2, 4)
type Bad struct{ A byte }
`,
			want: "malformed mkpackstruct pack directive",
		},
		{
			name: "empty pop",
			src: `package sample
//mkpackstruct:pack(pop)
type Bad struct{ A byte }
`,
			want: "empty stack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := writeSource(t, tt.src)
			_, err := GetPackInfo(filename, 8, filename)
			if err == nil {
				t.Fatal("GetPackInfo succeeded unexpectedly")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func TestNamedStructPackAndArrayStride(t *testing.T) {
	filename := writeSource(t, `package sample

//mkpackstruct:pack(push, 2)
type Inner struct {
	A byte
	B uint32
	C byte
}
//mkpackstruct:pack(pop)

type Outer struct {
	A byte
	B [2]Inner
	C byte
	D struct {
		A byte
		B uint32
		C byte
	}
}
`)

	info, err := GetPackInfo(filename, 8, filename)
	if err != nil {
		t.Fatalf("GetPackInfo: %v", err)
	}

	inner := structByName(t, info, "Inner")
	if inner.StructSize != 8 {
		t.Fatalf("Inner size = %d, want 8", inner.StructSize)
	}

	outer := structByName(t, info, "Outer")
	if outer.PackAlign != 1 || outer.Fields[1].Offset != 1 || outer.Fields[1].Size != 16 {
		t.Fatalf("Outer array field offset/size = %d/%d", outer.Fields[1].Offset, outer.Fields[1].Size)
	}
	if outer.Fields[3].StructInfo.PackAlign != 1 || outer.Fields[3].Size != 6 {
		t.Fatalf("inline struct pack/size = %d/%d", outer.Fields[3].StructInfo.PackAlign, outer.Fields[3].Size)
	}
}
