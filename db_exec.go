package sorm

import (
	"fmt"
	"github.com/n1xx1/builder"
)

func doExecTx(calldepth int, q DBTX, b *builder.Builder) error {
	sql1, args, err := b.ToSQL()
	if err != nil {
		return fmt.Errorf("exec error: %w", err)
	}

	sql1 = FormatQuery(q.Driver(), sql1)
	sql1, args = ConvertQuery(q.Driver(), sql1, args)

	_, err = timedExec(q, sql1, args, calldepth)
	if err != nil {
		return fmt.Errorf("exec error: %w", err)
	}
	return nil
}

func Exec(q DBTX, b *builder.Builder) error {
	return doExecTx(1, q, b)
}
