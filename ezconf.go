package ezconf

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/structs"
)

var validNameTag = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// ErrHelp is returned by Load when usage information was requested with -help or -h. It is the
// same sentinel as flag.ErrHelp, so errors.Is works against either.
var ErrHelp = flag.ErrHelp

// names claimed by the flags we add ourselves, which fields can't use
var reservedNames = map[string]bool{"help": true}

// Loader allows you to load your configuration from four sources, in order of priority (later overrides earlier):
//  1. The default values of your configuration struct
//  2. TOML files you specify (optional)
//  3. Set environment variables
//  4. Command line parameters
type Loader struct {
	name        string
	description string
	config      any
	files       []string
	args        []string

	// we hang onto this to print usage where needed
	flags *flag.FlagSet
}

// NewLoader creates a new EZLoader for the passed in configuration. `config` should be a pointer to a struct.
// `name` and `description` are used to build environment variables and help parameters. The list of files
// can be nil, or can contain optional files to read TOML configuration from in priority order. The first file
// found and parsed will end parsing of others, but there is no requirement that any file is found.
func NewLoader(config any, name string, description string, files []string) *Loader {
	return &Loader{
		name:        name,
		description: description,
		config:      config,
		files:       files,
		args:        os.Args[1:],
	}
}

// SetArgs allows you to override the command line arguments to be parsed. This is primarily useful for tests.
func (l *Loader) SetArgs(args ...string) {
	l.args = args
}

// MustLoad loads our configuration from our sources in the order of:
//  1. TOML files
//  2. Environment variables
//  3. Command line parameters
//
// If any error is encountered, the program will exit reporting the error and showing usage. If
// usage was requested with -help or -h, it is shown and the program exits.
func (l *Loader) MustLoad() {
	err := l.Load()
	if err == nil {
		return
	}

	// asking for usage isn't a configuration error, so don't report it as one
	if !errors.Is(err, ErrHelp) {
		fmt.Printf("Error while reading configuration: %s\n\n", err.Error())
	}

	l.Usage()
	os.Exit(1)
}

// Usage writes usage information for this configuration to stderr. It is a noop if called before
// Load or MustLoad, as the flags it describes haven't been built yet.
func (l *Loader) Usage() {
	if l.flags != nil {
		l.flags.Usage()
	}
}

// Load loads our configuration from our sources in the order of:
//  1. TOML files
//  2. Environment variables
//  3. Command line parameters
//
// If any error is encountered it is returned for the caller to process. Load never writes to
// stdout or stderr and never exits the program. If usage was requested with -help or -h, ErrHelp
// is returned and it is up to the caller to show usage via Usage.
func (l *Loader) Load() error {
	// first build our mapping of name snake_case -> structs.Field
	fields, err := buildFields(l.config)
	if err != nil {
		return err
	}

	// build our flags, silencing the flag package so that parse errors come back to us as errors
	// rather than being printed and exiting the program
	l.flags = buildFlags(l.name, l.description, fields, flag.ContinueOnError)
	l.flags.SetOutput(io.Discard)

	// parse them, then restore output so that callers showing usage get it on stderr
	flagValues, err := parseFlags(l.flags, l.args)
	l.flags.SetOutput(os.Stderr)
	if err != nil {
		return err
	}

	// if they asked for usage, let the caller decide how to present it
	if l.flags.Lookup("help").Value.String() == "true" {
		return ErrHelp
	}

	// read any found file into our config
	err = parseTOMLFiles(l.config, l.files)
	if err != nil {
		return err
	}

	// parse our environment
	envValues := parseEnv(l.name, fields)
	err = setValues(fields, envValues)
	if err != nil {
		return err
	}

	// set our flag values
	err = setValues(fields, flagValues)
	if err != nil {
		return err
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

		parsed, err := parseValue(f.Value(), value)
		if err != nil {
			return err
		}

		if err := f.Set(parsed); err != nil {
			return fmt.Errorf("unable to set field %s: %w", f.Name(), err)
		}
	}
	return nil
}

// parses the given string into the same type as the passed in current value
func parseValue(current any, value string) (any, error) {
	switch current.(type) {
	case int:
		i, err := strconv.ParseInt(value, 10, strconv.IntSize)
		if err != nil {
			return nil, err
		}
		return int(i), nil
	case int8:
		i, err := strconv.ParseInt(value, 10, 8)
		if err != nil {
			return nil, err
		}
		return int8(i), nil
	case int16:
		i, err := strconv.ParseInt(value, 10, 16)
		if err != nil {
			return nil, err
		}
		return int16(i), nil
	case int32:
		i, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return nil, err
		}
		return int32(i), nil
	case int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return int64(i), nil

	case uint:
		i, err := strconv.ParseUint(value, 10, strconv.IntSize)
		if err != nil {
			return nil, err
		}
		return uint(i), nil
	case uint8:
		i, err := strconv.ParseUint(value, 10, 8)
		if err != nil {
			return nil, err
		}
		return uint8(i), nil
	case uint16:
		i, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return nil, err
		}
		return uint16(i), nil
	case uint32:
		i, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, err
		}
		return uint32(i), nil
	case uint64:
		i, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return uint64(i), nil

	case float32:
		d, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return nil, err
		}
		return float32(d), nil
	case float64:
		d, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return float64(d), nil

	case bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}
		return b, nil

	case string:
		return value, nil

	case []string:
		parts, err := csv.NewReader(strings.NewReader(value)).Read()
		if err != nil {
			return nil, err
		}
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		return parts, nil

	case []int:
		parts, err := csv.NewReader(strings.NewReader(value)).Read()
		if err != nil {
			return nil, err
		}
		ints := make([]int, len(parts))
		for i, p := range parts {
			n, err := strconv.ParseInt(strings.TrimSpace(p), 10, strconv.IntSize)
			if err != nil {
				return nil, err
			}
			ints[i] = int(n)
		}
		return ints, nil

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
			return nil, err
		}
		return t, nil

	case slog.Level:
		var level slog.Level
		if err := level.UnmarshalText([]byte(value)); err != nil {
			return nil, err
		}
		return level, nil

	default:
		return nil, fmt.Errorf("unsupported type %T", current)
	}
}

func buildFields(config any) (*ezFields, error) {
	// config must be a non-nil pointer to a struct, otherwise its fields aren't settable and
	// every value we read would be silently discarded
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("config must be a non-nil pointer to a struct, got %T", config)
	}

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
				[]string,
				[]int,
				time.Time,
				slog.Level:
				name := f.Tag("name")
				if name == "" {
					name = CamelToSnake(f.Name())
				} else if !validNameTag.MatchString(name) {
					return nil, fmt.Errorf("invalid name tag %q for field %s, must be snake_case", name, f.Name())
				}
				if reservedNames[name] {
					return nil, fmt.Errorf("%s uses reserved name %q", f.Name(), name)
				}
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
