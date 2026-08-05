package repo

import (
	"selfcore/internal/model"

	"gorm.io/gorm"
)

type DistributorCategoryRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewDistributorCategoryRepo(db *gorm.DB) *DistributorCategoryRepo {
	return &DistributorCategoryRepo{db: db}
}

func (r *DistributorCategoryRepo) ForTenant(tenantID uint64) *DistributorCategoryRepo {
	return &DistributorCategoryRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *DistributorCategoryRepo) List() ([]model.DistributorCategory, error) {
	var list []model.DistributorCategory
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Order("sort ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *DistributorCategoryRepo) GetByID(id uint64) (*model.DistributorCategory, error) {
	var item model.DistributorCategory
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DistributorCategoryRepo) Create(item *model.DistributorCategory) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *DistributorCategoryRepo) Save(item *model.DistributorCategory) error {
	return r.db.Save(item).Error
}

func (r *DistributorCategoryRepo) Delete(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.DistributorCategory{}, id).Error
}

func (r *DistributorCategoryRepo) CountDistributors(categoryID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.Distributor{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("category_id = ?", categoryID).
		Count(&count).Error
	return count, err
}
