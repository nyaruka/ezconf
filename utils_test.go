package ezconf_test

import (
	"testing"

	"github.com/nyaruka/ezconf"
	"github.com/stretchr/testify/assert"
)

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
		snake := ezconf.CamelToSnake(tc.camel)
		assert.Equal(t, tc.snake, snake, "to snake mismatch for %s", tc.camel)
	}
}
