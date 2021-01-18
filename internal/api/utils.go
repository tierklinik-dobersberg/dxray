package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tierklinik-dobersberg/dxray/internal/app"
	"github.com/tierklinik-dobersberg/dxray/internal/dxr/fsdb"
	"github.com/tierklinik-dobersberg/dxray/internal/search"
	"github.com/tierklinik-dobersberg/service/server"
)

// createSutdyURLFactory returns a function that can be used
// to create a dicomweb:// URL for a study instance. The hostname
// is copied from the gin.Context.
func createStudyURLFactory(ctx *gin.Context) func(study, series, instance string) string {
	return func(study, series, instance string) string {
		host := ctx.Request.Host
		values := url.Values{}

		values.Add("seriesUID", series)
		values.Add("studyUID", study)
		values.Add("objectUID", instance)
		values.Add("requestType", "WADO")

		return fmt.Sprintf("dicomweb://%s/api/dxray/v1/wado?%s", host, values.Encode())
	}
}

// getStudyByUID lodas the study from the the FsDB that's identified
// by UID.
func getStudyByUID(ctx *gin.Context, uid string) (fsdb.Study, error) {
	appCtx := app.From(ctx)
	if appCtx == nil {
		return nil, errors.New("no app context")
	}

	key, err := appCtx.Indexer.Search(fmt.Sprintf("uid:%q", uid))
	if err != nil {
		server.AbortRequest(ctx, http.StatusBadRequest, err)
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

	std, err := search.Get(key[0], appCtx.FsDB)
	if err != nil {
		server.AbortRequest(ctx, http.StatusInternalServerError, err)
		return nil, err
	}

	return std, nil
}

// getNumberParam returns the value of the query parameter name
// parsed as a number.
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

// getNumberParamDefault is like getNumberParam but returns a default value
// in case the query parameter name is not set at all.
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
