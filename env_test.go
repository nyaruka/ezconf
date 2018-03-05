package ezconf

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestParseEnv(t *testing.T) {
	intStruct := struct {
		MyInt   int
		MyFloat float64
	}{
		MyInt:   32,
		MyFloat: 32.0,
	}

	tests := []struct {
		env      map[string]string
		fields   *ezFields
		expected map[string]ezValue
	}{
		{map[string]string{"FOO_MY_INT": "32", "FOO_IGNORE": "none"}, toFields(t, intStruct), map[string]ezValue{"my_int": ezValue{"FOO_MY_INT", "32"}}},
	}

	for _, tc := range tests {
		for k, v := range tc.env {
			os.Setenv(k, v)
		}
		val := parseEnv("foo", tc.fields)
		if !reflect.DeepEqual(val, tc.expected) {
			t.Errorf("parseEnv failed for env: %s, expected: %s, got: %s", tc.env, tc.expected, val)
		}
		for k := range tc.env {
			os.Setenv(k, "")
		}
	}
}

func TestBuildUsage(t *testing.T) {
	fields := toFields(t, allKinds{})
	expected := `Environment variables:
                 FOO_MY_BOOL - bool
                FOO_MY_FLOAT - float
                  FOO_MY_INT - int
               FOO_MY_STRING - string
                 FOO_MY_UINT - uint`
	usage := buildEnvUsage("foo", fields)
	if strings.Trim(usage, " \n\r\t") != strings.Trim(expected, " \n\r\t") {
		t.Errorf("Unexpected usage: %s", usage)
	}
}
