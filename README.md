## Describe

[![Go Report Card](https://goreportcard.com/badge/github.com/invertedv/describe)](https://goreportcard.com/report/github.com/invertedv/describe)
[![godoc](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/mod/github.com/invertedv/describe?tab=overview)

### Background

Describe is a package and a command to generate descriptive plots of fields in ClickHouse tables or queries.
For fields of type float, describe generates quantile plots.  For all other fields, it generates histograms.

Describe creates either files of the plots or displays them in a browser (or both).

Describe depends on [orca](https://github.com/plotly/orca).

### Parameters

ClickHouse credentials
- user
- pw


Source of the data. 

One of -q and -t needs to be specified.  If the entire table is run, Describe includes the comment for each field
in the title of the plot.
 
- -q - Query to pull the data, enclosed in quote.
- -t - Table name.
 

- -i - Image types.  One or more of: png,jpeg,html,pdf.webp,svg,eps,emf.  If none is specified, the plot(s) are sent
to the browser.
- -d - Directory.  The plots are placed in this directory.  Each image type is placed in a separate subdirectory. 
- show . If included, the plot is sent to the browser.

#### Missing Values.
By default, the results exclude values that indicate the data is missing.  Use -miss to disable this feature. 

- -mF - Value that indicates a missing float. Default: -1.
- -mI - Value that indicates a missing int.  Default: -1.
- -mS - Value that indicates a missing string. Default: !.
- -mD - Value that indicates a missing data. Default: 19700101
- -miss - If present, missing-value filter is disabled.

- -pdf - If present, the graphs are bundled into a pdf.


TODO: pdf
TODO: use default browser...


- -pdf: create a skeleton md with includes and then pandoc to pdf
- describe -q "select purpose from bk.loan"   : into browser
- describe -q "select * from bk.loan" -f png,html     <- check # of returns,
- describe -t bk.loan -d /home/will/describe -pdf

- pdf can't use -html
- mI, -mF, -mS, -mD
- q  (query)
- d  (output dir)
- f  (file root name)
