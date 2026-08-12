package repo

import (
	"fmt"
	"time"

	"selfcore/internal/dto"
	"selfcore/internal/model"

	"gorm.io/gorm"
)

type DashboardRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewDashboardRepo(db *gorm.DB) *DashboardRepo {
	return &DashboardRepo{db: db}
}

func (r *DashboardRepo) ForTenant(tenantID uint64) *DashboardRepo {
	return &DashboardRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *DashboardRepo) CountDistributors(activeOnly bool) (int64, error) {
	q := r.db.Model(&model.Distributor{}).Scopes(scopeTenant(r.tenantID))
	if activeOnly {
		q = q.Where("status = ?", 1)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountOffers(activeOnly bool) (int64, error) {
	q := r.db.Model(&model.SkuDistributorPrice{}).Scopes(scopeTenant(r.tenantID))
	if activeOnly {
		q = q.Where("status = ?", 1)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountSelfOrdersSince(dayStart *time.Time, excludeCancelled bool) (int64, error) {
	q := r.db.Model(&model.SelfOrder{}).Scopes(scopeTenant(r.tenantID))
	if excludeCancelled {
		q = q.Where("status <> ?", model.SelfOrderStatusCancelled)
	}
	q = scopeSelfOrderBusinessDay(q, dayStart)
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountSelfOrdersByStatusSince(status string, dayStart *time.Time) (int64, error) {
	q := r.db.Model(&model.SelfOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status = ?", status)
	q = scopeSelfOrderBusinessDay(q, dayStart)
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountSelfOrdersByStatusesSince(statuses []string, dayStart *time.Time) (int64, error) {
	q := r.db.Model(&model.SelfOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status IN ?", statuses)
	q = scopeSelfOrderBusinessDay(q, dayStart)
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountUnpaidSelfOrdersSince(dayStart *time.Time) (int64, error) {
	q := r.db.Model(&model.SelfOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("pay_status IN ?", []string{"unpaid", "partial"}).
		Where("status NOT IN ?", []string{
			model.SelfOrderStatusDraft, model.SelfOrderStatusCancelled,
		})
	q = scopeSelfOrderBusinessDay(q, dayStart)
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// scopeSelfOrderBusinessDay 按创建自营单日：created_at ∈ [dayStart, dayStart+1)
func scopeSelfOrderBusinessDay(q *gorm.DB, dayStart *time.Time) *gorm.DB {
	if dayStart == nil {
		return q
	}
	end := dayStart.Add(24 * time.Hour)
	return q.Where("created_at >= ? AND created_at < ?", *dayStart, end)
}

func (r *DashboardRepo) CountPOsByStatus(status string) (int64, error) {
	return r.CountPOsByStatusSince(status, nil)
}

func (r *DashboardRepo) CountPOsByStatusSince(status string, dayStart *time.Time) (int64, error) {
	q := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status = ?", status)
	q = scopePOBusinessDay(q, dayStart)
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountPOsByStatuses(statuses []string) (int64, error) {
	return r.CountPOsByStatusesSince(statuses, nil)
}

func (r *DashboardRepo) CountPOsByStatusesSince(statuses []string, dayStart *time.Time) (int64, error) {
	q := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status IN ?", statuses)
	q = scopePOBusinessDay(q, dayStart)
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountPOsByFulfillment(fulfillmentType string, excludeDraftCancel bool) (int64, error) {
	return r.CountPOsByFulfillmentSince(fulfillmentType, excludeDraftCancel, nil)
}

func (r *DashboardRepo) CountPOsByFulfillmentSince(fulfillmentType string, excludeDraftCancel bool, dayStart *time.Time) (int64, error) {
	q := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("fulfillment_type = ?", fulfillmentType)
	if excludeDraftCancel {
		q = q.Where("status NOT IN ?", []string{"draft", "cancelled"})
	}
	q = scopePOBusinessDay(q, dayStart)
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountUnpaidPOs() (int64, error) {
	return r.CountUnpaidPOsSince(nil)
}

func (r *DashboardRepo) CountUnpaidPOsSince(dayStart *time.Time) (int64, error) {
	q := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("pay_status IN ?", []string{"unpaid", "partial"}).
		Where("status NOT IN ?", []string{"draft", "cancelled"})
	q = scopePOBusinessDay(q, dayStart)
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// scopePOBusinessDay 按业务日筛选：COALESCE(ordered_at, created_at) 落在 [dayStart, dayStart+1)。
func scopePOBusinessDay(q *gorm.DB, dayStart *time.Time) *gorm.DB {
	if dayStart == nil {
		return q
	}
	dayEnd := dayStart.AddDate(0, 0, 1)
	return q.Where("COALESCE(ordered_at, created_at) >= ? AND COALESCE(ordered_at, created_at) < ?", *dayStart, dayEnd)
}

func (r *DashboardRepo) CountPOs() (int64, error) {
	var n int64
	err := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountPOsSince(since time.Time, excludeDraftCancel bool) (int64, error) {
	q := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("COALESCE(ordered_at, created_at) >= ?", since)
	if excludeDraftCancel {
		q = q.Where("status NOT IN ?", []string{"draft", "cancelled"})
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// CountDistOrdersOnDay 今日分销单（全类型）；excludeCancelled 时排除已取消。
func (r *DashboardRepo) CountDistOrdersOnDay(dayStart time.Time, excludeCancelled bool) (int64, error) {
	q := r.db.Model(&model.DistOrder{}).Scopes(scopeTenant(r.tenantID))
	if excludeCancelled {
		q = q.Where("status <> ?", model.DistStatusCancelled)
	}
	q = scopePOBusinessDay(q, &dayStart)
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *DashboardRepo) SumPOAmountSince(since time.Time) (float64, error) {
	var sum float64
	err := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status NOT IN ?", []string{"draft", "cancelled"}).
		Where("COALESCE(ordered_at, created_at) >= ?", since).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&sum).Error
	return sum, err
}

// SumDistCostAmountOnDay 业务日分销成本额（全部分销类型 total_amount）。
func (r *DashboardRepo) SumDistCostAmountOnDay(dayStart time.Time) (float64, error) {
	var sum float64
	q := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status NOT IN ?", []string{"draft", "cancelled"})
	q = scopePOBusinessDay(q, &dayStart)
	err := q.Select("COALESCE(SUM(total_amount), 0)").Scan(&sum).Error
	return sum, err
}

// SumSelfCostAmountOnDay 业务日自营成本额。
func (r *DashboardRepo) SumSelfCostAmountOnDay(dayStart time.Time) (float64, error) {
	var sum float64
	end := dayStart.Add(24 * time.Hour)
	err := r.db.Model(&model.SelfOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status <> ?", model.SelfOrderStatusCancelled).
		Where("created_at >= ? AND created_at < ?", dayStart, end).
		Select("COALESCE(SUM(cost_amount), 0)").
		Scan(&sum).Error
	return sum, err
}

// SumSelfCostAmountSince 自 since 起自营成本额累计。
func (r *DashboardRepo) SumSelfCostAmountSince(since time.Time) (float64, error) {
	var sum float64
	err := r.db.Model(&model.SelfOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status <> ?", model.SelfOrderStatusCancelled).
		Where("created_at >= ?", since).
		Select("COALESCE(SUM(cost_amount), 0)").
		Scan(&sum).Error
	return sum, err
}

const sqlDistSaleExpr = `CASE WHEN fulfillment_type = ? THEN sale_amount ELSE total_amount END`

// SumDistSaleAmountOnDay 今日全部分销类型销售额：直发按 sale_amount，批发按 total_amount。
func (r *DashboardRepo) SumDistSaleAmountOnDay(dayStart time.Time) (float64, error) {
	return r.sumDistSaleAmount(func(q *gorm.DB) *gorm.DB {
		return scopePOBusinessDay(q, &dayStart)
	})
}

// SumDistSaleAmountSince 自 since 起全部分销销售额累计。
func (r *DashboardRepo) SumDistSaleAmountSince(since time.Time) (float64, error) {
	return r.sumDistSaleAmount(func(q *gorm.DB) *gorm.DB {
		return q.Where("COALESCE(ordered_at, created_at) >= ?", since)
	})
}

func (r *DashboardRepo) sumDistSaleAmount(scopeFn func(*gorm.DB) *gorm.DB) (float64, error) {
	var sum float64
	q := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status NOT IN ?", []string{"draft", "cancelled"})
	q = scopeFn(q)
	err := q.Select(`COALESCE(SUM(`+sqlDistSaleExpr+`), 0)`, model.DistFulfillmentDropship).Scan(&sum).Error
	return sum, err
}

// SumSelfSaleAmountOnDay 今日自营销售额。
func (r *DashboardRepo) SumSelfSaleAmountOnDay(dayStart time.Time) (float64, error) {
	var sum float64
	end := dayStart.Add(24 * time.Hour)
	err := r.db.Model(&model.SelfOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status <> ?", model.SelfOrderStatusCancelled).
		Where("created_at >= ? AND created_at < ?", dayStart, end).
		Select("COALESCE(SUM(sale_amount), 0)").
		Scan(&sum).Error
	return sum, err
}

// SumSelfSaleAmountSince 自 since 起自营销售额累计。
func (r *DashboardRepo) SumSelfSaleAmountSince(since time.Time) (float64, error) {
	var sum float64
	err := r.db.Model(&model.SelfOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status <> ?", model.SelfOrderStatusCancelled).
		Where("created_at >= ?", since).
		Select("COALESCE(SUM(sale_amount), 0)").
		Scan(&sum).Error
	return sum, err
}

// SumSelfSaleAndCostWithCostOnDay 业务日自营：仅成本额 > 0 的订单销售额与成本（用于毛利）。
func (r *DashboardRepo) SumSelfSaleAndCostWithCostOnDay(dayStart time.Time) (saleAmount, costAmount float64, err error) {
	type row struct {
		SaleAmount float64
		CostAmount float64
	}
	var out row
	end := dayStart.Add(24 * time.Hour)
	err = r.db.Model(&model.SelfOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status <> ?", model.SelfOrderStatusCancelled).
		Where("cost_amount > 0").
		Where("created_at >= ? AND created_at < ?", dayStart, end).
		Select("COALESCE(SUM(sale_amount), 0) as sale_amount, COALESCE(SUM(cost_amount), 0) as cost_amount").
		Scan(&out).Error
	return out.SaleAmount, out.CostAmount, err
}

// SumDistSaleAndCostWithCostOnDay 业务日分销：仅成本额(total_amount) > 0 的订单销售额与成本（用于毛利）。
func (r *DashboardRepo) SumDistSaleAndCostWithCostOnDay(dayStart time.Time) (saleAmount, costAmount float64, err error) {
	type row struct {
		SaleAmount float64
		CostAmount float64
	}
	var out row
	q := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status NOT IN ?", []string{"draft", "cancelled"}).
		Where("total_amount > 0")
	q = scopePOBusinessDay(q, &dayStart)
	err = q.Select(
		`COALESCE(SUM(`+sqlDistSaleExpr+`), 0) as sale_amount, COALESCE(SUM(total_amount), 0) as cost_amount`,
		model.DistFulfillmentDropship,
	).Scan(&out).Error
	return out.SaleAmount, out.CostAmount, err
}

// SumUnpaidAmount 自营 + 分销待收余额合计：订单金额 − 已付（部分付款只计差额），排除草稿与取消。
func (r *DashboardRepo) SumUnpaidAmount() (float64, error) {
	var distSum float64
	err := r.db.Raw(`
		SELECT COALESCE(SUM(GREATEST(d.total_amount - COALESCE(p.paid, 0), 0)), 0)
		FROM dist_orders d
		LEFT JOIN (
			SELECT dist_order_id, SUM(pay_amount) AS paid
			FROM dist_receipts
			WHERE pay_status = ? AND tenant_id = ?
			GROUP BY dist_order_id
		) p ON p.dist_order_id = d.id
		WHERE d.tenant_id = ?
		  AND d.pay_status IN (?, ?)
		  AND d.status NOT IN (?, ?)
	`, model.DistPayStatusPaid, r.tenantID,
		r.tenantID,
		model.DistPayStatusUnpaid, model.DistPayStatusPartial,
		model.DistStatusDraft, model.DistStatusCancelled,
	).Scan(&distSum).Error
	if err != nil {
		return 0, err
	}
	var selfSum float64
	err = r.db.Raw(`
		SELECT COALESCE(SUM(GREATEST(s.sale_amount - COALESCE(p.paid, 0), 0)), 0)
		FROM self_orders s
		LEFT JOIN (
			SELECT self_order_id, SUM(pay_amount) AS paid
			FROM self_payments
			WHERE pay_status = ? AND tenant_id = ?
			GROUP BY self_order_id
		) p ON p.self_order_id = s.id
		WHERE s.tenant_id = ?
		  AND s.pay_status IN (?, ?)
		  AND s.status NOT IN (?, ?)
	`, model.DistPayStatusPaid, r.tenantID,
		r.tenantID,
		model.DistPayStatusUnpaid, model.DistPayStatusPartial,
		model.SelfOrderStatusDraft, model.SelfOrderStatusCancelled,
	).Scan(&selfSum).Error
	if err != nil {
		return 0, err
	}
	return distSum + selfSum, nil
}

type StatusCountRow struct {
	Status string
	Count  int64
}

func (r *DashboardRepo) GroupPOByStatus() ([]StatusCountRow, error) {
	var rows []StatusCountRow
	err := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&rows).Error
	return rows, err
}

type DistributorRankRow struct {
	DistributorID  uint64
	OrderCount  int64
	TotalAmount float64
}

func (r *DashboardRepo) TopDistributorsSince(since time.Time, limit int) ([]DistributorRankRow, error) {
	var rows []DistributorRankRow
	err := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status NOT IN ?", []string{"draft", "cancelled"}).
		Where("COALESCE(ordered_at, created_at) >= ?", since).
		Select("distributor_id, COUNT(*) as order_count, COALESCE(SUM(total_amount), 0) as total_amount").
		Group("distributor_id").
		Order("total_amount DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *DashboardRepo) CountDistinctDistributorsSince(since time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status NOT IN ?", []string{"draft", "cancelled"}).
		Where("COALESCE(ordered_at, created_at) >= ?", since).
		Select("COUNT(DISTINCT distributor_id)").
		Scan(&n).Error
	return n, err
}

func (r *DashboardRepo) RecentPOs(limit int) ([]model.DistOrder, error) {
	var list []model.DistOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Order("id DESC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

// NormalizeDashboardRange 闭区间日期，默认近 7 天，最长 90 天。
func NormalizeDashboardRange(start, end time.Time) (time.Time, time.Time, error) {
	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if start.IsZero() && end.IsZero() {
		end = today
		start = today.AddDate(0, 0, -6)
	} else {
		if start.IsZero() {
			start = end.AddDate(0, 0, -6)
		}
		if end.IsZero() {
			end = today
		}
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
		end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)
	}
	if end.Before(start) {
		start, end = end, start
	}
	if end.After(today) {
		end = today
	}
	days := int(end.Sub(start).Hours()/24) + 1
	if days > 90 {
		return time.Time{}, time.Time{}, fmt.Errorf("时间范围最长 90 天")
	}
	if days < 1 {
		return time.Time{}, time.Time{}, fmt.Errorf("无效时间范围")
	}
	return start, end, nil
}

const sqlPOBizDay = `COALESCE(ordered_at, created_at)`
const sqlSelfBizDay = `created_at`

// DailyAllTypesTrend 自营+全部分销按日趋势。
// 销售额/成本额含全部订单；毛利润仅统计成本额 > 0 的订单。
func (r *DashboardRepo) DailyAllTypesTrend(start, end time.Time) (points []dto.DashboardTrendPoint, err error) {
	start, end, err = NormalizeDashboardRange(start, end)
	if err != nil {
		return nil, err
	}
	endExclusive := end.AddDate(0, 0, 1)
	days := int(end.Sub(start).Hours()/24) + 1

	type dayAgg struct {
		Day          string
		OrderCount   int64
		SaleAmount   float64
		CostAmount   float64
		ProfitAmount float64
	}

	byDay := make(map[string]*dto.DashboardTrendPoint, days)
	ensure := func(day string) *dto.DashboardTrendPoint {
		if p, ok := byDay[day]; ok {
			return p
		}
		p := &dto.DashboardTrendPoint{Date: day}
		byDay[day] = p
		return p
	}

	var selfRows []dayAgg
	err = r.db.Model(&model.SelfOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Select(`to_char(date_trunc('day', `+sqlSelfBizDay+`), 'YYYY-MM-DD') as day,
			COUNT(*) as order_count,
			COALESCE(SUM(sale_amount), 0) as sale_amount,
			COALESCE(SUM(cost_amount), 0) as cost_amount,
			COALESCE(SUM(CASE WHEN cost_amount > 0 THEN sale_amount - cost_amount ELSE 0 END), 0) as profit_amount`).
		Where("status <> ?", model.SelfOrderStatusCancelled).
		Where(sqlSelfBizDay+" >= ? AND "+sqlSelfBizDay+" < ?", start, endExclusive).
		Group("day").
		Order("day ASC").
		Scan(&selfRows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range selfRows {
		p := ensure(row.Day)
		p.SelfOrderCount = row.OrderCount
		p.SelfSaleAmount = row.SaleAmount
		p.OrderCount += row.OrderCount
		p.SaleAmount += row.SaleAmount
		p.WholesaleAmount += row.CostAmount
		p.Profit += row.ProfitAmount
	}

	var distRows []dayAgg
	err = r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Select(`to_char(date_trunc('day', `+sqlPOBizDay+`), 'YYYY-MM-DD') as day,
			COUNT(*) as order_count,
			COALESCE(SUM(`+sqlDistSaleExpr+`), 0) as sale_amount,
			COALESCE(SUM(total_amount), 0) as cost_amount,
			COALESCE(SUM(CASE WHEN total_amount > 0 THEN (`+sqlDistSaleExpr+`) - total_amount ELSE 0 END), 0) as profit_amount`,
			model.DistFulfillmentDropship, model.DistFulfillmentDropship).
		Where("status NOT IN ?", []string{"draft", "cancelled"}).
		Where(sqlPOBizDay+" >= ? AND "+sqlPOBizDay+" < ?", start, endExclusive).
		Group("day").
		Order("day ASC").
		Scan(&distRows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range distRows {
		p := ensure(row.Day)
		p.OrderCount += row.OrderCount
		p.SaleAmount += row.SaleAmount
		p.WholesaleAmount += row.CostAmount
		p.Profit += row.ProfitAmount
	}

	points = make([]dto.DashboardTrendPoint, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		if p, ok := byDay[d]; ok {
			points = append(points, *p)
		} else {
			points = append(points, dto.DashboardTrendPoint{Date: d})
		}
	}
	return points, nil
}
