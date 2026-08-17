package admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"selfcore/internal/integrations/ordercore"
	"selfcore/internal/pkg/authcontext"
	"selfcore/internal/pkg/httputil"
	"selfcore/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	oc *ordercore.Client
}

func NewOrderHandler(oc *ordercore.Client) *OrderHandler {
	return &OrderHandler{oc: oc}
}

func (h *OrderHandler) Search(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	auth := authcontext.AuthorizationHeader(c)
	q := ordercore.OrderListQuery{
		SourceChannel:  strings.TrimSpace(c.Query("sourceChannel")),
		Status:         strings.TrimSpace(c.Query("status")),
		ShipStatus:     strings.TrimSpace(c.Query("shipStatus")),
		AllocType:      strings.TrimSpace(c.Query("allocType")),
		Keyword:        strings.TrimSpace(c.Query("keyword")),
		Platform:       strings.TrimSpace(c.Query("platform")),
		SalesChannel:   strings.TrimSpace(c.Query("salesChannel")),
		OrderedAtStart: strings.TrimSpace(c.Query("orderedAtStart")),
		OrderedAtEnd:   strings.TrimSpace(c.Query("orderedAtEnd")),
		ShippedAtStart: strings.TrimSpace(c.Query("shippedAtStart")),
		ShippedAtEnd:   strings.TrimSpace(c.Query("shippedAtEnd")),
		PayTimeStart:   strings.TrimSpace(c.Query("payTimeStart")),
		PayTimeEnd:     strings.TrimSpace(c.Query("payTimeEnd")),
		Page:           page,
		PageSize:       pageSize,
	}
	list, total, err := h.oc.ListOrders(c.Request.Context(), auth, q)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *OrderHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效订单 ID")
		return
	}
	auth := authcontext.AuthorizationHeader(c)
	order, err := h.oc.GetOrder(c.Request.Context(), auth, id)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, order)
}

func (h *OrderHandler) Ship(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效订单 ID")
		return
	}
	var req ordercore.ShipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	if strings.TrimSpace(req.ExpressNo) == "" {
		response.Fail(c, http.StatusBadRequest, "请填写物流单号")
		return
	}
	if strings.TrimSpace(req.ExpressCompany) == "" {
		response.Fail(c, http.StatusBadRequest, "请选择快递公司")
		return
	}
	// 代发采购侧「回传单号」默认回传电商平台
	cb := true
	req.Callback = &cb
	auth := authcontext.AuthorizationHeader(c)
	// 与前端断开解耦，避免浏览器超时取消导致订单中心回传中断
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 170*time.Second)
	defer cancel()
	order, err := h.oc.ShipOrder(ctx, auth, id, req)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, order)
}

func (h *OrderHandler) Decrypt(c *gin.Context) {
	var body struct {
		OrderIDs []uint64 `json:"orderIds"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.OrderIDs) == 0 {
		response.Fail(c, http.StatusBadRequest, "请传入 orderIds")
		return
	}
	auth := authcontext.AuthorizationHeader(c)
	result, err := h.oc.DecryptOrders(c.Request.Context(), auth, body.OrderIDs)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}
