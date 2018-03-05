package ezconf

import (
	"reflect"
	"testing"
)

type simpleStruct struct {
	MyInt  int
	MyBool bool

	MyInts []int

	Nested struct {
		NestedInt int
	}

	privInt int
}

func TestParsing(t *testing.T) {
	s := &simpleStruct{}
	err := parseTOMLFiles(s, []string{"notthere.toml", "simple.toml", "skipped.toml"}, true)
	if err != nil {
		t.Errorf("error encountered parsing: %s", err)
		return
	}

	if s.MyInt != 32 || s.MyBool != true || !reflect.DeepEqual(s.MyInts, []int{10, 20, 30}) || s.Nested.NestedInt != 64 {
		t.Errorf("unexpected parsed value: %+v", s)
	}
}
