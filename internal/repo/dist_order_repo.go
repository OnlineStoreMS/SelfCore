package repo

import (
	"fmt"
	"strings"
	"time"

	"selfcore/internal/model"

	"gorm.io/gorm"
)

type DistOrderRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewDistOrderRepo(db *gorm.DB) *DistOrderRepo {
	return &DistOrderRepo{db: db}
}

func (r *DistOrderRepo) ForTenant(tenantID uint64) *DistOrderRepo {
	return &DistOrderRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

type DistOrderListFilter struct {
	Status          string
	Statuses        []string // 多状态，优先于 Status
	PayStatuses      []string // unpaid|partial|paid
	ExcludeStatuses  []string
	FulfillmentType string
	DistributorID      uint64
	RefSoID         uint64
	RefTraceID      string
	Keyword         string
	CreatedAtStart  *time.Time
	CreatedAtEnd    *time.Time // 含当日：传次日 00:00 时用 < End
	OrderedAtStart  *time.Time
	OrderedAtEnd    *time.Time // 含当日：传次日 00:00 时用 < End；按业务日 COALESCE(ordered_at, created_at)
	SortBy          string // orderedAt | createdAt | id | totalAmount
	SortOrder       string // asc | desc
	Page            int
	PageSize        int
}

func (r *DistOrderRepo) List(f DistOrderListFilter) ([]model.DistOrder, int64, error) {
	q := r.db.Model(&model.DistOrder{}).Scopes(scopeTenant(r.tenantID))
	if len(f.Statuses) > 0 {
		q = q.Where("status IN ?", f.Statuses)
	} else if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if len(f.ExcludeStatuses) > 0 {
		q = q.Where("status NOT IN ?", f.ExcludeStatuses)
	}
	if len(f.PayStatuses) > 0 {
		q = q.Where("pay_status IN ?", f.PayStatuses)
	}
	if f.FulfillmentType != "" {
		q = q.Where("fulfillment_type = ?", f.FulfillmentType)
	}
	if f.DistributorID > 0 {
		q = q.Where("distributor_id = ?", f.DistributorID)
	}
	if f.RefSoID > 0 {
		q = q.Where("ref_so_id = ?", f.RefSoID)
	}
	if f.RefTraceID != "" {
		q = q.Where("ref_trace_id = ?", f.RefTraceID)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("dist_no ILIKE ?", like)
	}
	if f.CreatedAtStart != nil {
		q = q.Where("created_at >= ?", *f.CreatedAtStart)
	}
	if f.CreatedAtEnd != nil {
		q = q.Where("created_at < ?", *f.CreatedAtEnd)
	}
	// 采购时间筛选与工作台业务日一致：COALESCE(ordered_at, created_at)
	if f.OrderedAtStart != nil {
		q = q.Where("COALESCE(ordered_at, created_at) >= ?", *f.OrderedAtStart)
	}
	if f.OrderedAtEnd != nil {
		q = q.Where("COALESCE(ordered_at, created_at) < ?", *f.OrderedAtEnd)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DistOrder
	offset := (f.Page - 1) * f.PageSize
	err := q.Order(poListOrderClause(f.SortBy, f.SortOrder)).Offset(offset).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}

func poListOrderClause(sortBy, sortOrder string) string {
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(sortOrder), "asc") {
		dir = "ASC"
	}
	switch strings.TrimSpace(sortBy) {
	case "orderedAt", "ordered_at":
		// 采购时间为空时回退创建时间，保证排序稳定
		return fmt.Sprintf("COALESCE(ordered_at, created_at) %s, id %s", dir, dir)
	case "createdAt", "created_at":
		return fmt.Sprintf("created_at %s, id %s", dir, dir)
	case "totalAmount", "total_amount":
		return fmt.Sprintf("total_amount %s, id %s", dir, dir)
	case "id":
		return fmt.Sprintf("id %s", dir)
	default:
		return "COALESCE(ordered_at, created_at) DESC, id DESC"
	}
}

// ListDropshipMergeable 可结算合并的代发单：草稿/已下单且未付款，按创建时间窗口。
func (r *DistOrderRepo) ListDropshipMergeable(distributorID uint64, from, to time.Time) ([]model.DistOrder, error) {
	var list []model.DistOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ?", distributorID).
		Where("fulfillment_type = ?", model.DistFulfillmentDropship).
		Where("status IN ?", []string{model.DistStatusDraft, model.DistStatusConfirmed}).
		Where("pay_status = ?", model.DistPayStatusUnpaid).
		Where("created_at >= ? AND created_at < ?", from, to).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

func (r *DistOrderRepo) GetByID(id uint64) (*model.DistOrder, error) {
	var po model.DistOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&po, id).Error
	if err != nil {
		return nil, err
	}
	return &po, nil
}

func (r *DistOrderRepo) GetByDistNoWithItems(distNo string) (*model.DistOrder, error) {
	distNo = strings.TrimSpace(distNo)
	if distNo == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var po model.DistOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Where("dist_no = ?", distNo).
		First(&po).Error
	if err != nil {
		return nil, err
	}
	return &po, nil
}

func (r *DistOrderRepo) GetWithItems(id uint64) (*model.DistOrder, error) {
	var po model.DistOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("id ASC")
	}).First(&po, id).Error
	if err != nil {
		return nil, err
	}
	return &po, nil
}

func (r *DistOrderRepo) Create(po *model.DistOrder, items []model.DistOrderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		po.TenantID = r.tenantID
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].DistOrderID = po.ID
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		po.Items = items
		return nil
	})
}

