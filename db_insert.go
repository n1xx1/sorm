package sorm

import (
	"fmt"
	"github.com/n1xx1/builder"
	"reflect"
)

func doInsert(calldepth int, q DBTX, i interface{}) error {
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

	values := builder.Eq{}
	for _, f := range model.Fields {
		if f.IsAutoIncrement {
			continue
		}
		val := v.FieldByIndex(f.StructFieldPath)
		if val.Kind() != reflect.Ptr || !val.IsNil() {
			fieldName := fmt.Sprintf("[!%s.%s]", model.ModelName, f.Name)
			values[fieldName] = convertToDbType(val)
		}
	}

	sql1, args, err := b.Into("[" + model.ModelName + "]").Insert(values).ToSQL()
	if err != nil {
		return fmt.Errorf("sql builder: %w", err)
	}

	sql1 = FormatQuery(q.Driver(), sql1)
	sql1, args = ConvertQuery(q.Driver(), sql1, args)

	var id int64
	if q.Driver() == DriverMssql {
		sql1 += "; SELECT ID = CONVERT(BIGINT, SCOPE_IDENTITY())"

		rows, err := timedQuery(q, sql1, args, calldepth)
		if err != nil {
			return fmt.Errorf("database error: %w", err)
		}
		rows.Next()

		err = rows.Scan(&id)
		if err != nil {
			return fmt.Errorf("database error: %w", err)
		}
	} else {
		res, err := timedExec(q, sql1, args, calldepth)
		if err != nil {
			return fmt.Errorf("database error: %w", err)
		}
		id, err = res.LastInsertId()
		if err != nil {
			return fmt.Errorf("database error: %w", err)
		}
	}

	for _, f := range model.Fields {
		if f.IsAutoIncrement {
			elem := v.FieldByIndex(f.StructFieldPath)
			err := setFieldValue(elem, id, nil)

			if err != nil {
				return fmt.Errorf("autoincrement decode fail")
			}
		}
	}

	return nil
}

func Insert(q DBTX, i interface{}) error {
	return doInsert(1, q, i)
}
