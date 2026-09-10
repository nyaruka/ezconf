package ezconf

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type allKinds struct {
	MyInt      int
	MyUint     uint
	MyFloat    float64
	MyBool     bool
	MyString   string
	MyDatetime time.Time
}

type allTypes struct {
	MyInt      int
	MyInt8     int8
	MyInt16    int16
	MyInt32    int32
	MyInt64    int64
	MyUint     uint
	MyUint8    uint8
	MyUint16   uint16
	MyUint32   uint32
	MyUint64   uint64
	MyFloat32  float32
	MyFloat64  float64
	MyBool     bool
	MyString   string
	MyStrings  []string
	MyInts     []int
	MyDatetime time.Time
	MyLogLevel slog.Level
}

func toFields(t *testing.T, s any) *ezFields {
	return buildFields(s)
}

func TestSetValue(t *testing.T) {
	at := allTypes{}
	fields := toFields(t, &at)

	tests := []struct {
		key      string
		value    string
		hasErr   bool
		expected string
	}{
		{"my_int", "-48", false, "-48"},
		{"my_int", "wat", true, ""},
		{"my_int8", "-48", false, "-48"},
		{"my_int8", "wat", true, ""},
		{"my_int16", "-48", false, "-48"},
		{"my_int16", "wat", true, ""},
		{"my_int32", "-48", false, "-48"},
		{"my_int32", "wat", true, ""},
		{"my_int64", "-48", false, "-48"},
		{"my_int64", "wat", true, ""},

		{"my_uint", "48", false, "48"},
		{"my_uint", "wat", true, ""},
		{"my_uint8", "48", false, "48"},
		{"my_uint8", "wat", true, ""},
		{"my_uint16", "48", false, "48"},
		{"my_uint16", "wat", true, ""},
		{"my_uint32", "48", false, "48"},
		{"my_uint32", "wat", true, ""},
		{"my_uint64", "48", false, "48"},
		{"my_uint64", "wat", true, ""},

		{"my_float32", "12", false, "12"},
		{"my_float32", "wat", true, ""},
		{"my_float32", "1e300", true, ""},
		{"my_float64", "12", false, "12"},
		{"my_float64", "wat", true, ""},
		{"my_float64", "0.1234567890123456", false, "0.1234567890123456"},
		{"my_float64", "1e300", false, "1e+300"},

		{"my_bool", "true", false, "true"},
		{"my_bool", "wat", true, ""},

		{"my_string", "foozap", false, "foozap"},

		{"my_strings", "foo,bar,baz", false, "[foo bar baz]"},
		{"my_strings", "foo, bar , baz", false, "[foo bar baz]"},
		{"my_strings", "", true, ""},

		{"my_ints", "10,20,30", false, "[10 20 30]"},
		{"my_ints", "10, 20 , 30", false, "[10 20 30]"},
		{"my_ints", "wat", true, ""},

		{"my_datetime", "15:45:05", false, "0000-01-01 15:45:05 +0000 UTC"},
		{"my_datetime", "2018-04-03", false, "2018-04-03 00:00:00 +0000 UTC"},
		{"my_datetime", "2018-04-03T05:30:00Z", false, "2018-04-03 05:30:00 +0000 UTC"},
		{"my_datetime", "2018-04-03T05:30:00.123+07:00", false, "2018-04-03 05:30:00.123 +0700 +0700"},
		{"my_datetime", "notdate", true, ""},

		{"my_log_level", "info", false, "INFO"},
		{"my_log_level", "ERROR", false, "ERROR"},
		{"my_log_level", "crazy", true, ""},

		{"unknown", "", true, ""},
	}

	for _, tc := range tests {
		values := map[string]ezValue{tc.key: {tc.key, tc.value}}
		err := setValues(fields, values)
		if !tc.hasErr && err != nil {
			assert.NoError(t, err, "unexpected error setting %s to %s", tc.key, tc.value)
		}
		if tc.hasErr && err == nil {
			assert.Error(t, err, "expected error setting %s to %s", tc.key, tc.value)
		}
		field, found := fields.fields[tc.key]
		if found && !tc.hasErr && err == nil {
			strValue := fmt.Sprintf("%v", field.Value())
			assert.Equal(t, tc.expected, strValue)
		}
	}
}