func (r *DistOrderRepo) ReplaceItems(poID uint64, items []model.DistOrderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Scopes(scopeTenant(r.tenantID)).
			Where("dist_order_id = ?", poID).
			Delete(&model.DistOrderItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].DistOrderID = poID
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
}

func (r *DistOrderRepo) Save(po *model.DistOrder) error {
	return r.db.Save(po).Error
}

// UpdateHeaderTimes 强制更新单头创建时间 / 采购时间（合并后对齐最新单）。
func (r *DistOrderRepo) UpdateHeaderTimes(id uint64, createdAt time.Time, orderedAt *time.Time) error {
	fields := map[string]any{
		"created_at": createdAt,
		"updated_at": time.Now(),
	}
	if orderedAt != nil {
		fields["ordered_at"] = *orderedAt
	} else {
		fields["ordered_at"] = nil
	}
	return r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("id = ?", id).
		Updates(fields).Error
}

func (r *DistOrderRepo) SaveItem(item *model.DistOrderItem) error {
	return r.db.Save(item).Error
}

// ReassignPOSideData 将源单的物流/付款/附件挂到目标单（合并用，避免删源单时级联清掉）。
func (r *DistOrderRepo) ReassignPOSideData(fromDistOrderID, toDistOrderID uint64) error {
	if fromDistOrderID == 0 || toDistOrderID == 0 || fromDistOrderID == toDistOrderID {
		return nil
	}
	now := time.Now()
	for _, m := range []any{
		&model.DistShipment{},
		&model.DistReceipt{},
		&model.DistAttachment{},
	} {
		if err := r.db.Model(m).
			Scopes(scopeTenant(r.tenantID)).
			Where("dist_order_id = ?", fromDistOrderID).
			Updates(map[string]any{"dist_order_id": toDistOrderID, "updated_at": now}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *DistOrderRepo) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var shipmentIDs []uint64
		if err := tx.Model(&model.DistShipment{}).
			Scopes(scopeTenant(r.tenantID)).
			Where("dist_order_id = ?", id).
			Pluck("id", &shipmentIDs).Error; err != nil {
			return err
		}
		if len(shipmentIDs) > 0 {
			if err := tx.Scopes(scopeTenant(r.tenantID)).
				Where("shipment_id IN ?", shipmentIDs).
				Delete(&model.DistShipmentItem{}).Error; err != nil {
				return err
			}
		}
		for _, m := range []any{
			&model.DistShipment{},
			&model.DistReceipt{},
			&model.DistAttachment{},
		} {
			if err := tx.Scopes(scopeTenant(r.tenantID)).
				Where("dist_order_id = ?", id).
				Delete(m).Error; err != nil {
				return err
			}
		}
		if err := tx.Scopes(scopeTenant(r.tenantID)).
			Where("dist_order_id = ?", id).
			Delete(&model.DistOrderItem{}).Error; err != nil {
			return err
		}
		return tx.Scopes(scopeTenant(r.tenantID)).Delete(&model.DistOrder{}, id).Error
	})
}

func (r *DistOrderRepo) CountItems(poID uint64) (int64, error) {
	var n int64
	err := r.db.Model(&model.DistOrderItem{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("dist_order_id = ? AND cancelled = ?", poID, false).
		Count(&n).Error
	return n, err
}

// ItemSpecsByDistOrderIDs 批量汇总采购明细规格（跳过已撤回/空规格），同规格累加数量，输出如「规格 x2」。
func (r *DistOrderRepo) ItemSpecsByDistOrderIDs(poIDs []uint64) (map[uint64][]string, error) {
	out := make(map[uint64][]string, len(poIDs))
	if len(poIDs) == 0 {
		return out, nil
	}
	type row struct {
		DistOrderID     uint64 `gorm:"column:dist_order_id"`
		SkuSpecs string `gorm:"column:sku_specs"`
		Qty      int    `gorm:"column:qty"`
	}
	var rows []row
	err := r.db.Model(&model.DistOrderItem{}).
		Scopes(scopeTenant(r.tenantID)).
		Select("dist_order_id, sku_specs, qty").
		Where("dist_order_id IN ? AND cancelled = ? AND sku_specs <> ''", poIDs, false).
		Order("dist_order_id ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	order := make(map[uint64][]string, len(poIDs)) // first-seen order of specs
	qtyBy := make(map[uint64]map[string]int, len(poIDs))
	for _, row := range rows {
		spec := strings.TrimSpace(row.SkuSpecs)
		if spec == "" {
			continue
		}
		q := row.Qty
		if q <= 0 {
			q = 1
		}
		if _, ok := qtyBy[row.DistOrderID]; !ok {
			qtyBy[row.DistOrderID] = map[string]int{}
		}
		if _, ok := qtyBy[row.DistOrderID][spec]; !ok {
			order[row.DistOrderID] = append(order[row.DistOrderID], spec)
		}
		qtyBy[row.DistOrderID][spec] += q
	}
	for poID, specs := range order {
		lines := make([]string, 0, len(specs))
		for _, spec := range specs {
			q := qtyBy[poID][spec]
			if q > 1 {
				lines = append(lines, fmt.Sprintf("%s x%d", spec, q))
			} else {
				lines = append(lines, spec)
			}
		}
		out[poID] = lines
	}
	return out, nil
}

func (r *DistOrderRepo) NextDistNo() (string, error) {
	prefix := "PO" + time.Now().Format("20060102")
	var last string
	err := r.db.Model(&model.DistOrder{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("dist_no LIKE ?", prefix+"%").
		Order("dist_no DESC").
		Limit(1).
		Pluck("dist_no", &last).Error
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
