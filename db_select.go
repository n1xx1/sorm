package sorm

import (
	"fmt"
	"github.com/n1xx1/builder"
	"reflect"
)

func doSelect(calldepth int, q DBTX, i interface{}) error {
	var b *builder.Builder
	if q.Driver() == DriverMssql {
		b = builder.MsSQL()
	} else {
		b = builder.MySQL()
	}

	v := reflect.ValueOf(i)
	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	model := modelCache[v.Type()]
	if model == nil {
		panic("model not found")
	}

	selects := builder.Eq{}
	for _, f := range model.PrimaryFields {
		val := v.FieldByIndex(f.StructFieldPath)
		fieldName := fmt.Sprintf("[!%s.%s]", model.ModelName, f.Name)
		selects[fieldName] = val.Interface()
	}

	qs, err := doQuery(calldepth+1, q, b.From("["+model.ModelName+"]").Where(selects), model.ModelName)
	if err != nil {
		return err
	}
	defer qs.Close()

	return qs.First(i)
}

/// Select is a shortcut to Query(q, query, model) and then queryScanner.First(i),
/// where the query is obtained from the interface primary fields.
/// If the interface primary field is a zero value then it's ignored from the select
/// if you want to query for zero values for a string, for example, you are supposed to
/// have *string as the type of the field.
func Select(q DBTX, i interface{}) error {
	return doSelect(1, q, i)
}
