package describe

import (
	"fmt"
	"strings"

	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"github.com/dustin/go-humanize"
	"github.com/invertedv/chutils"
	s "github.com/invertedv/chutils/sql"
	"github.com/invertedv/utilities"
)

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
		Show:      show,
		Title:     "",
		YTitle:    "",
		STitle:    qry,
		Legend:    false,
		Height:    800,
		Width:     1000,
		FileName:  outFile,
		OutDir:    outDir,
		FileTypes: imageTypes,
	}

	if where != "" {
		pd.STitle = fmt.Sprintf("%s WHERE %s", qry, where)
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

	if e := utilities.Plotter(fig, nil, pd); e != nil {
		return e
	}

	return nil
}

func Table(table, outDir string, imageTypes []utilities.PlotlyImage, fds []*chutils.FieldDef,
	missStr, missInt, missFlt, missDt any, conn *chutils.Connect) error {
	fTypes, err := chutils.GetSystemFields(conn, "type", table)
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
			_, fld, _ = strings.Cut(field, ".")
		}

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

			where = fmt.Sprintf("%s %s %s", fld, oper, miss)
		}

		qry := fmt.Sprintf("SELECT %s FROM %s", field, table)
		if strings.Contains(fType, "Array") {
			qry = fmt.Sprintf("SELECT arrayJoin(%s) AS %s FROM %s", field, fld, table)
		}

		title := field

		// add the comment to the title
		comment, _ := chutils.GetSystemField(table, "comment", field, conn)
		if comment != "" {
			title = fmt.Sprintf("%s: %s", title, comment)
		}

		if e := FieldPlot(qry, fld, where, plotType, outDir, fld, title, imageTypes, false, conn); e != nil {
			panic(e)
		}
	}

	return nil
}
