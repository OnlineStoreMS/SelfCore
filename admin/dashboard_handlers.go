package admin

import (
	"selfcore/internal/pkg/authcontext"
	"selfcore/internal/pkg/httputil"
	"selfcore/internal/pkg/response"
	"selfcore/internal/service"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) ds(c *gin.Context) *service.DashboardService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *DashboardHandler) Stats(c *gin.Context) {
	stats, err := h.ds(c).Stats()
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, stats)
}

func (h *DashboardHandler) Trend(c *gin.Context) {
	data, err := h.ds(c).Trend(c.Query("startDate"), c.Query("endDate"))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, data)
}
