package ezconf

import (
	"fmt"
	"os"
	"testing"
	"time"
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
}

func toFields(t *testing.T, s interface{}) *ezFields {
	fields, err := buildFields(s)
	if err != nil {
		t.Errorf("error building fields for %+v: %s", s, err)
		t.FailNow()
	}
	return fields
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		camel string
		snake string
	}{
		{"CamelCase", "camel_case"},
		{"AWSAccessKey", "aws_access_key"},
		{"S3Region", "s3_region"},
		{"EC2Region", "ec2_region"},
		{"Route53", "route53"},
		{"AWS", "aws"},
		{"snake_case", "snake_case"},
		{"Snake_Camel", "snake_camel"},
		{"CamelCaseA", "camel_case_a"},
		{"CamelABCCaseDEF", "camel_abc_case_def"},
	}

	for _, tc := range tests {
		snake := CamelToSnake(tc.camel)
		if snake != tc.snake {
			t.Errorf("CamelToSnake of %s = %s instead of %s", tc.camel, snake, tc.snake)
		}
	}
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

		{"unknown", "", true, ""},
	}

	for _, tc := range tests {
		values := map[string]ezValue{tc.key: {tc.key, tc.value}}
		err := setValues(fields, values)
		if !tc.hasErr && err != nil {
			t.Errorf("unexpected error setting %s to %s: %s", tc.key, tc.value, err)
		}
		if tc.hasErr && err == nil {
			t.Errorf("expected error setting %s to %s", tc.key, tc.value)
		}
		field, found := fields.fields[tc.key]
		if found && !tc.hasErr && err == nil {
			strValue := fmt.Sprintf("%v", field.Value())
			if strValue != tc.expected {
				t.Errorf("value for %s not expected %s instead %s", tc.key, tc.expected, strValue)
			}
		}
	}
}

func TestEndToEnd(t *testing.T) {
	at := &allTypes{}
	conf := NewLoader(at, "foo", "description", []string{"missing.toml", "fields.toml", "simple.toml"})
	conf.args = []string{"-my-int=48", "-debug-conf"}
	err := conf.Load()
	if err != nil {
		t.Errorf("received error reading config: %s", err)
		return
	}
}

func TestPriority(t *testing.T) {
	at := &allTypes{MyInt: 16}
	conf := NewLoader(at, "foo", "description", []string{"missing.toml", "fields.toml", "simple.toml"})
	conf.args = []string{}
	conf.Load()

	if at.MyInt != 96 {
		t.Errorf("MyInt should be 96 from TOML is %d instead", at.MyInt)
	}

	// override with environment variable
	conf = NewLoader(at, "foo", "description", []string{"missing.toml", "fields.toml", "simple.toml"})
	conf.args = []string{}
	os.Setenv("FOO_MY_INT", "48")
	conf.Load()

	if at.MyInt != 48 {
		t.Errorf("MyInt should be 48 from Env is %d instead", at.MyInt)
	}

	// override with args
	conf = NewLoader(at, "foo", "description", []string{"missing.toml", "fields.toml", "simple.toml"})
	conf.args = []string{"-my-int=56"}
	os.Setenv("FOO_MY_INT", "48")
	conf.Load()

	if at.MyInt != 56 {
		t.Errorf("MyInt should be 56 from args is %d instead", at.MyInt)
	}

	// clear our env, args should take precedence now even though we are setting to the same as our new default
	os.Setenv("FOO_MY_INT", "")
	conf = NewLoader(at, "foo", "description", []string{"missing.toml", "fields.toml", "simple.toml"})
	conf.args = []string{"-my-int=56"}
	conf.Load()

	if at.MyInt != 56 {
		t.Errorf("MyInt should be 56 from args is %d instead", at.MyInt)
	}
}
