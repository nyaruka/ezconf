package ezconf

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type simpleStruct struct {
	MyInt      int
	MyBool     bool
	MyDatetime time.Time
	MyLogLevel slog.Level

	MyInts []int

	Nested struct {
		NestedInt int
	}
}

func TestParsing(t *testing.T) {
	s := &simpleStruct{}
	err := parseTOMLFiles(s, []string{"testdata/notthere.toml", "testdata/simple.toml", "testdata/skipped.toml"}, true)

	assert.NoError(t, err)
	assert.Equal(t, 32, s.MyInt)
	assert.True(t, s.MyBool)
	assert.Equal(t, []int{10, 20, 30}, s.MyInts)
	assert.Equal(t, 64, s.Nested.NestedInt)
	assert.Equal(t, time.Date(2018, 4, 3, 5, 30, 0, 0, time.UTC), s.MyDatetime)
}
