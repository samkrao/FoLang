package ast

import (
	"reflect"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

func errorObj(str string) helpers.ErrorInterface {

	stpos := helpers.NewPosition(1, 1, 1, 1, "", "", false)
	endpos := helpers.NewPosition(1, 1, 1, 1, "", "", false)
	return helpers.NewInvalidSyntaxError(*stpos, *endpos, str)

}

// CheckMapOrPointer returns 1 for a map, 2 for a pointer-to-map, 3 for other pointers, or 0 otherwise.
func CheckMapOrPointer(v any) int {
	t := reflect.TypeOf(v)
	k := t.Kind()

	switch k {
	case reflect.Map:
		return 1
	case reflect.Ptr:
		if t.Elem().Kind() == reflect.Map {
			return 2
		} else {
			return 3
		}
	default:
		return 0
	}

}
