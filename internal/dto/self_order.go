package dto

// SelfOrderItemInput 创建/更新自营单明细。
type SelfOrderItemInput struct {
	PimSkuID       uint64  `json:"pimSkuId"`
	SkuCode        string  `json:"skuCode"`
	ProductName    string  `json:"productName"`
	SkuSpecs       string  `json:"skuSpecs"`
	PicURL         string  `json:"picUrl"`
	Qty            int     `json:"qty"`
	SaleUnitPrice  float64 `json:"saleUnitPrice"`
	SaleAmount     float64 `json:"saleAmount"`
	InvSkuID       uint64  `json:"invSkuId"`
	InvSkuCode     string  `json:"invSkuCode"`
	CostUnitPrice  float64 `json:"costUnitPrice"`
	CostAmount     float64 `json:"costAmount"`
	RefSoID        uint64  `json:"refSoId"`
	RefOrderItemID uint64  `json:"refOrderItemId"`
	RefOrderNo     string  `json:"refOrderNo"`
	Remark         string  `json:"remark"`
}

// SelfOrderInput OrderCore / 管理端创建自营单。
type SelfOrderInput struct {
	WarehouseID   uint64               `json:"warehouseId"`
	RefSoID       uint64               `json:"refSoId"`
	RefTraceID    string               `json:"refTraceId"`
	SaleAmount    float64              `json:"saleAmount"`
	BuyerName     string               `json:"buyerName"`
	BuyerPhone    string               `json:"buyerPhone"`
	Address       string               `json:"address"`
	Remark        string               `json:"remark"`
	SourceChannel string               `json:"sourceChannel"`
	Platform      string               `json:"platform"`
	ShopName      string               `json:"shopName"`
	// ManualSourceName 手工单订单来源名称
	ManualSourceName string               `json:"manualSourceName"`
	BuyerRemark   string               `json:"buyerRemark"`
	SellerRemark  string               `json:"sellerRemark"`
	FenFaRemark   string               `json:"fenFaRemark"`
	PrinterRemark string               `json:"printerRemark"`
	OrderedAt     string               `json:"orderedAt"`
	// CreatedAt 可选：创建自营单时间；OrderCore 分配时传入 allocatedAt，历史回填同理
	CreatedAt string `json:"createdAt,omitempty"`
	// PayStatus 可选：paid 时创建即为已付款（电商订单由 OrderCore 传入）
	PayStatus string `json:"payStatus"`
	PaidAt    string `json:"paidAt"`
	Items     []SelfOrderItemInput `json:"items"`
}

type SelfOrderListItem struct {
	ID            uint64  `json:"id"`
	SoNo          string  `json:"soNo"`
	Status        string  `json:"status"`
	WarehouseID   uint64  `json:"warehouseId"`
	RefSoID       uint64  `json:"refSoId"`
	RefTraceID    string  `json:"refTraceId"`
	SaleAmount    float64 `json:"saleAmount"`
	CostAmount    float64 `json:"costAmount"`
	PayStatus     string  `json:"payStatus"`
	PaidAt        string  `json:"paidAt,omitempty"`
	BuyerName     string  `json:"buyerName"`
	BuyerPhone    string  `json:"buyerPhone"`
	SourceChannel string  `json:"sourceChannel"`
	Platform      string  `json:"platform"`
	ShopName      string  `json:"shopName"`
	ManualSourceName string `json:"manualSourceName"`
	BuyerRemark   string  `json:"buyerRemark"`
	SellerRemark  string  `json:"sellerRemark"`
	FenFaRemark   string  `json:"fenFaRemark"`
	PrinterRemark string  `json:"printerRemark"`
	SkuSpecs      string  `json:"skuSpecs"` // 明细规格汇总
	ItemCount     int     `json:"itemCount"`
	StockDeducted bool    `json:"stockDeducted"`
	StockError    string  `json:"stockError"`
	OrderedAt     string  `json:"orderedAt,omitempty"`
	ShippedAt     string  `json:"shippedAt,omitempty"`
	CreatedAt     string  `json:"createdAt"`
}

