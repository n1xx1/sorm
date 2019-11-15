package sorm

import (
	"fmt"
	"github.com/n1xx1/builder"
	"reflect"
)

func doFindTx(calldepth int, q DBTX, b *builder.Builder, dest interface{}) error {
	v := reflect.ValueOf(dest)
	if v.Type().Kind() != reflect.Ptr || v.Type().Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest parameter must be a pointer to slice")
	}
	v = v.Elem()

	elType := v.Type().Elem()
	destIsPtr := elType.Kind() == reflect.Ptr

	if destIsPtr {
		elType = elType.Elem()
	}

	var selectParams []interface{}

	model, isModel := modelCache[elType]
	if isModel {
		b = b.From("[" + model.ModelName + "]")
		selectParams = []interface{}{model.ModelName}
	}

	var d1 reflect.Value
	if !destIsPtr {
		// if []Type then we allocate only a single
		// store for our scanned values
		d1 = reflect.New(elType)
	}

	qs, err := doQuery(calldepth+1, q, b, selectParams...)
	if err != nil {
		return err
	}
	defer qs.Close()

	for qs.Next() {
		if destIsPtr {
			// if []*Type then we need a new store
			// for every scanned values
			d1 = reflect.New(elType)
		}

		if isModel {
			err := qs.Scan(d1.Interface())
			if err != nil {
				return err
			}
		} else {
			// it's not a model, use ScanTo
			err := qs.ScanTo(d1.Interface())
			if err != nil {
				return err
			}
		}

		appended := d1
		if !destIsPtr {
			appended = appended.Elem()
		}

		v.Set(reflect.Append(v, appended))
	}
	return nil
}

/// Find queries the database with the specified query (b) and fills the specified
/// slice of struct with their fields using the field name.
func Find(q DBTX, b *builder.Builder, dest interface{}) error {
	return doFindTx(1, q, b, dest)
}
