package sorm

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/n1xx1/builder"
	"reflect"
	"regexp"
	"strings"
)

type SelectedTable interface {
	get() *selectedTable
}

type selectedTable struct {
	alias       string
	model       *ModelInfo
	modelFields []*FieldInfo
	expr        builder.Column
}

func (s *selectedTable) get() *selectedTable {
	return s
}

var regexModel = regexp.MustCompile(`^(\w+)(?:\s*:\s*(\w+))?$`)

func SelectTable(modelName string, tags []string) SelectedTable {
	model, ok := modelNameCache[modelName]
	if !ok {
		panic(fmt.Errorf("unknown model %v", modelName))
	}
	var fields []*FieldInfo
	if len(tags) == 0 {
		fields = model.Fields
	} else {
		fields = model.FieldsWithTag(tags...)
	}
	return &selectedTable{
		alias:       "",
		model:       model,
		modelFields: fields,
	}
}

func SelectTableAlias(modelName string, alias string, tags []string) SelectedTable {
	model, ok := modelNameCache[modelName]
	if !ok {
		panic(fmt.Errorf("unknown model %v", modelName))
	}
	var fields []*FieldInfo
	if len(tags) == 0 {
		fields = model.Fields
	} else {
		fields = model.FieldsWithTag(tags...)
	}
	return &selectedTable{
		alias:       alias,
		model:       model,
		modelFields: fields,
	}
}

func SelectTableAliasFields(modelName string, alias string, fields []*FieldInfo) SelectedTable {
	model, ok := modelNameCache[modelName]
	if !ok {
		panic(fmt.Errorf("unknown model %v", modelName))
	}
	return &selectedTable{
		alias:       alias,
		model:       model,
		modelFields: fields,
	}
}

func getSelectedTable(i interface{}, off int) *selectedTable {
	if sel, ok := i.(SelectedTable); ok {
		return sel.get()
	}

	if c, ok := i.(builder.Column); ok {
		return &selectedTable{"", nil, nil, c}
	}
	if s, ok := i.(string); ok {
		m := regexModel.FindStringSubmatch(s)
		if m != nil {
			var modelName, alias string
			if m[2] != "" {
				alias = m[1]
				modelName = m[2]
			} else {
				alias = ""
				modelName = m[1]
			}
			model, ok := modelNameCache[modelName]
			if !ok {
				panic(fmt.Errorf("unknown model %v", modelName))
			}
			return &selectedTable{alias, model, model.Fields, nil}
		}

		// unnamed column
		return &selectedTable{"", nil, nil, builder.As(s, fmt.Sprintf("p%d", off))}
	}
	panic("unknown parameters for select")
}

type QueryScanner struct {
	selects       []*selectedTable
	offsets       []int
	scanToIndexes [][]int
	dest          []interface{}
	rows          *sql.Rows
	cols          []*sql.ColumnType
}

/// ScanTo scans every selected thing to a struct using the struct
/// field names
func (q *QueryScanner) ScanTo(dest interface{}) error {
	if len(q.selects) != 0 {
		return fmt.Errorf("ScanTo only allowed with manual selects")
	}

	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if q.scanToIndexes == nil {
		cols, err := q.rows.ColumnTypes()

		if err != nil {
			return err
		}
		q.scanToIndexes = make([][]int, len(cols))
		q.dest = make([]interface{}, len(cols))
		q.cols = cols
		cacheColumns(cols, v.Type(), q.scanToIndexes, q.dest)
	}

	err := q.rows.Scan(q.dest...)
	if err != nil {
		return err
	}

	for i, index := range q.scanToIndexes {
		if index != nil {
			rowVal := *(q.dest[i].(*interface{}))

			elem := v.FieldByIndex(index)
			err := setFieldValue(elem, rowVal, q.cols[i])

			if err != nil {
				return fmt.Errorf("on field %v: %w", v.Type().FieldByIndex(index).Name, err)
			}
		}
	}
	return nil
}

func cacheColumns(columns []*sql.ColumnType, typ reflect.Type, indexes [][]int, dest []interface{}) {
	for i, col := range columns {
		colName := strings.Replace(col.Name(), "_", "", -1)
		f, ok := typ.FieldByNameFunc(func(s string) bool {
			return strings.EqualFold(s, colName)
		})
		if ok {
			indexes[i] = f.Index
		}
		dest[i] = reflect.New(col.ScanType()).Interface()
	}
}

func (q *QueryScanner) Next() bool {
	return q.rows.Next()
}

