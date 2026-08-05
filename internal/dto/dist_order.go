package dto

type DistOrderItemInput struct {
	SkuID           uint64  `json:"skuId"`
	OfferID         uint64  `json:"offerId"`
	ProductName     string  `json:"productName"`
	SkuCode         string  `json:"skuCode"`
	SkuSpecs        string  `json:"skuSpecs"`
	PicURL          string  `json:"picUrl"`
	DistributorSkuCode string  `json:"distributorSkuCode"`
	Qty             int     `json:"qty" binding:"required,min=1"`
	SaleUnitPrice   float64 `json:"saleUnitPrice"`
	SaleAmount      float64 `json:"saleAmount"`
	UnitPrice       float64 `json:"unitPrice"`
	RefSoID         uint64  `json:"refSoId"`
	RefOrderNo      string  `json:"refOrderNo"`
	Remark          string  `json:"remark"`
}

type DistOrderInput struct {
	DistributorID          uint64                   `json:"distributorId" binding:"required"`
	FulfillmentType     string                   `json:"fulfillmentType"`
	Currency            string                   `json:"currency"`
	ExpectedArrivalDate string                   `json:"expectedArrivalDate"`
	WarehouseID         uint64                   `json:"warehouseId"`
	RefSoID             uint64                   `json:"refSoId"`
	RefTraceID          string                   `json:"refTraceId"`
	OrderedAt           string                   `json:"orderedAt"` // 采购时间；手工新建可改，默认当天
	SaleAmount          float64                  `json:"saleAmount"` // 销售侧订单总实付
	Remark              string                   `json:"remark"`
	Items               []DistOrderItemInput `json:"items" binding:"required,min=1,dive"`
}

type DistOrderItemDetail struct {
	ID              uint64  `json:"id"`
	SkuID           uint64  `json:"skuId"`
	OfferID         uint64  `json:"offerId"`
	ProductName     string  `json:"productName"`
	SkuCode         string  `json:"skuCode"`
	SkuSpecs        string  `json:"skuSpecs"`
	PicURL          string  `json:"picUrl"`
	DistributorSkuCode string  `json:"distributorSkuCode"`
	Qty             int     `json:"qty"`
	SaleUnitPrice   float64 `json:"saleUnitPrice"`
	SaleAmount      float64 `json:"saleAmount"`
	UnitPrice       float64 `json:"unitPrice"`
	LineAmount      float64 `json:"lineAmount"`
	ReceivedQty     int     `json:"receivedQty"`
	RefSoID         uint64  `json:"refSoId,omitempty"`
	RefOrderNo      string  `json:"refOrderNo,omitempty"`
	Cancelled       bool    `json:"cancelled"`
	Remark          string  `json:"remark"`
}

type DistOrderDetail struct {
	ID                  uint64                    `json:"id"`
	DistNo                string                    `json:"distNo"`
	DistributorID          uint64                    `json:"distributorId"`
	DistributorName        string                    `json:"distributorName"`
	DistributorCode        string                    `json:"distributorCode"`
	Status              string                    `json:"status"`
	TotalAmount         float64                   `json:"totalAmount"`
	SaleAmount          float64                   `json:"saleAmount"`
	Currency            string                    `json:"currency"`
	ExpectedArrivalDate string                    `json:"expectedArrivalDate,omitempty"`
	WarehouseID         uint64                    `json:"warehouseId"`
	FulfillmentType     string                    `json:"fulfillmentType"`
	RefSoID             uint64                    `json:"refSoId"`
	RefTraceID          string                    `json:"refTraceId"`
	BuyerID             uint64                    `json:"buyerId"`
	BuyerName           string                    `json:"buyerName"`
	PayStatus           string                    `json:"payStatus"`
	Remark              string                    `json:"remark"`
	OrderedAt           string                    `json:"orderedAt,omitempty"`
	CompletedAt         string                    `json:"completedAt,omitempty"`
	CreatedAt           string                    `json:"createdAt"`
	Items               []DistOrderItemDetail `json:"items"`
}

type DistOrderListItem struct {
	ID              uint64  `json:"id"`
	DistNo            string  `json:"distNo"`
	DistributorID      uint64  `json:"distributorId"`
	DistributorName    string  `json:"distributorName"`
	Status          string  `json:"status"`
	PayStatus       string  `json:"payStatus"`
	FulfillmentType string  `json:"fulfillmentType"`
	TotalAmount     float64 `json:"totalAmount"`
	SaleAmount      float64 `json:"saleAmount"`
	Currency        string  `json:"currency"`
	ItemCount       int     `json:"itemCount"`
	SkuSpecs        string  `json:"skuSpecs"` // 明细规格汇总（同规格累加数量，如「规格 x2」；分号分隔）
	RefSoID         uint64  `json:"refSoId,omitempty"`
	RefTraceID      string  `json:"refTraceId,omitempty"`
	OrderedAt       string  `json:"orderedAt,omitempty"`
	CreatedAt       string  `json:"createdAt"`
}

type MergeDistOrdersInput struct {
	SourceDistOrderIDs []uint64 `json:"sourceDistOrderIds" binding:"required,min=2"`
	TargetDistOrderID  uint64   `json:"targetDistOrderId"` // 可选；默认取 sourceDistOrderIds[0]
}

type MergeDistOrdersResult struct {
	*DistOrderDetail
	MergedFromDistNos []string `json:"mergedFromDistNos"`
	Relinked        int64    `json:"relinked"`
}

// DetachSalesOrderInput 从代发单中撤回某笔销售单（明细标灰作废 + 备注说明）。
type DetachSalesOrderInput struct {
	DistNo     string `json:"distNo"`
	OrderNo  string `json:"orderNo"`
	SoID     uint64 `json:"soId"`
	Reason   string `json:"reason"`
}

// UpdateDistOrderItemPriceInput 更新采购明细单价（已下单未付款也可改）。
type UpdateDistOrderItemPriceInput struct {
	ItemID    uint64  `json:"itemId" binding:"required"`
	UnitPrice float64 `json:"unitPrice"`
}

type UpdateDistOrderItemPricesInput struct {
	Items []UpdateDistOrderItemPriceInput `json:"items" binding:"required,min=1,dive"`
}
