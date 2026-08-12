package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, distributorH *DistributorHandler, priceH *PriceHandler, doH *DistOrderHandler, trackH *TrackingHandler, skuH *ProductSkuHandler, dashH *DashboardHandler, orderH *OrderHandler, selfOrderH *SelfOrderHandler, whH *WarehouseHandler) {
	g.GET("/dashboard/stats", dashH.Stats)
	g.GET("/dashboard/trend", dashH.Trend)

	g.GET("/orders/search", orderH.Search)
	g.POST("/orders/decrypt", orderH.Decrypt)
	g.GET("/orders/:id", orderH.Get)
	g.POST("/orders/:id/ship", orderH.Ship)

	g.GET("/self-orders", selfOrderH.List)
	g.GET("/self-orders/status-counts", selfOrderH.StatusCounts)
	g.POST("/self-orders", selfOrderH.Create)
	g.POST("/self-orders/cancel-by-ref-so", selfOrderH.CancelByRefSo)
	g.GET("/self-orders/:id", selfOrderH.Get)
	g.DELETE("/self-orders/:id", selfOrderH.Delete)
	g.POST("/self-orders/:id/submit", selfOrderH.Submit)
	g.POST("/self-orders/:id/mark-paid", selfOrderH.MarkPaid)
	g.POST("/self-orders/:id/complete", selfOrderH.Complete)
	g.POST("/self-orders/:id/ship", selfOrderH.Ship)
	g.POST("/self-orders/:id/retry-callback", selfOrderH.RetryCallback)
	g.POST("/self-orders/:id/retry-stock", selfOrderH.RetryStock)
	g.POST("/self-orders/:id/cancel", selfOrderH.Cancel)
	g.GET("/self-orders/:id/shipments", selfOrderH.ListShipments)
	g.POST("/self-orders/:id/shipments", selfOrderH.CreateShipment)
	g.POST("/self-orders/:id/shipments/sync-from-orders", selfOrderH.SyncShipmentsFromOrders)
	g.PATCH("/self-orders/:id/shipments/:shipmentId/status", selfOrderH.UpdateShipmentStatus)
	g.DELETE("/self-orders/:id/shipments/:shipmentId", selfOrderH.DeleteShipment)
	g.GET("/self-orders/:id/payments", selfOrderH.ListPayments)
	g.POST("/self-orders/:id/payments", selfOrderH.CreatePayment)
	g.PUT("/self-orders/:id/payments/:paymentId", selfOrderH.UpdatePayment)
	g.DELETE("/self-orders/:id/payments/:paymentId", selfOrderH.DeletePayment)
	g.GET("/self-orders/:id/attachments", selfOrderH.ListAttachments)
	g.POST("/self-orders/:id/attachments", selfOrderH.CreateAttachment)
	g.DELETE("/self-orders/:id/attachments/:attachmentId", selfOrderH.DeleteAttachment)
	g.PUT("/self-order-items/:itemId/inv-sku", selfOrderH.BindInvSku)
	g.PUT("/self-order-items/:itemId/cost", selfOrderH.UpdateItemCost)

	g.GET("/warehouse-skus/search", whH.SearchSkus)
	g.GET("/warehouses", whH.ListWarehouses)

	g.GET("/distributor-categories", distributorH.ListCategories)
	g.POST("/distributor-categories", distributorH.CreateCategory)
	g.PUT("/distributor-categories/:id", distributorH.UpdateCategory)
	g.DELETE("/distributor-categories/:id", distributorH.DeleteCategory)

	g.GET("/distributors", distributorH.List)
	g.POST("/distributors", distributorH.Create)
	g.GET("/distributors/:id", distributorH.Get)
	g.PUT("/distributors/:id", distributorH.Update)
	g.DELETE("/distributors/:id", distributorH.Delete)

	g.GET("/distributors/:id/addresses", distributorH.ListAddresses)
	g.POST("/distributors/:id/addresses", distributorH.CreateAddress)
	g.PUT("/distributors/:id/addresses/:addressId", distributorH.UpdateAddress)
	g.DELETE("/distributors/:id/addresses/:addressId", distributorH.DeleteAddress)

	g.GET("/distributors/:id/payment-accounts", distributorH.ListPaymentAccounts)
	g.POST("/distributors/:id/payment-accounts", distributorH.CreateReceiptAccount)
	g.PUT("/distributors/:id/payment-accounts/:accountId", distributorH.UpdateReceiptAccount)
	g.DELETE("/distributors/:id/payment-accounts/:accountId", distributorH.DeleteReceiptAccount)

	g.GET("/distributors/:id/payment-qrs", distributorH.ListPaymentQRs)
	g.POST("/distributors/:id/payment-qrs", distributorH.CreateReceiptQR)
	g.PUT("/distributors/:id/payment-qrs/:qrId", distributorH.UpdateReceiptQR)
	g.DELETE("/distributors/:id/payment-qrs/:qrId", distributorH.DeleteReceiptQR)

	g.GET("/sku-prices", priceH.List)
	g.POST("/sku-prices", priceH.Create)
	g.GET("/sku-prices/:id", priceH.Get)
	g.PUT("/sku-prices/:id", priceH.Update)
	g.DELETE("/sku-prices/:id", priceH.Delete)

	g.GET("/skus/:id/wholesale-options", priceH.SupplyOptions)

	g.GET("/product-skus/search", skuH.Search)
	g.GET("/products/search", skuH.SearchProducts)
	g.GET("/products/:id/skus", skuH.GetProductSkus)

	g.GET("/dist-orders", doH.List)
	g.POST("/dist-orders", doH.Create)
	g.POST("/dist-orders/merge", doH.Merge)
	g.POST("/dist-orders/detach-sales-order", doH.DetachSalesOrder)
	g.GET("/dist-orders/:id", doH.Get)
	g.PUT("/dist-orders/:id", doH.Update)
	g.PUT("/dist-orders/:id/item-prices", doH.UpdateItemPrices)
	g.POST("/dist-orders/:id/sync-purchase-prices", doH.SyncPurchasePrices)
	g.DELETE("/dist-orders/:id", doH.Delete)
	g.POST("/dist-orders/:id/submit", doH.Submit)
	g.POST("/dist-orders/:id/mark-paid", doH.MarkPaid)
	g.POST("/dist-orders/:id/complete", doH.Complete)
	g.POST("/dist-orders/:id/cancel", doH.Cancel)

	g.POST("/upload", trackH.Upload)

	g.GET("/dist-orders/:id/shipments", trackH.ListShipments)
	g.POST("/dist-orders/:id/shipments", trackH.CreateShipment)
	g.POST("/dist-orders/:id/shipments/sync-from-orders", trackH.SyncShipmentsFromOrders)
	g.PATCH("/dist-orders/:id/shipments/:shipmentId/status", trackH.UpdateShipmentStatus)
	g.DELETE("/dist-orders/:id/shipments/:shipmentId", trackH.DeleteShipment)

	g.GET("/dist-orders/:id/receipts", trackH.ListReceipts)
	g.POST("/dist-orders/:id/receipts", trackH.CreateReceipt)
	g.PUT("/dist-orders/:id/receipts/:receiptId", trackH.UpdateReceipt)
	g.DELETE("/dist-orders/:id/receipts/:receiptId", trackH.DeleteReceipt)

	g.GET("/dist-orders/:id/attachments", trackH.ListAttachments)
	g.POST("/dist-orders/:id/attachments", trackH.CreateAttachment)
	g.DELETE("/dist-orders/:id/attachments/:attachmentId", trackH.DeleteAttachment)
}
