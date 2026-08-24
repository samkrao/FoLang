package helpers

import (
	"fmt"
	"reflect"
	"strings"
)

// ExpectType casts r to type T using reflection, panicking if the types do not match.
func ExpectType[T any](r any) T {
	expectedType := reflect.TypeOf((*T)(nil)).Elem()
	recievedType := reflect.TypeOf(r)

	if expectedType == recievedType {
		return r.(T)
	}

	panic(fmt.Sprintf("Expected %T but instead recived %T inside ExpectType[T](r)\n", expectedType, recievedType))
}

// PrintType prints the kind and type of the given value to standard output.
func PrintType(T any) {
	fmt.Println(reflect.TypeOf(T).Kind())
	fmt.Println(reflect.TypeOf(T))

}

// GetType returns the type name of T as a string, with package prefixes removed.
func GetType(T interface{}) string {
	st := strings.Replace(fmt.Sprint(reflect.TypeOf(T)), "backend.", "", -1)
	return strings.Replace(st, "ast.", "", -1)
}

// GetTypeObj returns the reflect.Type of the given value.
func GetTypeObj(T interface{}) reflect.Type {
	return reflect.TypeOf(T)
}
