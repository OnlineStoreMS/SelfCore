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

type DistOrderHandler struct {
	svc *service.DistOrderService
}

func NewDistOrderHandler(svc *service.DistOrderService) *DistOrderHandler {
	return &DistOrderHandler{svc: svc}
}

func (h *DistOrderHandler) ps(c *gin.Context) *service.DistOrderService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *DistOrderHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	distributorID, _ := strconv.ParseUint(c.Query("distributorId"), 10, 64)
	refSoID, _ := strconv.ParseUint(c.Query("refSoId"), 10, 64)
	status := c.Query("status")
	if status == "in_transit" {
		status = "shipped"
	}
	list, total, err := h.ps(c).List(repo.DistOrderListFilter{
		Status: status, Statuses: splitCSV(c.Query("statuses")),
		PayStatuses: splitCSV(c.Query("payStatus")), ExcludeStatuses: splitCSV(c.Query("excludeStatuses")),
		FulfillmentType: c.Query("fulfillmentType"),
		DistributorID: distributorID, RefSoID: refSoID, RefTraceID: c.Query("refTraceId"),
		Keyword: c.Query("keyword"), SortBy: c.Query("sortBy"), SortOrder: c.Query("sortOrder"),
		CreatedAtStart: parsePOCreatedAtStart(c.Query("createdAtStart")),
		CreatedAtEnd:   parsePOCreatedAtEndExclusive(c.Query("createdAtEnd")),
		OrderedAtStart: parsePOCreatedAtStart(c.Query("orderedAtStart")),
		OrderedAtEnd:   parsePOCreatedAtEndExclusive(c.Query("orderedAtEnd")),
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// 兼容旧筛选值「运输中」
		if p == "in_transit" {
			p = "shipped"
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parsePOCreatedAtStart 解析创建日起始（含当日 00:00）。
func parsePOCreatedAtStart(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			if layout == "2006-01-02" {
				day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
				return &day
			}
			return &t
		}
	}
	return nil
}

// parsePOCreatedAtEndExclusive 解析创建日截止：日期按「含当日」转为次日 00:00，查询用 created_at < end。
func parsePOCreatedAtEndExclusive(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			if layout == "2006-01-02" {
				next := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
				return &next
			}
			// 带时分秒：按开区间上界 < end+1s 近似「含该秒」，这里直接用 < t+1ns 不方便；统一加 1 秒
			end := t.Add(time.Second)
			return &end
		}
	}
	return nil
}

func (h *DistOrderHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ps(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DistOrderHandler) Create(c *gin.Context) {
	var in dto.DistOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	claims := authcontext.Claims(c)
	buyerID := authcontext.UserID(c)
	buyerName := ""
	if claims != nil {
		buyerName = claims.DisplayName
	}
	item, err := h.ps(c).Create(&in, buyerID, buyerName)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *DistOrderHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.DistOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ps(c).Update(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DistOrderHandler) UpdateItemPrices(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.UpdateDistOrderItemPricesInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ps(c).UpdateItemPrices(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DistOrderHandler) SyncPurchasePrices(c *gin.Context) {
	response.Fail(c, http.StatusNotImplemented, "SyncPurchasePrices 未在 SelfCore 启用")
}

func (h *DistOrderHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ps(c).Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *DistOrderHandler) Merge(c *gin.Context) {
	var in dto.MergeDistOrdersInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.ps(c).Merge(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *DistOrderHandler) DetachSalesOrder(c *gin.Context) {
	var in dto.DetachSalesOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ps(c).DetachSalesOrder(&in)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httputil.HandleServiceError(c, err)
			return
		}
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{"purchaseOrder": item})
}

func (h *DistOrderHandler) Submit(c *gin.Context) {
	h.doAction(c, func(id uint64) (*dto.DistOrderDetail, error) {
		return h.ps(c).Submit(id)
	})
}

func (h *DistOrderHandler) MarkPaid(c *gin.Context) {
	h.doAction(c, func(id uint64) (*dto.DistOrderDetail, error) {
		return h.ps(c).MarkPaid(id)
	})
}

func (h *DistOrderHandler) Complete(c *gin.Context) {
	h.doAction(c, func(id uint64) (*dto.DistOrderDetail, error) {
		return h.ps(c).Complete(id)
	})
}

func (h *DistOrderHandler) Cancel(c *gin.Context) {
	h.doAction(c, func(id uint64) (*dto.DistOrderDetail, error) {
		return h.ps(c).Cancel(id)
	})
}

func (h *DistOrderHandler) doAction(c *gin.Context, fn func(uint64) (*dto.DistOrderDetail, error)) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := fn(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}
