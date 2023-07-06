package main

import (
	"flag"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/invertedv/chutils"
	"github.com/invertedv/describe"
	"github.com/invertedv/utilities"
	"math"
	"time"
)

func main() {
	// ClickHouse options
	const (
		maxMemoryDef  = 100000000000
		maxGroupByDef = 4000000000
	)

	const (
		defMs = "abcdefghijklmnopqrstuvwxyz"
		defMf = 1e10
		defMi = math.MinInt64
		defMd = "19700101"
	)
	var (
		conn *chutils.Connect
		err  error
	)

	host := flag.String("host", "127.0.0.1", "string") // ClickHouse db
	user := flag.String("user", "NA", "string")        // ClickHouse username
	pw := flag.String("pw", "NA", "string")            // password for user

	qry := flag.String("q", "NA", "string")
	table := flag.String("t", "NA", "string")

	pdf := flag.Bool("pdf", false, "bool")
	outDir := flag.String("d", "NA", "string")
	outFile := flag.String("f", "NA", "string")

	imgTypes := flag.String("i", "NA", "string")

	mI := flag.Int64("mI", defMi, "int64")
	mF := flag.Float64("mF", defMf, "float64")
	mS := flag.String("mS", defMs, "string")
	mD := flag.String("mD", defMd, "string")

	// ClickHouse options
	maxMemory := flag.Int64("memory", maxMemoryDef, "int64")
	maxGroupBy := flag.Int64("groupby", maxGroupByDef, "int64")

	flag.Parse()

	if conn, err = chutils.NewConnect(*host, *user, *pw, clickhouse.Settings{
		"max_memory_usage":                   *maxMemory,
		"max_bytes_before_external_group_by": *maxGroupBy,
	}); err != nil {
		panic(err)
	}
	defer func() { _ = conn.Close() }()

	qry := "SELECT * Except(lnID) FROM bk.final"
	fds, e := describe.GetFields(qry, conn)
	if e != nil {
		panic(e)
	}

	missInt, missFlt, missStr, missDt := -1, -1, "!", time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	images := []utilities.PlotlyImage{utilities.PlotlyPNG, utilities.PlotlyHTML}
	if e := describe.Table("bk.final", "/home/will/describe", images, fds, missStr, missInt, missFlt, missDt, conn); e != nil {
		panic(e)
	}

	// 1. -pdf: create a skeleton md with includes and then pandoc to pdf
	// - describe -q "select purpose from bk.loan"   : into browser
	// - describe -q "select * from bk.loan" -f png,html     <- check # of returns,
	// - describe -t bk.loan -d /home/will/describe -pdf

	// -pdf can't use -html
	// -mI, -mF, -mS, -mD
	// -q  (query)
	// -d  (output dir)
	// -f  (file root name)
}
