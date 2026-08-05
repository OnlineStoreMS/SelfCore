package admin

import (
	"net/http"
	"strconv"

	"selfcore/internal/dto"
	"selfcore/internal/pkg/authcontext"
	"selfcore/internal/pkg/httputil"
	"selfcore/internal/pkg/response"
	"selfcore/internal/repo"
	"selfcore/internal/service"

	"github.com/gin-gonic/gin"
)

type DistributorHandler struct {
	svc *service.DistributorService
}

func NewDistributorHandler(svc *service.DistributorService) *DistributorHandler {
	return &DistributorHandler{svc: svc}
}

func (h *DistributorHandler) ss(c *gin.Context) *service.DistributorService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *DistributorHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	categoryID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	list, total, err := h.ss(c).List(c.Query("keyword"), categoryID, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *DistributorHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DistributorHandler) Create(c *gin.Context) {
	var in dto.DistributorDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *DistributorHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.DistributorDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Update(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DistributorHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *DistributorHandler) ListCategories(c *gin.Context) {
	list, err := h.ss(c).ListCategories()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *DistributorHandler) CreateCategory(c *gin.Context) {
	var in dto.DistributorCategoryDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).CreateCategory(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *DistributorHandler) UpdateCategory(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.DistributorCategoryDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).UpdateCategory(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DistributorHandler) DeleteCategory(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).DeleteCategory(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *DistributorHandler) ListAddresses(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	list, err := h.ss(c).ListAddresses(distributorID, c.Query("type"))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *DistributorHandler) CreateAddress(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	var in dto.DistributorAddressDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).CreateAddress(distributorID, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *DistributorHandler) UpdateAddress(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	addressID, err := strconv.ParseUint(c.Param("addressId"), 10, 64)
	if err != nil || addressID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid address id")
		return
	}
	var in dto.DistributorAddressDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).UpdateAddress(distributorID, addressID, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DistributorHandler) DeleteAddress(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	addressID, err := strconv.ParseUint(c.Param("addressId"), 10, 64)
	if err != nil || addressID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid address id")
		return
	}
	if err := h.ss(c).DeleteAddress(distributorID, addressID); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *DistributorHandler) ListPaymentAccounts(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	list, err := h.ss(c).ListPaymentAccounts(distributorID)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *DistributorHandler) CreateReceiptAccount(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	var in dto.DistributorPaymentAccountDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).CreateReceiptAccount(distributorID, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *DistributorHandler) UpdateReceiptAccount(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	accountID, err := strconv.ParseUint(c.Param("accountId"), 10, 64)
	if err != nil || accountID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid account id")
		return
	}
	var in dto.DistributorPaymentAccountDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).UpdateReceiptAccount(distributorID, accountID, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DistributorHandler) DeleteReceiptAccount(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	accountID, err := strconv.ParseUint(c.Param("accountId"), 10, 64)
	if err != nil || accountID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid account id")
		return
	}
	if err := h.ss(c).DeleteReceiptAccount(distributorID, accountID); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *DistributorHandler) ListPaymentQRs(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	list, err := h.ss(c).ListPaymentQRs(distributorID)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *DistributorHandler) CreateReceiptQR(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	var in dto.DistributorPaymentQRDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).CreateReceiptQR(distributorID, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *DistributorHandler) UpdateReceiptQR(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	qrID, err := strconv.ParseUint(c.Param("qrId"), 10, 64)
	if err != nil || qrID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid qr id")
		return
	}
	var in dto.DistributorPaymentQRDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).UpdateReceiptQR(distributorID, qrID, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DistributorHandler) DeleteReceiptQR(c *gin.Context) {
	distributorID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid distributor id")
		return
	}
	qrID, err := strconv.ParseUint(c.Param("qrId"), 10, 64)
	if err != nil || qrID == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid qr id")
		return
	}
	if err := h.ss(c).DeleteReceiptQR(distributorID, qrID); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

type PriceHandler struct {
	svc *service.PriceService
}

func NewPriceHandler(svc *service.PriceService) *PriceHandler {
	return &PriceHandler{svc: svc}
}

func (h *PriceHandler) os(c *gin.Context) *service.PriceService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *PriceHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	skuID, _ := strconv.ParseUint(c.Query("skuId"), 10, 64)
	distributorID, _ := strconv.ParseUint(c.Query("distributorId"), 10, 64)
	list, total, err := h.os(c).List(repo.PriceListFilter{
		SkuID: skuID, DistributorID: distributorID, Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *PriceHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.os(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *PriceHandler) Create(c *gin.Context) {
	var in dto.SkuPriceDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.os(c).Create(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *PriceHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.SkuPriceDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.os(c).Update(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *PriceHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.os(c).Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *PriceHandler) SupplyOptions(c *gin.Context) {
	skuID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid sku id")
		return
	}
	dropshipOnly := c.Query("dropshipOnly") == "1" || c.Query("dropshipOnly") == "true"
	resp, err := h.os(c).SupplyOptions(skuID, dropshipOnly)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, resp)
}
