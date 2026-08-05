package repo

import (
	"fmt"
	"strings"
	"time"

	"selfcore/internal/model"

	"gorm.io/gorm"
)

type ShipmentRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewShipmentRepo(db *gorm.DB) *ShipmentRepo {
	return &ShipmentRepo{db: db}
}

func (r *ShipmentRepo) ForTenant(tenantID uint64) *ShipmentRepo {
	return &ShipmentRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *ShipmentRepo) ListByPO(poID uint64) ([]model.DistShipment, error) {
	var list []model.DistShipment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("dist_order_id = ?", poID).
		Preload("Items").
		Order("id DESC").
		Find(&list).Error
	return list, err
}

func (r *ShipmentRepo) GetByID(poID, id uint64) (*model.DistShipment, error) {
	var s model.DistShipment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("dist_order_id = ? AND id = ?", poID, id).
		Preload("Items").
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ShipmentRepo) FindByTrackingNo(poID uint64, trackingNo string) (*model.DistShipment, error) {
	trackingNo = strings.TrimSpace(trackingNo)
	if trackingNo == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var s model.DistShipment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("dist_order_id = ? AND tracking_no = ?", poID, trackingNo).
		Preload("Items").
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ShipmentRepo) Create(s *model.DistShipment, items []model.DistShipmentItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		s.TenantID = r.tenantID
		if err := tx.Create(s).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].ShipmentID = s.ID
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		s.Items = items
		return nil
	})
}

func (r *ShipmentRepo) Save(s *model.DistShipment) error {
	return r.db.Save(s).Error
}

func (r *ShipmentRepo) Delete(poID, id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Scopes(scopeTenant(r.tenantID)).
			Where("shipment_id = ?", id).
			Delete(&model.DistShipmentItem{}).Error; err != nil {
			return err
		}
		return tx.Scopes(scopeTenant(r.tenantID)).
			Where("dist_order_id = ? AND id = ?", poID, id).
			Delete(&model.DistShipment{}).Error
	})
}

// RemapItemDistOrderItemIDs 合并重建明细后，把物流明细的 dist_order_item_id 指到新 ID。
func (r *ShipmentRepo) RemapItemDistOrderItemIDs(oldToNew map[uint64]uint64) error {
	if len(oldToNew) == 0 {
		return nil
	}
	now := time.Now()
	for oldID, newID := range oldToNew {
		if oldID == 0 || newID == 0 || oldID == newID {
			continue
		}
		if err := r.db.Model(&model.DistShipmentItem{}).
			Scopes(scopeTenant(r.tenantID)).
			Where("dist_order_item_id = ?", oldID).
			Updates(map[string]any{"dist_order_item_id": newID, "updated_at": now}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *ShipmentRepo) NextShipmentNo() (string, error) {
	prefix := "SH" + time.Now().Format("20060102")
	var count int64
	if err := r.db.Model(&model.DistShipment{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("shipment_no LIKE ?", prefix+"%").
		Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%04d", prefix, count+1), nil
}

type ReceiptRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewReceiptRepo(db *gorm.DB) *ReceiptRepo {
	return &ReceiptRepo{db: db}
}

func (r *ReceiptRepo) ForTenant(tenantID uint64) *ReceiptRepo {
	return &ReceiptRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *ReceiptRepo) ListByPO(poID uint64) ([]model.DistReceipt, error) {
	var list []model.DistReceipt
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("dist_order_id = ?", poID).
		Order("id DESC").
		Find(&list).Error
	return list, err
}

func (r *ReceiptRepo) GetByID(poID, id uint64) (*model.DistReceipt, error) {
	var p model.DistReceipt
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("dist_order_id = ? AND id = ?", poID, id).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ReceiptRepo) Create(p *model.DistReceipt) error {
	p.TenantID = r.tenantID
	return r.db.Create(p).Error
}

func (r *ReceiptRepo) Save(p *model.DistReceipt) error {
	return r.db.Save(p).Error
}

func (r *ReceiptRepo) Delete(poID, id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).
		Where("dist_order_id = ? AND id = ?", poID, id).
		Delete(&model.DistReceipt{}).Error
}

func (r *ReceiptRepo) SumPaid(poID uint64) (float64, error) {
	var sum float64
	err := r.db.Model(&model.DistReceipt{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("dist_order_id = ? AND pay_status = ?", poID, model.DistPayStatusPaid).
		Select("COALESCE(SUM(pay_amount), 0)").
		Scan(&sum).Error
	return sum, err
}

type AttachmentRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewAttachmentRepo(db *gorm.DB) *AttachmentRepo {
	return &AttachmentRepo{db: db}
}

func (r *AttachmentRepo) ForTenant(tenantID uint64) *AttachmentRepo {
	return &AttachmentRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *AttachmentRepo) ListByPO(poID uint64) ([]model.DistAttachment, error) {
	var list []model.DistAttachment
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("dist_order_id = ?", poID).
		Order("id DESC").
		Find(&list).Error
	return list, err
}

func (r *AttachmentRepo) Create(a *model.DistAttachment) error {
	a.TenantID = r.tenantID
	return r.db.Create(a).Error
}

func (r *AttachmentRepo) Delete(poID, id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).
		Where("dist_order_id = ? AND id = ?", poID, id).
		Delete(&model.DistAttachment{}).Error
}
