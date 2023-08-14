// Package describe generates descriptive plots of ClickHouse tables and query results.
// There are two types of images generated: histograms and quantile plots.
// Quantile plots are generated for fields of type float.  Histograms are generated for
// fields of type string, date and int.  If you want a quantile plot of an int field, cast it as float.
//
// Values deemed "missing" in a field may be omitted from a graph.
//
// In addition, there is a func to create a simple markdown file of the images created.
//
// The command in the describe subdirectory.
package describe

import (
	"fmt"
	"os"
	"strings"

	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"github.com/dustin/go-humanize"
	"github.com/invertedv/chutils"
	"github.com/invertedv/utilities"
)

// TaskType is what we're asked to do:
//   - taskQuery: describe results of query
//   - taskTable: describe all the fields in a table
type TaskType int

const (
	TaskNone TaskType = 0 + iota
	TaskQuery
	TaskTable
	TaskXY
)

const (
	histogram = "histogram"
	quantile  = "quantile"
	xy        = "xy"
	SkipLevel = 1000 // a histogram isn't made if there are more than this many levels
)

// The RunDef struct holds the elements required to direct describe's activities.
type RunDef struct {
	Task TaskType // the kind of task to run

	Show         bool                    // if true, send the plots to the browser
	ImageTypesCh []utilities.PlotlyImage // type(s) of image files to create

	// one of these two must be specified
	Qry      string // query to pull data
	Table    string // table to pull data
	XY       string
	LineType string
	Color    string

	Title    string
	SubTitle string

	OutDir   string // directory for image files
	FileName string

	ImageTypes string // types of images to create

	MissStr, MissDt, MissInt, MissFlt any // values which indicate a field value is missing. Ignored if nil.

	Markdown string // if not nil, the name of a markdown file to create with the images in OutDir.

	Fds *chutils.TableDef // field defs of query results (not required if describing a table).
}

// FieldPlot builds the plot for a single field.
//   - qry. Query to pull the data.
//   - field. Field to keep from query.
//   - plotType.  ("histogram" or "quantile")
//   - outDir. Directory for output.
//   - title. Title for plot.
//   - imageTypes. Type(s) of images to produce.
//   - show. If true, push plot to browser.
//   - conn. Connector to ClickHouse.
func FieldPlot(runDetail *RunDef, xField, yField, where, plotType, title string, conn *chutils.Connect) error {
	var fig *grob.Fig

	pd := &utilities.PlotDef{
		Show:       runDetail.Show,
		Title:      "",
		YTitle:     "",
		STitle:     runDetail.SubTitle,
		Legend:     false,
		Height:     800,
		Width:      1000,
		FileName:   runDetail.FileName,
		OutDir:     runDetail.OutDir,
		ImageTypes: runDetail.ImageTypesCh,
	}

	// add where to query, subtitle
	if where != "" && runDetail.SubTitle == "" {
		pd.STitle = fmt.Sprintf("%s WHERE %s", runDetail.Qry, where)
		// note: single quotes screw up js
		pd.STitle = strings.ReplaceAll(pd.STitle, "'", "`")
	}

	pdTitle := xField
	if title != "" {
		pdTitle = title
	}

	switch plotType {
	case histogram:
		var (
			data *utilities.HistData
			err  error
		)

		if data, err = utilities.NewHistData(runDetail.Qry, xField, where, conn); err != nil {
			return err
		}

		if len(data.Levels) > SkipLevel {
			fmt.Printf("skipped %s: > %d levels\n", xField, SkipLevel)
			return nil
		}

		if title == "" {
			title = fmt.Sprintf("Histogram of %s<br>n: %s", pdTitle, humanize.Comma(data.Total))
		}

		pd.XTitle, pd.YTitle, pd.Title = "Level", "Proportion", title
		fig = data.Fig
	case quantile:
		var (
			data *utilities.QuantileData
			err  error
		)

		if data, err = utilities.NewQuantileData(runDetail.Qry, xField, where, conn); err != nil {
			return err
		}

		if title == "" {
			title = fmt.Sprintf("Quantile of %s<br>n: %s", pdTitle, humanize.Comma(data.Total))
		}

		pd.XTitle, pd.YTitle, pd.Title = "u", xField, title
		fig = data.Fig
	case xy:
		var (
			data *utilities.XYData
			err  error
		)
		flds := strings.Split(runDetail.XY, ",")
		xField = flds[0]

		if data, err = utilities.NewXYData(runDetail.Qry, where, runDetail.XY, runDetail.Color, runDetail.LineType, conn); err != nil {
			return err
		}
		if title == "" {
			title = fmt.Sprintf("XY plot of %s vs %s", xField, strings.Join(flds[1:], ", "))
		}

		pd.XTitle, pd.YTitle, pd.Title, pd.Legend = xField, yField, title, len(flds) > 2
		//fig = data.Fig
		fig = data.Fig
	default:
		return fmt.Errorf("unsupported plotType: %s, must be histogram or quantile", plotType)
	}

	if e := utilities.Plotter(fig, nil, pd); e != nil {
		return e
	}

	return nil
}

// getWhere builds the where statement to eliminate missing values
func getWhere(missInt, missFlt, missStr, missDt any, field, fType string) string {
	where := ""
	var missing any
	switch {
	case strings.Contains(fType, "Int"):
		missing = missInt
	case strings.Contains(fType, "Float"):
		missing = missFlt
	case strings.Contains(fType, "String"):
		missing = missStr
	case strings.Contains(fType, "Date"):
		missing = missDt
	}

	if missing != nil {
		miss := utilities.ToClickHouse(missing)
		oper := "!="

		if strings.Contains(fType, "Float") {
			oper = ">"
		}

		where = fmt.Sprintf("%s %s %s", field, oper, miss)
	}

	return where
}

