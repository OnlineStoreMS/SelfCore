package admin

import (
	"net/http"

	"selfcore/internal/integrations/warehousecore"
	"selfcore/internal/pkg/authcontext"
	"selfcore/internal/pkg/httputil"
	"selfcore/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type WarehouseHandler struct {
	wh *warehousecore.Client
}

func NewWarehouseHandler(wh *warehousecore.Client) *WarehouseHandler {
	return &WarehouseHandler{wh: wh}
}

func (h *WarehouseHandler) SearchSkus(c *gin.Context) {
	if h.wh == nil || !h.wh.Enabled() {
		response.Fail(c, http.StatusBadGateway, "WarehouseCore 未配置")
		return
	}
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.wh.SearchSkus(c.Request.Context(), authcontext.BearerToken(c), c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *WarehouseHandler) ListWarehouses(c *gin.Context) {
	if h.wh == nil || !h.wh.Enabled() {
		response.Fail(c, http.StatusBadGateway, "WarehouseCore 未配置")
		return
	}
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.wh.ListWarehouses(c.Request.Context(), authcontext.BearerToken(c), c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}
