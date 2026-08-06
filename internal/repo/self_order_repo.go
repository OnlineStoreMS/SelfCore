package repo

import (
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
	RefSoID        uint64
	Keyword        string
	ShipStatus     string // wait_ship | shipped（按 status 推导）
	OrderedAtStart *time.Time
	OrderedAtEnd   *time.Time
	ShippedAtStart *time.Time
	ShippedAtEnd   *time.Time
	Page           int
	PageSize       int
}

func (r *SelfOrderRepo) List(f SelfOrderListFilter) ([]model.SelfOrder, int64, error) {
	q := r.db.Model(&model.SelfOrder{}).Scopes(scopeTenant(r.tenantID))
	if len(f.Statuses) > 0 {
		q = q.Where("status IN ?", f.Statuses)
	} else if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if len(f.ExcludeStatuses) > 0 {
		q = q.Where("status NOT IN ?", f.ExcludeStatuses)
	}
	switch f.ShipStatus {
	case "wait_ship":
		q = q.Where("status = ?", model.SelfOrderStatusConfirmed)
	case "shipped":
		q = q.Where("status IN ?", []string{
			model.SelfOrderStatusPartialShipped,
			model.SelfOrderStatusShipped,
			model.SelfOrderStatusCompleted,
		})
	}
	if f.RefSoID > 0 {
		q = q.Where("ref_so_id = ?", f.RefSoID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("so_no ILIKE ? OR ref_trace_id ILIKE ?", like, like)
	}
	if f.OrderedAtStart != nil {
		q = q.Where("COALESCE(ordered_at, created_at) >= ?", *f.OrderedAtStart)
	}
	if f.OrderedAtEnd != nil {
		q = q.Where("COALESCE(ordered_at, created_at) <= ?", *f.OrderedAtEnd)
	}
	if f.ShippedAtStart != nil {
		q = q.Where("shipped_at >= ?", *f.ShippedAtStart)
	}
	if f.ShippedAtEnd != nil {
		q = q.Where("shipped_at <= ?", *f.ShippedAtEnd)
	}
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
	err := q.Order("COALESCE(ordered_at, created_at) DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
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
