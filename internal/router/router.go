package router

import (
	"path/filepath"

	"selfcore/admin"
	adminmw "selfcore/admin/middleware"
	"selfcore/internal/config"
	"selfcore/internal/integrations/ordercore"
	"selfcore/internal/integrations/productcore"
	"selfcore/internal/integrations/shippingcore"
	"selfcore/internal/integrations/warehousecore"
	jwtmgr "selfcore/internal/pkg/jwt"
	"selfcore/internal/repo"
	"selfcore/internal/service"
	"selfcore/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg))

	if cfg.Storage.Driver == "local" || cfg.Storage.Driver == "" {
		uploadDir := filepath.Join(cfg.Storage.LocalPath, cfg.Storage.Prefix)
		r.Static("/uploads", uploadDir)
	}

	store, err := storage.New(&cfg.Storage)
	if err != nil {
		panic(err)
	}

	repos := repo.New(db)
	distributorSvc := service.NewDistributorService(repos)
	priceSvc := service.NewPriceService(repos)
	doSvc := service.NewDistOrderService(repos)
	pcClient := productcore.NewClient(cfg.Integrations.ProductCoreAPIURL)
	ocClient := ordercore.NewClient(cfg.Integrations.OrderCoreAPIURL)
	whClient := warehousecore.NewClient(cfg.Integrations.WarehouseCoreAPIURL)
	shipClient := shippingcore.NewClient(cfg.Integrations.ShippingCoreAPIURL)
	trackSvc := service.NewTrackingService(repos)
	dashSvc := service.NewDashboardService(repos)
	selfOrderSvc := service.NewSelfOrderService(repos, whClient, ocClient, shipClient)

	distributorH := admin.NewDistributorHandler(distributorSvc)
	priceH := admin.NewPriceHandler(priceSvc)
	doH := admin.NewDistOrderHandler(doSvc)
	trackH := admin.NewTrackingHandler(trackSvc, store)
	skuH := admin.NewProductSkuHandler(pcClient)
	dashH := admin.NewDashboardHandler(dashSvc)
	orderH := admin.NewOrderHandler(ocClient)
	selfOrderH := admin.NewSelfOrderHandler(selfOrderSvc)
	whH := admin.NewWarehouseHandler(whClient)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "selfcore"})
	})

	v1 := r.Group("/api/v1")
	photoH := admin.NewPhotoUploadHandler(store)
	mobile := v1.Group("/mobile")
	{
		mobile.GET("/photo-upload/:token", photoH.MobileGet)
		mobile.POST("/photo-upload/:token", photoH.MobileUpload)
	}

	adminGroup := v1.Group("/admin")
	jwtMgr := jwtmgr.NewManager(cfg.Auth.JWTSecret)
	adminGroup.Use(adminmw.AdminAuth(&cfg.Auth, jwtMgr))
	adminGroup.POST("/photo-upload-sessions", photoH.CreateSession)
	adminGroup.GET("/photo-upload-sessions/:token", photoH.GetSession)
	admin.RegisterRoutes(adminGroup, distributorH, priceH, doH, trackH, skuH, dashH, orderH, selfOrderH, whH)

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.CORS.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin == ""
		for _, o := range origins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
