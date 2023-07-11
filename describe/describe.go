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
	null = "NA" // default value for string flags
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

	markDown := flag.String("markdown", null, "bool")
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

	if runDetail.MissInt, runDetail.MissFlt, runDetail.MissStr, runDetail.MissDt, runDetail.Markdown, err =
		setMissing(mI, mF, mS, mD, markDown, *noMiss); err != nil {
		panic(err)
	}

	// parseFlags parses the user input to fully populate runDetail and return a ClickHouse connection, if needed.
	if conn, err = parseFlags(runDetail, host, user, pw, maxMemory, maxGroupBy); err != nil {
		panic(err)
	}
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()

	// create graphs.
	if e := describe.Drive(runDetail, conn); e != nil {
		panic(e)
	}

	// create markdown file, if requested.
	if e := describe.Markdown(runDetail); e != nil {
		panic(e)
	}
}

// parseFlags fully populates runDetail from the user input
func parseFlags(runDetail *describe.RunDef, host, user, pw *string, maxMemory, maxGroupBy *int64) (*chutils.Connect, error) {
	// determine task
	if runDetail.Markdown != nil && *runDetail.Qry == null && *runDetail.Table == null {
		// just make markdown file
		runDetail.Task = describe.TaskNone
		runDetail.OutDir = checkNull(runDetail.OutDir)
		return nil, nil
	}

	conn, err := chutils.NewConnect(*host, *user, *pw, clickhouse.Settings{
		"max_memory_usage":                   *maxMemory,
		"max_bytes_before_external_group_by": *maxGroupBy,
	})
	if err != nil {
		return nil, err
	}

	if *runDetail.Qry == null {
		if *runDetail.Table == null {
			return nil, fmt.Errorf("both -q and -t cannot be omitted")
		}

		runDetail.Task = describe.TaskTable
	}

	if *runDetail.Qry != null {
		if *runDetail.Table != null {
			return nil, fmt.Errorf("cannot have both -q and -t")
		}

		runDetail.Task = describe.TaskQuery

		rdr := s.NewReader(*runDetail.Qry, conn)
		defer func() { _ = rdr.Close() }()

		if e := rdr.Init("", chutils.MergeTree); e != nil {
			return nil, e
		}

		runDetail.Fds = rdr.TableSpec().FieldDefs
	}

	// determine imageTypes
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
			return nil, fmt.Errorf("unknown image type: %s", img)
		}
	}

	// if no -d is specified, use current working directory
	if runDetail.ImageTypesCh != nil {
		// place in current working directory if not specified
		if *runDetail.OutDir == null {
			*runDetail.OutDir = "."
		}
	}

	if *runDetail.OutDir != null && runDetail.ImageTypesCh == nil {
		return nil, fmt.Errorf("must have -i if have -d")
	}

	// If there's no image type, then we must show to browser
	if runDetail.ImageTypesCh == nil {
		*runDetail.Show = true
	}

	return conn, nil
}

func checkNull(str *string) *string {
	if *str == null {
		return nil
	}

	return str
}

// setMissing sets up the missing values
func setMissing(mI, mF, mS, mD, markDown *string, noMiss bool) (missInt, missFlt, missStr, missDt any, mark *string, err error) {
	if noMiss {
		return nil, nil, nil, nil, nil, nil
	}

	mI, mF, missStr, mD, mark = checkNull(mI), checkNull(mF), *checkNull(mS), checkNull(mD), checkNull(markDown)

	if mF != nil {
		if missFlt, err = strconv.ParseFloat(*mF, 64); err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}

	if mI != nil {
		if missInt, err = strconv.ParseInt(*mI, 10, 64); err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}

	if mD != nil {
		dt, e := utilities.Any2Date(*mD)
		if e != nil {
			return nil, nil, nil, nil, nil, err
		}
		missDt = dt.Format("20060102")
	}

	return missInt, missFlt, missStr, missDt, mark, nil
}
