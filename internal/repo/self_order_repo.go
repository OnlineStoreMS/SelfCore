package repo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"selfcore/internal/model"

	"gorm.io/gorm"
)

type SelfOrderRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewSelfOrderRepo(db *gorm.DB) *SelfOrderRepo {
	return &SelfOrderRepo{db: db}
}

func (r *SelfOrderRepo) ForTenant(tenantID uint64) *SelfOrderRepo {
	return &SelfOrderRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

type SelfOrderListFilter struct {
	Status         string
	Statuses       []string
	ExcludeStatuses []string
	PayStatuses     []string
	RefSoID        uint64
	Keyword        string
	ShipStatus     string // wait_ship | partial_shipped | shipped
	// CreatedAt* 按创建自营单时间（≈分配时间）筛选
	CreatedAtStart *time.Time
	CreatedAtEnd   *time.Time
	// OrderedAt* 按销售单下单时间筛选
	OrderedAtStart *time.Time
	OrderedAtEnd   *time.Time
	ShippedAtStart *time.Time
	ShippedAtEnd   *time.Time
	Page           int
	PageSize       int
}

// applySelfOrderContextFilters 时间/关键词等上下文条件（不含 status / payStatus / shipStatus）。
func (r *SelfOrderRepo) applySelfOrderContextFilters(q *gorm.DB, f SelfOrderListFilter) *gorm.DB {
	if f.RefSoID > 0 {
		q = q.Where("ref_so_id = ?", f.RefSoID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"so_no ILIKE ? OR ref_trace_id ILIKE ? OR buyer_name ILIKE ? OR buyer_phone ILIKE ?",
			like, like, like, like,
		)
	}
	if f.CreatedAtStart != nil {
		q = q.Where("created_at >= ?", *f.CreatedAtStart)
	}
	if f.CreatedAtEnd != nil {
		q = q.Where("created_at <= ?", *f.CreatedAtEnd)
	}
	if f.OrderedAtStart != nil {
		q = q.Where("ordered_at >= ?", *f.OrderedAtStart)
	}
	if f.OrderedAtEnd != nil {
		q = q.Where("ordered_at <= ?", *f.OrderedAtEnd)
	}
	if f.ShippedAtStart != nil {
		q = q.Where("shipped_at >= ?", *f.ShippedAtStart)
	}
	if f.ShippedAtEnd != nil {
		q = q.Where("shipped_at <= ?", *f.ShippedAtEnd)
	}
	return q
}

func (r *SelfOrderRepo) applySelfOrderStatusFilters(q *gorm.DB, f SelfOrderListFilter) *gorm.DB {
	if len(f.Statuses) > 0 {
		q = q.Where("status IN ?", f.Statuses)
	} else if f.Status != "" {
		// 单据「已下单」= 未完成且未取消（含草稿/付款中/发货中）
		if f.Status == model.SelfOrderStatusOrdered {
			q = q.Where("status NOT IN ?", []string{
				model.SelfOrderStatusCompleted,
				model.SelfOrderStatusCancelled,
			})
		} else {
			q = q.Where("status = ?", f.Status)
		}
	}
	if len(f.ExcludeStatuses) > 0 {
		q = q.Where("status NOT IN ?", f.ExcludeStatuses)
	}
	if len(f.PayStatuses) > 0 {
		q = q.Where("pay_status IN ?", f.PayStatuses)
	}
	switch f.ShipStatus {
	case "wait_ship":
		// 待发货：仍有待发（含未开始发货 + 部分发货）；「部分发货」可单独再筛
		q = q.Where("status IN ?", []string{
			model.SelfOrderStatusDraft,
			model.SelfOrderStatusOrdered,
			model.SelfOrderStatusPaid,
			model.SelfOrderStatusConfirmed,
			model.SelfOrderStatusPartialShipped,
		})
	case "partial_shipped":
		q = q.Where("status = ?", model.SelfOrderStatusPartialShipped)
	case "shipped":
		// 已发货：全部发出（含已完成归档）
		q = q.Where("status IN ?", []string{
			model.SelfOrderStatusShipped,
			model.SelfOrderStatusCompleted,
		})
	}
	return q
}

func (r *SelfOrderRepo) List(f SelfOrderListFilter) ([]model.SelfOrder, int64, error) {
	q := r.db.Model(&model.SelfOrder{}).Scopes(scopeTenant(r.tenantID))
	q = r.applySelfOrderContextFilters(q, f)
	q = r.applySelfOrderStatusFilters(q, f)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := f.Page, f.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var list []model.SelfOrder
	err := q.Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// SelfOrderStatusCounts 列表页状态筛选数量（与时间/关键词上下文一致，不含当前 status 筛选）。
type SelfOrderStatusCounts struct {
	All      int64            `json:"all"`
	ByStatus map[string]int64 `json:"byStatus"`
	WaitShip int64            `json:"waitShip"`
	Unpaid   int64            `json:"unpaid"`
}

func (r *SelfOrderRepo) CountStatusFacets(f SelfOrderListFilter) (*SelfOrderStatusCounts, error) {
	newBase := func() *gorm.DB {
		return r.applySelfOrderContextFilters(
			r.db.Model(&model.SelfOrder{}).Scopes(scopeTenant(r.tenantID)),
			f,
		)
	}

	out := &SelfOrderStatusCounts{ByStatus: map[string]int64{}}
	if err := newBase().Count(&out.All).Error; err != nil {
		return nil, err
	}

	type statusRow struct {
		Status string
		Cnt    int64
	}
	var rows []statusRow
	if err := newBase().
		Select("status, count(*) as cnt").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out.ByStatus[row.Status] = row.Cnt
		switch row.Status {
		case model.SelfOrderStatusDraft, model.SelfOrderStatusOrdered, model.SelfOrderStatusPaid,
			model.SelfOrderStatusConfirmed, model.SelfOrderStatusPartialShipped:
			out.WaitShip += row.Cnt
		}
	}

	if err := newBase().
		Where("pay_status IN ?", []string{model.DistPayStatusUnpaid, model.DistPayStatusPartial}).
		Where("status NOT IN ?", []string{model.SelfOrderStatusDraft, model.SelfOrderStatusCancelled}).
		Count(&out.Unpaid).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SelfOrderRepo) GetByID(id uint64) (*model.SelfOrder, error) {
	var o model.SelfOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&o, id).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *SelfOrderRepo) GetWithItems(id uint64) (*model.SelfOrder, error) {
	var o model.SelfOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).Preload("Items").First(&o, id).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *SelfOrderRepo) FindActiveByRefSoID(refSoID uint64) (*model.SelfOrder, error) {
	if refSoID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var o model.SelfOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("ref_so_id = ? AND status <> ?", refSoID, model.SelfOrderStatusCancelled).
		Order("id DESC").
		First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *SelfOrderRepo) CountItems(selfOrderID uint64) (int64, error) {
	var n int64
	err := r.db.Model(&model.SelfOrderItem{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ?", selfOrderID).
		Count(&n).Error
	return n, err
}

// ItemSpecsBySelfOrderIDs 批量汇总明细规格（空规格跳过），同规格累加数量，输出如「规格 x2」。
func (r *SelfOrderRepo) ItemSpecsBySelfOrderIDs(ids []uint64) (map[uint64][]string, error) {
	out := make(map[uint64][]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		SelfOrderID uint64 `gorm:"column:self_order_id"`
		SkuSpecs    string `gorm:"column:sku_specs"`
		Qty         int    `gorm:"column:qty"`
	}
	var rows []row
	err := r.db.Model(&model.SelfOrderItem{}).
		Scopes(scopeTenant(r.tenantID)).
		Select("self_order_id, sku_specs, qty").
		Where("self_order_id IN ? AND sku_specs <> ''", ids).
		Order("self_order_id ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	order := make(map[uint64][]string, len(ids))
	qtyBy := make(map[uint64]map[string]int, len(ids))
	for _, row := range rows {
		spec := strings.TrimSpace(row.SkuSpecs)
		if spec == "" {
			continue
		}
		q := row.Qty
		if q <= 0 {
			q = 1
		}
		if _, ok := qtyBy[row.SelfOrderID]; !ok {
			qtyBy[row.SelfOrderID] = map[string]int{}
		}
		if _, ok := qtyBy[row.SelfOrderID][spec]; !ok {
			order[row.SelfOrderID] = append(order[row.SelfOrderID], spec)
		}
		qtyBy[row.SelfOrderID][spec] += q
	}
	for id, specs := range order {
		lines := make([]string, 0, len(specs))
		for _, spec := range specs {
			q := qtyBy[id][spec]
			if q > 1 {
				lines = append(lines, fmt.Sprintf("%s x%d", spec, q))
			} else {
				lines = append(lines, spec)
			}
		}
		out[id] = lines
	}
	return out, nil
}

func (r *SelfOrderRepo) Create(o *model.SelfOrder, items []model.SelfOrderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		o.TenantID = r.tenantID
		if err := tx.Create(o).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].SelfOrderID = o.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SelfOrderRepo) Save(o *model.SelfOrder) error {
	return r.db.Save(o).Error
}

func (r *SelfOrderRepo) SaveItem(it *model.SelfOrderItem) error {
	return r.db.Save(it).Error
}

func (r *SelfOrderRepo) GetItem(id uint64) (*model.SelfOrderItem, error) {
	var it model.SelfOrderItem
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&it, id).Error
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *SelfOrderRepo) NextSoNo() (string, error) {
	prefix := "SO" + time.Now().Format("20060102")
	var last string
	err := r.db.Model(&model.SelfOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("so_no LIKE ?", prefix+"%").
		Order("so_no DESC").
		Limit(1).
		Pluck("so_no", &last).Error
	if err != nil {
		return "", err
	}
	seq := 1
	if last != "" && len(last) > len(prefix) {
		var n int
		if _, scanErr := fmt.Sscanf(last[len(prefix):], "%d", &n); scanErr == nil && n >= 0 {
			seq = n + 1
		}
	}
	return fmt.Sprintf("%s%04d", prefix, seq), nil
}

func (r *SelfOrderRepo) NextShipmentNo() (string, error) {
	prefix := "SS" + time.Now().Format("20060102")
	var last string
	err := r.db.Model(&model.SelfShipment{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("shipment_no LIKE ?", prefix+"%").
		Order("shipment_no DESC").
		Limit(1).
		Pluck("shipment_no", &last).Error
	if err != nil {
		return "", err
	}
	seq := 1
	if last != "" && len(last) > len(prefix) {
		var n int
		if _, scanErr := fmt.Sscanf(last[len(prefix):], "%d", &n); scanErr == nil && n >= 0 {
			seq = n + 1
		}
	}
	return fmt.Sprintf("%s%04d", prefix, seq), nil
}

func (r *SelfOrderRepo) CreateShipment(sh *model.SelfShipment, items []model.SelfShipmentItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		sh.TenantID = r.tenantID
		if err := tx.Create(sh).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].ShipmentID = sh.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SelfOrderRepo) ListShipments(selfOrderID uint64) ([]model.SelfShipment, error) {
	var list []model.SelfShipment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ?", selfOrderID).
		Preload("Items").
		Order("id DESC").
		Find(&list).Error
	return list, err
}

func (r *SelfOrderRepo) SaveShipment(sh *model.SelfShipment) error {
	return r.db.Save(sh).Error
}

func (r *SelfOrderRepo) GetShipment(id uint64) (*model.SelfShipment, error) {
	var sh model.SelfShipment
	err := r.db.Scopes(scopeTenant(r.tenantID)).Preload("Items").First(&sh, id).Error
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

func (r *SelfOrderRepo) GetShipmentByID(selfOrderID, id uint64) (*model.SelfShipment, error) {
	var sh model.SelfShipment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ? AND id = ?", selfOrderID, id).
		Preload("Items").
		First(&sh).Error
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

func (r *SelfOrderRepo) FindShipmentByTrackingNo(selfOrderID uint64, trackingNo string) (*model.SelfShipment, error) {
	trackingNo = strings.TrimSpace(trackingNo)
	if trackingNo == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var sh model.SelfShipment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ? AND tracking_no = ?", selfOrderID, trackingNo).
		Preload("Items").
		First(&sh).Error
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

func (r *SelfOrderRepo) DeleteShipment(selfOrderID, shipmentID uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Scopes(scopeTenant(r.tenantID)).
			Where("shipment_id = ?", shipmentID).
			Delete(&model.SelfShipmentItem{}).Error; err != nil {
			return err
		}
		res := tx.Scopes(scopeTenant(r.tenantID)).
			Where("self_order_id = ? AND id = ?", selfOrderID, shipmentID).
			Delete(&model.SelfShipment{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *SelfOrderRepo) SumShippedQtyByItem(selfOrderID uint64) (map[uint64]int, error) {
	out := map[uint64]int{}
	type row struct {
		SelfOrderItemID uint64 `gorm:"column:self_order_item_id"`
		Total           int    `gorm:"column:total"`
	}
	var rows []row
	err := r.db.Table("self_shipment_items").
		Select("self_shipment_items.self_order_item_id, SUM(self_shipment_items.qty) as total").
		Joins("JOIN self_shipments ON self_shipments.id = self_shipment_items.shipment_id").
		Where("self_shipments.self_order_id = ? AND self_shipments.tenant_id = ?", selfOrderID, r.tenantID).
		Group("self_shipment_items.self_order_item_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.SelfOrderItemID] = row.Total
	}
	return out, nil
}

func (r *SelfOrderRepo) ListPayments(selfOrderID uint64) ([]model.SelfPayment, error) {
	var list []model.SelfPayment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ?", selfOrderID).
		Order("id DESC").
		Find(&list).Error
	return list, err
}

func (r *SelfOrderRepo) GetPayment(selfOrderID, id uint64) (*model.SelfPayment, error) {
	var p model.SelfPayment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ? AND id = ?", selfOrderID, id).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *SelfOrderRepo) CreatePayment(p *model.SelfPayment) error {
	p.TenantID = r.tenantID
	return r.db.Create(p).Error
}

func (r *SelfOrderRepo) SavePayment(p *model.SelfPayment) error {
	return r.db.Save(p).Error
}

func (r *SelfOrderRepo) DeletePayment(selfOrderID, id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ? AND id = ?", selfOrderID, id).
		Delete(&model.SelfPayment{}).Error
}

func (r *SelfOrderRepo) SumPaidPayments(selfOrderID uint64) (float64, error) {
	var sum float64
	err := r.db.Model(&model.SelfPayment{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ? AND pay_status = ?", selfOrderID, model.DistPayStatusPaid).
		Select("COALESCE(SUM(pay_amount), 0)").
		Scan(&sum).Error
	return sum, err
}

func (r *SelfOrderRepo) EarliestPaidAt(selfOrderID uint64) (*time.Time, error) {
	var paidAt sql.NullTime
	err := r.db.Model(&model.SelfPayment{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ? AND pay_status = ? AND paid_at IS NOT NULL", selfOrderID, model.DistPayStatusPaid).
		Select("MIN(paid_at)").
		Scan(&paidAt).Error
	if err != nil {
		return nil, err
	}
	if !paidAt.Valid {
		return nil, nil
	}
	t := paidAt.Time
	return &t, nil
}

func (r *SelfOrderRepo) ListAttachments(selfOrderID uint64) ([]model.SelfAttachment, error) {
	var list []model.SelfAttachment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ?", selfOrderID).
		Order("id DESC").
		Find(&list).Error
	return list, err
}

func (r *SelfOrderRepo) GetAttachment(selfOrderID, id uint64) (*model.SelfAttachment, error) {
	var a model.SelfAttachment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ? AND id = ?", selfOrderID, id).
		First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *SelfOrderRepo) CreateAttachment(a *model.SelfAttachment) error {
	a.TenantID = r.tenantID
	return r.db.Create(a).Error
}

func (r *SelfOrderRepo) DeleteAttachment(selfOrderID, id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ? AND id = ?", selfOrderID, id).
		Delete(&model.SelfAttachment{}).Error
}

func (r *SelfOrderRepo) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var shipmentIDs []uint64
		if err := tx.Model(&model.SelfShipment{}).
			Scopes(scopeTenant(r.tenantID)).
			Where("self_order_id = ?", id).
			Pluck("id", &shipmentIDs).Error; err != nil {
			return err
		}
		if len(shipmentIDs) > 0 {
			if err := tx.Scopes(scopeTenant(r.tenantID)).
				Where("shipment_id IN ?", shipmentIDs).
				Delete(&model.SelfShipmentItem{}).Error; err != nil {
				return err
			}
		}
		for _, m := range []any{
			&model.SelfShipment{},
			&model.SelfPayment{},
			&model.SelfAttachment{},
		} {
			if err := tx.Scopes(scopeTenant(r.tenantID)).
				Where("self_order_id = ?", id).
				Delete(m).Error; err != nil {
				return err
			}
		}
		if err := tx.Scopes(scopeTenant(r.tenantID)).
			Where("self_order_id = ?", id).
			Delete(&model.SelfOrderItem{}).Error; err != nil {
			return err
		}
		return tx.Scopes(scopeTenant(r.tenantID)).Delete(&model.SelfOrder{}, id).Error
	})
}

func (r *SelfOrderRepo) FindLatestShipment(selfOrderID uint64) (*model.SelfShipment, error) {
	var sh model.SelfShipment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("self_order_id = ?", selfOrderID).
		Order("id DESC").
		First(&sh).Error
	if err != nil {
		return nil, err
	}
	return &sh, nil
}
