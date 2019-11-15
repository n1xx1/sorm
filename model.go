package db

import (
	"fmt"
	"reflect"
	"strings"
)

func tagContain(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

type TableName interface {
	TableName() string
}

type ModelInfo struct {
	TableName string
	ModelName string
	Type      reflect.Type

	PrimaryFields []*FieldInfo
	Fields        []*FieldInfo
	ForeignFields []*ForeignInfo

	fieldNameMap   map[string]*FieldInfo
	fieldDbNameMap map[string]*FieldInfo
}

func (m *ModelInfo) FieldByName(name string) *FieldInfo {
	return m.fieldNameMap[name]
}
func (m *ModelInfo) FieldByDbName(dbName string) *FieldInfo {
	return m.fieldDbNameMap[dbName]
}

// FieldsWithTag returns the list of fields that have the specified tag.
// if one of the tags is "primary" then all the primary fields are also returned
// if one of the tags is "autoincrement" then the autoincrement field is returned
func (m *ModelInfo) FieldsWithTag(tag ...string) []*FieldInfo {
	var ret []*FieldInfo

	for _, f := range m.Fields {
		for _, t := range tag {
			if t == "primary" && f.IsPrimary {
				ret = append(ret, f)
				break
			}
			if t == "autoincrement" && f.IsAutoIncrement {
				ret = append(ret, f)
				break
			}
			if tagContain(f.Tags, t) {
				ret = append(ret, f)
				break
			}
		}
	}
	return ret
}

type ForeignKind int

const (
	OneToOne ForeignKind = iota
	OneToMany
)

type FieldInfo struct {
	StructField     reflect.StructField
	StructFieldPath []int

	Name        string
	Index       int
	Tags        []string
	ForeignInfo *ForeignInfo

	DbName          string
	IsPrimary       bool
	IsAutoIncrement bool
	IsForeign       bool
}

type ForeignInfo struct {
	Name       string
	Field      *FieldInfo
	Model      string
	JoinTable  string
	joinColumn string
	Kind       ForeignKind
}

func (f *ForeignInfo) JoinColumn() string {
	if f.joinColumn == "" {
		m := ModelByName(f.Model)
		if len(m.PrimaryFields) != 1 {
			panic(fmt.Sprintf("model %s is expected to have only one primary field", f.Model))
		}
		return m.PrimaryFields[0].Name
	}
	return f.joinColumn
}

func (f *FieldInfo) HasTag(tag string) bool {
	return tagContain(f.Tags, tag)
}

var modelCache = map[reflect.Type]*ModelInfo{}
var modelNameCache = map[string]*ModelInfo{}

func GetAllModels() []string {
	var ret []string
	for k := range modelNameCache {
		ret = append(ret, k)
	}
	return ret
}

func ModelByName(model string) *ModelInfo {
	return modelNameCache[model]
}
func ModelByType(typ reflect.Type) *ModelInfo {
	return modelCache[typ]
}

func AddModel(tbl TableName) {
	typ := reflect.TypeOf(tbl)
	computeFieldCache(tbl.TableName(), typ.Elem())
}

func computeFieldCacheRec(model *ModelInfo, typ reflect.Type, path []int) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)

		var fieldPath []int
		fieldPath = append(fieldPath, path...)
		fieldPath = append(fieldPath, i)

		isStruct := f.Type.Kind() == reflect.Struct ||
			(f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct)

		dbfk := f.Tag.Get("dbfk")
		tag := f.Tag.Get("db")
		if tag == "" {
			if isStruct {
				computeFieldCacheRec(model, f.Type, fieldPath)
			}
			continue
		}
		if tag == "-" {
			continue
		}

		tagParts := strings.Split(tag, ",")
		name := tagParts[0]

		if name == "" {
			panic("name can't be empty")
		}

		field := &FieldInfo{
			StructField:     f,
			DbName:          name,
			Name:            f.Name,
			Index:           len(model.Fields),
			StructFieldPath: fieldPath,
		}

		dbtags := f.Tag.Get("dbtags")
		if dbtags != "" {
			field.Tags = strings.Split(dbtags, ",")
		}

		for _, tag := range tagParts[1:] {
			switch {
			case tag == "autoincrement":
				field.IsAutoIncrement = true
			case tag == "primary":
				field.IsPrimary = true
				model.PrimaryFields = append(model.PrimaryFields, field)
			default:
				panic(fmt.Sprintf("invalid attribute specified in tag 'db' for field %s in model %s", name, model.ModelName))
			}
		}

		if dbfk != "" {
			dbfkParts := strings.Split(dbfk, ",")
			foreignModel := dbfkParts[0]

			foreign := &ForeignInfo{
				Field: field,
				Model: foreignModel,
				Kind:  OneToOne,
			}
			for _, tag := range dbfkParts[1:] {
				if tag1 := strings.TrimPrefix(tag, "table:"); len(tag1) != len(tag) {
					foreign.Kind = OneToMany
					foreign.JoinTable = tag1
				} else if tag1 := strings.TrimPrefix(tag, "col:"); len(tag1) != len(tag) {
					foreign.joinColumn = tag1
				}
			}

			field.ForeignInfo = foreign
			field.IsForeign = true
			model.ForeignFields = append(model.ForeignFields, foreign)
		}

		model.Fields = append(model.Fields, field)
		model.fieldNameMap[f.Name] = field
		model.fieldDbNameMap[name] = field
	}
}

func computeFieldCache(tableName string, typ reflect.Type) {
	model := CreateModelInfo(tableName, typ)
	modelCache[typ] = model
	modelNameCache[typ.Name()] = model
}

func CreateModelInfo(tableName string, typ reflect.Type) *ModelInfo {
	model := &ModelInfo{
		TableName: tableName,
		ModelName: typ.Name(),
		Type:      typ,

		fieldNameMap:   map[string]*FieldInfo{},
		fieldDbNameMap: map[string]*FieldInfo{},
	}
	computeFieldCacheRec(model, typ, nil)
	return model
}
