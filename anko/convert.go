package anko

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/mattn/anko/env"
)

// importToX defines type coercion functions
func importToX(e *env.Env) {

	e.Define("bool", func(v interface{}) bool {
		rv := reflect.ValueOf(v)
		if !rv.IsValid() {
			return false
		}
		nt := reflect.TypeOf(true)
		if rv.Type().ConvertibleTo(nt) {
			return rv.Convert(nt).Bool()
		}
		if rv.Type().ConvertibleTo(reflect.TypeOf(1.0)) && rv.Convert(reflect.TypeOf(1.0)).Float() > 0.0 {
			return true
		}
		if rv.Kind() == reflect.String {
			s := strings.ToLower(v.(string))
			if s == "y" || s == "yes" {
				return true
			}
			b, err := strconv.ParseBool(s)
			if err == nil {
				return b
			}
		}
		return false
	})

	e.Define("string", func(v interface{}) string {
		if b, ok := v.([]byte); ok {
			return string(b)
		}
		return fmt.Sprint(v)
	})

	e.Define("int", func(v interface{}) int64 {
		rv := reflect.ValueOf(v)
		if !rv.IsValid() {
			return 0
		}
		nt := reflect.TypeOf(1)
		if rv.Type().ConvertibleTo(nt) {
			return rv.Convert(nt).Int()
		}
		if rv.Kind() == reflect.String {
			i, err := strconv.ParseInt(v.(string), 10, 64)
			if err == nil {
				return i
			}
			f, err := strconv.ParseFloat(v.(string), 64)
			if err == nil {
				return int64(f)
			}
		}
		if rv.Kind() == reflect.Bool {
			if v.(bool) {
				return 1
			}
		}
		return 0
	})

	e.Define("float", func(v interface{}) float64 {
		rv := reflect.ValueOf(v)
		if !rv.IsValid() {
			return 0
		}
		nt := reflect.TypeOf(1.0)
		if rv.Type().ConvertibleTo(nt) {
			return rv.Convert(nt).Float()
		}
		if rv.Kind() == reflect.String {
			f, err := strconv.ParseFloat(v.(string), 64)
			if err == nil {
				return f
			}
		}
		if rv.Kind() == reflect.Bool {
			if v.(bool) {
				return 1.0
			}
		}
		return 0.0
	})

	e.Define("char", func(s rune) string {
		return string(s)
	})

	e.Define("rune", func(s string) rune {
		if len(s) == 0 {
			return 0
		}
		return []rune(s)[0]
	})
}
