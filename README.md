## Describe

[![Go Report Card](https://goreportcard.com/badge/github.com/invertedv/describe)](https://goreportcard.com/report/github.com/invertedv/describe)
[![godoc](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/mod/github.com/invertedv/describe?tab=overview)

### Background

Describe is a package and a command to generate descriptive plots of fields in ClickHouse tables or queries.
For fields of type float, describe generates quantile plots.  For all other fields, it generates histograms.

Describe creates either files of the plots or displays them in a browser (or both).

Describe depends on [orca](https://github.com/plotly/orca).

### Parameters

#### ClickHouse credentials
- user <user name>
- pw <password>

#### Source of the data. 

One of -q and -t needs to be specified.  If the entire table is run, Describe includes the comment for each field
in the title of the plot.
 
- -q \<"SELECT * FROM..."\>. Query to pull the data, enclosed in quote.
- -t \<db.table\>. Table name.
 
#### Outputs
- -i \<image type\>. Image types.  One or more of: png,jpeg,html,pdf.webp,svg,eps,emf.  If none is specified, the plot(s) are sent
to the browser.
- -d - Directory.  Directory for output images.
- -b \<browser\>. Browser for images. If omitted, the system default is used.
- -show - If included, the plot is sent to the browser.
 
 
Images are placed in subdirectories of -d accroding to image type. For example, if you have

    -i png,html

then two subdirectories are created - png and html - for images of the corresponding type.

Filenames are the name of the field.

#### Missing Values.
By default, the results exclude values that indicate the data is missing.  Use -miss to disable this feature. 

- -mF \<value\>. Value that indicates a missing float. Default: -1.
- -mI \<value\>. Value that indicates a missing int.  Default: -1.
- -mS \<value\>. Value that indicates a missing string. Default: !.
- -mD \<value\>. Value that indicates a missing data. Default: 19700101
- -miss - If present, missing-value filter is disabled.

- -pdf - If present, the graphs are bundled into a pdf.  pdf cannot use html input.

#### Parameter Combinations

1. -i requires -d
2. -show is implied if -i is omitted.
3. 

#### Examples

describe -q "select purpose from bk.loan"   
describe -q "select * from bk.loan" -i png,html
describe -t bk.loan -d ~/describe -pdf

#### Images

Histograms are not produced for fields that have more than 1000 distinct levels.
