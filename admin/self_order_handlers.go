package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"selfcore/internal/dto"
	"selfcore/internal/pkg/authcontext"
	"selfcore/internal/pkg/httputil"
	"selfcore/internal/pkg/response"
	"selfcore/internal/repo"
	"selfcore/internal/service"

	"github.com/gin-gonic/gin"
)

type SelfOrderHandler struct {
	svc *service.SelfOrderService
}

func NewSelfOrderHandler(svc *service.SelfOrderService) *SelfOrderHandler {
	return &SelfOrderHandler{svc: svc}
}

func (h *SelfOrderHandler) ps(c *gin.Context) *service.SelfOrderService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *SelfOrderHandler) listFilterFromQuery(c *gin.Context) repo.SelfOrderListFilter {
	refSoID, _ := strconv.ParseUint(c.Query("refSoId"), 10, 64)
	return repo.SelfOrderListFilter{
		Status: c.Query("status"), Statuses: splitCSV(c.Query("statuses")),
		ExcludeStatuses: splitCSV(c.Query("excludeStatuses")),
		PayStatuses:     splitCSV(c.Query("payStatus")),
		RefSoID: refSoID, Keyword: c.Query("keyword"),
		ShipStatus:     c.Query("shipStatus"),
		CreatedAtStart: parsePOCreatedAtStart(firstNonEmptyQuery(c, "createdAtStart", "orderedAtStart")),
		CreatedAtEnd:   parseDateTimeInclusiveEnd(firstNonEmptyQuery(c, "createdAtEnd", "orderedAtEnd")),
		ShippedAtStart: parsePOCreatedAtStart(c.Query("shippedAtStart")),
		ShippedAtEnd:   parseDateTimeInclusiveEnd(c.Query("shippedAtEnd")),
	}
}

func (h *SelfOrderHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	f := h.listFilterFromQuery(c)
	f.Page, f.PageSize = page, pageSize
	list, total, err := h.ps(c).List(f)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

// StatusCounts 自营单状态数量（按时间/关键词上下文，不含 status 筛选本身）。
func (h *SelfOrderHandler) StatusCounts(c *gin.Context) {
	f := h.listFilterFromQuery(c)
	// 数量统计忽略当前状态/付款筛选，避免 tab 数字随选中项变空
	f.Status, f.Statuses, f.ExcludeStatuses, f.PayStatuses, f.ShipStatus = "", nil, nil, nil, ""
	counts, err := h.ps(c).CountStatusFacets(f)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, counts)
}

func (h *SelfOrderHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	item, err := h.ps(c).Get(id)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "自营单不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *SelfOrderHandler) Create(c *gin.Context) {
	var in dto.SelfOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := h.ps(c).Create(c.Request.Context(), authcontext.BearerToken(c), &in)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Created(c, item)
}

func (h *SelfOrderHandler) BindInvSku(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("itemId"), 10, 64)
	if err != nil || itemID == 0 {
		response.Fail(c, http.StatusBadRequest, "无效明细 ID")
		return
	}
	var in dto.BindInvSkuInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := h.ps(c).BindInvSku(itemID, &in)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "明细不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *SelfOrderHandler) UpdateItemCost(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("itemId"), 10, 64)
	if err != nil || itemID == 0 {
		response.Fail(c, http.StatusBadRequest, "无效明细 ID")
		return
	}
	var in dto.UpdateItemCostInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := h.ps(c).UpdateItemCost(itemID, &in)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "明细不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *SelfOrderHandler) ListShipments(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	list, err := h.ps(c).ListShipments(id)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *SelfOrderHandler) CreateShipment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	var in dto.SelfShipmentCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := h.ps(c).CreateShipment(c.Request.Context(), authcontext.BearerToken(c), id, &in)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "自营单不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Created(c, item)
}

