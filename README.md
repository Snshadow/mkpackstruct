# mkpackstruct

mkpackstruct generates go file for packing struct, which can be useful for using structs with [packed attribute](https://gcc.gnu.org/onlinedocs/gcc/Common-Type-Attributes.html#index-packed-type-attribute) from gcc or with [pack pragma](https://learn.microsoft.com/en-us/cpp/preprocessor/pack)(especially with `#pragma pack(1)`) from MSVC. Instead of using `reflect` package to create packed struct at runtime, this repository seeks to create functions for packing in advance, to reduce runtime overheads.

## Features

It reads the single go file with struct type declarations, then creates `ToPackedByte()` method for each structs in go file like

```go
func (s *SomeStruct) ToPackedByte() []byte {
    var b Bytes.Buffer
    // write buffer with packed offsets and sizes...

    return b.Bytes()
}
```

which returns byte slice which has serialized struct data without any padding.

It also creates generic function `ToStruct[P PackedStruct](st P) (P, error)` for unpacking structs from serialized byte slice by creating type union for structs in the specified go file,

```go
func ToStruct[P PackedStruct](buf []byte) (P, error) {
    var result any

    // fill in struct...

    return st.(P)
}
```

note that this function returns an error if the size of the byte slice does not match the packed size of the struct.

## Usage

### go run

```cmd
go run github.com/Snshadow/mkpackstruct <go_filename> <output>
```

If output is not specified, file will be written at `srcdir/<go_filename>_gopack_${GOARCH}.go` by default.

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

_need_pack.go_

```go
//go:generate go run github.com/Snshadow/mkpackstruct $GOFILE

// struct declarations...
```
