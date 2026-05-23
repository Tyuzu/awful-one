package utils

import (
	"reflect"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func SortAndSlice[T any](
	items *[]T,
	sortDef bson.D,
	skip int64,
	limit int64,
) {
	if items == nil || len(*items) == 0 || limit <= 0 {
		return
	}

	field, direction := parseSort(sortDef)

	if field != "" {
		sort.SliceStable(*items, func(i, j int) bool {
			vi := reflect.ValueOf((*items)[i])
			vj := reflect.ValueOf((*items)[j])

			fi := indirectField(vi, field)
			fj := indirectField(vj, field)

			if !fi.IsValid() || !fj.IsValid() {
				return false
			}

			less := compareValues(fi, fj)
			if direction < 0 {
				return !less
			}
			return less
		})
	}

	start := int(skip)
	if start >= len(*items) {
		*items = []T{}
		return
	}

	end := start + int(limit)
	if end > len(*items) {
		end = len(*items)
	}

	*items = (*items)[start:end]
}

func parseSort(sortDef bson.D) (string, int) {
	if len(sortDef) == 0 {
		return "", 1
	}

	dir := 1
	switch v := sortDef[0].Value.(type) {
	case int:
		dir = v
	case int32:
		dir = int(v)
	case int64:
		dir = int(v)
	}

	return sortDef[0].Key, dir
}

func indirectField(v reflect.Value, field string) reflect.Value {
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	return v.FieldByNameFunc(func(name string) bool {
		return equalFold(name, field)
	})
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca == cb {
			continue
		}
		if ca|0x20 != cb|0x20 {
			return false
		}
	}
	return true
}

func compareValues(a, b reflect.Value) bool {
	if !a.IsValid() || !b.IsValid() {
		return false
	}

	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() < b.Int()
	case reflect.Float32, reflect.Float64:
		return a.Float() < b.Float()
	case reflect.String:
		return a.String() < b.String()
	case reflect.Struct:
		if ta, ok := a.Interface().(time.Time); ok {
			if tb, ok := b.Interface().(time.Time); ok {
				return ta.Before(tb)
			}
		}
	}

	return false
}