func (h *SelfOrderHandler) UpdateShipmentStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	shipmentID, err := strconv.ParseUint(c.Param("shipmentId"), 10, 64)
	if err != nil || shipmentID == 0 {
		response.Fail(c, http.StatusBadRequest, "无效发货 ID")
		return
	}
	var in dto.SelfShipmentStatusInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := h.ps(c).UpdateShipmentStatus(id, shipmentID, in.Status)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "记录不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *SelfOrderHandler) DeleteShipment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	shipmentID, err := strconv.ParseUint(c.Param("shipmentId"), 10, 64)
	if err != nil || shipmentID == 0 {
		response.Fail(c, http.StatusBadRequest, "无效发货 ID")
		return
	}
	if err := h.ps(c).DeleteShipment(id, shipmentID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "记录不存在")
			return
		}
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *SelfOrderHandler) SyncShipmentsFromOrders(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	var in dto.SyncShipmentsFromOrdersInput
	_ = c.ShouldBindJSON(&in)
	if v := c.Query("refSoId"); v != "" {
		if refID, perr := strconv.ParseUint(v, 10, 64); perr == nil {
			in.RefSoID = refID
		}
	}
	result, err := h.ps(c).SyncShipmentsFromOrders(c.Request.Context(), id, authcontext.BearerToken(c), &in)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "自营单不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *SelfOrderHandler) SyncShipmentsByRefSo(c *gin.Context) {
	var in dto.SyncShipmentsFromOrdersInput
	if err := c.ShouldBindJSON(&in); err != nil || in.RefSoID == 0 {
		response.Fail(c, http.StatusBadRequest, "请提供 refSoId")
		return
	}
	result, err := h.ps(c).SyncShipmentsByRefSoID(c.Request.Context(), authcontext.BearerToken(c), in.RefSoID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *SelfOrderHandler) ListAttachments(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	list, err := h.ps(c).ListAttachments(id)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "自营单不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *SelfOrderHandler) CreateAttachment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	var in dto.SelfAttachmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := h.ps(c).CreateAttachment(id, authcontext.UserID(c), &in)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "自营单不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Created(c, item)
}

func (h *SelfOrderHandler) DeleteAttachment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	attachmentID, err := strconv.ParseUint(c.Param("attachmentId"), 10, 64)
	if err != nil || attachmentID == 0 {
		response.Fail(c, http.StatusBadRequest, "无效附件 ID")
		return
	}
	if err := h.ps(c).DeleteAttachment(id, attachmentID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "记录不存在")
			return
		}
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *SelfOrderHandler) Ship(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	var in dto.SelfShipInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := h.ps(c).Ship(c.Request.Context(), authcontext.BearerToken(c), id, &in)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "自营单不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *SelfOrderHandler) RetryStock(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	item, err := h.ps(c).RetryStockDeduct(c.Request.Context(), authcontext.BearerToken(c), id)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "自营单不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *SelfOrderHandler) RetryCallback(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	shipmentID, _ := strconv.ParseUint(c.Query("shipmentId"), 10, 64)
	item, err := h.ps(c).RetryCallback(c.Request.Context(), authcontext.BearerToken(c), id, shipmentID)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "记录不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *SelfOrderHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	if err := h.ps(c).Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *SelfOrderHandler) Submit(c *gin.Context) {
	h.doAction(c, func(id uint64) (*dto.SelfOrderDetail, error) {
		return h.ps(c).Submit(id)
	})
}

func (h *SelfOrderHandler) MarkPaid(c *gin.Context) {
	h.doAction(c, func(id uint64) (*dto.SelfOrderDetail, error) {
		return h.ps(c).MarkPaid(id)
	})
}

func (h *SelfOrderHandler) Complete(c *gin.Context) {
	h.doAction(c, func(id uint64) (*dto.SelfOrderDetail, error) {
		return h.ps(c).Complete(id)
	})
}

func (h *SelfOrderHandler) doAction(c *gin.Context, fn func(uint64) (*dto.SelfOrderDetail, error)) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	item, err := fn(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *SelfOrderHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	var in dto.SelfCancelInput
	_ = c.ShouldBindJSON(&in)
	item, err := h.ps(c).CancelWithReason(id, in.Reason)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "自营单不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *SelfOrderHandler) CancelByRefSo(c *gin.Context) {
	var in dto.SelfCancelByRefInput
	if err := c.ShouldBindJSON(&in); err != nil || in.RefSoID == 0 {
		response.Fail(c, http.StatusBadRequest, "请提供 refSoId")
		return
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = "撤回分配"
	}
	list, err := h.ps(c).CancelByRefSoID(in.RefSoID, reason)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *SelfOrderHandler) DeleteByRefSo(c *gin.Context) {
	var in dto.SelfDeleteByRefInput
	if err := c.ShouldBindJSON(&in); err != nil || in.RefSoID == 0 {
		response.Fail(c, http.StatusBadRequest, "请提供 refSoId")
		return
	}
	n, err := h.ps(c).DeleteByRefSoID(in.RefSoID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{"deleted": n})
}

func (h *SelfOrderHandler) ListPayments(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	list, err := h.ps(c).ListPayments(id)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "自营单不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *SelfOrderHandler) CreatePayment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	var in dto.SelfPaymentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := h.ps(c).CreatePayment(c.Request.Context(), authcontext.BearerToken(c), id, &in)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "自营单不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Created(c, item)
}

func (h *SelfOrderHandler) UpdatePayment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	paymentID, err := strconv.ParseUint(c.Param("paymentId"), 10, 64)
	if err != nil || paymentID == 0 {
		response.Fail(c, http.StatusBadRequest, "无效付款 ID")
		return
	}
	var in dto.SelfPaymentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := h.ps(c).UpdatePayment(c.Request.Context(), authcontext.BearerToken(c), id, paymentID, &in)
	if errors.Is(err, service.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, "记录不存在")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *SelfOrderHandler) DeletePayment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "无效 ID")
		return
	}
	paymentID, err := strconv.ParseUint(c.Param("paymentId"), 10, 64)
	if err != nil || paymentID == 0 {
		response.Fail(c, http.StatusBadRequest, "无效付款 ID")
		return
	}
	if err := h.ps(c).DeletePayment(c.Request.Context(), authcontext.BearerToken(c), id, paymentID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "记录不存在")
			return
		}
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func parseDateTimeInclusiveEnd(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "T", " ")
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			if layout == "2006-01-02" {
				t = t.Add(24*time.Hour - time.Second)
			}
			return &t
		}
	}
	return nil
}


func firstNonEmptyQuery(c *gin.Context, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(c.Query(k)); v != "" {
			return v
		}
	}
	return ""
}
