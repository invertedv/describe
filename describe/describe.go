package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/invertedv/chutils"
	s "github.com/invertedv/chutils/sql"
	"github.com/invertedv/describe"
	"github.com/invertedv/utilities"
	"golang.org/x/term"
)

const (
	null = "NA" // default value for string flags
)

var (
	//go:embed helpString.txt
	helpString string
)

func main() {
	// ClickHouse options
	const (
		maxMemoryDef  = 100000000000
		maxGroupByDef = 4000000000
		threadsDef    = 6
	)

	var (
		conn *chutils.Connect
		err  error
	)

	host := flag.String("host", "127.0.0.1", "string") // ClickHouse db
	user := flag.String("user", null, "string")        // ClickHouse username
	pw := flag.String("pw", null, "string")            // password for user

	runDetail := &describe.RunDef{}

	qry := flag.String("q", null, "string")
	table := flag.String("t", null, "string")

	markDown := flag.String("markdown", null, "bool")
	outDir := flag.String("d", null, "string")
	fileName := flag.String("f", null, "string")

	show := flag.Bool("show", false, "bool")
	imageTypes := flag.String("i", null, "string")
	xy := flag.String("xy", null, "bool")

	// values to recognize a field is missing
	mI := flag.String("mI", "-1", "int64")
	mF := flag.String("mF", "-1", "float64")
	mS := flag.String("mS", "!", "string")
	mD := flag.String("mD", "19700101", "string")
	noMiss := flag.Bool("miss", false, "bool")
	help := flag.Bool("h", false, "bool")
	title := flag.String("title", null, "string")
	subTitle := flag.String("subtitle", null, "string")
	lineType := flag.String("lineType", "m", "string")
	color := flag.String("color", "black", "string")
	height := flag.Int64("height", 800, "int64")
	width := flag.Int64("width", 1000, "int64")
	xlim := flag.String("xlim", null, "string")
	ylim := flag.String("ylim", null, "string")
	box := flag.Bool("box", false, "bool")
	log := flag.Bool("log", false, "bool")
	xlab := flag.String("xlab", null, "string")
	ylab := flag.String("ylab", null, "string")

	browser := flag.String("b", "xdg-open", "string")

	// ClickHouse options
	maxMemory := flag.Int64("memory", maxMemoryDef, "int64")
	maxGroupBy := flag.Int64("groupby", maxGroupByDef, "int64")
	threads := flag.Int64("threads", threadsDef, "int64")

	flag.Parse()

	runDetail.Qry, runDetail.Table, runDetail.OutDir = toEmpty(qry), toEmpty(table), toEmpty(outDir)
	runDetail.ImageTypes, runDetail.XY, runDetail.Title = toEmpty(imageTypes), toEmpty(xy), toEmpty(title)
	runDetail.SubTitle, runDetail.LineType, runDetail.Show = toEmpty(subTitle), toEmpty(lineType), *show
	runDetail.Color, runDetail.FileName = toEmpty(color), toEmpty(fileName)
	runDetail.Height, runDetail.Width, runDetail.Box = float64(*height), float64(*width), *box
	runDetail.Xlim, runDetail.Ylim, runDetail.Log = limits(xlim), limits(ylim), *log
	runDetail.Xlab, runDetail.Ylab = toEmpty(xlab), toEmpty(ylab)

	if *help {
		fmt.Println(helpString)
		os.Exit(0)
	}

	if *user == null {
		*user = getValue("ClickHouse User: ")
	}

	if *pw == null {
		*pw = getPW("Clickhouse Password: ")
	}

	utilities.Browser = *browser

	if err = setMissing(mI, mF, mS, mD, markDown, *noMiss, runDetail); err != nil {
		panic(err)
	}

	// parseFlags parses the user input to fully populate runDetail and return a ClickHouse connection, if needed.
	if conn, err = parseFlags(runDetail, host, user, pw, maxMemory, maxGroupBy, threads); err != nil {
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
func parseFlags(runDetail *describe.RunDef, host, user, pw *string, maxMemory, maxGroupBy, threads *int64) (*chutils.Connect, error) {
	// determine task
	if runDetail.Markdown != "" && runDetail.Qry == "" && runDetail.Table == "" {
		// just make markdown file
		runDetail.Task = describe.TaskNone
		return nil, nil
	}

	conn, err := chutils.NewConnect(*host, *user, *pw, clickhouse.Settings{
		"max_memory_usage":                   *maxMemory,
		"max_bytes_before_external_group_by": *maxGroupBy,
		"max_threads":                        *threads,
	})

	if err != nil {
		return nil, err
	}

	if runDetail.Qry == "" {
		if runDetail.Table == "" {
			return nil, fmt.Errorf("both -q and -t cannot be omitted")
		}

		runDetail.Task = describe.TaskTable
	}

	if runDetail.Qry != "" {
		if runDetail.Table != "" {
			return nil, fmt.Errorf("cannot have both -q and -t")
		}

		runDetail.Task = describe.TaskQuery

		if runDetail.XY != "" {
			runDetail.Task = describe.TaskXY
		}

		rdr := s.NewReader(runDetail.Qry, conn)
		defer func() { _ = rdr.Close() }()

		if e := rdr.Init("", chutils.MergeTree); e != nil {
			return nil, e
		}

		runDetail.Fds = rdr.TableSpec()
	}

	// determine imageTypes
	for _, img := range strings.Split(strings.ReplaceAll(runDetail.ImageTypes, " ", ""), ",") {
		if img == "" {
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
		if runDetail.OutDir == "" {
			runDetail.OutDir = "."
		}
	}

	if runDetail.OutDir != "" && runDetail.ImageTypesCh == nil {
		return nil, fmt.Errorf("must have -i if have -d")
	}

	// If there's no image type, then we must show to browser
	if runDetail.ImageTypesCh == nil {
		runDetail.Show = true
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
func setMissing(mI, mF, mS, mD, markDown *string, noMiss bool, runDetail *describe.RunDef) error {
	var err error
	if noMiss {
		return nil
	}

	mI, mF, runDetail.MissStr, mD, runDetail.Markdown = checkNull(mI), checkNull(mF), toEmpty(mS), checkNull(mD), toEmpty(markDown)

	if mF != nil {
		if runDetail.MissFlt, err = strconv.ParseFloat(*mF, 64); err != nil {
			return err
		}
	}

	if mI != nil {
		if runDetail.MissInt, err = strconv.ParseInt(*mI, 10, 64); err != nil {
			return err
		}
	}

	if mD != nil {
		dt, e := utilities.Any2Date(*mD)
		if e != nil {
			return err
		}
		runDetail.MissDt = dt.Format("20060102")
	}

	return nil
}

// getValue returns a value for the user
func getValue(prompt string) string {
	rdr := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	txt, _ := rdr.ReadString('\n')
	return strings.ReplaceAll(txt, "\n", "")
}

func getPW(prompt string) string {
	fmt.Print(prompt)
	pass, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	return string(pass)
}

func toEmpty(inStr *string) string {
	if inStr == nil || *inStr == null {
		return ""
	}

	return *inStr
}

func limits(asStr *string) []float64 {
	var (
		min, max float64
		err      error
	)
	if *asStr == null {
		return nil
	}

	lims := strings.Split(strings.ReplaceAll(*asStr, " ", ""), ",")
	if len(lims) != 2 {
		panic(fmt.Errorf("need two limits, max and min. Got %s", *asStr))
	}

	if min, err = strconv.ParseFloat(lims[0], 64); err != nil {
		panic(err)
	}

	if max, err = strconv.ParseFloat(lims[1], 64); err != nil {
		panic(err)
	}

	if min >= max {
		panic(fmt.Errorf("max is not greater than min: %s", *asStr))
	}

	return []float64{min, max}
}
