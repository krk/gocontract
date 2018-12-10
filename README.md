# gocontract

`gocontract` is a very simple utility which detects uninitialized and struct fields in a Go file. Detection is opt-in.

## Usage

```
gocontract [file]

    file    Go source file.
```

## Example

Input file: `example.go`

```go
package example

type Abc struct {
	val *int `json:"config" require:"assignment,NewAbc ,  NewAbcOther"`
}

func NewAbc() Abc {
	val := 42
	return Abc{
		val: &val}
}

func NewAbcOther() *Abc {
	return &Abc{}
}
```

Output:

```
./gocontract example.go
example.go uninitialized struct field Abc.val in NewAbcOther
```
