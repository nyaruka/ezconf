package ezconf

import (
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildFlags(t *testing.T) {
	as := allTypes{
		MyInt:      32,
		MyFloat32:  16,
		MyBool:     true,
		MyString:   "foobar",
		MyDatetime: time.Date(2018, 3, 5, 12, 30, 0, 0, time.UTC),
	}
	fs := buildFlags("foo", "description", toFields(t, &as), flag.ContinueOnError)

	flags := []struct {
		name  string
		usage string
		def   string
	}{
		{"my-int", "set value for my_int", "32"},
		{"my-float32", "set value for my_float32", "16"},
		{"my-bool", "set value for my_bool", "true"},
		{"my-string", "set value for my_string", "foobar"},
		{"my-datetime", "set value for my_datetime", "2018-03-05T12:30:00Z"},
	}

	for _, ef := range flags {
		f := fs.Lookup(ef.name)
		if f == nil {
			t.Errorf("did not find flag with name: %s", ef.name)
			continue
		}
		if f.Usage != ef.usage {
			t.Errorf("usage '%s' does not match expected '%s'", f.Usage, ef.usage)
		}
		if f.DefValue != ef.def {
			t.Errorf("default '%s' does not match expected '%s'", f.DefValue, ef.def)
		}
	}

	// print our usage
	fs.Usage()

	// parse with invalid args
	_, err := parseFlags(fs, []string{"-unknown=bar"})
	if err == nil {
		t.Errorf("should have errored with invalid args")
	}

	// finally parse our flags
	args := []string{
		"-my-string=foozap",
		"-my-int32=65",
		"-my-bool=false",
		"-my-datetime=2018-04-05T12:30:00Z",
	}
	values, err := parseFlags(fs, args)
	if err != nil {
		t.Errorf("received error parsing flags")
		return
	}

	tcs := []struct {
		key    string
		rawKey string
		value  string
	}{
		{"my_int32", "my-int32", "65"},
		{"my_bool", "my-bool", "false"},
		{"my_string", "my-string", "foozap"},
		{"my_datetime", "my-datetime", "2018-04-05T12:30:00Z"},
	}

	for _, tc := range tcs {
		v, found := values[tc.key]
		if !found {
			t.Errorf("did not find value with key: %s", tc.key)
			continue
		}

		assert.Equal(t, tc.rawKey, v.rawKey, "raw key mismatch for key %s", tc.key)
		assert.Equal(t, tc.value, v.value, "value mismatch for key %s", tc.key)
	}
}

func TestBuildFlagsEmbedded(t *testing.T) {
	// fields of embedded structs get flags with their defaults and help like any other
	c := &ExtendedConfig{BaseConfig: BaseConfig{CoreConfig: CoreConfig{Timeout: 30}, DB: "postgres://default/db"}}
	fs := buildFlags("foo", "description", toFields(t, c), flag.ContinueOnError)

	f := fs.Lookup("timeout")
	assert.NotNil(t, f)
	assert.Equal(t, "the request timeout in seconds", f.Usage)
	assert.Equal(t, "30", f.DefValue)

	f = fs.Lookup("opensearch")
	assert.NotNil(t, f)
	assert.Equal(t, "the OpenSearch URL", f.Usage)

	f = fs.Lookup("db")
	assert.NotNil(t, f)
	assert.Equal(t, "postgres://default/db", f.DefValue)

	values, err := parseFlags(fs, []string{"-timeout=60", "-db=postgres://flag/db", "-sentry-dsn=https://sentry"})
	assert.NoError(t, err)
	assert.Equal(t, map[string]ezValue{
		"timeout":    {"timeout", "60"},
		"db":         {"db", "postgres://flag/db"},
		"sentry_dsn": {"sentry-dsn", "https://sentry"},
	}, values)
}
