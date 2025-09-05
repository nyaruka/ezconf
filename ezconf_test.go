package ezconf

import (
	"fmt"
	"log/slog"
	"os"
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
	MyDatetime time.Time
	MyLogLevel slog.Level
}

func toFields(t *testing.T, s any) *ezFields {
	fields, err := buildFields(s)
	if err != nil {
		t.Errorf("error building fields for %+v: %s", s, err)
		t.FailNow()
	}
	return fields
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
		{"my_float64", "12", false, "12"},
		{"my_float64", "wat", true, ""},

		{"my_bool", "true", false, "true"},
		{"my_bool", "wat", true, ""},

		{"my_string", "foozap", false, "foozap"},

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
	conf.SetArgs("-my-int=48", "-my-log-level=error", "-debug-conf")
	err := conf.Load()
	assert.NoError(t, err)
	assert.Equal(t, 48, at.MyInt)
	assert.Equal(t, slog.LevelError, at.MyLogLevel)
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
