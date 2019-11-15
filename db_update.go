package sorm

import (
	"fmt"
	"github.com/n1xx1/builder"
	"reflect"
)

func doUpdate(calldepth int, q DBTX, i interface{}, otherValues ...builder.Eq) error {
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
	values := builder.Eq{}
	for _, f := range model.Fields {
		val := v.FieldByIndex(f.StructFieldPath)
		if reflect.Zero(val.Type()).Interface() != val.Interface() {
			fieldName := fmt.Sprintf("[!%s.%s]", model.ModelName, f.Name)
			if f.IsPrimary {
				selects[fieldName] = convertToDbType(val)
			} else {
				values[fieldName] = convertToDbType(val)
			}
		}
	}
	for _, eq := range otherValues {
		for k, v := range eq {
			values[k] = v
		}
	}

	sql1, args, err := b.From("[" + model.ModelName + "]").Where(selects).Update(values).ToSQL()
	if err != nil {
		return fmt.Errorf("sql builder error: %w", err)
	}

	sql1 = FormatQuery(q.Driver(), sql1)
	sql1, args = ConvertQuery(q.Driver(), sql1, args)

	_, err = timedExec(q, sql1, args, calldepth)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

// Update updates the row the model represent using it's primary fields for the WHERE
// and all the non-zero values for the VALUES. Please notice that bool zero value is false,
// so you should either use *bool in the model or pass custom values for the update.
func Update(q DBTX, i interface{}, otherValues ...builder.Eq) error {
	return doUpdate(1, q, i, otherValues...)
}
