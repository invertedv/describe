package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/invertedv/chutils"
	s "github.com/invertedv/chutils/sql"
	"github.com/invertedv/describe"
	"github.com/invertedv/utilities"
)

const (
	null = "NA"
)

func main() {
	// ClickHouse options
	const (
		maxMemoryDef  = 100000000000
		maxGroupByDef = 4000000000
	)

	var (
		conn *chutils.Connect
		err  error
	)

	host := flag.String("host", "127.0.0.1", "string") // ClickHouse db
	user := flag.String("user", null, "string")        // ClickHouse username
	pw := flag.String("pw", null, "string")            // password for user

	runDetail := &describe.RunDef{}

	runDetail.Qry = flag.String("q", null, "string")
	runDetail.Table = flag.String("t", null, "string")

	runDetail.PDF = flag.Bool("pdf", false, "bool")
	runDetail.OutDir = flag.String("d", null, "string")

	runDetail.Show = flag.Bool("show", false, "bool")
	runDetail.ImageTypes = flag.String("i", null, "string")

	// values to recognize a field is missing
	mI := flag.String("mI", "-1", "int64")
	mF := flag.String("mF", "-1", "float64")
	mS := flag.String("mS", "!", "string")
	mD := flag.String("mD", "19700101", "string")
	noMiss := flag.Bool("miss", false, "bool")

	browser := flag.String("b", "xdg-open", "string")

	// ClickHouse options
	maxMemory := flag.Int64("memory", maxMemoryDef, "int64")
	maxGroupBy := flag.Int64("groupby", maxGroupByDef, "int64")

	flag.Parse()

	utilities.Browser = *browser

	if runDetail.MissInt, runDetail.MissFlt, runDetail.MissStr, runDetail.MissDt, err = setMissing(mI, mF, mS, mD, *noMiss); err != nil {
		panic(err)
	}

	if conn, err = chutils.NewConnect(*host, *user, *pw, clickhouse.Settings{
		"max_memory_usage":                   *maxMemory,
		"max_bytes_before_external_group_by": *maxGroupBy,
	}); err != nil {
		panic(err)
	}
	defer func() { _ = conn.Close() }()

	if e := parseFlags(runDetail, conn); e != nil {
		panic(e)
	}

	if e := describe.Drive(runDetail, conn); e != nil {
		panic(e)
	}
}

// -o: requires -i
func parseFlags(runDetail *describe.RunDef, conn *chutils.Connect) error {
	// determine task
	if *runDetail.Qry == null {
		if *runDetail.Table == null {
			return fmt.Errorf("both -q and -t cannot be omitted")
		}

		runDetail.Task = describe.TaskTable
	}

	if *runDetail.Qry != null {
		if *runDetail.Table != null {
			return fmt.Errorf("cannot have both -q and -t")
		}

		runDetail.Task = describe.TaskQuery

		rdr := s.NewReader(*runDetail.Qry, conn)
		defer func() { _ = rdr.Close() }()

		if e := rdr.Init("", chutils.MergeTree); e != nil {
			return e
		}

		runDetail.Fds = rdr.TableSpec().FieldDefs
	}

	// determine imageTypes (and show if no imageTypes)
	for _, img := range strings.Split(strings.ReplaceAll(*runDetail.ImageTypes, " ", ""), ",") {
		if img == null {
			break
		}

		switch img {
		case "png":
			runDetail.ImageTypesCh = append(runDetail.ImageTypesCh, utilities.PlotlyPNG)
		case "jpeg":
			runDetail.ImageTypesCh = append(runDetail.ImageTypesCh, utilities.PlotlyJPEG)
		case "html":
			runDetail.ImageTypesCh = append(runDetail.ImageTypesCh, utilities.PlotlyHTML)
		case "pdf":
			runDetail.ImageTypesCh = append(runDetail.ImageTypesCh, utilities.PlotlyPDF)
		case "wepb":
			runDetail.ImageTypesCh = append(runDetail.ImageTypesCh, utilities.PlotlyWEBP)
		case "eps":
			runDetail.ImageTypesCh = append(runDetail.ImageTypesCh, utilities.PlotlyEPS)
		case "emf":
			runDetail.ImageTypesCh = append(runDetail.ImageTypesCh, utilities.PlotlyEMF)
		case "svg":
			runDetail.ImageTypesCh = append(runDetail.ImageTypesCh, utilities.PlotlySVG)
		default:
			return fmt.Errorf("unknown image type: %s", img)
		}
	}

	if runDetail.ImageTypesCh != nil {
		if *runDetail.OutDir == null {
			return fmt.Errorf("must have -o and -f if have -i")
		}
	}

	if *runDetail.OutDir != null && runDetail.ImageTypesCh == nil {
		return fmt.Errorf("must have -i if have -o")
	}

	// If there's no image type, then we must show to browswer
	if runDetail.ImageTypesCh == nil {
		*runDetail.Show = true
	}

	return nil
}

func checkNull(str *string) *string {
	if *str == null {
		return nil
	}

	return str
}

func setMissing(mI, mF, mS, mD *string, noMiss bool) (missInt, missFlt, missStr, missDt any, err error) {
	if noMiss {
		return nil, nil, nil, nil, nil
	}

	mI, mF, missStr, mD = checkNull(mI), checkNull(mF), *checkNull(mS), checkNull(mD)

	if mF != nil {
		if missFlt, err = strconv.ParseFloat(*mF, 64); err != nil {
			return nil, nil, nil, nil, err
		}
	}

	if mI != nil {
		if missInt, err = strconv.ParseInt(*mI, 10, 64); err != nil {
			return nil, nil, nil, nil, err
		}
	}

	if mD != nil {
		dt, e := utilities.Any2Date(*mD)
		if e != nil {
			return nil, nil, nil, nil, err
		}
		missDt = dt.Format("20060102")
	}

	return missInt, missFlt, missStr, missDt, nil
}
