package sorm

import (
	"fmt"
	"regexp"
	"strconv"
)

func SqlEscape(mssql bool, table string) string {
	if mssql {
		return "[" + table + "]"
	}
	return "`" + table + "`"
}

var regexField = regexp.MustCompile(`\[(!?)([a-zA-Z_][a-zA-Z0-9_]*)(?:\.([a-zA-Z_][a-zA-Z0-9_]*))?]`)
var regexParam = regexp.MustCompile(`\?`)
var regexParamMs = regexp.MustCompile(`@p(\d+)`)

func ConvertQuery(isMssql bool, query string, args []interface{}) (string, []interface{}) {
	if isMssql {
		return query, args
	}
	realArgs := make([]interface{}, 0, len(args))
	query = ReplaceAllStringSubmatchFunc(regexParamMs, query, func(groups []string) string {
		pos, _ := strconv.ParseInt(groups[1], 10, 32)
		realArgs = append(realArgs, args[pos-1])
		return "?"
	})
	return query, realArgs
}

func FormatQuery(isMssql bool, query string) string {
	i := 0
	query = ReplaceAllStringSubmatchFunc(regexParam, query, func(groups []string) string {
		i++
		return fmt.Sprintf("@p%d", i)
	})

	query, err := PerformQueryMacro(query, isMssql)
	if err != nil {
		panic(err)
	}
	query = ReplaceAllStringSubmatchFunc(regexField, query, func(groups []string) string {
		if m, ok := modelNameCache[groups[2]]; ok {
			if groups[3] != "" {
				if groups[1] == "!" {
					return SqlEscape(isMssql, m.TableName) + "." + fieldName(isMssql, m, groups[3])
				}
				return fieldName(isMssql, m, groups[3])
			} else {
				return SqlEscape(isMssql, m.TableName)
			}
		}
		return groups[0]
	})

	return query
}

func fieldName(mssql bool, model *ModelInfo, field string) string {
	if f, ok := model.fieldNameMap[field]; ok {
		return SqlEscape(mssql, f.DbName)
	}
	return field
}
