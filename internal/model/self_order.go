package model

import "time"

// SelfOrder 自营履约单据（承接 OrderCore self_ship，对接仓储扣库）。
type SelfOrder struct {
	ID            uint64     `gorm:"primaryKey" json:"id"`
	TenantID      uint64     `gorm:"index;not null" json:"tenantId"`
	SoNo          string     `gorm:"size:32;not null" json:"soNo"`
	Status        string     `gorm:"size:32;not null;default:ordered" json:"status"`
	WarehouseID   uint64     `gorm:"index;default:0" json:"warehouseId"`
	RefSoID       uint64     `gorm:"index;default:0" json:"refSoId"`
	RefTraceID    string     `gorm:"size:64;index" json:"refTraceId"`
	SaleAmount    float64    `gorm:"type:decimal(14,2);not null;default:0" json:"saleAmount"`
	CostAmount    float64    `gorm:"type:decimal(14,2);not null;default:0" json:"costAmount"`
	BuyerName     string     `gorm:"size:64" json:"buyerName"`
	BuyerPhone    string     `gorm:"size:32" json:"buyerPhone"`
	Address       string     `gorm:"type:text" json:"address"`
	Remark        string     `gorm:"type:text" json:"remark"` // 系统备注（如 OMS自营 / 取消原因）
	SourceChannel string     `gorm:"size:32;index" json:"sourceChannel"`
	Platform      string     `gorm:"size:32" json:"platform"`
	ShopName      string     `gorm:"size:128" json:"shopName"`
	// ManualSourceName 手工单订单来源名称（来自 OrderCore 手工订单来源字典）
	ManualSourceName string `gorm:"size:128" json:"manualSourceName"`
	BuyerRemark   string     `gorm:"type:text" json:"buyerRemark"`
	SellerRemark  string     `gorm:"type:text" json:"sellerRemark"`
	FenFaRemark   string     `gorm:"type:text" json:"fenFaRemark"`
	PrinterRemark string     `gorm:"type:text" json:"printerRemark"`
	StockDeducted bool       `gorm:"default:false;index" json:"stockDeducted"`
	StockError    string     `gorm:"type:text" json:"stockError"`
	// PayStatus 付款进度：unpaid|partial|paid（对齐分销/采购）
	PayStatus string `gorm:"size:16;default:unpaid;index" json:"payStatus"`
	// PaidAt 最早一笔已付时间（有付款记录时填写）
	PaidAt      *time.Time `json:"paidAt,omitempty"`
	OrderedAt   *time.Time `json:"orderedAt"`
	ShippedAt   *time.Time `json:"shippedAt"`
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	Items       []SelfOrderItem `gorm:"foreignKey:SelfOrderID" json:"items,omitempty"`
}

