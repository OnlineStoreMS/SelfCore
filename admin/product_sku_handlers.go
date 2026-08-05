package admin

import (
	"net/http"
	"strconv"
	"strings"

	"selfcore/internal/integrations/productcore"
	"selfcore/internal/pkg/httputil"
	"selfcore/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type ProductSkuHandler struct {
	pc *productcore.Client
}

func NewProductSkuHandler(pc *productcore.Client) *ProductSkuHandler {
	return &ProductSkuHandler{pc: pc}
}

func (h *ProductSkuHandler) Search(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		response.Fail(c, http.StatusBadRequest, "请输入 SKU 编码、规格值或商品关键字")
		return
	}
	page, pageSize := httputil.ParsePage(c)
	auth := c.GetHeader("Authorization")
	list, total, err := h.pc.SearchSkus(c.Request.Context(), auth, keyword, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *ProductSkuHandler) SearchProducts(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	auth := c.GetHeader("Authorization")
	list, total, err := h.pc.SearchProducts(c.Request.Context(), auth, c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *ProductSkuHandler) GetProductSkus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid product id")
		return
	}
	auth := c.GetHeader("Authorization")
	item, err := h.pc.GetProductSkus(c.Request.Context(), auth, id)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, item)
}