type SelfOrderItemDTO struct {
	ID             uint64  `json:"id"`
	PimSkuID       uint64  `json:"pimSkuId"`
	SkuCode        string  `json:"skuCode"`
	ProductName    string  `json:"productName"`
	SkuSpecs       string  `json:"skuSpecs"`
	PicURL         string  `json:"picUrl"`
	Qty            int     `json:"qty"`
	SaleUnitPrice  float64 `json:"saleUnitPrice"`
	SaleAmount     float64 `json:"saleAmount"`
	InvSkuID       uint64  `json:"invSkuId"`
	InvSkuCode     string  `json:"invSkuCode"`
	CostUnitPrice  float64 `json:"costUnitPrice"`
	CostAmount     float64 `json:"costAmount"`
	RefSoID        uint64  `json:"refSoId"`
	RefOrderItemID uint64  `json:"refOrderItemId"`
	RefOrderNo     string  `json:"refOrderNo"`
	ParentSelfOrderItemID uint64 `json:"parentSelfOrderItemId,omitempty"`
	SplitKind             string `json:"splitKind,omitempty"`
	ShipPlanLineID        uint64 `json:"shipPlanLineId,omitempty"`
	Remark         string  `json:"remark"`
}

type SelfOrderDetail struct {
	ID            uint64             `json:"id"`
	SoNo          string             `json:"soNo"`
	Status        string             `json:"status"`
	WarehouseID   uint64             `json:"warehouseId"`
	RefSoID       uint64             `json:"refSoId"`
	RefTraceID    string             `json:"refTraceId"`
	SaleAmount    float64            `json:"saleAmount"`
	CostAmount    float64            `json:"costAmount"`
	PayStatus     string             `json:"payStatus"`
	PaidAt        string             `json:"paidAt,omitempty"`
	BuyerName     string             `json:"buyerName"`
	BuyerPhone    string             `json:"buyerPhone"`
	Address       string             `json:"address"`
	Remark        string             `json:"remark"`
	SourceChannel string             `json:"sourceChannel"`
	Platform      string             `json:"platform"`
	ShopName      string             `json:"shopName"`
	ManualSourceName string          `json:"manualSourceName"`
	BuyerRemark   string             `json:"buyerRemark"`
	SellerRemark  string             `json:"sellerRemark"`
	FenFaRemark   string             `json:"fenFaRemark"`
	PrinterRemark string             `json:"printerRemark"`
	StockDeducted bool               `json:"stockDeducted"`
	StockError    string             `json:"stockError"`
	OrderedAt     string             `json:"orderedAt,omitempty"`
	ShippedAt     string             `json:"shippedAt,omitempty"`
	CompletedAt   string             `json:"completedAt,omitempty"`
	CreatedAt     string             `json:"createdAt"`
	UpdatedAt     string             `json:"updatedAt"`
	Items         []SelfOrderItemDTO `json:"items"`
}

type BindInvSkuInput struct {
	InvSkuID      uint64  `json:"invSkuId" binding:"required"`
	InvSkuCode    string  `json:"invSkuCode"`
	CostUnitPrice float64 `json:"costUnitPrice"`
}

type UpdateItemCostInput struct {
	CostUnitPrice float64 `json:"costUnitPrice"`
}

type SelfShipInput struct {
	ExpressCompany string `json:"expressCompany" binding:"required"`
	ExpressNo      string `json:"expressNo" binding:"required"`
	Remark         string `json:"remark"`
	Callback       *bool  `json:"callback"` // 默认 true：回传订单中心
}

type SelfCancelByRefInput struct {
	RefSoID uint64 `json:"refSoId" binding:"required"`
	Reason  string `json:"reason"`
}

type SelfDeleteByRefInput struct {
	RefSoID uint64 `json:"refSoId" binding:"required"`
}

// SyncSplitItemsByRefSoInput 订单中心拆分计划同步到关联自营单。
type SyncSplitItemsByRefSoInput struct {
	RefSoID uint64                 `json:"refSoId" binding:"required"`
	Mode    string                 `json:"mode"`
	Lines   []SyncSplitItemLineIn  `json:"lines"`
}

