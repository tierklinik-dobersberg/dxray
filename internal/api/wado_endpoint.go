package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tierklinik-dobersberg/service/server"
)

// http://dicom.nema.org/medical/dicom/current/output/chtml/part18/chapter_9.html
func WadoEndpoint(grp gin.IRouter) {
	grp.GET("wado", func(ctx *gin.Context) {
		requestType := ctx.Query("requestType")
		studyUID := ctx.Query("studyUID")
		seriesUID := ctx.Query("seriesUID")
		objectUID := ctx.Query("objectUID")
		contentType := ctx.Query("contentType")

		if contentType != "" && (contentType != "application/dicom" && contentType != "image/jpeg") {
			ctx.AbortWithStatus(http.StatusNotAcceptable)
			return
		}

		if studyUID == "" || seriesUID == "" || objectUID == "" {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if requestType != "WADO" {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}

		std, err := getStudyByUID(ctx, studyUID)
		if err != nil {
			return
		}

		if err := std.Load(); err != nil {
			server.AbortRequest(ctx, 0, err)
			return
		}

		model, _ := std.Model()
		for _, series := range model.Patient.Visit.Study.Series {
			if series.UID == seriesUID {
				for _, instance := range series.Instances {
					if instance.UID == objectUID {
						// check if we should return application/dicom or the thumbnail image
						var path string
						if contentType == "image/jpeg" {
							path = strings.Replace(instance.Data.DICOMPath, "I_", "S128_", 1)
							path = strings.Replace(path, ".dcm", ".jpg", 1)
							path = std.RealPath(path)
						} else {
							path = std.RealPath(instance.Data.DICOMPath)
						}

						ctx.File(path)
						return
					}
				}
			}
		}

		ctx.AbortWithStatus(http.StatusNotFound)
	})
}
