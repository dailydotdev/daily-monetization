package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
)

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

// ThisMethodName returns the name of the method or function from which this function was called.
func ThisMethodName() (res string) {
	fpcs := make([]uintptr, 1)

	res = "error: could not determine the current method name"
	// Skip 2 levels to get the caller
	n := runtime.Callers(2, fpcs)
	if n == 0 {
		return
	}

	caller := runtime.FuncForPC(fpcs[0] - 1)
	if caller == nil {
		return
	}

	tkns := strings.Split(caller.Name(), ".")

	return tkns[len(tkns)-1]
}

func EnvGet[T string | int | int64 | bool](key string, fallback ...T) (result T, err error) {

	v, ok := os.LookupEnv(key)
	if !ok {

		if len(fallback) > 0 {
			result = fallback[0]
			return
		}

		err = fmt.Errorf("env variable %s not found", key)
		return
	}

	var ret any
	switch any(result).(type) {
	case int:
		ret, err = strconv.Atoi(v)
	case int64:
		ret, err = strconv.ParseInt(v, 10, 64)
	case string:
		ret = v
	case bool:
		ret, err = strconv.ParseBool(v)
	}

	if err != nil {
		return
	}

	result = ret.(T)
	return
}

func AllBool[T any](s []T, f func(t T) bool) bool {
	for _, i := range s {
		if !f(i) {
			return false
		}
	}

	return true
}

func AnyBool[T any](s []T, f func(t T) bool) bool {
	for _, i := range s {
		if f(i) {
			return true
		}
	}

	return false
}

func Map[T any, M any](s []T, f func(T) M) []M {
	var result []M
	for _, t := range s {
		result = append(result, f(t))
	}
	return result
}

func SliceToMap[T any, M comparable](s []T, f func(T) M) map[M]T {
	m := make(map[M]T, len(s))
	for _, t := range s {
		m[f(t)] = t
	}
	return m
}

func SliceToMapKeys[T comparable, V any](s []T, f func(T) V) map[T]V {
	m := make(map[T]V, len(s))
	for _, t := range s {
		m[t] = f(t)
	}
	return m
}

func Contains[T comparable](s []T, e T) bool {
	for _, i := range s {
		if i == e {
			return true
		}
	}

	return false
}

func ContainsAny[T comparable](k []T, l []T) bool {
	for _, i := range k {
		for _, k := range l {
			if i == k {
				return true
			}
		}
	}

	return false
}

func ContainsAll[T comparable](k []T, l []T) bool {
	if len(l) > len(k) {
		return false
	}

	for _, i := range l {
		if !Contains[T](k, i) {
			return false
		}
	}

	return true
}

func FilterAny[T comparable](from, to []T) (result []T) {
	return mapset.NewSet[T](from...).Difference(mapset.NewSet[T](to...)).ToSlice()
}

func FilterAnyFunc[T any, M any](from []T, to []M, filter func(T, M) bool) (result, removed []T) {

	for _, f := range from {
		remove := false
		for _, t := range to {
			if filter(f, t) {
				remove = true
			}
		}
		if remove {
			removed = append(removed, f)
			continue
		}

		result = append(result, f)
	}

	return
}

func Default[T comparable](value T, fallback T) (result T) {
	if value == result {
		return fallback
	}

	return value
}

func PaginateAsync[T any](slice []T, pageSize int) <-chan []T {

	ch := make(chan []T)

	go func(s []T) {

		defer close(ch)

		for {
			if len(s) == 0 {
				return
			}

			if len(s) <= pageSize {
				ch <- s[:]
				return
			}

			send := s[:pageSize]
			ch <- send

			s = s[pageSize:]
		}

	}(slice)

	return ch
}

func PaginateSync[T any](slice []T, pageSize int) func() []T {

	skip := 0

	return func() []T {

		if pageSize == 0 {
			return nil
		}

		start := skip

		skip = skip + pageSize

		if start == len(slice) {
			return nil
		}

		if len(slice) == 0 {
			return nil
		}

		if skip > len(slice) {
			skip = len(slice)
		}

		return slice[start:skip]
	}

}

func GetIndexes[T any](iterable []T, f func(T) bool) (result []int) {

	for i, val := range iterable {
		if f(val) {
			result = append(result, i)
		}
	}

	return
}

func MapToSlice[K comparable, V any](m map[K]V) (result []K) {

	for k := range m {
		result = append(result, k)
	}

	return
}

func ConvertToInterfaceSlice[T any](s []T) (result []interface{}) {
	result = make([]interface{}, len(s))
	for i, v := range s {
		result[i] = v
	}
	return
}

func BatchGenerator[T any](counter int, maxBatchSize int, object T, f func(int) int) <-chan []T {

	c := make(chan []T, counter/maxBatchSize)

	go func() {
		defer close(c)

		b := make([]T, 0)
		batchSize := maxBatchSize
		if f != nil {
			batchSize = f(maxBatchSize)
		}

		for i := 0; i < counter; i++ {

			if len(b) < batchSize {
				b = append(b, object)
			} else {
				c <- b
				b = make([]T, 0)
				batchSize = maxBatchSize
				if f != nil {
					batchSize = f(maxBatchSize)
				}
				b = append(b, object)
			}
		}

		if len(b) > 0 {
			c <- b
		}

	}()

	return c
}

func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

func Values[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

func ConvertI(in any, out any) error {

	if reflect.TypeOf(out).Kind() != reflect.Ptr {
		return errors.New("out should be pointer")
	}

	switch v := in.(type) {
	case []byte:
		return json.Unmarshal(v, out)
	case string:
		return json.Unmarshal([]byte(v), out)
	default:
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, out)
	}
}

func Convert[T any](in any) (T, error) {
	var v T
	return v, ConvertI(in, &v)
}

func ConvertSlice[T any](in []T) []any {
	slice := make([]any, len(in))
	for i, v := range in {
		slice[i] = v
	}

	return slice
}

func Ptr[T any](t T) *T {
	return &t
}
