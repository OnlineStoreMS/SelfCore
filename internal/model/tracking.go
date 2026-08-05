package model

import "time"

type DistShipment struct {
	ID                  uint64     `gorm:"primaryKey" json:"id"`
	TenantID            uint64     `gorm:"index;not null" json:"tenantId"`
	DistOrderID                uint64     `gorm:"index;not null" json:"distOrderId"`
	ShipmentNo          string     `gorm:"size:32;not null" json:"shipmentNo"`
	Status              string     `gorm:"size:32;not null;default:pending" json:"status"`
	CarrierCode         string     `gorm:"size:32" json:"carrierCode"`
	CarrierName         string     `gorm:"size:64" json:"carrierName"`
	TrackingNo          string     `gorm:"size:64" json:"trackingNo"`
	ShippedAt           *time.Time `json:"shippedAt"`
	ExpectedArrivalDate *time.Time `json:"expectedArrivalDate"`
	DeliveredAt         *time.Time `json:"deliveredAt"`
	ShipFromAddressID   uint64     `json:"shipFromAddressId"`
	ReceiverName        string     `gorm:"size:64" json:"receiverName"`
	ReceiverPhone       string     `gorm:"size:32" json:"receiverPhone"`
	ReceiverAddress     string     `gorm:"size:255" json:"receiverAddress"`
	Remark              string     `gorm:"type:text" json:"remark"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	Items               []DistShipmentItem `gorm:"foreignKey:ShipmentID" json:"items,omitempty"`
}

func (DistShipment) TableName() string { return "dist_shipments" }

type DistShipmentItem struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	TenantID   uint64    `gorm:"index;not null" json:"tenantId"`
	ShipmentID uint64    `gorm:"index;not null" json:"shipmentId"`
	DistOrderItemID   uint64    `gorm:"index;not null" json:"distOrderItemId"`
	Qty        int       `gorm:"not null" json:"qty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (DistShipmentItem) TableName() string { return "dist_shipment_items" }

type DistReceipt struct {
	ID            uint64     `gorm:"primaryKey" json:"id"`
	TenantID      uint64     `gorm:"index;not null" json:"tenantId"`
	DistOrderID          uint64     `gorm:"index;not null" json:"distOrderId"`
	PayAmount     float64    `gorm:"type:decimal(14,2);not null" json:"payAmount"`
	PayMethod     string     `gorm:"size:32" json:"payMethod"`
	PayAccount    string     `gorm:"size:128" json:"payAccount"`
	PayeeAccount  string     `gorm:"size:128" json:"payeeAccount"`
	PayeeName     string     `gorm:"size:128" json:"payeeName"`
	PayStatus     string     `gorm:"size:16;default:paid" json:"payStatus"`
	PaidAt        *time.Time `json:"paidAt"`
	Remark        string     `gorm:"type:text" json:"remark"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (DistReceipt) TableName() string { return "dist_receipts" }

type DistAttachment struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	TenantID   uint64    `gorm:"index;not null" json:"tenantId"`
	DistOrderID       uint64    `gorm:"index;not null" json:"distOrderId"`
	PaymentID  uint64    `json:"paymentId"`
	ShipmentID uint64    `gorm:"index" json:"shipmentId"`
	FileType   string    `gorm:"size:32;not null" json:"fileType"`
	FileName   string    `gorm:"size:255;not null" json:"fileName"`
	FileURL    string    `gorm:"size:512;not null" json:"fileUrl"`
	UploadedBy uint64    `json:"uploadedBy"`
	Remark     string    `gorm:"type:text" json:"remark"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (DistAttachment) TableName() string { return "dist_attachments" }
