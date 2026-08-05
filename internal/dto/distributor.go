package dto

type DistributorCategoryDTO struct {
	Name     string `json:"name" binding:"required"`
	ParentID uint64 `json:"parentId"`
	Sort     int    `json:"sort"`
	Status   int8   `json:"status"`
	Remark   string `json:"remark"`
}

type DistributorDTO struct {
	CategoryID          uint64 `json:"categoryId"`
	CategoryName        string `json:"categoryName"`
	Code                string `json:"code" binding:"required"`
	Name                string `json:"name" binding:"required"`
	ShortName           string `json:"shortName"`
	Status              int8   `json:"status"`
	BuyerName            string `json:"buyerName"`
	CutOffTime           string `json:"cutOffTime"`
	ArrivalDays          int    `json:"arrivalDays"`
	PaymentDays          int    `json:"paymentDays"`
	SettlementCycle      string `json:"settlementCycle"`      // "" | day | week | month | custom
	SettlementCustomDays int    `json:"settlementCustomDays"` // custom 时有效
	SettlementMergeTime   string `json:"settlementMergeTime"`   // HH:mm，默认 18:30
	AutoCreateDropshipPO  bool   `json:"autoCreateDropshipPO"`  // 自动创建代发分销订单
	SyncPurchasePriceFrom string `json:"syncPurchasePriceFrom"` // "" | fen_fa_remark | alloc_remark | seller_remark | printer_remark
	ContactName           string `json:"contactName"`
	Address             string `json:"address"`
	OfficePhone         string `json:"officePhone"`
	Mobile              string `json:"mobile"`
	Phone               string `json:"phone"`
	WangwangID          string `json:"wangwangId"`
	QQ                  string `json:"qq"`
	Email               string `json:"email"`
	Website             string `json:"website"`
	Remark              string `json:"remark"`
	DefaultPaymentTerms string `json:"defaultPaymentTerms"`
	BankName            string `json:"bankName"`
	BankAccount         string `json:"bankAccount"`
	AccountName         string `json:"accountName"`
}

type DistributorAddressDTO struct {
	AddressType string `json:"addressType"`
	Label       string `json:"label" binding:"required"`
	ContactName string `json:"contactName"`
	Phone       string `json:"phone"`
	Province    string `json:"province"`
	City        string `json:"city"`
	District    string `json:"district"`
	Address     string `json:"address"`
	IsDefault   bool   `json:"isDefault"`
	Status      int8   `json:"status"`
}

type DistributorPaymentAccountDTO struct {
	Label       string `json:"label" binding:"required"`
	AccountType string `json:"accountType"`
	BankName    string `json:"bankName"`
	BankAccount string `json:"bankAccount"`
	AccountName string `json:"accountName"`
	IsDefault   bool   `json:"isDefault"`
	Status      int8   `json:"status"`
	Remark      string `json:"remark"`
}

type DistributorPaymentQRDTO struct {
	Label       string `json:"label" binding:"required"`
	PayType     string `json:"payType"`
	ImageURL    string `json:"imageUrl" binding:"required"`
	AccountName string `json:"accountName"`
	IsDefault   bool   `json:"isDefault"`
	Status      int8   `json:"status"`
	Remark      string `json:"remark"`
}

type SkuPriceDTO struct {
	SkuID             uint64  `json:"skuId" binding:"required"`
	DistributorID        uint64  `json:"distributorId" binding:"required"`
	DistributorSkuCode   string  `json:"distributorSkuCode"`
	WholesalePrice       float64 `json:"wholesalePrice" binding:"required"`
	Currency          string  `json:"currency"`
	MinOrderQty       int     `json:"minOrderQty"`
	LeadTimeDays      int     `json:"leadTimeDays"`
	ShipFromAddressID uint64  `json:"shipFromAddressId"`
	SupportsDropship  bool    `json:"supportsDropship"`
	SupportsWholesale bool    `json:"supportsWholesale"`
	IsPrimary         bool    `json:"isPrimary"`
	Priority          int     `json:"priority"`
	Status            int8    `json:"status"`
	Remark            string  `json:"remark"`
}

type ShipFromBrief struct {
	Label    string `json:"label"`
	Province string `json:"province"`
	City     string `json:"city"`
	District string `json:"district"`
	Address  string `json:"address"`
}

type WholesaleOptionPrice struct {
	OfferID          uint64         `json:"offerId"`
	DistributorID       uint64         `json:"distributorId"`
	DistributorName     string         `json:"distributorName"`
	DistributorCode     string         `json:"distributorCode"`
	DistributorSkuCode  string         `json:"distributorSkuCode"`
	WholesalePrice      float64        `json:"wholesalePrice"`
	Currency         string         `json:"currency"`
	SupportsDropship bool           `json:"supportsDropship"`
	SupportsWholesale bool          `json:"supportsWholesale"`
	LeadTimeDays     int            `json:"leadTimeDays"`
	IsPrimary        bool           `json:"isPrimary"`
	Priority         int            `json:"priority"`
	ShipFrom         *ShipFromBrief `json:"shipFrom,omitempty"`
}

type WholesaleOptionsResp struct {
	SkuID  uint64              `json:"skuId"`
	Offers []WholesaleOptionPrice `json:"offers"`
}

type PriceDetail struct {
	ID                uint64  `json:"id"`
	SkuID             uint64  `json:"skuId"`
	DistributorID        uint64  `json:"distributorId"`
	DistributorName      string  `json:"distributorName"`
	DistributorCode      string  `json:"distributorCode"`
	DistributorSkuCode   string  `json:"distributorSkuCode"`
	WholesalePrice       float64 `json:"wholesalePrice"`
	Currency          string  `json:"currency"`
	MinOrderQty       int     `json:"minOrderQty"`
	LeadTimeDays      int     `json:"leadTimeDays"`
	ShipFromAddressID uint64  `json:"shipFromAddressId"`
	ShipFromLabel     string  `json:"shipFromLabel"`
	ShipFromCity      string  `json:"shipFromCity"`
	SupportsDropship  bool    `json:"supportsDropship"`
	SupportsWholesale bool    `json:"supportsWholesale"`
	IsPrimary         bool    `json:"isPrimary"`
	Priority          int     `json:"priority"`
	Status            int8    `json:"status"`
	Remark            string  `json:"remark"`
}
