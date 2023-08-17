## Describe

[![Go Report Card](https://goreportcard.com/badge/github.com/invertedv/describe)](https://goreportcard.com/report/github.com/invertedv/describe)
[![godoc](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/mod/github.com/invertedv/describe?tab=overview)

### Background

Describe is a package and a command to generate descriptive plots of fields in ClickHouse tables or queries.
For fields of type float, describe generates quantile plots.  For all other fields, it generates histograms.

Describe creates either files of the plots or displays them in a browser (or both).

Describe depends on [orca](https://github.com/plotly/orca).

### Parameters

#### Help

- -h

Prints help.

#### ClickHouse credentials
- -user <user name>
- -pw <password>

#### Source of the data. 

One of -q and -t needs to be specified.  If the entire table is run, Describe includes the comment for each field
in the title of the plot.
 
- -q \<"SELECT * FROM..."\>. Query to pull the data, enclosed in quote.
- -t \<db.table\>. Table name.
 
#### Outputs
- -xy <'xField,yField1,..yFieldk'>.  If this flag is used, an XY plot is created. The input must be a query.
  The field names to plot are enclosed in quotes and are comma-separated. This syntax works, too:
    -  -xy <'field'> which plots 'field' against an index 0,1,2...
    - -lineType 'm,l' line types for xy plots (m=marker, l=line)
    - -color colors for xy plots (e.g. 'black' 'red',...)
    - -f - Filename.  Optional root file name for output graphs (no extension )
    - -box Do boxplots.
- -i \<image type\>. Image types.  One or more of: png, jpeg, html, pdf, webp, svg, eps, emf.  If none is specified, the plot(s) are sent
to the browser.
- -d - Directory.  Directory for output images. Defaults to the working directory.
- -b \<browser\>. Browser for images. If omitted, the system default is used.
- -show - If included, the plot is (also) sent to the browser. -show is assumed if -d and -i are omitted.
- -title - If included, the plots are titled with this value.
- -subtitle - Optional subtitle.
- -threads - If included, maximum # of threads for ClickHouse to use.
- -width - plot width (default 1000)
- -height - plot heith (default 800)
- -xlim <min,max>  x-axis range.
- -ylim <min,max>  y-axis range.
- -log Plot y-axis on log scale.
 
 Images are placed in subdirectories of -d according to image type. For example, if you have

    -i png,html

then two subdirectories are created - png and html - for images of the corresponding type.

Image filenames are the name of the field.

#### Missing Values.
By default, the results exclude values that indicate the data is missing.  Use -miss to disable this feature. 

- -mF \<value\>. Value that indicates a missing float. Default: -1.
- -mI \<value\>. Value that indicates a missing int.  Default: -1.
- -mS \<value\>. Value that indicates a missing string. Default: !.
- -mD \<value\>. Value that indicates a missing data. Default: 19700101
- -miss - If present, missing-value filter is disabled.

- -markdown \<filename\> - If present, the graphs are bundled into a markdown file \<filename\>.  
Requires -d parameter to point to the directory of images. If the input files are
html, markdown uses links.  Otherwise, the graphs are included. From markdown, you can include it in
a Jekyll (GitHub pages) site, or you can convert it to PDF.  If the -d path is relative, the links are
relative. -markdown is run standalone, outside of creating the images. Why? Well, we'd have to add another
flag to specify which image type to use.  

#### Parameter Combinations

1. -i requires -d
2. -show is implied if -i is omitted.
3. If -d is omitted, -d is set to the working directory. 

#### Examples


    describe -q "select purpose from bk.loan" -user <user> -pw <pw>  

Runs the query, sending the graph to the default browser.

    describe -q "select * from bk.loan" -i png,html -user <user> -pw <pw>

Runs the query, pulling all the fields from bk.loan.  Both png and html files are produced. These are placed in
the current working directory. One could, instead, use:

    describe -q bk.loan -i png,html -user <user> -pw <pw>

which will include the field comments in the graphs.

    describe -t bk.loan -d figs/png -markdown figs.md

Creates a markdown file, figs.md, in the current working directory with the images in figs/png.

    describe -q 'select ltv, cltv from bk.loan' -xy 'ltv,cltv' -show

produces a cross plot of cltv (y-axis) vs ltv (x-axis) in the default browser

#### Images

Histograms are not produced for fields that have more than 1000 distinct levels.
