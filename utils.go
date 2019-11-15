package sorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/n1xx1/builder"
	"log"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func CloneBuilder(b *builder.Builder) *builder.Builder {
	b1 := new(builder.Builder)
	*b1 = *b
	return b1
}

func PagedQuery(q DBTX, current int, rpp int, counter *builder.Builder, selector *builder.Builder, dest interface{}) (int, error) {
	// TODO: maybe perPage sanitization should be moved somewhere else
	if rpp != 10 && rpp != 20 && rpp != 50 {
		rpp = 10
	}

	total, err := doCountTx(1, q, counter)
	if err != nil {
		return 0, err
	}

	selector = selector.Limit(rpp, current*rpp)

	err = doFindTx(1, q, selector, dest)
	if err != nil {
		return 0, err
	}
	return total, nil
}

var dbl = log.New(os.Stderr, "\r\n", 0)

func timedQuery(q DBTX, sql1 string, args []interface{}, calldepth int) (*sql.Rows, error) {
	start := time.Now()
	rows, err := q.Query(sql1, args...)
	if q.debugMode() {
		dbl.Println(logFormatter(sql1, args, fileLocation(calldepth), time.Since(start))...)
	}
	return rows, err
}

func timedExec(q DBTX, sql1 string, args []interface{}, calldepth int) (sql.Result, error) {
	start := time.Now()
	res, err := q.Exec(sql1, args...)
	if q.debugMode() {
		dbl.Println(logFormatter(sql1, args, fileLocation(calldepth), time.Since(start))...)
	}
	return res, err
}

func fileLocation(calldepth int) string {
	_, file, line, ok := runtime.Caller(calldepth + 3)
	if !ok {
		file = "???"
		line = 0
	}
	return fmt.Sprintf("%v:%v", file, line)
}

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

var sqlRegexp = regexp.MustCompile(`(?:\?|@p(\d+))`)

func logFormatter(sql string, args []interface{}, file string, duration time.Duration) []interface{} {
	currentTime := "\n\033[33m[" + time.Now().Format("2006-01-02 15:04:05") + "]\033[0m"
	source := fmt.Sprintf("\033[35m(%v)\033[0m", file)

	messages := []interface{}{source, currentTime}

	if duration > 0 {
		messages = append(messages, fmt.Sprintf(" \033[36;1m[%.2fms]\033[0m ", float64(duration.Nanoseconds()/1e4)/100.0))
	}

	formattedValues := make([]string, len(args))
	for i, value := range args {
		indirectValue := reflect.Indirect(reflect.ValueOf(value))
		if indirectValue.IsValid() {
			value = indirectValue.Interface()
			if t, ok := value.(time.Time); ok {
				formattedValues[i] = fmt.Sprintf("%#v", t.Format("2006-01-02 15:04:05"))
			} else if b, ok := value.([]byte); ok {
				if str := string(b); isPrintable(str) {
					formattedValues[i] = fmt.Sprintf("'%v'", str)
				} else {
					formattedValues[i] = "'<binary>'"
				}
			} else if r, ok := value.(driver.Valuer); ok {
				if value, err := r.Value(); err == nil && value != nil {
					formattedValues[i] = fmt.Sprintf("%v", value)
				} else {
					formattedValues[i] = "NULL"
				}
			} else if s, ok := value.(string); ok {
				formattedValues[i] = fmt.Sprintf("'%v'", strings.Replace(s, "'", "\\'", -1))
			} else {
				formattedValues[i] = fmt.Sprintf("%v", value)
			}
		} else {
			formattedValues[i] = "NULL"
		}
	}

	index := 0
	formattedSql := ReplaceAllStringSubmatchFunc(sqlRegexp, sql, func(groups []string) string {
		if groups[1] != "" {
			pos, _ := strconv.ParseInt(groups[1], 10, 32)
			index = int(pos) - 1
		}
		ret := formattedValues[index]
		index++
		return ret
	})

	messages = append(messages, formattedSql)
	return messages
}

func ReplaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func(groups []string) string) string {
	result := ""
	lastIndex := 0

	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		var groups []string
		for i := 0; i < len(v); i += 2 {
			if v[i] >= 0 && v[i+1] >= 0 {
				groups = append(groups, str[v[i]:v[i+1]])
			} else {
				groups = append(groups, "")
			}
		}
		result += str[lastIndex:v[0]] + repl(groups)
		lastIndex = v[1]
	}

	return result + str[lastIndex:]
}
