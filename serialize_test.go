package db

import (
	"fmt"
	"reflect"
	"testing"
)

type Test struct {
	A int32
	B float32
	C string
}

func (t *Test) TableName() string {
	return "test"
}

var result1 *Test

func ConvertToInt32(v interface{}) int32 {
	switch t := v.(type) {
	case int:
		return int32(t)
	case int8:
		return int32(t)
	case int16:
		return int32(t)
	case int32:
		return int32(t)
	case int64:
		return int32(t)
	case float32:
		return int32(t)
	case float64:
		return int32(t)
	}
	panic("wrong type for v")
}

func ConvertToFloat32(v interface{}) float32 {
	switch t := v.(type) {
	case int:
		return float32(t)
	case int8:
		return float32(t)
	case int16:
		return float32(t)
	case int32:
		return float32(t)
	case int64:
		return float32(t)
	case float32:
		return float32(t)
	case float64:
		return float32(t)
	}
	panic("wrong type for v")
}

func BenchmarkStructAccess(b *testing.B) {
	var t *Test
	for i := 0; i < b.N; i++ {
		row := []interface{}{int(1), float64(2), "hello"}
		t = &Test{ConvertToInt32(row[0]), ConvertToFloat32(row[1]), row[2].(string)}
	}
	result1 = t
}

var result2 interface{}

func BenchmarkReflectAccess(b *testing.B) {
	AddModel(&Test{})
	m := modelNameCache["Test"]
	b.ResetTimer()

	var t interface{}
	for i := 0; i < b.N; i++ {
		row := []interface{}{int(1), float32(2), "hello"}
		tv := reflect.New(m.Type)
		for i, f := range m.Fields {
			rowVal := *(row[i].(*interface{}))
			elem := tv.Elem().FieldByIndex(f.StructFieldPath)

			err := setFieldValue(elem, rowVal, nil)
			if err != nil {
				panic(fmt.Errorf("on field %v.%v: %w", m.ModelName, f.Name, err))
			}
		}
		t = tv.Interface()
	}
	result2 = t
}
