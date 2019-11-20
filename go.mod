module github.com/tierklinik-dobersberg/dxray

go 1.13

require (
	github.com/RoaringBitmap/roaring v0.4.21 // indirect
	github.com/apex/log v1.1.1
	github.com/blevesearch/bleve v0.8.1
	github.com/blevesearch/go-porterstemmer v1.0.2 // indirect
	github.com/blevesearch/segment v0.0.0-20160915185041-762005e7a34f // indirect
	github.com/couchbase/vellum v0.0.0-20190829182332-ef2e028c01fd // indirect
	github.com/gin-gonic/gin v1.4.0
	github.com/grailbio/go-dicom v0.0.0-20190117035129-c30d9eaca591
	github.com/steveyen/gtreap v0.0.0-20150807155958-0abe01ef9be2 // indirect
	github.com/tierklinik-dobersberg/micro v0.0.0-20191115074518-f640789e0dfb
	github.com/ugorji/go v1.1.7 // indirect
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
)

replace github.com/tierklinik-dobersberg/micro => ../micro
