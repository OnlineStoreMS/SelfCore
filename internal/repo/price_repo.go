package repo

import (
	"selfcore/internal/model"

	"gorm.io/gorm"
)

type PriceRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewPriceRepo(db *gorm.DB) *PriceRepo {
	return &PriceRepo{db: db}
}

func (r *PriceRepo) ForTenant(tenantID uint64) *PriceRepo {
	return &PriceRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

type PriceListFilter struct {
	SkuID      uint64
	DistributorID uint64
	Keyword    string
	Page       int
	PageSize   int
}

func (r *PriceRepo) List(f PriceListFilter) ([]model.SkuDistributorPrice, int64, error) {
	q := r.db.Model(&model.SkuDistributorPrice{}).Scopes(scopeTenant(r.tenantID))
	if f.SkuID > 0 {
		q = q.Where("sku_id = ?", f.SkuID)
	}
	if f.DistributorID > 0 {
		q = q.Where("distributor_id = ?", f.DistributorID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SkuDistributorPrice
	offset := (f.Page - 1) * f.PageSize
	err := q.Order("sku_id ASC, is_primary DESC, priority DESC, wholesale_price ASC, id ASC").
		Offset(offset).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}

func (r *PriceRepo) ListBySku(skuID uint64, activeOnly bool) ([]model.SkuDistributorPrice, error) {
	q := r.db.Scopes(scopeTenant(r.tenantID)).Where("sku_id = ?", skuID)
	if activeOnly {
		q = q.Where("status = 1")
	}
	var list []model.SkuDistributorPrice
	err := q.Order("is_primary DESC, priority DESC, wholesale_price ASC").Find(&list).Error
	return list, err
}

func (r *PriceRepo) GetByID(id uint64) (*model.SkuDistributorPrice, error) {
	var item model.SkuDistributorPrice
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PriceRepo) Create(item *model.SkuDistributorPrice) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *PriceRepo) Save(item *model.SkuDistributorPrice) error {
	return r.db.Save(item).Error
}

func (r *PriceRepo) Delete(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.SkuDistributorPrice{}, id).Error
}

func (r *PriceRepo) ClearPrimary(skuID uint64, exceptID uint64) error {
	q := r.db.Model(&model.SkuDistributorPrice{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("sku_id = ?", skuID)
	if exceptID > 0 {
		q = q.Where("id <> ?", exceptID)
	}
	return q.Update("is_primary", false).Error
}
