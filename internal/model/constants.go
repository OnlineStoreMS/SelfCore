package model

const (
	DistStatusDraft           = "draft"
	DistStatusConfirmed       = "confirmed"
	DistStatusPaid            = "paid"
	DistStatusPartialShipped  = "partial_shipped" // 部分发货
	DistStatusShipped         = "shipped"         // 已发货（原 in_transit「运输中」）
	DistStatusPartialReceived = "partial_received"
	DistStatusCompleted       = "completed"
	DistStatusCancelled       = "cancelled"

	DistPayStatusUnpaid  = "unpaid"
	DistPayStatusPartial = "partial"
	DistPayStatusPaid    = "paid"

	DistFulfillmentWholesale = "wholesale"
	DistFulfillmentDropship  = "dropship"

	AddressTypeShip   = "ship"
	AddressTypeReturn = "return"

	ShipmentStatusPending   = "pending"
	ShipmentStatusShipped   = "shipped"
	ShipmentStatusInTransit = "in_transit"
	ShipmentStatusDelivered = "delivered"
	ShipmentStatusException = "exception"

	AttachmentTypeDistSalesOrder    = "dist_sales_order"
	AttachmentTypePaymentScreenshot = "payment_screenshot"
	AttachmentTypeShipmentPhoto     = "shipment_photo" // 发货记录 / 物流单号照片等
	AttachmentTypeContract          = "contract"
	AttachmentTypeOther             = "other"

	// 自营订单状态（对齐供应链：草稿→已下单→已付款→发货→完成）
	SelfOrderStatusDraft          = "draft"
	SelfOrderStatusOrdered        = "ordered"
	SelfOrderStatusPaid           = "paid"
	SelfOrderStatusPartialShipped = "partial_shipped"
	SelfOrderStatusShipped        = "shipped"
	SelfOrderStatusCompleted      = "completed"
	SelfOrderStatusCancelled      = "cancelled"

	// 拆分发货（对齐订单中心）
	SplitKindPartial = "partial"
	SplitKindFull    = "full"

	// SelfOrderStatusConfirmed 历史别名（已迁移为 ordered/paid，仅兼容旧数据读取）
	SelfOrderStatusConfirmed = "confirmed"
)
