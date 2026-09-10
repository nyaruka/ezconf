package ezconf

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/naoina/toml"
	"github.com/naoina/toml/ast"
)

// Iterates the list of files, parsing the first that is found and loading the
// result into our config struct. If no files are passed in or
// no files are found, this is a noop.
func parseTOMLFiles(fields *ezFields, files []string) error {
	// search through our list of files, stopping when we find one
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			// not finding a file is ok, we just move on
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		// if we can't parse this file as TOML, that's a nogo
		if err := parseTOML(data, fields); err != nil {
			return err
		}

		// we break at the first file we find
		break
	}

	return nil
}

// Parses the given TOML document into our config struct. The fields of embedded structs are promoted so are
// set from top level keys just like the config struct's own fields, never from a table named after the
// embedded field. So we split the top level keys between the config struct and its embedded structs, rejecting
// any that none of them define, and then decode each struct from the keys it owns.
func parseTOML(data []byte, fields *ezFields) error {
	table, err := toml.Parse(data)
	if err != nil {
		return err
	}

	// process keys in document order so that errors point at the first problem
	keys := make([]string, 0, len(table.Fields))
	for key := range table.Fields {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		li, lj := tomlLine(table.Fields[keys[i]]), tomlLine(table.Fields[keys[j]])
		return li < lj || (li == lj && keys[i] < keys[j])
	})

	owned := make([]map[string]any, len(fields.structs))
	for i := range owned {
		owned[i] = make(map[string]any)
	}

	for _, key := range keys {
		owner := -1
		for i, es := range fields.structs {
			if definesTOMLKey(reflect.TypeOf(es.ptr).Elem(), key) {
				owner = i
				break
			}
		}
		if owner < 0 {
			return fmt.Errorf("line %d: unknown key '%s'", tomlLine(table.Fields[key]), key)
		}
		owned[owner][key] = table.Fields[key]
	}

	for i, es := range fields.structs {
		sub := *table
		sub.Fields = owned[i]
		if err := tomlConfig.UnmarshalTable(&sub, es.ptr); err != nil {
			return err
		}
	}

	return nil
}

// We build our own decoder config that uses our own CamelToSnake and is a bit stricter with
// matching of fields in our TOML file. (they must match CamelToSnake)
var tomlConfig = &toml.Config{
	NormFieldName: camelNormalizer,
	FieldToKey:    camelKey,
}

// Returns whether the given struct type has a field which the decoder would set from the given TOML key,
// matching the same way it does. Embedded structs don't count as they're decoded from the top level keys
// they own rather than as a table.
func definesTOMLKey(typ reflect.Type, key string) bool {
	normKey := camelNormalizer(typ, key)

	for i := range typ.NumField() {
		ft := typ.Field(i)
		if !ft.IsExported() || (ft.Anonymous && ft.Type.Kind() == reflect.Struct) {
			continue
		}

		// a toml tag names the key explicitly and isn't normalized
		if tag, _, _ := strings.Cut(ft.Tag.Get("toml"), ","); tag != "" {
			if tag == key {
				return true
			}
			continue
		}

		if camelNormalizer(typ, ft.Name) == normKey {
			return true
		}
	}
	return false
}

// returns the line that the given top level entry of a TOML document appears on
func tomlLine(node any) int {
	switch n := node.(type) {
	case *ast.KeyValue:
		return n.Line
	case *ast.Table:
		return n.Line
	case []*ast.Table:
		return n[0].Line
	}
	return 0
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
