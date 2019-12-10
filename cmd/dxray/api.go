package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/apex/log"
	"github.com/gin-gonic/gin"
	"github.com/tierklinik-dobersberg/dxray/pkg/dxr/fsdb"
	"github.com/tierklinik-dobersberg/dxray/pkg/ohif"
	"github.com/tierklinik-dobersberg/dxray/pkg/search"
	"github.com/tierklinik-dobersberg/micro/pkg/api"
)

const (
	// ActionSearch defines the IAM action that a user must be allowed
	// in order to search for studies
	ActionSearch = "dxray:study.search"

	// ActionReadStudy defines the IAM action that a user must be allowed
	// in order to read a study
	ActionReadStudy = "dxray:study.read"
)

// API is the RESTful API of dxray
var API = api.Module{
	Name: "api",
	Setup: func(r api.Router) error {
		r.GET("/search", searchStudies(r))

		// TODO(ppacher): we should try to extract some study metadata
		// and not just check if the user has permission to read all studies

		r.GET("/ohif/:study", ohifStudyJSON(r))
		r.GET("/wado", wadoURI(r))

		/*
			r.GET("/search", auth.Permission(ActionSearch, nil), searchStudies(r))

			// TODO(ppacher): we should try to extract some study metadata
			// and not just check if the user has permission to read all studies

			r.GET("/ohif/:study", auth.Permission(ActionReadStudy, nil), ohifStudyJSON(r))
			r.GET("/wado", auth.Permission(ActionReadStudy, nil), wadoURI(r))
		*/
		return nil
	},
}

func getStudyURL(ctx *gin.Context) func(study, series, instance string) string {
	return func(study, series, instance string) string {
		host := ctx.Request.Host
		scheme := "https"
		if ctx.Request.TLS != nil {
			scheme = "https"
		}

		values := url.Values{}

		values.Add("seriesUID", series)
		values.Add("studyUID", study)
		values.Add("objectUID", instance)
		values.Add("requestType", "WADO")

		_ = scheme
		url := fmt.Sprintf("dicomweb://%s/wado?%s", host, values.Encode())
		log.Infof("url: %q", url)

		return url
	}
}

func searchStudies(r api.Router) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		index := r.GetKey(ContextKeyIndexer).(*StudyIndexer)
		db, _ := r.GetKey(ContextKeyDXR).(*DXR).Open()

		term := ctx.Query("q")
		results, err := index.Search(term)
		if err != nil {
			r.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		models := make([]*ohif.StudyJSON, 0, len(results))
		for _, key := range results {
			s, err := search.Get(key, db)
			if err != nil {
				r.Log().WithFields(log.Fields{
					"error": err.Error(),
					"key":   key,
				}).Errorf("failed to open study")
				continue
			}

			m, err := ohif.JSONFromDXR(s, getStudyURL(ctx), false)
			if err != nil {
				r.Log().WithFields(log.Fields{
					"error": err.Error(),
					"key":   key,
				}).Errorf("failed to get JSON representaion")
				continue
			}

			models = append(models, m)
		}

		ctx.JSON(http.StatusOK, models)
	}
}

func ohifStudyJSON(r api.Router) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uid := ctx.Param("study")

		std, err := getStudyByUID(ctx, uid, r)
		if err != nil {
			return
		}

		model, err := ohif.JSONFromDXR(std, getStudyURL(ctx), true)
		if err != nil {
			r.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)

		if err := enc.Encode(map[string]interface{}{
			"studies": []interface{}{model},
		}); err != nil {
			r.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		ctx.Data(http.StatusOK, "application/json", buf.Bytes())
	}
}

// http://dicom.nema.org/medical/dicom/current/output/chtml/part18/chapter_9.html
func wadoURI(r api.Router) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestType := ctx.Query("requestType")
		studyUID := ctx.Query("studyUID")
		seriesUID := ctx.Query("seriesUID")
		objectUID := ctx.Query("objectUID")
		contentType := ctx.Query("contentType")

		if contentType != "" && contentType != "application/dicom" {
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

		std, err := getStudyByUID(ctx, studyUID, r)
		if err != nil {
			return
		}

		if err := std.Load(); err != nil {
			r.AbortRequest(ctx, 0, err)
			return
		}

		model, _ := std.Model()
		for _, series := range model.Patient.Visit.Study.Series {
			if series.UID == seriesUID {
				for _, instance := range series.Instances {
					if instance.UID == objectUID {
						path := std.RealPath(instance.Data.DICOMPath)

						ctx.File(path)
						return
					}
				}
			}
		}

		ctx.AbortWithStatus(http.StatusNotFound)
	}
}

func getStudyByUID(ctx *gin.Context, uid string, r api.Router) (fsdb.Study, error) {
	index := r.GetKey(ContextKeyIndexer).(*StudyIndexer)
	db, _ := r.GetKey(ContextKeyDXR).(*DXR).Open()

	key, err := index.Search(fmt.Sprintf("uid:%q", uid))
	if err != nil {
		r.AbortRequest(ctx, http.StatusBadRequest, err)
		return nil, err
	}

	if len(key) == 0 {
		ctx.AbortWithStatus(http.StatusNotFound)
		return nil, errors.New("aborted")
	}

	if len(key) > 1 {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return nil, errors.New("aborted")
	}

	std, err := search.Get(key[0], db)
	if err != nil {
		r.AbortRequest(ctx, http.StatusInternalServerError, err)
		return nil, err
	}

	return std, nil
}
