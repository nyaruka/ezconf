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

	MyInts    []int
	MyStrings []string

	Nested struct {
		NestedInt int
	}
}

func TestParsing(t *testing.T) {
	s := &simpleStruct{}
	err := parseTOMLFiles(toFields(t, s), []string{"testdata/notthere.toml", "testdata/simple.toml", "testdata/skipped.toml"})

	assert.NoError(t, err)
	assert.Equal(t, 32, s.MyInt)
	assert.True(t, s.MyBool)
	assert.Equal(t, []int{10, 20, 30}, s.MyInts)
	assert.Equal(t, []string{"foo", "bar"}, s.MyStrings)
	assert.Equal(t, 64, s.Nested.NestedInt)
	assert.Equal(t, time.Date(2018, 4, 3, 5, 30, 0, 0, time.UTC), s.MyDatetime)
}

func TestParsingEmbedded(t *testing.T) {
	type Nested struct {
		NestedInt int
	}
	type Inner struct {
		CoreConfig
		Nested Nested
	}
	type config struct {
		Inner
		MyInt int
	}

	parse := func(doc string) (*config, error) {
		c := &config{}
		return c, parseTOML([]byte(doc), toFields(t, c))
	}

	// fields of embedded structs are set from top level keys at any depth, honouring name tags, and
	// non-embedded struct fields in an embedded struct are still tables
	c, err := parse("my_int = 32\ntimeout = 30\nopensearch = \"http://localhost:9200\"\n\n[nested]\nnested_int = 64\n")
	assert.NoError(t, err)
	assert.Equal(t, 32, c.MyInt)
	assert.Equal(t, 30, c.Timeout)
	assert.Equal(t, "http://localhost:9200", c.OpenSearch)
	assert.Equal(t, 64, c.Nested.NestedInt)

	// keys that nothing defines are still an error
	_, err = parse("my_int = 32\nfoo = 1\n")
	assert.EqualError(t, err, "line 2: unknown key 'foo'")

	// including the promoted field's default name when a name tag overrides it
	_, err = parse("open_search = \"http://localhost:9200\"\n")
	assert.EqualError(t, err, "line 1: unknown key 'open_search'")

	// and an embedded struct can't be set as a table named after it
	_, err = parse("my_int = 32\n\n[inner]\ntimeout = 30\n")
	assert.EqualError(t, err, "line 3: unknown key 'inner'")

	_, err = parse("[[inner]]\ntimeout = 30\n")
	assert.EqualError(t, err, "line 1: unknown key 'inner'")

	// a toml tag on a promoted field is honoured as it would be by the decoder
	type Tagged struct {
		Renamed string `toml:"custom_key"`
	}
	type taggedConfig struct {
		Tagged
	}
	tc := &taggedConfig{}
	assert.NoError(t, parseTOML([]byte("custom_key = \"x\"\n"), toFields(t, tc)))
	assert.Equal(t, "x", tc.Renamed)
	assert.EqualError(t, parseTOML([]byte("renamed = \"x\"\n"), toFields(t, tc)), "line 1: unknown key 'renamed'")

	// values of the wrong type for a promoted field are an error like any other
	_, err = parse("timeout = \"soon\"\n")
	assert.Error(t, err)

	// as is a document that isn't valid TOML
	_, err = parse("my_int = \n")
	assert.Error(t, err)
}
