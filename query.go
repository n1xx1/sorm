package sorm

import (
	"fmt"
	"regexp"
	"strconv"
)

func SqlEscape(driver Driver, table string) string {
	if driver == DriverMssql {
		return "[" + table + "]"
	}
	return "`" + table + "`"
}

var regexField = regexp.MustCompile(`\[(!?)([a-zA-Z_][a-zA-Z0-9_]*)(?:\.([a-zA-Z_][a-zA-Z0-9_]*))?]`)
var regexParam = regexp.MustCompile(`\?`)
var regexParamMs = regexp.MustCompile(`@p(\d+)`)

func ConvertQuery(driver Driver, query string, args []interface{}) (string, []interface{}) {
	if driver == DriverMssql {
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

func FormatQuery(driver Driver, query string) string {
	i := 0
	query = ReplaceAllStringSubmatchFunc(regexParam, query, func(groups []string) string {
		i++
		return fmt.Sprintf("@p%d", i)
	})

	query, err := PerformQueryMacro(query, driver)
	if err != nil {
		panic(err)
	}
	query = ReplaceAllStringSubmatchFunc(regexField, query, func(groups []string) string {
		if m, ok := modelNameCache[groups[2]]; ok {
			if groups[3] != "" {
				if groups[1] == "!" {
					return SqlEscape(driver, m.TableName) + "." + fieldName(driver, m, groups[3])
				}
				return fieldName(driver, m, groups[3])
			} else {
				return SqlEscape(driver, m.TableName)
			}
		}
		return groups[0]
	})

	return query
}

func fieldName(driver Driver, model *ModelInfo, field string) string {
	if f, ok := model.fieldNameMap[field]; ok {
		return SqlEscape(driver, f.DbName)
	}
	return field
}
