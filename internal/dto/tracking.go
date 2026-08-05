package dto

type ShipmentItemInput struct {
	DistOrderItemID uint64 `json:"distOrderItemId" binding:"required"`
	Qty      int    `json:"qty" binding:"required,min=1"`
}

type ShipmentInput struct {
	CarrierCode         string              `json:"carrierCode"`
	CarrierName         string              `json:"carrierName"`
	TrackingNo          string              `json:"trackingNo"`
	ExpectedArrivalDate string              `json:"expectedArrivalDate"`
	ShipFromAddressID   uint64              `json:"shipFromAddressId"`
	ReceiverName        string              `json:"receiverName"`
	ReceiverPhone       string              `json:"receiverPhone"`
	ReceiverAddress     string              `json:"receiverAddress"`
	Remark              string              `json:"remark"`
	Items               []ShipmentItemInput `json:"items"`
}

type ShipmentStatusInput struct {
	Status string `json:"status" binding:"required"`
}

type ShipmentItemDetail struct {
	ID       uint64 `json:"id"`
	DistOrderItemID uint64 `json:"distOrderItemId"`
	SkuID    uint64 `json:"skuId"`
	Qty      int    `json:"qty"`
}

type ShipmentDetail struct {
	ID                  uint64               `json:"id"`
	PoID                uint64               `json:"distOrderId"`
	ShipmentNo          string               `json:"shipmentNo"`
	Status              string               `json:"status"`
	CarrierCode         string               `json:"carrierCode"`
	CarrierName         string               `json:"carrierName"`
	TrackingNo          string               `json:"trackingNo"`
	ShippedAt           string               `json:"shippedAt,omitempty"`
	ExpectedArrivalDate string               `json:"expectedArrivalDate,omitempty"`
	DeliveredAt         string               `json:"deliveredAt,omitempty"`
	ReceiverName        string               `json:"receiverName"`
	ReceiverPhone       string               `json:"receiverPhone"`
	ReceiverAddress     string               `json:"receiverAddress"`
	Remark              string               `json:"remark"`
	Items               []ShipmentItemDetail `json:"items"`
	CreatedAt           string               `json:"createdAt"`
}

type ReceiptInput struct {
	PayAmount    float64 `json:"payAmount" binding:"required"`
	PayMethod    string  `json:"payMethod"`
	PayAccount   string  `json:"payAccount"`
	PayeeAccount string  `json:"payeeAccount"`
	PayeeName    string  `json:"payeeName"`
	PayStatus    string  `json:"payStatus"`
	PaidAt       string  `json:"paidAt"`
	Remark       string  `json:"remark"`
}

type ReceiptDetail struct {
	ID           uint64  `json:"id"`
	PoID         uint64  `json:"distOrderId"`
	PayAmount    float64 `json:"payAmount"`
	PayMethod    string  `json:"payMethod"`
	PayAccount   string  `json:"payAccount"`
	PayeeAccount string  `json:"payeeAccount"`
	PayeeName    string  `json:"payeeName"`
	PayStatus    string  `json:"payStatus"`
	PaidAt       string  `json:"paidAt,omitempty"`
	Remark       string  `json:"remark"`
	CreatedAt    string  `json:"createdAt"`
}

type AttachmentInput struct {
	PaymentID  uint64 `json:"paymentId"`
	ShipmentID uint64 `json:"shipmentId"`
	FileType   string `json:"fileType" binding:"required"`
	FileName   string `json:"fileName" binding:"required"`
	FileURL    string `json:"fileUrl" binding:"required"`
	Remark     string `json:"remark"`
}

type AttachmentDetail struct {
	ID         uint64 `json:"id"`
	PoID       uint64 `json:"distOrderId"`
	PaymentID  uint64 `json:"paymentId"`
	ShipmentID uint64 `json:"shipmentId"`
	FileType   string `json:"fileType"`
	FileName   string `json:"fileName"`
	FileURL    string `json:"fileUrl"`
	UploadedBy uint64 `json:"uploadedBy"`
	Remark     string `json:"remark"`
	CreatedAt  string `json:"createdAt"`
}

type UploadResult struct {
	URL      string `json:"url"`
	FileName string `json:"fileName"`
}

type SyncShipmentsFromOrdersInput struct {
	RefSoID uint64 `json:"refSoId"` // 可选：仅同步某一销售单
}

type SyncShipmentsFromOrdersResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}
