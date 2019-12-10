module github.com/tierklinik-dobersberg/dxray

go 1.13

require (
	github.com/RoaringBitmap/roaring v0.4.21 // indirect
	github.com/apex/log v1.1.1
	github.com/blevesearch/bleve v0.8.1
	github.com/blevesearch/go-porterstemmer v1.0.2 // indirect
	github.com/blevesearch/segment v0.0.0-20160915185041-762005e7a34f // indirect
	github.com/couchbase/vellum v0.0.0-20190829182332-ef2e028c01fd // indirect
	github.com/gin-contrib/static v0.0.0-20191128031702-f81c604d8ac2
	github.com/gin-gonic/gin v1.5.0
	github.com/gobuffalo/packr/v2 v2.7.1
	github.com/grailbio/go-dicom v0.0.0-20190117035129-c30d9eaca591
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/steveyen/gtreap v0.0.0-20150807155958-0abe01ef9be2 // indirect
	github.com/tierklinik-dobersberg/micro v0.0.0-20191115074518-f640789e0dfb
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
)

replace github.com/tierklinik-dobersberg/micro => ../micro
