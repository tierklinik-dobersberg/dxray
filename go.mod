module github.com/tierklinik-dobersberg/dxray

go 1.13

require (
	github.com/apex/log v1.1.1
	github.com/blevesearch/bleve v1.0.14
	github.com/gin-gonic/gin v1.6.3
	github.com/grailbio/go-dicom v0.0.0-20190117035129-c30d9eaca591
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/kr/pretty v0.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/ppacher/system-conf v0.3.0
	github.com/tierklinik-dobersberg/logger v0.0.0-20201214100914-9fd1564ce006
	github.com/tierklinik-dobersberg/service v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

replace github.com/ppacher/system-conf => ../system-conf

replace github.com/tierklinik-dobersberg/service => ../service