func gatherFindInformation(index int, destSlice interface{}, table *selectedTable) (reflect.Type, bool, error) {
	v := reflect.ValueOf(destSlice)
	if v.Type().Kind() != reflect.Ptr || v.Type().Elem().Kind() != reflect.Slice {
		return nil, false, fmt.Errorf("dest parameter %d must be a pointer to slice", index)
	}
	v = v.Elem()

	elType := v.Type().Elem()
	destIsPtr := elType.Kind() == reflect.Ptr

	if destIsPtr {
		elType = elType.Elem()
	}

	if table.model != nil {
		if mcached, ok := modelCache[elType]; !ok || table.model != mcached {
			return nil, false, fmt.Errorf("wrong parameter type, expected model %s", table.model.ModelName)
		}
	}

	return elType, destIsPtr, nil
}

func (q *QueryScanner) Find(dest ...interface{}) error {
	scanned := make([]interface{}, len(dest))

	scannedType := make([]reflect.Type, len(dest))
	destIsPtr := make([]bool, len(dest))

	if len(dest) != len(q.selects) {
		return errors.New("unmatched selects and dest length")
	}

	for i, s := range q.selects {
		var err error
		scannedType[i], destIsPtr[i], err = gatherFindInformation(i, dest[i], s)
		if err != nil {
			return err
		}
		if destIsPtr[i] {
			scanned[i] = reflect.New(scannedType[i]).Interface()
		}
	}

	for q.Next() {
		for i := range q.selects {
			if !destIsPtr[i] {
				scanned[i] = reflect.New(scannedType[i]).Interface()
			}
		}
		err := q.Scan(scanned...)
		if err != nil {
			return err
		}
		for i := range q.selects {
			appended := reflect.ValueOf(scanned[i])
			if !destIsPtr[i] {
				appended = appended.Elem()
			}
			v := reflect.ValueOf(dest[i]).Elem()
			v.Set(reflect.Append(v, appended))
		}
	}
	return nil
}

func (q *QueryScanner) Scan(dest ...interface{}) error {
	// manual selects when q.selects is empty
	if len(q.selects) == 0 {
		err := q.rows.Scan(dest...)
		if err != nil {
			return fmt.Errorf("scan error: %w", err)
		}
		return nil
	}

	if len(dest) != len(q.selects) {
		return fmt.Errorf("unmatched selects and dest length")
	}

	err := q.rows.Scan(q.dest...)
	if err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	for i, s := range q.selects {
		if s.expr != nil {
			offset := q.offsets[i]
			rowVal := q.dest[offset]
			rowVal = *(rowVal.(*interface{}))

			elem := reflect.ValueOf(dest[i])
			err = setFieldValue(elem, rowVal, q.cols[offset])

			if err != nil {
				return err
			}
			continue
		}
		err = encodeFromRow(s, q.dest, q.offsets[i], dest[i], q.cols)
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *QueryScanner) First(dest ...interface{}) error {
	if !q.Next() {
		return ErrEmptyResult
	}
	return q.Scan(dest...)
}

func (q *QueryScanner) Close() error {
	return q.rows.Close()
}

func doQuery(calldepth int, q DBTX, b *builder.Builder, selectParams ...interface{}) (*QueryScanner, error) {
	var selects []*selectedTable
	var offsets []int

	if len(selectParams) != 0 {
		selects = make([]*selectedTable, len(selectParams))
		for i, s := range selectParams {
			selects[i] = getSelectedTable(s, i)
		}

		offsets = make([]int, len(selects))

		var builderSelects []interface{}
		for i, s := range selects {
			offsets[i] = len(builderSelects)

			if s.expr != nil {
				builderSelects = append(builderSelects, s.expr)
				continue
			}
			sel := selectInBuilder(s, offsets[i])
			builderSelects = append(builderSelects, sel...)
		}

		b = b.Select(builderSelects...)
	}

	sql1, args, err := b.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("sql builder error: %w", err)
	}

	sql1 = FormatQuery(q.Driver(), sql1)
	sql1, args = ConvertQuery(q.Driver(), sql1, args)

	rows, err := timedQuery(q, sql1, args, calldepth)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	rowCols, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	dest := make([]interface{}, len(rowCols))
	for i, col := range rowCols {
		dest[i] = reflect.New(col.ScanType()).Interface()
	}

	return &QueryScanner{selects: selects, dest: dest, offsets: offsets, rows: rows, cols: rowCols}, nil
}

/// Query queries the database with the specified query (b) with the models you want
/// the returned QueryScanner can be used to decode to the interfaces of the models you selected
func Query(q DBTX, b *builder.Builder, selectParams ...interface{}) (*QueryScanner, error) {
	return doQuery(1, q, b, selectParams...)
}
