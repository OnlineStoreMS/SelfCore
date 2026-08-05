package admin

import (
	"net/http"
	"strconv"
	"strings"

	"selfcore/internal/dto"
	"selfcore/internal/pkg/authcontext"
	"selfcore/internal/pkg/httputil"
	"selfcore/internal/pkg/response"
	"selfcore/internal/service"
	"selfcore/internal/storage"

	"github.com/gin-gonic/gin"
)

type TrackingHandler struct {
	svc     *service.TrackingService
	storage storage.Storage
}

func NewTrackingHandler(svc *service.TrackingService, store storage.Storage) *TrackingHandler {
	return &TrackingHandler{svc: svc, storage: store}
}

func (h *TrackingHandler) ts(c *gin.Context) *service.TrackingService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func parseDistOrderID(c *gin.Context) (uint64, error) {
	return httputil.ParseID(c)
}

func parseReceiptID(c *gin.Context) (uint64, error) {
	idStr := c.Param("receiptId")
	if idStr == "" {
		idStr = c.Param("paymentId") // backward compat
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}

func (h *TrackingHandler) ListShipments(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	list, err := h.ts(c).ListShipments(poID)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *TrackingHandler) CreateShipment(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	var in dto.ShipmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ts(c).CreateShipment(poID, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *TrackingHandler) SyncShipmentsFromOrders(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	var in dto.SyncShipmentsFromOrdersInput
	_ = c.ShouldBindJSON(&in)
	if v := c.Query("refSoId"); v != "" {
		if id, perr := strconv.ParseUint(v, 10, 64); perr == nil {
			in.RefSoID = id
		}
	}
	auth := c.GetHeader("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		tok := authcontext.BearerToken(c)
		if tok != "" {
			auth = "Bearer " + tok
		}
	}
	result, err := h.ts(c).SyncShipmentsFromOrders(c.Request.Context(), poID, auth, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *TrackingHandler) UpdateShipmentStatus(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	shipmentID, err := strconv.ParseUint(c.Param("shipmentId"), 10, 64)
	if err != nil || shipmentID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid shipment id")
		return
	}
	var in dto.ShipmentStatusInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ts(c).UpdateShipmentStatus(poID, shipmentID, in.Status)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *TrackingHandler) DeleteShipment(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	shipmentID, err := strconv.ParseUint(c.Param("shipmentId"), 10, 64)
	if err != nil || shipmentID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid shipment id")
		return
	}
	if err := h.ts(c).DeleteShipment(poID, shipmentID); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *TrackingHandler) ListReceipts(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	list, err := h.ts(c).ListReceipts(poID)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *TrackingHandler) CreateReceipt(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	var in dto.ReceiptInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ts(c).CreateReceipt(poID, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *TrackingHandler) UpdateReceipt(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	paymentID, err := strconv.ParseUint(c.Param("paymentId"), 10, 64)
	if err != nil || paymentID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid payment id")
		return
	}
	var in dto.ReceiptInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ts(c).UpdateReceipt(poID, paymentID, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *TrackingHandler) DeleteReceipt(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	paymentID, err := strconv.ParseUint(c.Param("paymentId"), 10, 64)
	if err != nil || paymentID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid payment id")
		return
	}
	if err := h.ts(c).DeleteReceipt(poID, paymentID); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *TrackingHandler) ListAttachments(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	list, err := h.ts(c).ListAttachments(poID)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *TrackingHandler) CreateAttachment(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	var in dto.AttachmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ts(c).CreateAttachment(poID, authcontext.UserID(c), &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *TrackingHandler) DeleteAttachment(c *gin.Context) {
	poID, err := parseDistOrderID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid po id")
		return
	}
	attachmentID, err := strconv.ParseUint(c.Param("attachmentId"), 10, 64)
	if err != nil || attachmentID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid attachment id")
		return
	}
	if err := h.ts(c).DeleteAttachment(poID, attachmentID); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *TrackingHandler) Upload(c *gin.Context) {
	if h.storage == nil {
		response.Fail(c, http.StatusInternalServerError, "存储未配置")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "请选择文件")
		return
	}
	subdir := strings.Trim(c.PostForm("subdir"), "/")
	if subdir == "" {
		subdir = "po"
	}
	url, err := h.storage.Upload(file, subdir)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, dto.UploadResult{URL: url, FileName: file.Filename})
}
