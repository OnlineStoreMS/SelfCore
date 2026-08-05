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
)