// SelfPayment 自营单付款记录。
type SelfPayment struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	TenantID     uint64     `gorm:"index;not null" json:"tenantId"`
	SelfOrderID  uint64     `gorm:"index;not null" json:"selfOrderId"`
	PayAmount    float64    `gorm:"type:decimal(14,2);not null" json:"payAmount"`
	PayMethod    string     `gorm:"size:32" json:"payMethod"`
	PayAccount   string     `gorm:"size:128" json:"payAccount"`
	PayeeAccount string     `gorm:"size:128" json:"payeeAccount"`
	PayeeName    string     `gorm:"size:128" json:"payeeName"`
	PayStatus    string     `gorm:"size:16;default:paid" json:"payStatus"`
	PaidAt       *time.Time `json:"paidAt"`
	Remark       string     `gorm:"type:text" json:"remark"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (SelfPayment) TableName() string { return "self_payments" }

func (SelfOrder) TableName() string { return "self_orders" }

type SelfOrderItem struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	TenantID      uint64    `gorm:"index;not null" json:"tenantId"`
	SelfOrderID   uint64    `gorm:"index;not null" json:"selfOrderId"`
	PimSkuID      uint64    `gorm:"index;default:0" json:"pimSkuId"`
	SkuCode       string    `gorm:"size:64" json:"skuCode"`
	ProductName   string    `gorm:"size:512" json:"productName"`
	SkuSpecs      string    `gorm:"size:256" json:"skuSpecs"`
	PicURL        string    `gorm:"size:512" json:"picUrl"`
	Qty           int       `gorm:"not null;default:1" json:"qty"`
	SaleUnitPrice float64   `gorm:"type:decimal(12,2);not null;default:0" json:"saleUnitPrice"`
	SaleAmount    float64   `gorm:"type:decimal(14,2);not null;default:0" json:"saleAmount"`
	InvSkuID      uint64    `gorm:"index;default:0" json:"invSkuId"`
	InvSkuCode    string    `gorm:"size:64" json:"invSkuCode"`
	CostUnitPrice float64   `gorm:"type:decimal(12,2);not null;default:0" json:"costUnitPrice"`
	CostAmount    float64   `gorm:"type:decimal(14,2);not null;default:0" json:"costAmount"`
	RefSoID        uint64    `gorm:"index;default:0" json:"refSoId"`
	RefOrderItemID uint64    `gorm:"index;default:0" json:"refOrderItemId"`
	RefOrderNo     string    `gorm:"size:64;index" json:"refOrderNo"`
	Remark         string    `gorm:"type:text" json:"remark"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (SelfOrderItem) TableName() string { return "self_order_items" }

type SelfShipment struct {
	ID                  uint64     `gorm:"primaryKey" json:"id"`
	TenantID            uint64     `gorm:"index;not null" json:"tenantId"`
	SelfOrderID         uint64     `gorm:"index;not null" json:"selfOrderId"`
	ShipmentNo          string     `gorm:"size:32;not null" json:"shipmentNo"`
	Status              string     `gorm:"size:32;not null;default:shipped" json:"status"`
	CarrierCode         string     `gorm:"size:32" json:"carrierCode"`
	CarrierName         string     `gorm:"size:64" json:"carrierName"`
	TrackingNo          string     `gorm:"size:64" json:"trackingNo"`
	ShippedAt           *time.Time `json:"shippedAt"`
	ExpectedArrivalDate *time.Time `json:"expectedArrivalDate"`
	DeliveredAt         *time.Time `json:"deliveredAt"`
	ReceiverName        string     `gorm:"size:64" json:"receiverName"`
	ReceiverPhone       string     `gorm:"size:32" json:"receiverPhone"`
	ReceiverAddress     string     `gorm:"size:255" json:"receiverAddress"`
	CallbackOK          bool       `gorm:"default:false" json:"callbackOk"`
	StockDeducted       bool       `gorm:"default:false" json:"stockDeducted"`
	Remark              string     `gorm:"type:text" json:"remark"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	Items               []SelfShipmentItem `gorm:"foreignKey:ShipmentID" json:"items,omitempty"`
}

func (SelfShipment) TableName() string { return "self_shipments" }

type SelfShipmentItem struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	TenantID        uint64    `gorm:"index;not null" json:"tenantId"`
	ShipmentID      uint64    `gorm:"index;not null" json:"shipmentId"`
	SelfOrderItemID uint64    `gorm:"index;not null" json:"selfOrderItemId"`
	Qty             int       `gorm:"not null" json:"qty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (SelfShipmentItem) TableName() string { return "self_shipment_items" }

type SelfAttachment struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `gorm:"index;not null" json:"tenantId"`
	SelfOrderID uint64    `gorm:"index;not null" json:"selfOrderId"`
	PaymentID   uint64    `gorm:"index" json:"paymentId"`
	ShipmentID  uint64    `gorm:"index" json:"shipmentId"`
	FileType    string    `gorm:"size:32;not null" json:"fileType"`
	FileName    string    `gorm:"size:255;not null" json:"fileName"`
	FileURL     string    `gorm:"size:512;not null" json:"fileUrl"`
	UploadedBy  uint64    `json:"uploadedBy"`
	Remark      string    `gorm:"type:text" json:"remark"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (SelfAttachment) TableName() string { return "self_attachments" }