func TestEndToEnd(t *testing.T) {
	at := &allTypes{}
	conf := NewLoader(at, "foo", "description", []string{"testdata/missing.toml", "testdata/fields.toml", "testdata/simple.toml"})
	conf.SetArgs("-my-int=48", "-my-log-level=error")
	err := conf.Load()
	assert.NoError(t, err)
	assert.Equal(t, 48, at.MyInt)
	assert.Equal(t, slog.LevelError, at.MyLogLevel)
}

func TestNameTagValidation(t *testing.T) {
	tests := []struct {
		name string
		err  string
	}{
		{"opensearch", ""},
		{"num_workers", ""},
		{"s3_bucket", ""},
		{"a", ""},
		{"OpenSearch", `invalid name tag "OpenSearch" for field F, must be snake_case`},
		{"open-search", `invalid name tag "open-search" for field F, must be snake_case`},
		{"_opensearch", `invalid name tag "_opensearch" for field F, must be snake_case`},
		{"opensearch_", `invalid name tag "opensearch_" for field F, must be snake_case`},
		{"open__search", `invalid name tag "open__search" for field F, must be snake_case`},
		{"123", `invalid name tag "123" for field F, must be snake_case`},
		{"open search", `invalid name tag "open search" for field F, must be snake_case`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// we can't dynamically set struct tags, so test the regex directly
			valid := validNameTag.MatchString(tc.name)
			if tc.err == "" {
				assert.True(t, valid, "expected %q to be valid", tc.name)
			} else {
				assert.False(t, valid, "expected %q to be invalid", tc.name)
			}
		})
	}

	// test that buildFields panics for invalid name tag
	type badConfig struct {
		F string `name:"Open-Search"`
	}
	assert.PanicsWithValue(t, `invalid name tag "Open-Search" for field F, must be snake_case`, func() { buildFields(&badConfig{}) })

	// test that buildFields accepts valid name tag
	type goodConfig struct {
		F string `name:"opensearch"`
	}
	assert.Contains(t, buildFields(&goodConfig{}).fields, "opensearch")
}

func TestNameTag(t *testing.T) {
	type config struct {
		OpenSearch string `name:"opensearch" help:"the OpenSearch URL"`
		NumWorkers int    `help:"the number of workers"`
	}

	// test that buildFields uses name tag for name
	c := &config{}
	fields := toFields(t, c)
	assert.Contains(t, fields.fields, "opensearch")
	assert.Contains(t, fields.fields, "num_workers")
	assert.NotContains(t, fields.fields, "open_search")

	// test flag name uses name tag (opensearch not open-search)
	c = &config{OpenSearch: "http://default", NumWorkers: 4}
	conf := NewLoader(c, "foo", "description", nil)
	conf.SetArgs("-opensearch=http://localhost:9200", "-num-workers=8")
	err := conf.Load()
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:9200", c.OpenSearch)
	assert.Equal(t, 8, c.NumWorkers)

	// test env var uses name tag (FOO_OPENSEARCH not FOO_OPEN_SEARCH)
	c = &config{OpenSearch: "http://default", NumWorkers: 4}
	conf = NewLoader(c, "foo", "description", nil)
	conf.SetArgs()
	os.Setenv("FOO_OPENSEARCH", "http://from-env")
	defer os.Setenv("FOO_OPENSEARCH", "")
	err = conf.Load()
	assert.NoError(t, err)
	assert.Equal(t, "http://from-env", c.OpenSearch)

	// test TOML uses name tag
	c = &config{OpenSearch: "http://default", NumWorkers: 4}
	conf = NewLoader(c, "foo", "description", []string{"testdata/conftag.toml"})
	conf.SetArgs()
	os.Setenv("FOO_OPENSEARCH", "")
	err = conf.Load()
	assert.NoError(t, err)
	assert.Equal(t, "http://from-toml", c.OpenSearch)
}

