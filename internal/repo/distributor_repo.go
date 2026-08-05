package repo

import (
	"time"

	"selfcore/internal/model"

	"gorm.io/gorm"
)

type DistributorRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewDistributorRepo(db *gorm.DB) *DistributorRepo {
	return &DistributorRepo{db: db}
}

func (r *DistributorRepo) ForTenant(tenantID uint64) *DistributorRepo {
	return &DistributorRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *DistributorRepo) List(keyword string, categoryID uint64, page, pageSize int) ([]model.Distributor, int64, error) {
	q := r.db.Model(&model.Distributor{}).Scopes(scopeTenant(r.tenantID))
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name ILIKE ? OR code ILIKE ? OR short_name ILIKE ?", like, like, like)
	}
	if categoryID > 0 {
		q = q.Where("category_id = ?", categoryID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Distributor
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// ListWithSettlementCycle 已配置结算周期的分销商（跨租户，供调度器用）。
func (r *DistributorRepo) ListWithSettlementCycle() ([]model.Distributor, error) {
	var list []model.Distributor
	err := r.db.Where("settlement_cycle IN ?", []string{"day", "week", "month", "custom"}).
		Order("tenant_id ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *DistributorRepo) UpdateSettlementLastRunAt(id uint64, at time.Time) error {
	return r.db.Model(&model.Distributor{}).Where("id = ?", id).Update("settlement_last_run_at", at).Error
}

func (r *DistributorRepo) GetByID(id uint64) (*model.Distributor, error) {
	var item model.Distributor
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DistributorRepo) GetByCode(code string) (*model.Distributor, error) {
	var item model.Distributor
	err := r.db.Scopes(scopeTenant(r.tenantID)).Where("code = ?", code).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DistributorRepo) Create(item *model.Distributor) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *DistributorRepo) Save(item *model.Distributor) error {
	return r.db.Save(item).Error
}

func (r *DistributorRepo) Delete(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.Distributor{}, id).Error
}

func (r *DistributorRepo) ListAddresses(distributorID uint64, addressType string) ([]model.DistributorAddress, error) {
	var list []model.DistributorAddress
	q := r.db.Scopes(scopeTenant(r.tenantID)).Where("distributor_id = ?", distributorID)
	if addressType != "" {
		q = q.Where("address_type = ?", addressType)
	}
	err := q.Order("is_default DESC, id ASC").Find(&list).Error
	return list, err
}

func (r *DistributorRepo) GetAddress(distributorID, addressID uint64) (*model.DistributorAddress, error) {
	var item model.DistributorAddress
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ? AND id = ?", distributorID, addressID).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DistributorRepo) CreateAddress(item *model.DistributorAddress) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *DistributorRepo) SaveAddress(item *model.DistributorAddress) error {
	return r.db.Save(item).Error
}

func (r *DistributorRepo) DeleteAddress(distributorID, addressID uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ? AND id = ?", distributorID, addressID).
		Delete(&model.DistributorAddress{}).Error
}

func (r *DistributorRepo) ClearDefaultAddress(distributorID uint64, addressType string, exceptID uint64) error {
	q := r.db.Model(&model.DistributorAddress{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ?", distributorID)
	if addressType != "" {
		q = q.Where("address_type = ?", addressType)
	}
	if exceptID > 0 {
		q = q.Where("id <> ?", exceptID)
	}
	return q.Update("is_default", false).Error
}

func (r *DistributorRepo) ListPaymentAccounts(distributorID uint64) ([]model.DistributorPaymentAccount, error) {
	var list []model.DistributorPaymentAccount
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ?", distributorID).
		Order("is_default DESC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *DistributorRepo) GetPaymentAccount(distributorID, accountID uint64) (*model.DistributorPaymentAccount, error) {
	var item model.DistributorPaymentAccount
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ? AND id = ?", distributorID, accountID).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DistributorRepo) CreateReceiptAccount(item *model.DistributorPaymentAccount) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *DistributorRepo) SavePaymentAccount(item *model.DistributorPaymentAccount) error {
	return r.db.Save(item).Error
}

func (r *DistributorRepo) DeleteReceiptAccount(distributorID, accountID uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ? AND id = ?", distributorID, accountID).
		Delete(&model.DistributorPaymentAccount{}).Error
}

func (r *DistributorRepo) ClearDefaultPaymentAccount(distributorID uint64, exceptID uint64) error {
	q := r.db.Model(&model.DistributorPaymentAccount{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ?", distributorID)
	if exceptID > 0 {
		q = q.Where("id <> ?", exceptID)
	}
	return q.Update("is_default", false).Error
}

func (r *DistributorRepo) ListPaymentQRs(distributorID uint64) ([]model.DistributorPaymentQR, error) {
	var list []model.DistributorPaymentQR
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ?", distributorID).
		Order("is_default DESC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *DistributorRepo) GetPaymentQR(distributorID, qrID uint64) (*model.DistributorPaymentQR, error) {
	var item model.DistributorPaymentQR
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ? AND id = ?", distributorID, qrID).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DistributorRepo) CreateReceiptQR(item *model.DistributorPaymentQR) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *DistributorRepo) SavePaymentQR(item *model.DistributorPaymentQR) error {
	return r.db.Save(item).Error
}

func (r *DistributorRepo) DeleteReceiptQR(distributorID, qrID uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ? AND id = ?", distributorID, qrID).
		Delete(&model.DistributorPaymentQR{}).Error
}

func (r *DistributorRepo) ClearDefaultPaymentQR(distributorID uint64, exceptID uint64) error {
	q := r.db.Model(&model.DistributorPaymentQR{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("distributor_id = ?", distributorID)
	if exceptID > 0 {
		q = q.Where("id <> ?", exceptID)
	}
	return q.Update("is_default", false).Error
}
