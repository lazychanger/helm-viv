package utils

import (
	"strconv"
	"strings"
)

type Flags struct {
	flags map[string][]string
}

func (f *Flags) GetStringSlice(keys ...string) []string {
	f.init()
	val := make([]string, 0)
	for _, key := range keys {
		if vals, ok := f.flags[key]; ok {
			val = append(val, vals...)
		}
	}
	return val
}

func (f *Flags) GetString(keys ...string) string {
	f.init()
	for _, key := range keys {
		if vals, ok := f.flags[key]; ok && len(vals) > 0 {
			return vals[0]
		}
	}

	return ""
}

func (f *Flags) GetInt(keys ...string) int {
	f.init()
	val := f.GetString(keys...)
	intVal, _ := strconv.Atoi(val)
	return intVal
}

func (f *Flags) GetBool(key string) bool {
	f.init()

	if vals, ok := f.flags[key]; ok {
		if len(vals) == 0 {
			return true
		}
		res, _ := strconv.ParseBool(f.flags[key][0])
		return res
	}
	return false
}

func (f *Flags) set(key string, val ...string) {
	f.init()
	if _, ok := f.flags[key]; !ok {
		f.flags[key] = make([]string, 0)
	}
	f.flags[key] = append(f.flags[key], val...)
}

func (f *Flags) init() {
	if f.flags == nil {
		f.flags = make(map[string][]string)
	}
}

func ParseFlags(args []string) *Flags {
	flags := &Flags{}

	next := false
	key := ""
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			if next {
				next = false
				if key != "" {
					flags.set(key, arg)
				}
			}
			continue
		}

		if next && key != "" {
			flags.set(key)
			key = ""
			next = false
		}

		if strings.HasPrefix(arg, "--") {
			idx := strings.Index(arg, "=")
			if idx > -1 {
				flags.set(strings.Trim(arg[0:idx], "-"), arg[idx+1:])
			} else {
				key = strings.Trim(arg, "-")
				next = true
			}
			continue
		}

		key = strings.Trim(arg, "-")
		next = true
	}

	// if last is type bool
	if next && key != "" {
		flags.set(key)
		key = ""
		next = false
	}

	return flags
}

func DefaultValue[T int | string | bool](val, eqValue, defaultValue T) T {
	if val == eqValue {
		return defaultValue
	}
	return val
}

func StringDefaultValue(val, defaultValue string) string {
	return DefaultValue(val, "", defaultValue)
}

func IntDefaultValue(val, defaultValue int) int {
	return DefaultValue(val, 0, defaultValue)
}

func BoolDefaultValue(val, defaultValue bool) bool {

	return DefaultValue(val, false, defaultValue)
}

func StringSliceDefaultValue(val, defaultValue []string) []string {
	if val == nil || len(val) == 0 {
		return defaultValue
	}
	return val
}
