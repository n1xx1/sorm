package sorm

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// simple function that parse a macro (text starts from the first character inside the parentesis), and
// returns its args and parsed length
func parseMacroArguments(texts string) (int, []string) {
	text := []byte(texts)

	argOff := 0
	i := 0
	opened := 0
	var args []string
	for i < len(text) {
		r, rw := utf8.DecodeRune(text[i:])

		switch r {
		case '(':
			opened++
		case ')':
			if opened == 0 {
				args = append(args, strings.TrimSpace(string(text[argOff:i])))
				return i + rw, args
			} else {
				opened -= 1
			}
		case ',':
			if opened == 0 {
				args = append(args, strings.TrimSpace(string(text[argOff:i])))
				argOff = i + rw
			}
		}
		i += rw
	}
	return 0, nil
}

func macroFuncIf(args []string, driver Driver) (string, error) {
	if len(args) != 3 {
		return "", fmt.Errorf("wrong argument count for IF! (expected 3, got %d instead)", len(args))
	}
	return fmt.Sprintf("CASE WHEN %s THEN %s ELSE %s END", args[0], args[1], args[2]), nil
}
func macroFuncGt0(args []string, driver Driver) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("wrong argument count for GT0! (expected 2, got %d instead)", len(args))
	}
	return fmt.Sprintf("CASE WHEN %s > 0 THEN %s ELSE %s END", args[0], args[0], args[1]), nil
}
func macroFuncMin(args []string, driver Driver) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("wrong argument count for MIN! (expected 2 or more, got %d instead)", len(args))
	}
	if driver == DriverMssql {
		values := "(" + strings.Join(args, "),(") + ")"
		return fmt.Sprintf("(SELECT MIN(i) FROM (VALUES %s) AS T(i))", values), nil
	}
	values := strings.Join(args, ",")
	return fmt.Sprintf("LEAST(%s)", values), nil
}
func macroFuncMax(args []string, driver Driver) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("wrong argument count for MAX! (expected 2 or more, got %d instead)", len(args))
	}
	if driver == DriverMssql {
		values := "(" + strings.Join(args, "),(") + ")"
		return fmt.Sprintf("(SELECT MAX(i) FROM (VALUES %s) AS T(i))", values), nil
	}
	values := "SELECT " + args[0] + " AS i UNION SELECT " + strings.Join(args[1:], " UNION SELECT ")
	return fmt.Sprintf("(SELECT MAX(v.i) FROM (%s) v)", values), nil
}
func macroFuncAddMonths(args []string, driver Driver) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("wrong argument count for ADDMONTH! (expected 2, got %d instead)", len(args))
	}
	if driver == DriverMssql {
		return fmt.Sprintf("DATEADD(month, %s, %s)", args[1], args[0]), nil
	} else {
		return fmt.Sprintf("DATE_ADD(%s, INTERVAL %s MONTH)", args[0], args[1]), nil
	}
}

type MacroFunc func(args []string, driver Driver) (string, error)

var macroFuncs = map[string]MacroFunc{
	"IF":       macroFuncIf,
	"GT0":      macroFuncGt0,
	"MIN":      macroFuncMin,
	"MAX":      macroFuncMax,
	"ADDMONTH": macroFuncAddMonths,
}

var regexMacro = regexp.MustCompile(`(?:$|\W)([a-zA-Z]\w)!\(`)

func init() {
	var exp strings.Builder
	for macro := range macroFuncs {
		if exp.Len() > 0 {
			exp.WriteString(`|`)
		}
		exp.WriteString(regexp.QuoteMeta(macro))
	}
	regexMacro = regexp.MustCompile(`(?:$|\W)(` + exp.String() + `)!\(`)
}

/// PerformQueryMacro transforms some macro in the input
///   IF!(a, b, c)  --> CASE WHEN a THEN b ELSE c END,
///   GT0!(a, b)    --> IF!(a > 0, a, b)
func PerformQueryMacro(input string, driver Driver) (string, error) {
	for {
		indexes := regexMacro.FindStringSubmatchIndex(input)
		if indexes != nil {
			begin := indexes[2]
			end := indexes[1]

			l, args := parseMacroArguments(input[end:])

			var repl string
			var err error
			macro := input[indexes[2]:indexes[3]]

			if fn, ok := macroFuncs[macro]; ok {
				repl, err = fn(args, driver)
				if err != nil {
					return "", err
				}
			} else {
				return "", fmt.Errorf("unknown macro %s", macro)
			}

			input = input[:begin] + repl + input[end+l:]
			continue
		}
		break
	}
	return input, nil
}
