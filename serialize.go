package sorm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

func selectInBuilder(s *selectedTable, offset int) []interface{} {
	selects := make([]interface{}, len(s.modelFields))

	for i, f := range s.modelFields {
		if s.alias == "" {
			selects[i] = fmt.Sprintf("[!%s.%s] as q%d", s.model.ModelName, f.Name, offset+i)
		} else {
			selects[i] = fmt.Sprintf("%s.[%s.%s] as q%d", s.alias, s.model.ModelName, f.Name, offset+i)
		}
	}
	return selects
}

func convertToDbType(src reflect.Value) interface{} {
	if src.Kind() == reflect.Bool {
		if src.Bool() {
			return 1
		} else {
			return 0
		}
	}
	return src.Interface()
}

func createStorage(dest reflect.Value) reflect.Value {
	for dest.Kind() == reflect.Ptr {
		dest.Set(reflect.New(dest.Type().Elem()))
		dest = dest.Elem()
	}
	return dest
}
func convertInt(dest reflect.Value, v int64) bool {
	dest = createStorage(dest)
	switch dest.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dest.SetInt(v)
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		dest.SetUint(uint64(v))
		return true
	case reflect.Float32, reflect.Float64:
		dest.SetFloat(float64(v))
		return true
	case reflect.Bool:
		dest.SetBool(v != 0)
		return true
	}
	return false
}
func convertUint(dest reflect.Value, v uint64) bool {
	dest = createStorage(dest)
	switch dest.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dest.SetInt(int64(v))
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		dest.SetUint(v)
		return true
	case reflect.Float32, reflect.Float64:
		dest.SetFloat(float64(v))
		return true
	case reflect.Bool:
		dest.SetBool(v != 0)
		return true
	}
	return false
}
func convertFloat(dest reflect.Value, v float64) bool {
	dest = createStorage(dest)
	switch dest.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dest.SetInt(int64(v))
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		dest.SetUint(uint64(v))
		return true
	case reflect.Float32, reflect.Float64:
		dest.SetFloat(v)
		return true
	}
	return false
}
func convertBool(dest reflect.Value, v bool) bool {
	i := 0
	if v {
		i = 1
	}
	dest = createStorage(dest)
	switch dest.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dest.SetInt(int64(i))
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		dest.SetUint(uint64(i))
		return true
	case reflect.Bool:
		dest.SetBool(v)
		return true
	}
	return false
}

func setFieldValue(dest reflect.Value, src interface{}, ctype *sql.ColumnType) error {
	if !dest.CanAddr() && dest.Kind() == reflect.Ptr {
		// if the value is not addressable we can't modify it with Set,
		// thus if it's a pointer we are supposed to dereference it once
		dest = dest.Elem()
	}

	success := false
	switch s := src.(type) {
	case nil:
		dest.Set(reflect.Zero(dest.Type()))
		success = true
	case int, int8, int16, int32, int64:
		v := reflect.ValueOf(s).Int()
		success = convertInt(dest, v)
	case uint, uint8, uint16, uint32, uint64, uintptr:
		v := reflect.ValueOf(s).Uint()
		success = convertUint(dest, v)
	case float32, float64:
		v := reflect.ValueOf(s).Float()
		success = convertFloat(dest, v)
	case string:
		dest = createStorage(dest)
		if dest.Kind() == reflect.String {
			dest.SetString(s)
			success = true
		}
	case []uint8:
		dest = createStorage(dest)
		if dest.Kind() == reflect.String {
			dest.SetString(string(s))
			success = true
		}
		if ctype.DatabaseTypeName() == "MONEY" || ctype.DatabaseTypeName() == "SMALLMONEY" {
			f, err := strconv.ParseFloat(string(s), 64)
			if err == nil {
				dest.SetFloat(f)
				success = true
			}
		}
	case time.Time:
		dest = createStorage(dest)
		v := reflect.ValueOf(s)
		if dest.Type() == v.Type() {
			dest.Set(v)
			success = true
		}
	case bool:
		success = convertBool(dest, s)
	default:
		_ = s
	}

	if !success {
		return fmt.Errorf("unmatched types (got %v, expected %v)", reflect.TypeOf(src), dest.Type())
	}

	return nil
}

func encodeFromRow(s *selectedTable, row []interface{}, offset int, t interface{}, cols []*sql.ColumnType) error {
	tv := reflect.ValueOf(t)
	if tv.Kind() != reflect.Ptr {
		return fmt.Errorf("t must be a pointer")
	}

	for i, f := range s.modelFields {
		colTyp := cols[offset+i]
		rowVal := *(row[offset+i].(*interface{}))
		elem := tv.Elem().FieldByIndex(f.StructFieldPath)

		err := setFieldValue(elem, rowVal, colTyp)
		if err != nil {
			return fmt.Errorf("on field %v.%v: %w", s.model.ModelName, f.Name, err)
		}
	}
	return nil
}
