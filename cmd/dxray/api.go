package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tierklinik-dobersberg/dxray/internal/dxr/fsdb"
	"github.com/tierklinik-dobersberg/dxray/internal/index"
	"github.com/tierklinik-dobersberg/dxray/internal/ohif"
	"github.com/tierklinik-dobersberg/dxray/internal/search"
	"github.com/tierklinik-dobersberg/logger"
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

		r.GET("/list", listStudies(r))

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
		values := url.Values{}

		values.Add("seriesUID", series)
		values.Add("studyUID", study)
		values.Add("objectUID", instance)
		values.Add("requestType", "WADO")

		return fmt.Sprintf("dicomweb://%s/wado?%s", host, values.Encode())
	}
}

func searchStudies(r api.Router) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		log := logger.From(ctx.Request.Context())

		index := r.GetKey(ContextKeyIndexer).(*index.StudyIndexer)
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
				log.WithFields(logger.Fields{
					"error": err.Error(),
					"key":   key,
				}).Errorf("failed to open study")
				continue
			}

			m, err := ohif.JSONFromDXR(ctx.Request.Context(), s, getStudyURL(ctx), false)
			if err != nil {
				log.WithFields(logger.Fields{
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

		model, err := ohif.JSONFromDXR(ctx.Request.Context(), std, getStudyURL(ctx), true)
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
	}
}

func listStudies(r api.Router) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		db, _ := r.GetKey(ContextKeyDXR).(*DXR).Open()

		limit, err := getNumberParamDefault(ctx, "limit", 100)
		if err != nil {
			r.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		offset, err := getNumberParamDefault(ctx, "offset", 0)
		if err != nil {
			r.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		volumes, err := db.VolumeNames()
		if err != nil {
			r.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		if len(volumes) == 0 {
			ctx.JSON(http.StatusOK, []interface{}{})
			return
		}

		// reverse sort the volume names so the newest are on top
		sort.Sort(sort.Reverse(sort.StringSlice(volumes)))

		result := make([]interface{}, limit)

		volIdx := 0
		vol, err := db.OpenVolumeByName(volumes[volIdx])
		if err != nil {
			r.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		i := 0
		// iterate volumes
	L:
		for {
			styIdx := 0
			studies, err := vol.Studies()
			if err != nil {
				r.AbortRequest(ctx, http.StatusInternalServerError, err)
				return
			}

			// sort the in reverse order so newest are on top
			sort.Sort(sort.Reverse(sort.StringSlice(studies)))

			// iterate studies
			for {
				if i-offset >= limit {
					break L
				}

				if styIdx >= len(studies) {
					break
				}

				if i-offset >= 0 {

					study, err := vol.OpenStudyByName(studies[styIdx])
					if err != nil {
						r.AbortRequest(ctx, http.StatusInternalServerError, err)
						return
					}

					m, err := ohif.JSONFromDXR(ctx.Request.Context(), study, getStudyURL(ctx), false)
					if err != nil {
						r.AbortRequest(ctx, http.StatusInternalServerError, err)
						return
					}

					result[i-offset] = m
				}

				styIdx++
				i++
			}

			if volIdx+1 > len(volumes) {
				break L
			}

			volIdx++

			vol, err = db.OpenVolumeByName(volumes[volIdx])
			if err != nil {
				r.AbortRequest(ctx, http.StatusInternalServerError, err)
				return
			}
		}

		ctx.JSON(http.StatusOK, result)
	}
}

func getStudyByUID(ctx *gin.Context, uid string, r api.Router) (fsdb.Study, error) {
	index := r.GetKey(ContextKeyIndexer).(*index.StudyIndexer)
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

func getNumberParam(ctx *gin.Context, name string) (int, bool, error) {
	v := ctx.Query(name)
	if v != "" {
		var err error

		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false, err
		}

		return int(i), true, nil
	}

	return 0, false, nil
}

func getNumberParamDefault(ctx *gin.Context, name string, def int) (int, error) {
	v, set, err := getNumberParam(ctx, name)
	if err != nil {
		return 0, err
	}

	if !set {
		return def, nil
	}

	return v, nil
}
