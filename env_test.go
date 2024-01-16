package ezconf

import (
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
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
		{map[string]string{"FOO_MY_INT": "32", "FOO_IGNORE": "none"}, toFields(t, intStruct), map[string]ezValue{"my_int": {"FOO_MY_INT", "32"}}},
	}

	for _, tc := range tests {
		for k, v := range tc.env {
			os.Setenv(k, v)
		}

		val := parseEnv("foo", tc.fields)
		assert.Equal(t, tc.expected, val, "parseEnv failed for env: %s", tc.env)

		for k := range tc.env {
			os.Setenv(k, "")
		}
	}
}

func TestBuildUsage(t *testing.T) {
	stripWhitespace := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, s)
	}

	fields := toFields(t, allKinds{})
	expected := `Environment variables:
				 FOO_MY_BOOL - bool
             FOO_MY_DATETIME - datetime
                FOO_MY_FLOAT - float
                  FOO_MY_INT - int
               FOO_MY_STRING - string
                 FOO_MY_UINT - uint`
	usage := buildEnvUsage("foo", fields)

	assert.Equal(t, stripWhitespace(expected), stripWhitespace(usage))
}
