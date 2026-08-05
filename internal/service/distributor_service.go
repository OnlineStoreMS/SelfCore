package service

import (
	"errors"
	"strings"

	"selfcore/internal/dto"
	"selfcore/internal/model"
	"selfcore/internal/repo"

	"gorm.io/gorm"
)

type DistributorService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewDistributorService(repos *repo.Repos) *DistributorService {
	return &DistributorService{repos: repos}
}

func (s *DistributorService) ForTenant(tenantID uint64) *DistributorService {
	return &DistributorService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *DistributorService) List(keyword string, categoryID uint64, page, pageSize int) ([]model.Distributor, int64, error) {
	list, total, err := s.repos.Distributor.ForTenant(s.tenantID).List(keyword, categoryID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		normalizeDistributorPhones(&list[i])
	}
	return list, total, nil
}

func (s *DistributorService) Get(id uint64) (*model.Distributor, error) {
	item, err := s.repos.Distributor.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	normalizeDistributorPhones(item)
	return item, nil
}

func (s *DistributorService) Create(in *dto.DistributorDTO) (*model.Distributor, error) {
	r := s.repos.Distributor.ForTenant(s.tenantID)
	if _, err := r.GetByCode(in.Code); err == nil {
		return nil, ErrDuplicateCode
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	syncPhoneFields(in)
	if err := s.resolveCategoryName(in); err != nil {
		return nil, err
	}
	item := distributorFromDTO(in)
	if err := r.Create(item); err != nil {
		if isDuplicateKey(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	normalizeDistributorPhones(item)
	return item, nil
}

func (s *DistributorService) Update(id uint64, in *dto.DistributorDTO) (*model.Distributor, error) {
	r := s.repos.Distributor.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if in.Code != item.Code {
		if other, err := r.GetByCode(in.Code); err == nil && other.ID != id {
			return nil, ErrDuplicateCode
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	syncPhoneFields(in)
	if err := s.resolveCategoryName(in); err != nil {
		return nil, err
	}
	applyDistributorDTO(item, in)
	if err := r.Save(item); err != nil {
		if isDuplicateKey(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	normalizeDistributorPhones(item)
	return item, nil
}

func (s *DistributorService) Delete(id uint64) error {
	r := s.repos.Distributor.ForTenant(s.tenantID)
	if _, err := r.GetByID(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return r.Delete(id)
}

func (s *DistributorService) ListCategories() ([]model.DistributorCategory, error) {
	return s.repos.DistributorCategory.ForTenant(s.tenantID).List()
}

func (s *DistributorService) GetCategory(id uint64) (*model.DistributorCategory, error) {
	item, err := s.repos.DistributorCategory.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *DistributorService) CreateCategory(in *dto.DistributorCategoryDTO) (*model.DistributorCategory, error) {
	r := s.repos.DistributorCategory.ForTenant(s.tenantID)
	item := &model.DistributorCategory{
		Name: in.Name, ParentID: in.ParentID, Sort: in.Sort,
		Status: defaultStatus(in.Status), Remark: in.Remark,
	}
	if err := r.Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *DistributorService) UpdateCategory(id uint64, in *dto.DistributorCategoryDTO) (*model.DistributorCategory, error) {
	r := s.repos.DistributorCategory.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.Name = in.Name
	item.ParentID = in.ParentID
	item.Sort = in.Sort
	item.Status = defaultStatus(in.Status)
	item.Remark = in.Remark
	if err := r.Save(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *DistributorService) DeleteCategory(id uint64) error {
	r := s.repos.DistributorCategory.ForTenant(s.tenantID)
	if _, err := r.GetByID(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	count, err := r.CountDistributors(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrBadRequest
	}
	return r.Delete(id)
}

func (s *DistributorService) ListAddresses(distributorID uint64, addressType string) ([]model.DistributorAddress, error) {
	if _, err := s.Get(distributorID); err != nil {
		return nil, err
	}
	return s.repos.Distributor.ForTenant(s.tenantID).ListAddresses(distributorID, addressType)
}

func defaultAddressType(t string) string {
	if t == model.AddressTypeReturn {
		return model.AddressTypeReturn
	}
	return model.AddressTypeShip
}

func (s *DistributorService) CreateAddress(distributorID uint64, in *dto.DistributorAddressDTO) (*model.DistributorAddress, error) {
	if _, err := s.Get(distributorID); err != nil {
		return nil, err
	}
	r := s.repos.Distributor.ForTenant(s.tenantID)
	addrType := defaultAddressType(in.AddressType)
	item := &model.DistributorAddress{
		DistributorID: distributorID, AddressType: addrType, Label: in.Label, ContactName: in.ContactName,
		Phone: in.Phone, Province: in.Province, City: in.City,
		District: in.District, Address: in.Address, IsDefault: in.IsDefault,
		Status: defaultStatus(in.Status),
	}
	if item.IsDefault {
		_ = r.ClearDefaultAddress(distributorID, addrType, 0)
	}
	if err := r.CreateAddress(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *DistributorService) UpdateAddress(distributorID, addressID uint64, in *dto.DistributorAddressDTO) (*model.DistributorAddress, error) {
	r := s.repos.Distributor.ForTenant(s.tenantID)
	item, err := r.GetAddress(distributorID, addressID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	addrType := defaultAddressType(in.AddressType)
	if addrType == "" {
		addrType = defaultAddressType(item.AddressType)
	}
	item.AddressType = addrType
	item.Label = in.Label
	item.ContactName = in.ContactName
	item.Phone = in.Phone
	item.Province = in.Province
	item.City = in.City
	item.District = in.District
	item.Address = in.Address
	item.IsDefault = in.IsDefault
	item.Status = defaultStatus(in.Status)
	if item.IsDefault {
		_ = r.ClearDefaultAddress(distributorID, addrType, addressID)
	}
	if err := r.SaveAddress(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *DistributorService) DeleteAddress(distributorID, addressID uint64) error {
	r := s.repos.Distributor.ForTenant(s.tenantID)
	if _, err := r.GetAddress(distributorID, addressID); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return r.DeleteAddress(distributorID, addressID)
}

func defaultAccountType(t string) string {
	if t == "" {
		return "bank"
	}
	return t
}

func defaultPayType(t string) string {
	if t == "" {
		return "wechat"
	}
	return t
}

func (s *DistributorService) ListPaymentAccounts(distributorID uint64) ([]model.DistributorPaymentAccount, error) {
	if _, err := s.Get(distributorID); err != nil {
		return nil, err
	}
	return s.repos.Distributor.ForTenant(s.tenantID).ListPaymentAccounts(distributorID)
}

func (s *DistributorService) CreateReceiptAccount(distributorID uint64, in *dto.DistributorPaymentAccountDTO) (*model.DistributorPaymentAccount, error) {
	if _, err := s.Get(distributorID); err != nil {
		return nil, err
	}
	r := s.repos.Distributor.ForTenant(s.tenantID)
	item := &model.DistributorPaymentAccount{
		DistributorID:  distributorID,
		Label:       in.Label,
		AccountType: defaultAccountType(in.AccountType),
		BankName:    in.BankName,
		BankAccount: in.BankAccount,
		AccountName: in.AccountName,
		IsDefault:   in.IsDefault,
		Status:      defaultStatus(in.Status),
		Remark:      in.Remark,
	}
	if item.IsDefault {
		_ = r.ClearDefaultPaymentAccount(distributorID, 0)
	}
	if err := r.CreateReceiptAccount(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *DistributorService) UpdateReceiptAccount(distributorID, accountID uint64, in *dto.DistributorPaymentAccountDTO) (*model.DistributorPaymentAccount, error) {
	r := s.repos.Distributor.ForTenant(s.tenantID)
	item, err := r.GetPaymentAccount(distributorID, accountID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.Label = in.Label
	item.AccountType = defaultAccountType(in.AccountType)
	item.BankName = in.BankName
	item.BankAccount = in.BankAccount
	item.AccountName = in.AccountName
	item.IsDefault = in.IsDefault
	item.Status = defaultStatus(in.Status)
	item.Remark = in.Remark
	if item.IsDefault {
		_ = r.ClearDefaultPaymentAccount(distributorID, accountID)
	}
	if err := r.SavePaymentAccount(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *DistributorService) DeleteReceiptAccount(distributorID, accountID uint64) error {
	r := s.repos.Distributor.ForTenant(s.tenantID)
	if _, err := r.GetPaymentAccount(distributorID, accountID); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return r.DeleteReceiptAccount(distributorID, accountID)
}

func (s *DistributorService) ListPaymentQRs(distributorID uint64) ([]model.DistributorPaymentQR, error) {
	if _, err := s.Get(distributorID); err != nil {
		return nil, err
	}
	return s.repos.Distributor.ForTenant(s.tenantID).ListPaymentQRs(distributorID)
}

func (s *DistributorService) CreateReceiptQR(distributorID uint64, in *dto.DistributorPaymentQRDTO) (*model.DistributorPaymentQR, error) {
	if _, err := s.Get(distributorID); err != nil {
		return nil, err
	}
	r := s.repos.Distributor.ForTenant(s.tenantID)
	item := &model.DistributorPaymentQR{
		DistributorID:  distributorID,
		Label:       in.Label,
		PayType:     defaultPayType(in.PayType),
		ImageURL:    in.ImageURL,
		AccountName: in.AccountName,
		IsDefault:   in.IsDefault,
		Status:      defaultStatus(in.Status),
		Remark:      in.Remark,
	}
	if item.IsDefault {
		_ = r.ClearDefaultPaymentQR(distributorID, 0)
	}
	if err := r.CreateReceiptQR(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *DistributorService) UpdateReceiptQR(distributorID, qrID uint64, in *dto.DistributorPaymentQRDTO) (*model.DistributorPaymentQR, error) {
	r := s.repos.Distributor.ForTenant(s.tenantID)
	item, err := r.GetPaymentQR(distributorID, qrID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.Label = in.Label
	item.PayType = defaultPayType(in.PayType)
	item.ImageURL = in.ImageURL
	item.AccountName = in.AccountName
	item.IsDefault = in.IsDefault
	item.Status = defaultStatus(in.Status)
	item.Remark = in.Remark
	if item.IsDefault {
		_ = r.ClearDefaultPaymentQR(distributorID, qrID)
	}
	if err := r.SavePaymentQR(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *DistributorService) DeleteReceiptQR(distributorID, qrID uint64) error {
	r := s.repos.Distributor.ForTenant(s.tenantID)
	if _, err := r.GetPaymentQR(distributorID, qrID); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return r.DeleteReceiptQR(distributorID, qrID)
}

func (s *DistributorService) resolveCategoryName(in *dto.DistributorDTO) error {
	if in.CategoryID == 0 {
		in.CategoryName = ""
		return nil
	}
	if in.CategoryName != "" {
		return nil
	}
	cat, err := s.repos.DistributorCategory.ForTenant(s.tenantID).GetByID(in.CategoryID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	in.CategoryName = cat.Name
	return nil
}

func distributorFromDTO(in *dto.DistributorDTO) *model.Distributor {
	item := &model.Distributor{
		CategoryID: in.CategoryID, CategoryName: in.CategoryName,
		Code: in.Code, Name: in.Name, ShortName: in.ShortName,
		Status: defaultStatus(in.Status), BuyerName: in.BuyerName,
		CutOffTime: defaultCutOffTime(in.CutOffTime),
		ArrivalDays: in.ArrivalDays, PaymentDays: in.PaymentDays,
		SettlementCycle:      normalizeSettlementCycle(in.SettlementCycle),
		SettlementCustomDays: normalizeSettlementCustomDays(normalizeSettlementCycle(in.SettlementCycle), in.SettlementCustomDays),
		SettlementMergeTime:   defaultSettlementMergeTime(in.SettlementMergeTime),
		AutoCreateDropshipPO:  in.AutoCreateDropshipPO,
		SyncPurchasePriceFrom: normalizeSyncPurchasePriceFrom(in.SyncPurchasePriceFrom),
		ContactName: in.ContactName, Address: in.Address,
		OfficePhone: in.OfficePhone, Mobile: in.Mobile, Phone: in.Phone,
		WangwangID: in.WangwangID, QQ: in.QQ, Email: in.Email,
		Website: in.Website, Remark: in.Remark,
		DefaultPaymentTerms: in.DefaultPaymentTerms,
		BankName: in.BankName, BankAccount: in.BankAccount, AccountName: in.AccountName,
	}
	return item
}

func applyDistributorDTO(item *model.Distributor, in *dto.DistributorDTO) {
	item.CategoryID = in.CategoryID
	item.CategoryName = in.CategoryName
	item.Code = in.Code
	item.Name = in.Name
	item.ShortName = in.ShortName
	item.Status = defaultStatus(in.Status)
	item.BuyerName = in.BuyerName
	item.CutOffTime = defaultCutOffTime(in.CutOffTime)
	item.ArrivalDays = in.ArrivalDays
	item.PaymentDays = in.PaymentDays
	item.SettlementCycle = normalizeSettlementCycle(in.SettlementCycle)
	item.SettlementCustomDays = normalizeSettlementCustomDays(item.SettlementCycle, in.SettlementCustomDays)
	item.SettlementMergeTime = defaultSettlementMergeTime(in.SettlementMergeTime)
	item.AutoCreateDropshipPO = in.AutoCreateDropshipPO
	item.SyncPurchasePriceFrom = normalizeSyncPurchasePriceFrom(in.SyncPurchasePriceFrom)
	item.ContactName = in.ContactName
	item.Address = in.Address
	item.OfficePhone = in.OfficePhone
	item.Mobile = in.Mobile
	item.Phone = in.Phone
	item.WangwangID = in.WangwangID
	item.QQ = in.QQ
	item.Email = in.Email
	item.Website = in.Website
	item.Remark = in.Remark
	item.DefaultPaymentTerms = in.DefaultPaymentTerms
	item.BankName = in.BankName
	item.BankAccount = in.BankAccount
	item.AccountName = in.AccountName
}

func syncPhoneFields(in *dto.DistributorDTO) {
	if in.Mobile == "" && in.Phone != "" {
		in.Mobile = in.Phone
	}
	if in.Phone == "" && in.Mobile != "" {
		in.Phone = in.Mobile
	}
}

func normalizeDistributorPhones(item *model.Distributor) {
	if item.Mobile == "" && item.Phone != "" {
		item.Mobile = item.Phone
	}
}

func defaultCutOffTime(v string) string {
	if v == "" {
		return "00:01"
	}
	return v
}

func defaultSettlementMergeTime(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "18:30"
	}
	return v
}

func normalizeSyncPurchasePriceFrom(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case model.SyncPurchasePriceFenFa, "fenfa", "fen_fa":
		return model.SyncPurchasePriceFenFa
	case model.SyncPurchasePriceAlloc, "alloc":
		return model.SyncPurchasePriceAlloc
	case model.SyncPurchasePriceSeller, "seller":
		return model.SyncPurchasePriceSeller
	case model.SyncPurchasePricePrinter, "printer":
		return model.SyncPurchasePricePrinter
	default:
		return ""
	}
}

func normalizeSettlementCycle(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "day", "week", "month", "custom":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return ""
	}
}

func normalizeSettlementCustomDays(cycle string, days int) int {
	if cycle != "custom" {
		return 0
	}
	if days < 1 {
		return 1
	}
	if days > 365 {
		return 365
	}
	return days
}

func defaultStatus(v int8) int8 {
	if v == 0 {
		return 1
	}
	return v
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "duplicate") || contains(msg, "unique")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (stringIndex(s, sub) >= 0)))
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
