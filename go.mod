module github.com/tierklinik-dobersberg/dxray

go 1.13

require (
	github.com/apex/log v1.1.1
	github.com/blevesearch/bleve v1.0.14
	github.com/gin-gonic/gin v1.6.3
	github.com/grailbio/go-dicom v0.0.0-20190117035129-c30d9eaca591
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/tierklinik-dobersberg/logger v0.0.0-20201125171257-d519c7625406
	github.com/tierklinik-dobersberg/micro v0.0.0-20191115074518-f640789e0dfb
	golang.org/x/crypto v0.0.0-20190621222207-cc06ce4a13d4 // indirect
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
)

replace github.com/tierklinik-dobersberg/micro => ../micro
