// Copied from github.com/samber/lo v1.53.0 (MIT; see LICENSE):
// https://github.com/samber/lo/blob/v1.53.0/type_manipulation.go
// https://github.com/samber/lo/blob/v1.53.0/condition.go
// https://github.com/samber/lo/blob/v1.53.0/map.go
// https://github.com/samber/lo/blob/v1.53.0/slice.go
// https://github.com/samber/lo/blob/v1.53.0/string.go
// Only the helpers used by is-healthy are retained; implementations are unchanged.
package lo

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ToPtr[T any](x T) *T {
	return &x
}

func FromPtr[T any](x *T) T {
	if x == nil {
		return Empty[T]()
	}
	return *x
}

func Empty[T any]() T {
	var zero T
	return zero
}

func Coalesce[T comparable](values ...T) (T, bool) {
	var zero T
	for i := range values {
		if values[i] != zero {
			return values[i], true
		}
	}
	return zero, false
}

func CoalesceOrEmpty[T comparable](v ...T) T {
	result, _ := Coalesce(v...)
	return result
}

func Ternary[T any](condition bool, ifOutput, elseOutput T) T {
	if condition {
		return ifOutput
	}
	return elseOutput
}

func Keys[K comparable, V any](in ...map[K]V) []K {
	size := 0
	for i := range in {
		size += len(in[i])
	}
	result := make([]K, 0, size)
	for i := range in {
		for k := range in[i] {
			result = append(result, k)
		}
	}
	return result
}

func Map[T, R any](collection []T, transform func(item T, index int) R) []R {
	result := make([]R, len(collection))
	for i := range collection {
		result[i] = transform(collection[i], i)
	}
	return result
}

func Compact[T comparable, Slice ~[]T](collection Slice) Slice {
	var zero T
	result := make(Slice, 0, len(collection))
	for i := range collection {
		if collection[i] != zero {
			result = append(result, collection[i])
		}
	}
	return result
}

func Capitalize(str string) string {
	return cases.Title(language.English).String(str)
}

func Ellipsis(str string, length int) string {
	str = strings.TrimSpace(str)
	const ellipsis = "..."
	cutPosition := 0
	for i := range str {
		if length == len(ellipsis) {
			cutPosition = i
		}
		if length--; length < 0 {
			return strings.TrimSpace(str[:cutPosition]) + ellipsis
		}
	}
	return str
}
