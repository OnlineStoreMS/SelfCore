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

// SumDropshipSaleAndPurchaseOnDay 今日代发销售额 / 采购金额。
// 仅统计已填采购金额（total_amount > 0）的代发单；业务日，排除草稿与取消。
func (r *DashboardRepo) SumDropshipSaleAndPurchaseOnDay(dayStart time.Time) (saleAmount, wholesaleAmount float64, err error) {
	type row struct {
		SaleAmount     float64
		WholesaleAmount float64
	}
	var out row
	q := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("fulfillment_type = ?", model.DistFulfillmentDropship).
		Where("status NOT IN ?", []string{"draft", "cancelled"}).
		Where("total_amount > 0")
	q = scopePOBusinessDay(q, &dayStart)
	err = q.Select(
		"COALESCE(SUM(sale_amount), 0) as sale_amount, COALESCE(SUM(total_amount), 0) as purchase_amount",
	).Scan(&out).Error
	return out.SaleAmount, out.WholesaleAmount, err
}

func (r *DashboardRepo) SumUnpaidAmount() (float64, error) {
	var sum float64
	err := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("pay_status IN ?", []string{"unpaid", "partial"}).
		Where("status NOT IN ?", []string{"draft", "cancelled"}).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&sum).Error
	return sum, err
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

// DailyDropshipTrend 代发按日趋势：订单量含全部有效代发；销售额/采购额/毛利仅统计 total_amount > 0。
func (r *DashboardRepo) DailyDropshipTrend(start, end time.Time) (points []dto.DashboardTrendPoint, err error) {
	start, end, err = NormalizeDashboardRange(start, end)
	if err != nil {
		return nil, err
	}
	endExclusive := end.AddDate(0, 0, 1)
	days := int(end.Sub(start).Hours()/24) + 1

	type row struct {
		Day            string
		OrderCount     int64
		SaleAmount     float64
		WholesaleAmount float64
		Profit         float64
	}
	var rows []row
	err = r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Select(`to_char(date_trunc('day', `+sqlPOBizDay+`), 'YYYY-MM-DD') as day,
			COUNT(*) as order_count,
			COALESCE(SUM(CASE WHEN total_amount > 0 THEN sale_amount ELSE 0 END), 0) as sale_amount,
			COALESCE(SUM(CASE WHEN total_amount > 0 THEN total_amount ELSE 0 END), 0) as purchase_amount,
			COALESCE(SUM(CASE WHEN total_amount > 0 THEN sale_amount - total_amount ELSE 0 END), 0) as profit`).
		Where("fulfillment_type = ?", model.DistFulfillmentDropship).
		Where("status NOT IN ?", []string{"draft", "cancelled"}).
		Where(sqlPOBizDay+" >= ? AND "+sqlPOBizDay+" < ?", start, endExclusive).
		Group("day").
		Order("day ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	byDay := make(map[string]dto.DashboardTrendPoint, len(rows))
	for _, r0 := range rows {
		byDay[r0.Day] = dto.DashboardTrendPoint{
			Date:           r0.Day,
			OrderCount:     r0.OrderCount,
			SaleAmount:     r0.SaleAmount,
			WholesaleAmount: r0.WholesaleAmount,
			Profit:         r0.Profit,
		}
	}
	points = make([]dto.DashboardTrendPoint, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		if p, ok := byDay[d]; ok {
			points = append(points, p)
		} else {
			points = append(points, dto.DashboardTrendPoint{Date: d})
		}
	}
	return points, nil
}
