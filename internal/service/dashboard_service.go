package service

import (
	"fmt"
	"time"

	"selfcore/internal/dto"
	"selfcore/internal/model"
	"selfcore/internal/repo"
)

type DashboardService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewDashboardService(repos *repo.Repos) *DashboardService {
	return &DashboardService{repos: repos}
}

func (s *DashboardService) ForTenant(tenantID uint64) *DashboardService {
	return &DashboardService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *DashboardService) Stats() (*dto.DashboardStats, error) {
	r := s.repos.Dashboard.ForTenant(s.tenantID)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	week := today.AddDate(0, 0, -6)
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	year := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	inProgressStatuses := []string{
		model.DistStatusPaid,
		model.DistStatusPartialShipped,
		model.DistStatusShipped,
		model.DistStatusPartialReceived,
	}
	shippedStatuses := []string{
		model.DistStatusPartialShipped,
		model.DistStatusShipped,
	}
	selfWaitShipStatuses := []string{
		model.SelfOrderStatusOrdered,
		model.SelfOrderStatusPaid,
		model.SelfOrderStatusConfirmed,
	}
	distWaitShipStatuses := []string{
		model.DistStatusConfirmed,
		model.DistStatusPaid,
	}

	out := &dto.DashboardStats{}

	var err error
	// 工作场景 · 今日：第一行自营 / 第二行分销
	if out.Workbench.SelfOrderPO, err = r.CountSelfOrdersSince(&today, true); err != nil {
		return nil, err
	}
	if out.Workbench.SelfUnpaidPO, err = r.CountUnpaidSelfOrdersSince(&today); err != nil {
		return nil, err
	}
	if out.Workbench.SelfDraftPO, err = r.CountSelfOrdersByStatusSince(model.SelfOrderStatusDraft, &today); err != nil {
		return nil, err
	}
	if out.Workbench.SelfWaitShipPO, err = r.CountSelfOrdersByStatusesSince(selfWaitShipStatuses, &today); err != nil {
		return nil, err
	}
	if out.Workbench.DistOrderPO, err = r.CountDistOrdersOnDay(today, true); err != nil {
		return nil, err
	}
	if out.Workbench.DropshipPO, err = r.CountPOsByFulfillmentSince(model.DistFulfillmentDropship, true, &today); err != nil {
		return nil, err
	}
	if out.Workbench.WholesalePO, err = r.CountPOsByFulfillmentSince(model.DistFulfillmentWholesale, true, &today); err != nil {
		return nil, err
	}
	if out.Workbench.DraftPO, err = r.CountPOsByStatusSince(model.DistStatusDraft, &today); err != nil {
		return nil, err
	}
	if out.Workbench.ConfirmedPO, err = r.CountPOsByStatusSince(model.DistStatusConfirmed, &today); err != nil {
		return nil, err
	}
	out.Workbench.OrderedPO = out.Workbench.ConfirmedPO
	if out.Workbench.UnpaidPO, err = r.CountUnpaidPOsSince(&today); err != nil {
		return nil, err
	}
	if out.Workbench.DistWaitShipPO, err = r.CountPOsByStatusesSince(distWaitShipStatuses, &today); err != nil {
		return nil, err
	}
	if out.Workbench.InTransitPO, err = r.CountPOsByStatusesSince(shippedStatuses, &today); err != nil {
		return nil, err
	}
	if out.Workbench.PartialReceivedPO, err = r.CountPOsByStatusSince(model.DistStatusPartialReceived, &today); err != nil {
		return nil, err
	}
	if out.Workbench.ActiveOffers, err = r.CountOffers(true); err != nil {
		return nil, err
	}
	if out.Workbench.TodayDistSaleAmount, err = r.SumDistSaleAmountOnDay(today); err != nil {
		return nil, err
	}
	if out.Workbench.TodaySelfSaleAmount, err = r.SumSelfSaleAmountOnDay(today); err != nil {
		return nil, err
	}
	if out.Workbench.WeekSelfSaleAmount, err = r.SumSelfSaleAmountSince(week); err != nil {
		return nil, err
	}
	if out.Workbench.MonthSelfSaleAmount, err = r.SumSelfSaleAmountSince(month); err != nil {
		return nil, err
	}
	if out.Workbench.MonthDistSaleAmount, err = r.SumDistSaleAmountSince(month); err != nil {
		return nil, err
	}
	distCostToday, err := r.SumDistCostAmountOnDay(today)
	if err != nil {
		return nil, err
	}
	selfCostToday, err := r.SumSelfCostAmountOnDay(today)
	if err != nil {
		return nil, err
	}
	// 毛利润仅统计成本额 > 0 的订单；销售额卡片仍含成本为 0 的单
	selfSaleForProfit, selfCostForProfit, err := r.SumSelfSaleAndCostWithCostOnDay(today)
	if err != nil {
		return nil, err
	}
	distSaleForProfit, distCostForProfit, err := r.SumDistSaleAndCostWithCostOnDay(today)
	if err != nil {
		return nil, err
	}
	out.Workbench.TodayDropshipSaleAmount = selfSaleForProfit + distSaleForProfit
	out.Workbench.TodayDropshipWholesaleAmount = selfCostForProfit + distCostForProfit
	out.Workbench.TodayDropshipProfit = out.Workbench.TodayDropshipSaleAmount - out.Workbench.TodayDropshipWholesaleAmount

	if out.Distributor.Total, err = r.CountDistributors(false); err != nil {
		return nil, err
	}
	if out.Distributor.Active, err = r.CountDistributors(true); err != nil {
		return nil, err
	}
	if out.Distributor.OfferCount, err = r.CountOffers(false); err != nil {
		return nil, err
	}
	if out.Distributor.OrderedThisMonth, err = r.CountDistinctDistributorsSince(month); err != nil {
		return nil, err
	}

	if out.DistOrder.Total, err = r.CountPOs(); err != nil {
		return nil, err
	}
	if out.DistOrder.Draft, err = r.CountPOsByStatus(model.DistStatusDraft); err != nil {
		return nil, err
	}
	if out.DistOrder.InProgress, err = r.CountPOsByStatuses(inProgressStatuses); err != nil {
		return nil, err
	}
	if out.DistOrder.Completed, err = r.CountPOsByStatus(model.DistStatusCompleted); err != nil {
		return nil, err
	}
	if out.DistOrder.Cancelled, err = r.CountPOsByStatus(model.DistStatusCancelled); err != nil {
		return nil, err
	}
	if out.DistOrder.TodayCount, err = r.CountPOsSince(today, true); err != nil {
		return nil, err
	}
	if out.DistOrder.WeekCount, err = r.CountPOsSince(week, true); err != nil {
		return nil, err
	}
	if out.DistOrder.MonthCount, err = r.CountPOsSince(month, true); err != nil {
		return nil, err
	}

	out.Cost.TodayAmount = selfCostToday + distCostToday
	if out.Cost.WeekAmount, err = r.SumPOAmountSince(week); err != nil {
		return nil, err
	}
	if selfCostWeek, err2 := r.SumSelfCostAmountSince(week); err2 != nil {
		return nil, err2
	} else {
		out.Cost.WeekAmount += selfCostWeek
	}
	if out.Cost.MonthAmount, err = r.SumPOAmountSince(month); err != nil {
		return nil, err
	}
	if selfCostMonth, err2 := r.SumSelfCostAmountSince(month); err2 != nil {
		return nil, err2
	} else {
		out.Cost.MonthAmount += selfCostMonth
	}
	if out.Cost.YearAmount, err = r.SumPOAmountSince(year); err != nil {
		return nil, err
	}
	if selfCostYear, err2 := r.SumSelfCostAmountSince(year); err2 != nil {
		return nil, err2
	} else {
		out.Cost.YearAmount += selfCostYear
	}
	if out.Cost.UnpaidAmount, err = r.SumUnpaidAmount(); err != nil {
		return nil, err
	}

	statusRows, err := r.GroupPOByStatus()
	if err != nil {
		return nil, err
	}
	out.StatusBreakdown = make([]dto.DashboardStatusCount, 0, len(statusRows))
	for _, row := range statusRows {
		out.StatusBreakdown = append(out.StatusBreakdown, dto.DashboardStatusCount{
			Status: row.Status,
			Count:  row.Count,
		})
	}

	rankRows, err := r.TopDistributorsSince(month, 8)
	if err != nil {
		return nil, err
	}
	sr := s.repos.Distributor.ForTenant(s.tenantID)
	out.TopDistributors = make([]dto.DashboardDistributorRank, 0, len(rankRows))
	for _, row := range rankRows {
		item := dto.DashboardDistributorRank{
			DistributorID:  row.DistributorID,
			OrderCount:  row.OrderCount,
			TotalAmount: row.TotalAmount,
		}
		if sup, err := sr.GetByID(row.DistributorID); err == nil {
			item.DistributorName = sup.Name
		}
		out.TopDistributors = append(out.TopDistributors, item)
	}

	recent, err := r.RecentPOs(8)
	if err != nil {
		return nil, err
	}
	out.RecentOrders = make([]dto.DistOrderListItem, 0, len(recent))
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	for _, po := range recent {
		item := dto.DistOrderListItem{
			ID: po.ID, DistNo: po.DistNo, DistributorID: po.DistributorID,
			Status: po.Status, PayStatus: po.PayStatus,
			FulfillmentType: po.FulfillmentType,
			TotalAmount: po.TotalAmount, Currency: po.Currency,
			RefSoID: po.RefSoID, RefTraceID: po.RefTraceID,
			CreatedAt: formatTime(po.CreatedAt),
		}
		if po.OrderedAt != nil {
			item.OrderedAt = formatTimePtr(po.OrderedAt)
		}
		if sup, err := sr.GetByID(po.DistributorID); err == nil {
			item.DistributorName = sup.Name
		}
		if n, err := pr.CountItems(po.ID); err == nil {
			item.ItemCount = int(n)
		}
		out.RecentOrders = append(out.RecentOrders, item)
	}

	return out, nil
}