type SyncSplitItemLineIn struct {
	RefOrderItemID       uint64 `json:"refOrderItemId"`
	ParentRefOrderItemID uint64 `json:"parentRefOrderItemId"`
	SkuName              string `json:"skuName"`
	Qty                  int    `json:"qty"`
	ShipPlanLineID       uint64 `json:"shipPlanLineId"`
	SplitKind            string `json:"splitKind"`
}

type SyncSplitItemsByRefSoResult struct {
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

type SelfCancelInput struct {
	Reason string `json:"reason"`
}


type SelfShipmentItemInput struct {
	SelfOrderItemID uint64 `json:"selfOrderItemId" binding:"required"`
	Qty             int    `json:"qty" binding:"required,min=1"`
}

type SelfShipmentCreateInput struct {
	CarrierCode         string                  `json:"carrierCode"`
	CarrierName         string                  `json:"carrierName"`
	TrackingNo          string                  `json:"trackingNo"`
	ExpectedArrivalDate string                  `json:"expectedArrivalDate"`
	ReceiverName        string                  `json:"receiverName"`
	ReceiverPhone       string                  `json:"receiverPhone"`
	ReceiverAddress     string                  `json:"receiverAddress"`
	Remark              string                  `json:"remark"`
	Callback            *bool                   `json:"callback"`
	Items               []SelfShipmentItemInput `json:"items"`
}

type SelfShipmentStatusInput struct {
	Status string `json:"status" binding:"required"`
}

type SelfShipmentItemDTO struct {
	ID              uint64 `json:"id"`
	SelfOrderItemID uint64 `json:"selfOrderItemId"`
	Qty             int    `json:"qty"`
}

type SelfShipmentDTO struct {
	ID                  uint64                `json:"id"`
	SelfOrderID         uint64                `json:"selfOrderId"`
	ShipmentNo          string                `json:"shipmentNo"`
	Status              string                `json:"status"`
	CarrierCode         string                `json:"carrierCode"`
	CarrierName         string                `json:"carrierName"`
	TrackingNo          string                `json:"trackingNo"`
	ShippedAt           string                `json:"shippedAt,omitempty"`
	ExpectedArrivalDate string                `json:"expectedArrivalDate,omitempty"`
	DeliveredAt         string                `json:"deliveredAt,omitempty"`
	CallbackOK          bool                  `json:"callbackOk"`
	StockDeducted       bool                  `json:"stockDeducted"`
	ReceiverName        string                `json:"receiverName"`
	ReceiverPhone       string                `json:"receiverPhone"`
	ReceiverAddress     string                `json:"receiverAddress"`
	Remark              string                `json:"remark"`
	Items               []SelfShipmentItemDTO `json:"items"`
	CreatedAt           string                `json:"createdAt"`
}

type SelfAttachmentInput struct {
	PaymentID  uint64 `json:"paymentId"`
	ShipmentID uint64 `json:"shipmentId"`
	FileType   string `json:"fileType" binding:"required"`
	FileName   string `json:"fileName" binding:"required"`
	FileURL    string `json:"fileUrl" binding:"required"`
	Remark     string `json:"remark"`
}

type SelfAttachmentDTO struct {
	ID          uint64 `json:"id"`
	SelfOrderID uint64 `json:"selfOrderId"`
	PaymentID   uint64 `json:"paymentId"`
	ShipmentID  uint64 `json:"shipmentId"`
	FileType    string `json:"fileType"`
	FileName    string `json:"fileName"`
	FileURL     string `json:"fileUrl"`
	UploadedBy  uint64 `json:"uploadedBy"`
	Remark      string `json:"remark"`
	CreatedAt   string `json:"createdAt"`
}

type SelfPaymentInput struct {
	PayAmount    float64 `json:"payAmount" binding:"required"`
	PayMethod    string  `json:"payMethod"`
	PayAccount   string  `json:"payAccount"`
	PayeeAccount string  `json:"payeeAccount"`
	PayeeName    string  `json:"payeeName"`
	PayStatus    string  `json:"payStatus"`
	PaidAt       string  `json:"paidAt"`
	Remark       string  `json:"remark"`
}

type SelfPaymentDetail struct {
	ID           uint64  `json:"id"`
	SelfOrderID  uint64  `json:"selfOrderId"`
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
