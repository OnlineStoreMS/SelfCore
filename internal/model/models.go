package model

import "time"

type DistributorCategory struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `gorm:"index;not null" json:"tenantId"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	ParentID  uint64    `gorm:"index;default:0" json:"parentId"`
	Sort      int       `gorm:"default:0" json:"sort"`
	Status    int8      `gorm:"default:1;not null" json:"status"`
	Remark    string    `gorm:"type:text" json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (DistributorCategory) TableName() string { return "distributor_categories" }

type Distributor struct {
	ID                  uint64    `gorm:"primaryKey" json:"id"`
	TenantID            uint64    `gorm:"index;not null" json:"tenantId"`
	CategoryID          uint64    `gorm:"index" json:"categoryId"`
	CategoryName        string    `gorm:"size:64" json:"categoryName"`
	Code                string    `gorm:"size:64;not null" json:"code"`
	Name                string    `gorm:"size:128;not null" json:"name"`
	ShortName           string    `gorm:"size:64" json:"shortName"`
	Status              int8      `gorm:"default:1;not null" json:"status"`
	BuyerName           string     `gorm:"size:64" json:"buyerName"`
	CutOffTime          string     `gorm:"size:16;default:00:01" json:"cutOffTime"`
	ArrivalDays         int        `json:"arrivalDays"`
	PaymentDays         int        `json:"paymentDays"`
	// SettlementCycle: 空=不启用；day|week|month|custom（T+1：合并时刻处理上一完整周期）
	SettlementCycle      string     `gorm:"size:16;default:''" json:"settlementCycle"`
	SettlementCustomDays int        `gorm:"default:0" json:"settlementCustomDays"`
	SettlementMergeTime  string     `gorm:"size:8;default:18:30" json:"settlementMergeTime"` // HH:mm 归档合并时刻（处理上一周期）
	SettlementLastRunAt  *time.Time `json:"settlementLastRunAt,omitempty"`
	// AutoCreateDropshipPO 开启后：同步时自动分配到该分销商会建代发分销订单（不补历史；手工改分配不受此开关约束）
	AutoCreateDropshipPO bool `gorm:"not null;default:false" json:"autoCreateDropshipPO"`
	// SyncPurchasePriceFrom 合并时刻从订单备注同步采购价：空=关闭；fen_fa_remark|alloc_remark|seller_remark|printer_remark
	SyncPurchasePriceFrom string `gorm:"size:32;default:''" json:"syncPurchasePriceFrom"`
	ContactName         string    `gorm:"size:64" json:"contactName"`
	Address             string    `gorm:"size:255" json:"address"`
	OfficePhone         string    `gorm:"size:32" json:"officePhone"`
	Mobile              string    `gorm:"size:32" json:"mobile"`
	Phone               string    `gorm:"size:32" json:"phone"`
	WangwangID          string    `gorm:"size:64" json:"wangwangId"`
	QQ                  string    `gorm:"size:32" json:"qq"`
	Email               string    `gorm:"size:128" json:"email"`
	Website             string    `gorm:"size:255" json:"website"`
	Remark              string    `gorm:"type:text" json:"remark"`
	DefaultPaymentTerms string    `gorm:"size:255" json:"defaultPaymentTerms"`
	BankName            string    `gorm:"size:128" json:"bankName"`
	BankAccount         string    `gorm:"size:64" json:"bankAccount"`
	AccountName         string    `gorm:"size:128" json:"accountName"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

func (Distributor) TableName() string { return "distributors" }

// 同步采购价来源字段（分销商 SyncPurchasePriceFrom）
const (
	SyncPurchasePriceFenFa   = "fen_fa_remark"
	SyncPurchasePriceAlloc   = "alloc_remark"
	SyncPurchasePriceSeller  = "seller_remark"
	SyncPurchasePricePrinter = "printer_remark"
)

type DistributorAddress struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	TenantID     uint64    `gorm:"index;not null" json:"tenantId"`
	DistributorID   uint64    `gorm:"index;not null" json:"distributorId"`
	AddressType  string    `gorm:"size:16;not null;default:ship;index" json:"addressType"` // ship / return
	Label        string    `gorm:"size:64;not null" json:"label"`
	ContactName  string    `gorm:"size:64" json:"contactName"`
	Phone        string    `gorm:"size:32" json:"phone"`
	Province     string    `gorm:"size:32" json:"province"`
	City         string    `gorm:"size:32" json:"city"`
	District     string    `gorm:"size:32" json:"district"`
	Address      string    `gorm:"size:255" json:"address"`
	IsDefault    bool      `gorm:"not null;default:false" json:"isDefault"`
	Status       int8      `gorm:"default:1;not null" json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (DistributorAddress) TableName() string { return "distributor_addresses" }

// DistributorPaymentAccount 分销商收款账户（银行/支付宝/微信等）
type DistributorPaymentAccount struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `gorm:"index;not null" json:"tenantId"`
	DistributorID  uint64    `gorm:"index;not null" json:"distributorId"`
	Label       string    `gorm:"size:64;not null" json:"label"`
	AccountType string    `gorm:"size:32;not null;default:bank" json:"accountType"` // bank / alipay / wechat / other
	BankName    string    `gorm:"size:128" json:"bankName"`
	BankAccount string    `gorm:"size:64" json:"bankAccount"`
	AccountName string    `gorm:"size:128" json:"accountName"`
	IsDefault   bool      `gorm:"not null;default:false" json:"isDefault"`
	Status      int8      `gorm:"default:1;not null" json:"status"`
	Remark      string    `gorm:"size:255" json:"remark"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (DistributorPaymentAccount) TableName() string { return "distributor_payment_accounts" }

// DistributorPaymentQR 分销商收款码
type DistributorPaymentQR struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `gorm:"index;not null" json:"tenantId"`
	DistributorID  uint64    `gorm:"index;not null" json:"distributorId"`
	Label       string    `gorm:"size:64;not null" json:"label"`
	PayType     string    `gorm:"size:32;not null;default:wechat" json:"payType"` // wechat / alipay / other
	ImageURL    string    `gorm:"size:512;not null" json:"imageUrl"`
	AccountName string    `gorm:"size:128" json:"accountName"`
	IsDefault   bool      `gorm:"not null;default:false" json:"isDefault"`
	Status      int8      `gorm:"default:1;not null" json:"status"`
	Remark      string    `gorm:"size:255" json:"remark"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (DistributorPaymentQR) TableName() string { return "distributor_payment_qrs" }

type SkuDistributorPrice struct {
	ID                 uint64    `gorm:"primaryKey" json:"id"`
	TenantID           uint64    `gorm:"index;not null" json:"tenantId"`
	SkuID              uint64    `gorm:"index;not null" json:"skuId"`
	DistributorID         uint64    `gorm:"index;not null" json:"distributorId"`
	DistributorSkuCode    string    `gorm:"size:64" json:"distributorSkuCode"`
	WholesalePrice        float64   `gorm:"type:decimal(12,2);not null" json:"wholesalePrice"`
	Currency           string    `gorm:"size:8;default:CNY" json:"currency"`
	MinOrderQty        int       `gorm:"default:1" json:"minOrderQty"`
	LeadTimeDays       int       `json:"leadTimeDays"`
	ShipFromAddressID  uint64    `gorm:"index" json:"shipFromAddressId"`
	SupportsDropship   bool      `gorm:"not null;default:false" json:"supportsDropship"`
	SupportsWholesale  bool      `gorm:"not null;default:true" json:"supportsWholesale"`
	IsPrimary          bool      `gorm:"not null;default:false" json:"isPrimary"`
	Priority           int       `gorm:"default:0" json:"priority"`
	Status             int8      `gorm:"default:1;not null" json:"status"`
	Remark             string    `gorm:"type:text" json:"remark"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

func (SkuDistributorPrice) TableName() string { return "sku_distributor_prices" }

type DistOrder struct {
	ID                  uint64     `gorm:"primaryKey" json:"id"`
	TenantID            uint64     `gorm:"index;not null" json:"tenantId"`
	DistNo                string     `gorm:"size:32;not null" json:"distNo"`
	DistributorID          uint64     `gorm:"index;not null" json:"distributorId"`
	Status              string     `gorm:"size:32;not null;default:draft" json:"status"`
	TotalAmount         float64    `gorm:"type:decimal(14,2);not null;default:0" json:"totalAmount"`   // 采购总额
	SaleAmount          float64    `gorm:"type:decimal(14,2);not null;default:0" json:"saleAmount"`    // 销售侧订单总金额（实付合计）
	Currency            string     `gorm:"size:8;default:CNY" json:"currency"`
	ExpectedArrivalDate *time.Time `json:"expectedArrivalDate"`
	WarehouseID         uint64     `json:"warehouseId"`
	FulfillmentType     string     `gorm:"size:16;default:wholesale" json:"fulfillmentType"`
	RefTraceID          string     `gorm:"type:text" json:"refTraceId"`
	RefSoID             uint64     `json:"refSoId"`
	BuyerID             uint64     `json:"buyerId"`
	BuyerName           string     `gorm:"size:64" json:"buyerName"`
	PayStatus           string     `gorm:"size:16;default:unpaid" json:"payStatus"`
	Remark              string     `gorm:"type:text" json:"remark"`
	OrderedAt           *time.Time `json:"orderedAt"`
	CompletedAt         *time.Time `json:"completedAt"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	Items               []DistOrderItem `gorm:"foreignKey:DistOrderID" json:"items,omitempty"`
}

func (DistOrder) TableName() string { return "dist_orders" }

type DistOrderItem struct {
	ID                  uint64     `gorm:"primaryKey" json:"id"`
	TenantID            uint64     `gorm:"index;not null" json:"tenantId"`
	DistOrderID                uint64     `gorm:"index;not null" json:"distOrderId"`
	SkuID               uint64     `gorm:"index;not null;default:0" json:"skuId"`
	OfferID             uint64     `json:"offerId"`
	ProductName         string     `gorm:"size:512" json:"productName"`
	SkuCode             string     `gorm:"size:64" json:"skuCode"`         // 我方商家编码
	SkuSpecs            string     `gorm:"size:256" json:"skuSpecs"`       // 规格
	PicURL              string     `gorm:"size:512" json:"picUrl"`         // SKU 图片
	DistributorSkuCode     string     `gorm:"size:64" json:"distributorSkuCode"` // 对方货号
	Qty                 int        `gorm:"not null" json:"qty"`
	SaleUnitPrice       float64    `gorm:"type:decimal(12,2);not null;default:0" json:"saleUnitPrice"` // 实付摊到单价
	SaleAmount          float64    `gorm:"type:decimal(14,2);not null;default:0" json:"saleAmount"`     // 实付金额（销售单）
	UnitPrice           float64    `gorm:"type:decimal(12,2);not null" json:"unitPrice"`                 // 分销订单价
	LineAmount          float64    `gorm:"type:decimal(14,2);not null" json:"lineAmount"`                // 采购小计
	ExpectedArrivalDate *time.Time `json:"expectedArrivalDate"`
	ReceivedQty         int        `gorm:"default:0" json:"receivedQty"`
	RefSoID             uint64     `gorm:"index;default:0" json:"refSoId"`     // 关联销售单 ID
	RefOrderNo          string     `gorm:"size:64;index" json:"refOrderNo"`    // 关联销售单号
	Cancelled           bool       `gorm:"default:false;index" json:"cancelled"` // 销售单撤回后代发明细作废
	Remark              string     `gorm:"type:text" json:"remark"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

func (DistOrderItem) TableName() string { return "dist_order_items" }