func (s *DashboardService) Trend(startDate, endDate string) (*dto.DashboardTrend, error) {
	r := s.repos.Dashboard.ForTenant(s.tenantID)
	var start, end time.Time
	var err error
	if startDate != "" {
		start, err = time.ParseInLocation("2006-01-02", startDate, time.Local)
		if err != nil {
			return nil, fmt.Errorf("%w: startDate 格式应为 YYYY-MM-DD", ErrBadRequest)
		}
	}
	if endDate != "" {
		end, err = time.ParseInLocation("2006-01-02", endDate, time.Local)
		if err != nil {
			return nil, fmt.Errorf("%w: endDate 格式应为 YYYY-MM-DD", ErrBadRequest)
		}
	}
	start, end, err = repo.NormalizeDashboardRange(start, end)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	points, err := r.DailyAllTypesTrend(start, end)
	if err != nil {
		return nil, err
	}
	out := &dto.DashboardTrend{
		StartDate: start.Format("2006-01-02"),
		EndDate:   end.Format("2006-01-02"),
		Points:    points,
	}
	for _, p := range points {
		out.OrderCount += p.OrderCount
		out.SaleAmount += p.SaleAmount
		out.WholesaleAmount += p.WholesaleAmount
		out.Profit += p.Profit
	}
	return out, nil
}
