package api

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/tierklinik-dobersberg/dxray/internal/app"
	"github.com/tierklinik-dobersberg/dxray/internal/ohif"
	"github.com/tierklinik-dobersberg/service/server"
)

// ListStudiesEndpoint allows listing all studies with support
// for pagination (using limit and offset query parameters)
//
// GET /api/list
func ListStudiesEndpoint(grp gin.IRouter) {
	grp.GET("list", func(ctx *gin.Context) {
		appCtx := app.From(ctx)
		if appCtx == nil {
			return
		}

		limit, err := getNumberParamDefault(ctx, "limit", 100)
		if err != nil {
			server.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		offset, err := getNumberParamDefault(ctx, "offset", 0)
		if err != nil {
			server.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		volumes, err := appCtx.FsDB.VolumeNames()
		if err != nil {
			server.AbortRequest(ctx, http.StatusInternalServerError, err)
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
		vol, err := appCtx.FsDB.OpenVolumeByName(volumes[volIdx])
		if err != nil {
			server.AbortRequest(ctx, http.StatusInternalServerError, err)
			return
		}

		i := 0
		// iterate volumes
	L:
		for {
			styIdx := 0
			studies, err := vol.Studies()
			if err != nil {
				server.AbortRequest(ctx, http.StatusInternalServerError, err)
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
						server.AbortRequest(ctx, http.StatusInternalServerError, err)
						return
					}

					m, err := ohif.JSONFromDXR(ctx.Request.Context(), study, createStudyURLFactory(ctx), false)
					if err != nil {
						server.AbortRequest(ctx, http.StatusInternalServerError, err)
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

			vol, err = appCtx.FsDB.OpenVolumeByName(volumes[volIdx])
			if err != nil {
				server.AbortRequest(ctx, http.StatusInternalServerError, err)
				return
			}
		}

		ctx.JSON(http.StatusOK, result)
	})
}