// Table generates plots for all the fields in the table.
func Table(runDetail *RunDef, conn *chutils.Connect) error {
	// get data types
	fTypes, err := chutils.GetSystemFields(conn, "type", runDetail.Table)
	if err != nil {
		return err
	}

	for field, fType := range fTypes {
		plotType := histogram
		if strings.Contains(fType, "Float") {
			plotType = quantile
		}
		fmt.Println(field)

		fld := field

		// if the field is a nested array, we create an output name that has the form: <array>_<field>
		if strings.Contains(fType, "Array") {
			if array, fieldName, nested := strings.Cut(field, "."); nested {
				// rename from array.field to array_field
				fld = fmt.Sprintf("%s_%s", array, fieldName)
			}
		}

		where := getWhere(runDetail.MissInt, runDetail.MissFlt, runDetail.MissStr, runDetail.MissDt, fld, fType)

		runDetail.Qry = fmt.Sprintf("SELECT %s FROM %s", field, runDetail.Table)

		// If the field is an array, we need to do an arrayJoin
		if strings.Contains(fType, "Array") {
			runDetail.Qry = fmt.Sprintf("SELECT arrayJoin(%s) AS %s FROM %s", field, fld, runDetail.Table)
		}

		var title string
		// add the comment to the title
		comment, _ := chutils.GetSystemField(runDetail.Table, "comment", field, conn)

		switch {
		case runDetail.Title != "":
			title = runDetail.Title
		case comment != "":
			title = fmt.Sprintf("%s: %s", title, comment)
		default:
			title = field
		}

		runDetail.FileName = fld
		if e := FieldPlot(runDetail, fld, "", where, plotType, title, conn); e != nil {
			return e
		}
	}

	return nil
}

// Multiple creates the graphs for a query (as opposed to a table)
func Multiple(runDetail *RunDef, conn *chutils.Connect) error {
	fds := runDetail.Fds.FieldDefs
	for ind := 0; ind < len(fds); ind++ {
		fd := fds[ind]
		plotType := histogram
		if fd.ChSpec.Base == chutils.ChFloat {
			plotType = quantile
		}

		where := getWhere(runDetail.MissInt, runDetail.MissFlt, runDetail.MissStr, runDetail.MissDt, fd.Name, fd.ChSpec.Base.String())

		runDetail.FileName = fd.Name
		if e := FieldPlot(runDetail, fd.Name, "", where, plotType, runDetail.Title, conn); e != nil {
			return e
		}
	}

	return nil
}

// XY creates an XY graphs for a query
func XY(runDetail *RunDef, conn *chutils.Connect) error {
	flds := strings.Split(runDetail.XY, ",")
	where := ""

	for _, fld := range flds {
		var (
			xFd *chutils.FieldDef
			e   error
		)
		field := strings.Trim(fld, " ")
		if _, xFd, e = runDetail.Fds.Get(field); e != nil {
			return e
		}

		whr := getWhere(runDetail.MissInt, runDetail.MissFlt, runDetail.MissStr, runDetail.MissDt, field, xFd.ChSpec.Base.String())

		if where == "" {
			where = whr
		} else {
			where = fmt.Sprintf("%s AND %s", where, whr)
		}
	}

	if runDetail.FileName == "" {
		runDetail.FileName = fmt.Sprintf("%sVs%s", strings.Join(flds, "_"))
	}

	if e := FieldPlot(runDetail, "", "", where, "xy", runDetail.Title, conn); e != nil {
		return e
	}

	return nil
}

// Drive runs the appropriate task
func Drive(runDetail *RunDef, conn *chutils.Connect) error {
	switch runDetail.Task {
	case TaskTable:
		return Table(runDetail, conn)
	case TaskQuery:
		return Multiple(runDetail, conn)
	case TaskXY:
		return XY(runDetail, conn)
	}

	return nil
}

// Markdown creates a simple markdown file of the images in OutDir
func Markdown(runDetail *RunDef) error {
	if runDetail.Markdown == "" {
		return nil
	}

	if runDetail.Task != TaskNone {
		return fmt.Errorf("cannot create markdown in same run as image creation")
	}

	var (
		mdFile *os.File
		err    error
	)

	if mdFile, err = os.Create(runDetail.Markdown); err != nil {
		return err
	}
	defer func() { _ = mdFile.Close() }()

	var outDir []os.DirEntry
	var info os.FileInfo

	if info, err = os.Stat(runDetail.OutDir); err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", runDetail.OutDir)
	}

	if outDir, err = os.ReadDir(runDetail.OutDir); err != nil {
		return err
	}

	for _, fig := range outDir {
		file := "png/"
		if runDetail.OutDir != "" {
			file = fmt.Sprintf("%s%s", utilities.Slash(runDetail.OutDir), fig.Name())
		}
		label, ext, _ := strings.Cut(fig.Name(), ".")

		if !utilities.Has(ext, ",", "png,jpeg,html,pdf,webp,svg,eps,emf") {
			continue
		}

		line := "### "
		// if not html, insert image rather than a link
		if !strings.Contains(fig.Name(), "html") {
			line = "### !"
		}

		line = fmt.Sprintf("%s[%s](%s)\n", line, label, file)
		if _, e := mdFile.WriteString(line); e != nil {
			return e
		}
	}

	return nil
}
