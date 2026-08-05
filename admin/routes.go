package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, distributorH *DistributorHandler, priceH *PriceHandler, doH *DistOrderHandler, trackH *TrackingHandler, skuH *ProductSkuHandler, dashH *DashboardHandler, orderH *OrderHandler) {
	g.GET("/dashboard/stats", dashH.Stats)
	g.GET("/dashboard/trend", dashH.Trend)

	g.GET("/orders/search", orderH.Search)
	g.POST("/orders/decrypt", orderH.Decrypt)
	g.GET("/orders/:id", orderH.Get)
	g.POST("/orders/:id/ship", orderH.Ship)

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
