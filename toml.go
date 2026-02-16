package ezconf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/naoina/toml"
)

// Iterates the list of files, parsing the first that is found and loading the
// result into the passed in struct pointer. If no files are passed in or
// no files are found, this is a noop.
func parseTOMLFiles(config any, files []string, debug bool) error {
	// search through our list of files, stopping when we find one
	for i, file := range files {
		toml, err := os.ReadFile(file)
		if err != nil {
			// not finding a file is ok, we just move on
			if os.IsNotExist(err) {
				if debug {
					fmt.Printf("CONF: Skipping missing TOML file: %s\n", file)
				}
				continue
			}
			return err
		}
		if debug {
			fmt.Printf("CONF: Parsing TOML file: %s\n", file)
		}
		decoder := newDecoder(bytes.NewReader(toml))
		err = decoder.Decode(config)

		// if we can't parse this file as TOML, that's a nogo
		if err != nil {
			return err
		}
		if debug {
			for i = i + 1; i < len(files); i++ {
				fmt.Printf("CONF: Previous file found, skipping TOML file: %s\n", files[i])
			}
		}

		// we break at the first file we find
		break
	}

	return nil
}

// We build our own decoder that uses our own CamelToSnake and is a bit stricter with
// matching of fields in our TOML file. (they must match CamelToSnake)
func newDecoder(r io.Reader) *toml.Decoder {
	tomlConfig := &toml.Config{
		NormFieldName: camelNormalizer,
		FieldToKey:    camelKey,
	}
	return tomlConfig.NewDecoder(r)
}

// resolveNameTag checks if a struct field has a `name` tag and returns it if present.
// Returns an empty string if the field doesn't exist or doesn't have a `name` tag.
func resolveNameTag(typ reflect.Type, field string) string {
	if typ.Kind() == reflect.Struct {
		if sf, ok := typ.FieldByName(field); ok {
			return sf.Tag.Get("name")
		}
	}
	return ""
}

// Satisfies the NormFieldName interface and is used to match TOML keys to struct fields.
// The function runs for both input keys and struct field names and should return a string
// that makes the two match.
func camelNormalizer(typ reflect.Type, keyOrField string) string {
	if name := resolveNameTag(typ, keyOrField); name != "" {
		return name
	}
	return CamelToSnake(keyOrField)
}

// Satisfies the FieldToKey interface and determines the TOML key of a struct field when encoding.
//
// Note that FieldToKey is not used for fields which define a TOML key through the struct tag.
func camelKey(typ reflect.Type, field string) string {
	if name := resolveNameTag(typ, field); name != "" {
		return name
	}
	return CamelToSnake(field)
}
