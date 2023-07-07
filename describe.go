package describe

import (
	"fmt"
	"github.com/MetalBlueberry/go-plotly/offline"
	"strings"

	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"github.com/dustin/go-humanize"
	"github.com/invertedv/chutils"
	s "github.com/invertedv/chutils/sql"
	"github.com/invertedv/utilities"
)

// task is what we're asked to do:
//   - taskSingle: describe one field
//   - taskMultiple: describe more than one field but as a query, not a table
//   - taskTable: describe all the fields in a table
type TaskType int

const (
	TaskQuery TaskType = 0 + iota
	TaskTable
)

type RunDef struct {
	Task TaskType

	Show         *bool
	ImageTypesCh []utilities.PlotlyImage

	Qry   *string
	Table *string

	FileRoot *string
	OutDir   *string

	ImageTypes *string

	MissStr, MissDt, MissInt, MissFlt any

	PDF *bool

	Fds map[int]*chutils.FieldDef
}

func GetFields(qry string, conn *chutils.Connect) (fields []*chutils.FieldDef, err error) {
	rdr := s.NewReader(qry, conn)
	if e := rdr.Init("", chutils.MergeTree); e != nil {
		return nil, e
	}

	for _, fd := range rdr.TableSpec().FieldDefs {
		fields = append(fields, fd)
	}

	return fields, nil
}

func FieldPlot(qry, field, where, plotType, outDir, outFile, title string, imageTypes []utilities.PlotlyImage,
	show bool, conn *chutils.Connect) error {
	var fig *grob.Fig

	pd := &utilities.PlotDef{
		Show:       show,
		Title:      "",
		YTitle:     "",
		STitle:     "",
		Legend:     false,
		Height:     800,
		Width:      1000,
		FileName:   outFile,
		OutDir:     outDir,
		ImageTypes: imageTypes,
	}

	if where != "" {
		pd.STitle = fmt.Sprintf("%s WHERE %s", qry, where)
		// note: single quotes screw up js
		pd.STitle = strings.ReplaceAll(pd.STitle, "'", "`")
	}

	pdTitle := field
	if title != "" {
		pdTitle = title
	}

	switch plotType {
	case "histogram":
		var (
			data *utilities.HistData
			err  error
		)

		if data, err = utilities.NewHistData(qry, field, where, conn); err != nil {
			return err
		}

		if len(data.Levels) > 1000 {
			fmt.Printf("skipped %s: > 1000 levels\n", field)
			return nil
		}

		pd.XTitle, pd.Title, pd.YTitle = "Level", fmt.Sprintf("Histogram of %s<br>n: %s", pdTitle, humanize.Comma(data.Total)), "Proportion"
		fig = data.Fig
	case "quantile":
		var (
			data *utilities.QuantileData
			err  error
		)

		if data, err = utilities.NewQuantileData(qry, field, where, conn); err != nil {
			return err
		}

		pd.XTitle, pd.YTitle, pd.Title = "u", field, fmt.Sprintf("Quantile of %s<br>n: %s", pdTitle, humanize.Comma(data.Total))
		fig = data.Fig
	default:
		return fmt.Errorf("unsupported plotType: %s, must be histogram or quantile", plotType)
	}

	offline.ToHtml(fig, "/home/will/tmp/test.html")
	if e := utilities.Plotter(fig, nil, pd); e != nil {
		return e
	}

	return nil
}

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

func Table(runDetail *RunDef, conn *chutils.Connect) error {
	// get data types
	fTypes, err := chutils.GetSystemFields(conn, "type", *runDetail.Table)
	if err != nil {
		return err
	}

	for field, fType := range fTypes {
		plotType := "histogram"
		if strings.Contains(fType, "Float") {
			plotType = "quantile"
		}
		fmt.Println(field)

		fld := field
		if strings.Contains(fType, "Array") {
			array, field, _ := strings.Cut(field, ".")
			// rename from array.field to array_field
			fld = fmt.Sprintf("%s_%s", array, field)
		}

		where := getWhere(runDetail.MissInt, runDetail.MissFlt, runDetail.MissStr, runDetail.MissDt, fld, fType)

		qry := fmt.Sprintf("SELECT %s FROM %s", field, *runDetail.Table)
		if strings.Contains(fType, "Array") {
			qry = fmt.Sprintf("SELECT arrayJoin(%s) AS %s FROM %s", field, fld, *runDetail.Table)
		}

		title := field

		// add the comment to the title
		comment, _ := chutils.GetSystemField(*runDetail.Table, "comment", field, conn)
		if comment != "" {
			title = fmt.Sprintf("%s: %s", title, comment)
		}

		if e := FieldPlot(qry, fld, where, plotType, *runDetail.OutDir, fld, title, runDetail.ImageTypesCh, *runDetail.Show, conn); e != nil {
			return e
		}
	}

	return nil
}

func Multiple(runDetail *RunDef, conn *chutils.Connect) error {
	for ind := 0; ind < len(runDetail.Fds); ind++ {
		fd := runDetail.Fds[ind]
		plotType := "histogram"
		if fd.ChSpec.Base == chutils.ChFloat {
			plotType = "quantile"
		}

		where := getWhere(runDetail.MissInt, runDetail.MissFlt, runDetail.MissStr, runDetail.MissDt, fd.Name, fmt.Sprintf("%v", fd.ChSpec.Base))

		if e := FieldPlot(*runDetail.Qry, fd.Name, where, plotType, *runDetail.OutDir, fd.Name, "",
			runDetail.ImageTypesCh, *runDetail.Show, conn); e != nil {
			return e
		}
	}

	return nil
}

func Drive(runDetail *RunDef, conn *chutils.Connect) error {
	switch runDetail.Task {
	case TaskTable:
		return Table(runDetail, conn)
	case TaskQuery:
		return Multiple(runDetail, conn)
	}

	return nil
}
