package db

import (
	"fmt"
	"github.com/n1xx1/builder"
	"reflect"
)

// basically doFind but for a signle result
func doScanSingle(calldepth int, q DBTX, b *builder.Builder, dest interface{}) error {
	elType := reflect.TypeOf(dest)
	if elType.Kind() != reflect.Ptr {
		return fmt.Errorf("dest parameter must be a pointer")
	}
	elType = elType.Elem()
	isStruct := elType.Kind() == reflect.Struct

	var selectParams []interface{}

	model, isModel := modelCache[elType]
	if isModel {
		b = b.From("[" + model.ModelName + "]")
		selectParams = []interface{}{model.ModelName}
	}

	qs, err := doQuery(calldepth+1, q, b, selectParams...)
	if err != nil {
		return err
	}
	defer qs.Close()

	if !qs.Next() {
		return ErrEmptyResult
	}

	if isModel || !isStruct {
		// if !isStruct then it's going to do a manual scan as we want
		err = qs.Scan(dest)
	} else {
		err = qs.ScanTo(dest)
	}

	if err != nil {
		return err
	}
	return nil
}

func doScan(calldepth int, q DBTX, b *builder.Builder, dests ...interface{}) error {
	qs, err := doQuery(calldepth+1, q, b)
	if err != nil {
		return err
	}
	defer qs.Close()

	if !qs.Next() {
		return ErrEmptyResult
	}
	return qs.Scan(dests...)
}

// Scan queries the database with the specified query (b) and fills the specified
// slice of struct with their fields using the field name.
func Scan(q DBTX, b *builder.Builder, dest ...interface{}) error {
	if len(dest) == 1 {
		return doScanSingle(1, q, b, dest[0])
	} else {
		return doScan(1, q, b, dest...)
	}
}
