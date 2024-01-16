package ezconf

import (
	"reflect"
	"testing"
	"time"
)

type simpleStruct struct {
	MyInt      int
	MyBool     bool
	MyDatetime time.Time

	MyInts []int

	Nested struct {
		NestedInt int
	}
}

func TestParsing(t *testing.T) {
	s := &simpleStruct{}
	err := parseTOMLFiles(s, []string{"testdata/notthere.toml", "testdata/simple.toml", "testdata/skipped.toml"}, true)
	if err != nil {
		t.Errorf("error encountered parsing: %s", err)
		return
	}

	if s.MyInt != 32 || s.MyBool != true || !reflect.DeepEqual(s.MyInts, []int{10, 20, 30}) || s.Nested.NestedInt != 64 || !s.MyDatetime.Equal(time.Date(2018, 4, 3, 5, 30, 0, 0, time.UTC)) {
		t.Errorf("unexpected parsed value: %+v", s)
	}
}
