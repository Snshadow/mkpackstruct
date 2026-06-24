# mkpackstruct

mkpackstruct generates go file for packing struct, which can be useful for using structs with [aligned attribute](https://gcc.gnu.org/onlinedocs/gcc/Common-Attributes.html#index-aligned) from gcc or with [pack pragma](https://learn.microsoft.com/en-us/cpp/preprocessor/pack) from MSVC. Instead of using `reflect` package with tags to create packed struct at runtime, this repository seeks to create functions for packing in advance, to reduce runtime overheads.

## Features

It reads single go file with struct type declarations, then creates `ToPackedByte()` method for each structs in go file as

```go
func (s *SomeStruct) ToPackedByte() []byte {
    var b Bytes.Buffer
    // write buffer with packed offsets and sizes...

    return b.Bytes()
}
```

which returns byte slice with serialized struct data using the configured pack layout.

It also creates generic function `ToStruct[P PackedStruct](st P) (P, error)` for unpacking structs from serialized byte slice by creating type union for structs in the specified go file,

```go
func ToStruct[P PackedStruct](buf []byte) (P, error) {
    var result any

    // fill in struct...

    return st.(P)
}
```

note that this function returns an error if the size of the byte slice does not match the packed size of the struct.

Only structs declared while a `mkpackstruct:pack` directive is active are generated. Use Go comments to change the current pack alignment in source order:

```go
//mkpackstruct:pack(1)
type Packed1 struct {
    Flag byte
    Size uint32
}

//mkpackstruct:pack(push, 2)
type Packed2 struct {
    Flag byte
    Size uint32
}
//mkpackstruct:pack(pop)

type AnotherPacked1 struct {
    Flag byte
    Size uint32
}

//mkpackstruct:pack()
type NotGenerated struct {
    Flag byte
    Size uint32
}
```

Supported directives are `pack(N)`, `pack()`, `pack(push)`, `pack(push, N)`, and `pack(pop)`, where `N` is `1`, `2`, `4`, `8`, or `16`. `pack()` clears the active pack context, so following structs are not generated until another pack directive sets an active alignment.

## Usage

### go run

```cmd
go run github.com/Snshadow/mkpackstruct <go_filename> <output>
```

If output is not specified, generated file will be written at `srcdir/<go_filename>_packstruct_${GOARCH}.go` by default. If such build constraint is not needed, specify the output path like `<go_filename>_packstruct.go`. Set `GOARCH` value to create function for wanted architecture.

---

### go generate

For convenience, one would use `tools.go` design with `go generate`

_tools.go_

```go
//go:build tools

package main

import (
    _ "github.com/Snshadow/mkpackstruct"
)
```

_need\_pack.go_

```go
//go:generate go run github.com/Snshadow/mkpackstruct $GOFILE

// struct declarations...
```
