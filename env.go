package ezconf

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func parseEnv(name string, fields *ezFields) map[string]ezValue {
	values := make(map[string]ezValue)
	for _, snake := range fields.keys {
		env := strings.ToUpper(fmt.Sprintf("%s_%s", name, snake))
		value := os.Getenv(env)
		if value != "" {
			values[snake] = ezValue{env, value}
		}
	}
	return values
}

func buildEnvUsage(name string, fields *ezFields) string {
	usage := strings.Builder{}
	usage.WriteString("Environment variables:\n")

	for _, snake := range fields.keys {
		f := fields.fields[snake]

		env := strings.ToUpper(fmt.Sprintf("%s_%s", name, snake))
		switch f.Value().(type) {
		case int, int8, int16, int32, int64:
			fmt.Fprintf(&usage, "    % 40s - int\n", env)
		case uint, uint8, uint16, uint32, uint64:
			fmt.Fprintf(&usage, "    % 40s - uint\n", env)
		case float32, float64:
			fmt.Fprintf(&usage, "    % 40s - float\n", env)
		case bool:
			fmt.Fprintf(&usage, "    % 40s - bool\n", env)
		case string:
			fmt.Fprintf(&usage, "    % 40s - string\n", env)
		case []int:
			fmt.Fprintf(&usage, "    % 40s - comma separated integer list\n", env)
		case []string:
			fmt.Fprintf(&usage, "    % 40s - comma separated string list\n", env)
		case time.Time:
			fmt.Fprintf(&usage, "    % 40s - datetime\n", env)
		}
	}
	return usage.String()
}
