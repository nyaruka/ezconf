package ezconf

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/structs"
)

// EZLoader allows you to load your configuration from four sources, in order of priority (later overrides earlier):
//  1. The default values of your configuration struct
//  2. TOML files you specify (optional)
//  3. Set environment variables
//  4. Command line parameters
type EZLoader struct {
	name        string
	description string
	config      any
	args        []string
	files       []string

	// we hang onto this to print usage where needed
	flags *flag.FlagSet
}

// NewLoader creates a new EZLoader for the passed in configuration. `config` should be a pointer to a struct.
// `name` and `description` are used to build environment variables and help parameters. The list of files
// can be nil, or can contain optional files to read TOML configuration from in priority order. The first file
// found and parsed will end parsing of others, but there is no requirement that any file is found.
func NewLoader(config any, name string, description string, files []string) *EZLoader {
	return &EZLoader{
		name:        name,
		description: description,
		config:      config,
		files:       files,
		args:        os.Args[1:],
	}
}

// SetArgs allows you to override the command line arguments to be parsed. This is primarily useful for tests.
func (ez *EZLoader) SetArgs(args ...string) {
	ez.args = args
}

// MustLoad loads our configuration from our sources in the order of:
//  1. TOML files
//  2. Environment variables
//  3. Command line parameters
//
// If any error is encountered, the program will exit reporting the error and showing usage.
func (ez *EZLoader) MustLoad() {
	err := ez.Load()
	if err != nil {
		fmt.Printf("Error while reading configuration: %s\n\n", err.Error())
		ez.flags.Usage()
		os.Exit(1)
	}
}

// Load loads our configuration from our sources in the order of:
//  1. TOML files
//  2. Environment variables
//  3. Command line parameters
//
// If any error is encountered it is returned for the caller to process.
func (ez *EZLoader) Load() error {
	// first build our mapping of name snake_case -> structs.Field
	fields, err := buildFields(ez.config)
	if err != nil {
		return err
	}

	// build our flags
	ez.flags = buildFlags(ez.name, ez.description, fields, flag.ExitOnError)

	// parse them
	flagValues, err := parseFlags(ez.flags, ez.args)
	if err != nil {
		return err
	}

	// if they asked for usage, show it
	if ez.flags.Lookup("help").Value.String() == "true" {
		ez.flags.Usage()
		os.Exit(1)
	}

	// if they asked for config debug, show it
	debug := false
	if ez.flags.Lookup("debug-conf").Value.String() == "true" {
		debug = true
	}

	if debug {
		printFields("Default overridable values:", fields)
	}

	// read any found file into our config
	err = parseTOMLFiles(ez.config, ez.files, debug)
	if err != nil {
		return err
	}

	if debug {
		printFields("Overridable values after TOML parsing:", fields)
	}

	// parse our environment
	envValues := parseEnv(ez.name, fields)
	err = setValues(fields, envValues)
	if err != nil {
		return err
	}

	// set our flag values
	err = setValues(fields, flagValues)
	if err != nil {
		return err
	}

	if debug {
		printValues("Command line overrides:", flagValues)
		printValues("Environment overrides:", envValues)
		printFields("Final top level values:", fields)
	}

	return nil
}

func setValues(fields *ezFields, values map[string]ezValue) error {
	// iterates all passed in values, attempting to set them, returning an error if
	// there are any type mismatches
	for name, cValue := range values {
		value := cValue.value

		f, found := fields.fields[name]
		if !found {
			return fmt.Errorf("unknown key '%s' for value '%s'", name, value)
		}

		switch f.Value().(type) {
		case int:
			i, err := strconv.ParseInt(value, 10, strconv.IntSize)
			if err != nil {
				return err
			}
			f.Set(int(i))
		case int8:
			i, err := strconv.ParseInt(value, 10, 8)
			if err != nil {
				return err
			}
			f.Set(int8(i))
		case int16:
			i, err := strconv.ParseInt(value, 10, 16)
			if err != nil {
				return err
			}
			f.Set(int16(i))
		case int32:
			i, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return err
			}
			f.Set(int32(i))
		case int64:
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			f.Set(int64(i))

		case uint:
			i, err := strconv.ParseUint(value, 10, strconv.IntSize)
			if err != nil {
				return err
			}
			f.Set(uint(i))

		case uint8:
			i, err := strconv.ParseUint(value, 10, 8)
			if err != nil {
				return err
			}
			f.Set(uint8(i))
		case uint16:
			i, err := strconv.ParseUint(value, 10, 16)
			if err != nil {
				return err
			}
			f.Set(uint16(i))
		case uint32:
			i, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return err
			}
			f.Set(uint32(i))
		case uint64:
			i, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			f.Set(uint64(i))

		case float32:
			d, err := strconv.ParseFloat(value, 32)
			if err != nil {
				return err
			}
			f.Set(float32(d))
		case float64:
			d, err := strconv.ParseFloat(value, 32)
			if err != nil {
				return err
			}
			f.Set(float64(d))

		case bool:
			b, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}
			f.Set(b)

		case string:
			f.Set(value)

		case time.Time:
			var t time.Time
			var err error

			switch {
			case !strings.Contains(value, ":"):
				t, err = time.Parse("2006-01-02", value)
			case !strings.Contains(value, "-"):
				t, err = time.Parse("15:04:05.999999999", value)
			default:
				for _, format := range timeFormats {
					t, err = time.Parse(format, value)
					if err == nil {
						break
					}
				}
			}

			if err != nil {
				return err
			}

			f.Set(t)

		case slog.Level:
			var level slog.Level
			err := level.UnmarshalText([]byte(value))
			if err != nil {
				return err
			}
			f.Set(level)
		}
	}
	return nil
}

func buildFields(config any) (*ezFields, error) {
	fields := make(map[string]*structs.Field)
	s := structs.New(config)
	for _, f := range s.Fields() {
		if f.IsExported() {
			switch f.Value().(type) {
			case int, int8, int16, int32, int64,
				uint, uint8, uint16, uint32, uint64,
				float32, float64,
				bool,
				string,
				time.Time,
				slog.Level:
				name := CamelToSnake(f.Name())
				dupe, found := fields[name]
				if found {
					return nil, fmt.Errorf("%s name collides with %s", dupe.Name(), f.Name())
				}
				fields[name] = f
			}
		}
	}

	// build our keys and sort them
	keys := make([]string, 0)
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return &ezFields{keys, fields}, nil
}

// utility struct for holding the snaked key, raw key (env all caps or flag) along with a read value
type ezValue struct {
	rawKey string
	value  string
}

// utility struct that holds our fields and an ordered list of the keys for predictable iteration
type ezFields struct {
	keys   []string
	fields map[string]*structs.Field
}

func printFields(header string, fields *ezFields) {
	fmt.Printf("CONF: %s\n", header)
	for _, k := range fields.keys {
		field := fields.fields[k]
		fmt.Printf("CONF: % 40s = %v\n", field.Name(), field.Value())
	}
	fmt.Println()
}

func printValues(header string, values map[string]ezValue) {
	fmt.Printf("CONF: %s\n", header)
	for _, v := range values {
		fmt.Printf("CONF: % 40s = %s\n", v.rawKey, v.value)
	}
	fmt.Println()
}