func TestPriority(t *testing.T) {
	at := &allTypes{MyInt: 16}
	conf := NewLoader(at, "foo", "description", []string{"testdata/missing.toml", "testdata/fields.toml", "testdata/simple.toml"})
	conf.SetArgs()
	conf.Load()

	assert.Equal(t, 96, at.MyInt)

	// override with environment variable
	conf = NewLoader(at, "foo", "description", []string{"testdata/missing.toml", "testdata/fields.toml", "testdata/simple.toml"})
	conf.SetArgs()
	os.Setenv("FOO_MY_INT", "48")
	conf.Load()

	assert.Equal(t, 48, at.MyInt)

	// override with args
	conf = NewLoader(at, "foo", "description", []string{"testdata/missing.toml", "testdata/fields.toml", "testdata/simple.toml"})
	conf.SetArgs("-my-int=56")
	os.Setenv("FOO_MY_INT", "48")
	conf.Load()

	assert.Equal(t, 56, at.MyInt)

	// clear our env, args should take precedence now even though we are setting to the same as our new default
	os.Setenv("FOO_MY_INT", "")
	conf = NewLoader(at, "foo", "description", []string{"testdata/missing.toml", "testdata/fields.toml", "testdata/simple.toml"})
	conf.SetArgs("-my-int=56")
	conf.Load()

	assert.Equal(t, 56, at.MyInt)
}

func TestConfigMustBePointer(t *testing.T) {
	// a struct passed by value isn't settable, so we panic rather than silently discarding values
	assert.PanicsWithValue(t, "config must be a non-nil pointer to a struct, got ezconf.allTypes", func() { buildFields(allTypes{}) })

	assert.PanicsWithValue(t, "config must be a non-nil pointer to a struct, got *ezconf.allTypes", func() { buildFields((*allTypes)(nil)) })

	i := 32
	assert.PanicsWithValue(t, "config must be a non-nil pointer to a struct, got *int", func() { buildFields(&i) })

	// and the loader surfaces it as a panic rather than reporting a successful load
	conf := NewLoader(allTypes{}, "foo", "description", nil)
	conf.SetArgs("-my-int=48")
	assert.Panics(t, func() { conf.Load() })
}

func TestReservedNames(t *testing.T) {
	// fields can't claim the names of the flags we add ourselves
	type helpConfig struct {
		Help bool
	}
	assert.PanicsWithValue(t, `Help uses reserved name "help"`, func() { buildFields(&helpConfig{}) })

	// -h is documented as a usage alias, so a field can't claim it either
	type hConfig struct {
		H bool
	}
	assert.PanicsWithValue(t, `H uses reserved name "h"`, func() { buildFields(&hConfig{}) })

	type taggedConfig struct {
		Something bool `name:"help"`
	}
	assert.PanicsWithValue(t, `Something uses reserved name "help"`, func() { buildFields(&taggedConfig{}) })

	// and the loader panics with the same message rather than doing so inside the flag package
	conf := NewLoader(&helpConfig{}, "foo", "description", nil)
	conf.SetArgs()
	assert.PanicsWithValue(t, `Help uses reserved name "help"`, func() { conf.Load() })
}

func TestLoadDoesNotExit(t *testing.T) {
	// an unparseable flag comes back as an error rather than exiting the program
	at := &allTypes{}
	conf := NewLoader(at, "foo", "description", nil)
	conf.SetArgs("-not-a-real-flag=1")
	assert.EqualError(t, conf.Load(), "flag provided but not defined: -not-a-real-flag")

	// as does a value of the wrong type
	conf = NewLoader(at, "foo", "description", nil)
	conf.SetArgs("-my-int=wat")
	assert.Error(t, conf.Load())
}

