package api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tierklinik-dobersberg/dxray/internal/ohif"
	"github.com/tierklinik-dobersberg/service/server"
)

func OHIFEndpoint(grp gin.IRouter) {
	grp.GET("ohif/:study", func(ctx *gin.Context) {
		uid := ctx.Param("study")

		std, err := getStudyByUID(ctx, uid)
		if err != nil {
			return
		}

		model, err := ohif.JSONFromDXR(ctx.Request.Context(), std, createStudyURLFactory(ctx), true)
		if err != nil {
			server.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)

		if err := enc.Encode(map[string]interface{}{
			"studies": []interface{}{model},
		}); err != nil {
			server.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		ctx.Data(http.StatusOK, "application/json", buf.Bytes())
	})
}
