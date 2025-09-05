package ezconf

import (
	"strings"
	"time"
	"unicode"
)

// CamelToSnake converts a CamelCase strings to a snake_case using the following algorithm:
//
//  1. for every transition from upper->lowercase insert an underscore before the uppercase character
//
//  2. for every transition fro lowercase->uppercase insert an underscore before the uppercase
//
//  3. lowercase resulting string
//
//     Examples:
//     CamelCase -> camel_case
//     AWSConfig -> aws_config
//     IPAddress -> ip_address
//     S3MediaPrefix -> s3_media_prefix
//     Route53Region -> route53_region
//     CamelCaseA -> camel_case_a
//     CamelABCCaseDEF -> camel_abc_case_def
func CamelToSnake(camel string) string {
	snakes := make([]string, 0, 4)
	snake := strings.Builder{}
	runes := []rune(camel)

	// two transitions:
	//    we are upper, next is lower
	//    we are lower, next is upper
	for i := range runes {
		r := runes[i]
		hasNext := i+1 < len(runes)

		if snake.Len() == 0 {
			// no snake, just append to it
			snake.WriteRune(r)
		} else if r == '_' {
			// if we are at an underscore, that's a boundary, create a snake but ignore the underscore
			snakes = append(snakes, snake.String())
			snake.Reset()
		} else if unicode.IsLower(r) && hasNext && unicode.IsUpper(runes[i+1]) {
			// if we are lowercase and the next item is uppercase, that's a transtion
			snake.WriteRune(r)
			snakes = append(snakes, snake.String())
			snake.Reset()
		} else if unicode.IsUpper(r) && hasNext && unicode.IsLower(runes[i+1]) {
			// if we are uppercase and the next item is lowercase, that's a transition
			snakes = append(snakes, snake.String())
			snake.Reset()
			snake.WriteRune(r)
		} else {
			// otherwise, add to our current snake
			snake.WriteRune(r)
		}
	}

	// if we have a trailing snake, add it
	if snake.Len() > 0 {
		snakes = append(snakes, snake.String())
	}

	// join everything together with _ and lowercase
	return strings.ToLower(strings.Join(snakes, "_"))
}

// TOML supported datetime formats
var timeFormats = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05.999999999",
}

func formatDatetime(t time.Time) string {
	return t.Format(timeFormats[0])
}