func TestLoadHelp(t *testing.T) {
	for _, arg := range []string{"-help", "-h"} {
		at := &allTypes{}
		conf := NewLoader(at, "foo", "description", nil)
		conf.SetArgs(arg)

		err := conf.Load()
		assert.True(t, errors.Is(err, ErrHelp), "expected ErrHelp for %s, got %v", arg, err)

		// usage is the caller's to show, and is available once Load has run
		buf := &strings.Builder{}
		conf.flags.SetOutput(buf)
		conf.Usage()
		assert.Contains(t, buf.String(), "Usage of foo:")
		assert.Contains(t, buf.String(), "FOO_MY_INT - int")
	}

	// Usage before Load is a noop rather than a panic
	assert.NotPanics(t, func() { NewLoader(&allTypes{}, "foo", "description", nil).Usage() })
}

func TestLoadIsSilent(t *testing.T) {
	// Load must not write to stdout or stderr, even on error
	oldOut, oldErr := os.Stdout, os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout, os.Stderr = w, w
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()

	for _, args := range [][]string{{"-not-a-real-flag=1"}, {"-help"}, {"-my-int=wat"}} {
		conf := NewLoader(&allTypes{}, "foo", "description", nil)
		conf.SetArgs(args...)
		conf.Load()
	}

	w.Close()
	written, _ := io.ReadAll(r)
	assert.Empty(t, string(written), "Load wrote to stdout/stderr")
}

func TestUsageFollowsStderr(t *testing.T) {
	conf := NewLoader(&allTypes{}, "foo", "description", nil)
	conf.SetArgs()
	assert.NoError(t, conf.Load())

	// stderr redirected after Load returned must still receive usage, so Load can't have
	// captured the old one when it restored output
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	conf.Usage()
	w.Close()
	os.Stderr = old

	out, _ := io.ReadAll(r)
	assert.Contains(t, string(out), "Usage of foo:")
}

// a config struct embedding another which in turn embeds another, so that fields are promoted from two levels down
type CoreConfig struct {
	Timeout    int    `help:"the request timeout in seconds"`
	OpenSearch string `name:"opensearch" help:"the OpenSearch URL"`
}

type BaseConfig struct {
	CoreConfig
	DB       string     `help:"the database URL"`
	LogLevel slog.Level `help:"the logging level"`
	Networks []string
}

type ExtendedConfig struct {
	BaseConfig
	SentryDSN  string `help:"the Sentry DSN"`
	LimitsMode string
}

