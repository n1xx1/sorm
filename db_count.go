package db

import (
	"github.com/n1xx1/builder"
)

func doCountTx(calldepth int, q DBTX, b *builder.Builder) (int, error) {
	qs, err := doQuery(calldepth+1, q, b, "COUNT(*)")
	if err != nil {
		return 0, err
	}
	defer qs.Close()

	var count int
	err = qs.First(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

/// Count queries the database with the specified query (b) with SELECT COUNT(*), and returns the number
func Count(q DBTX, b *builder.Builder) (int, error) {
	return doCountTx(1, q, b)
}
