package internal

import "reflect"

func IsBytesType[T any]() bool {
	return reflect.TypeFor[T]() == reflect.TypeFor[[]byte]()
}