func TestEmbeddedStructs(t *testing.T) {
	// fields of embedded structs are discovered at any depth alongside the struct's own fields
	fields := toFields(t, &ExtendedConfig{})
	assert.Equal(t, []string{"db", "limits_mode", "log_level", "networks", "opensearch", "sentry_dsn", "timeout"}, fields.keys)

	// and the name, reserved name and collision checks apply to them too, panicking as they're development errors
	type Reserved struct {
		Help bool
	}
	type reservedConfig struct {
		Reserved
	}
	assert.PanicsWithValue(t, `Reserved.Help uses reserved name "help"`, func() { buildFields(&reservedConfig{}) })

	type Tagged struct {
		F string `name:"Bad-Name"`
	}
	type taggedConfig struct {
		Tagged
	}
	assert.PanicsWithValue(t, `invalid name tag "Bad-Name" for field Tagged.F, must be snake_case`, func() { buildFields(&taggedConfig{}) })

	type outerCollision struct {
		BaseConfig
		DB string
	}
	assert.PanicsWithValue(t, "DB name collides with BaseConfig.DB", func() { buildFields(&outerCollision{}) })

	type nameTagCollision struct {
		CoreConfig
		Opensearch string
	}
	assert.PanicsWithValue(t, "Opensearch name collides with CoreConfig.OpenSearch", func() { buildFields(&nameTagCollision{}) })

	type A struct {
		X int
	}
	type B struct {
		X int
	}
	type embeddedCollision struct {
		A
		B
	}
	assert.PanicsWithValue(t, "A.X name collides with B.X", func() { buildFields(&embeddedCollision{}) })

	// embedded pointers would need allocating before their fields could be set, so aren't supported
	type pointerConfig struct {
		*BaseConfig
	}
	assert.PanicsWithValue(t, "embedded field BaseConfig must be a struct, not a pointer", func() { buildFields(&pointerConfig{}) })

	type base struct {
		DB string
	}
	type unexportedConfig struct {
		base
	}
	assert.PanicsWithValue(t, "embedded struct base must be exported", func() { buildFields(&unexportedConfig{}) })

	// and the loader panics rather than silently ignoring the embedded fields
	conf := NewLoader(&pointerConfig{}, "foo", "description", nil)
	conf.SetArgs()
	assert.PanicsWithValue(t, "embedded field BaseConfig must be a struct, not a pointer", func() { conf.Load() })

	// an embedded non-struct is just a field named after its type
	type levelConfig struct {
		slog.Level
	}
	fields = toFields(t, &levelConfig{})
	assert.Equal(t, []string{"level"}, fields.keys)

	// non-embedded struct fields still aren't loaded
	type nestedConfig struct {
		Nested CoreConfig
	}
	fields = toFields(t, &nestedConfig{})
	assert.Empty(t, fields.keys)
}

func TestEmbeddedStructsEndToEnd(t *testing.T) {
	c := &ExtendedConfig{
		BaseConfig: BaseConfig{CoreConfig: CoreConfig{Timeout: 10}, DB: "postgres://default/db"},
		LimitsMode: "enforce",
	}
	conf := NewLoader(c, "foo", "description", []string{"testdata/missing.toml", "testdata/embedded.toml"})
	conf.SetArgs("-timeout=60", "-sentry-dsn=https://from-flag@sentry")
	os.Setenv("FOO_DB", "postgres://from-env/db")
	os.Setenv("FOO_OPENSEARCH", "http://from-env:9200")
	defer os.Setenv("FOO_DB", "")
	defer os.Setenv("FOO_OPENSEARCH", "")

	assert.NoError(t, conf.Load())

	// promoted fields are set from TOML, env and flags with the usual priority, regardless of depth
	assert.Equal(t, 60, c.Timeout)
	assert.Equal(t, "http://from-env:9200", c.OpenSearch)
	assert.Equal(t, "postgres://from-env/db", c.DB)
	assert.Equal(t, slog.LevelWarn, c.LogLevel)
	assert.Equal(t, []string{"10.0.0.0/8", "192.168.0.0/16"}, c.Networks)
	assert.Equal(t, "https://from-flag@sentry", c.SentryDSN)
	assert.Equal(t, "observe", c.LimitsMode)

	// and usage lists them alongside the struct's own fields
	buf := &strings.Builder{}
	conf.flags.SetOutput(buf)
	conf.Usage()
	assert.Contains(t, buf.String(), "-timeout int\n    \tthe request timeout in seconds (default 10)")
	assert.Contains(t, buf.String(), "-opensearch string\n    \tthe OpenSearch URL")
	assert.Contains(t, buf.String(), "-sentry-dsn string\n    \tthe Sentry DSN")
	assert.Contains(t, buf.String(), "FOO_DB - string")
	assert.Contains(t, buf.String(), "FOO_TIMEOUT - int")
	assert.Contains(t, buf.String(), "FOO_OPENSEARCH - string")

	// a TOML key that nothing defines is still an error
	conf = NewLoader(&ExtendedConfig{}, "foo", "description", []string{"testdata/simple.toml"})
	conf.SetArgs()
	assert.EqualError(t, conf.Load(), "line 2: unknown key 'my_int'")
}
