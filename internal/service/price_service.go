package service

import (
	"errors"

	"selfcore/internal/dto"
	"selfcore/internal/model"
	"selfcore/internal/repo"

	"gorm.io/gorm"
)

type PriceService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewPriceService(repos *repo.Repos) *PriceService {
	return &PriceService{repos: repos}
}

func (s *PriceService) ForTenant(tenantID uint64) *PriceService {
	return &PriceService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *PriceService) List(f repo.PriceListFilter) ([]dto.PriceDetail, int64, error) {
	list, total, err := s.repos.Offer.ForTenant(s.tenantID).List(f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.PriceDetail, 0, len(list))
	for _, item := range list {
		out = append(out, s.toDetail(&item))
	}
	return out, total, nil
}

func (s *PriceService) Get(id uint64) (*dto.PriceDetail, error) {
	item, err := s.repos.Offer.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	detail := s.toDetail(item)
	return &detail, nil
}

func (s *PriceService) Create(in *dto.SkuPriceDTO) (*dto.PriceDetail, error) {
	if err := s.validateRefs(in.DistributorID, in.ShipFromAddressID); err != nil {
		return nil, err
	}
	r := s.repos.Offer.ForTenant(s.tenantID)
	item := s.fromDTO(in)
	if item.Currency == "" {
		item.Currency = "CNY"
	}
	if item.MinOrderQty <= 0 {
		item.MinOrderQty = 1
	}
	if item.IsPrimary {
		_ = r.ClearPrimary(item.SkuID, 0)
	}
	if err := r.Create(item); err != nil {
		if isDuplicateKey(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	detail := s.toDetail(item)
	return &detail, nil
}

func (s *PriceService) Update(id uint64, in *dto.SkuPriceDTO) (*dto.PriceDetail, error) {
	r := s.repos.Offer.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.validateRefs(in.DistributorID, in.ShipFromAddressID); err != nil {
		return nil, err
	}
	item.SkuID = in.SkuID
	item.DistributorID = in.DistributorID
	item.DistributorSkuCode = in.DistributorSkuCode
	item.WholesalePrice = in.WholesalePrice
	if in.Currency != "" {
		item.Currency = in.Currency
	}
	item.MinOrderQty = in.MinOrderQty
	if item.MinOrderQty <= 0 {
		item.MinOrderQty = 1
	}
	item.LeadTimeDays = in.LeadTimeDays
	item.ShipFromAddressID = in.ShipFromAddressID
	item.SupportsDropship = in.SupportsDropship
	item.SupportsWholesale = in.SupportsWholesale
	item.IsPrimary = in.IsPrimary
	item.Priority = in.Priority
	item.Status = defaultStatus(in.Status)
	item.Remark = in.Remark
	if item.IsPrimary {
		_ = r.ClearPrimary(item.SkuID, id)
	}
	if err := r.Save(item); err != nil {
		if isDuplicateKey(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	detail := s.toDetail(item)
	return &detail, nil
}

func (s *PriceService) Delete(id uint64) error {
	r := s.repos.Offer.ForTenant(s.tenantID)
	if _, err := r.GetByID(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return r.Delete(id)
}

func (s *PriceService) SupplyOptions(skuID uint64, dropshipOnly bool) (*dto.WholesaleOptionsResp, error) {
	list, err := s.repos.Offer.ForTenant(s.tenantID).ListBySku(skuID, true)
	if err != nil {
		return nil, err
	}
	resp := &dto.WholesaleOptionsResp{SkuID: skuID, Offers: make([]dto.WholesaleOptionPrice, 0, len(list))}
	sr := s.repos.Distributor.ForTenant(s.tenantID)
	for _, item := range list {
		if dropshipOnly && !item.SupportsDropship {
			continue
		}
		distributor, _ := sr.GetByID(item.DistributorID)
		opt := dto.WholesaleOptionPrice{
			OfferID: item.ID, DistributorID: item.DistributorID,
			DistributorSkuCode: item.DistributorSkuCode, WholesalePrice: item.WholesalePrice,
			Currency: item.Currency, SupportsDropship: item.SupportsDropship,
			SupportsWholesale: item.SupportsWholesale, LeadTimeDays: item.LeadTimeDays,
			IsPrimary: item.IsPrimary, Priority: item.Priority,
		}
		if distributor != nil {
			opt.DistributorName = distributor.Name
			opt.DistributorCode = distributor.Code
		}
		if item.ShipFromAddressID > 0 {
			if addr, err := sr.GetAddress(item.DistributorID, item.ShipFromAddressID); err == nil {
				opt.ShipFrom = &dto.ShipFromBrief{
					Label: addr.Label, Province: addr.Province,
					City: addr.City, District: addr.District, Address: addr.Address,
				}
			}
		}
		resp.Offers = append(resp.Offers, opt)
	}
	return resp, nil
}

func (s *PriceService) validateRefs(distributorID, addressID uint64) error {
	if _, err := s.repos.Distributor.ForTenant(s.tenantID).GetByID(distributorID); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if addressID == 0 {
		return nil
	}
	if _, err := s.repos.Distributor.ForTenant(s.tenantID).GetAddress(distributorID, addressID); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return nil
}

func (s *PriceService) fromDTO(in *dto.SkuPriceDTO) *model.SkuDistributorPrice {
	return &model.SkuDistributorPrice{
		SkuID: in.SkuID, DistributorID: in.DistributorID, DistributorSkuCode: in.DistributorSkuCode,
		WholesalePrice: in.WholesalePrice, Currency: in.Currency, MinOrderQty: in.MinOrderQty,
		LeadTimeDays: in.LeadTimeDays, ShipFromAddressID: in.ShipFromAddressID,
		SupportsDropship: in.SupportsDropship, SupportsWholesale: in.SupportsWholesale,
		IsPrimary: in.IsPrimary, Priority: in.Priority, Status: defaultStatus(in.Status),
		Remark: in.Remark,
	}
}

func (s *PriceService) toDetail(item *model.SkuDistributorPrice) dto.PriceDetail {
	detail := dto.PriceDetail{
		ID: item.ID, SkuID: item.SkuID, DistributorID: item.DistributorID,
		DistributorSkuCode: item.DistributorSkuCode, WholesalePrice: item.WholesalePrice,
		Currency: item.Currency, MinOrderQty: item.MinOrderQty,
		LeadTimeDays: item.LeadTimeDays, ShipFromAddressID: item.ShipFromAddressID,
		SupportsDropship: item.SupportsDropship, SupportsWholesale: item.SupportsWholesale,
		IsPrimary: item.IsPrimary, Priority: item.Priority, Status: item.Status, Remark: item.Remark,
	}
	sr := s.repos.Distributor.ForTenant(s.tenantID)
	if distributor, err := sr.GetByID(item.DistributorID); err == nil {
		detail.DistributorName = distributor.Name
		detail.DistributorCode = distributor.Code
	}
	if item.ShipFromAddressID > 0 {
		if addr, err := sr.GetAddress(item.DistributorID, item.ShipFromAddressID); err == nil {
			detail.ShipFromLabel = addr.Label
			detail.ShipFromCity = addr.City
		}
	}
	return detail
}
