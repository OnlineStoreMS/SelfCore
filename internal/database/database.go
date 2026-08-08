package database

import (
	"fmt"
	"os"
	"path/filepath"

	"selfcore/internal/config"
	"selfcore/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.PostgresDSN)
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.SQLitePath)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.DistributorCategory{},
		&model.Distributor{},
		&model.DistributorAddress{},
		&model.DistributorPaymentAccount{},
		&model.DistributorPaymentQR{},
		&model.SkuDistributorPrice{},
		&model.DistOrder{},
		&model.DistOrderItem{},
		&model.DistShipment{},
		&model.DistShipmentItem{},
		&model.DistReceipt{},
		&model.DistAttachment{},
		&model.SelfOrder{},
		&model.SelfOrderItem{},
		&model.SelfShipment{},
		&model.SelfShipmentItem{},
		&model.SelfPayment{},
		&model.SelfAttachment{},
	); err != nil {
		return err
	}
	_ = db.Exec(`UPDATE distributor_addresses SET address_type = 'ship' WHERE address_type IS NULL OR address_type = ''`).Error
	_ = db.Exec(`UPDATE dist_orders SET status = 'shipped' WHERE status = 'in_transit'`).Error
	_ = db.Exec(`UPDATE dist_orders SET status = 'confirmed' WHERE status = 'ordered'`).Error
	// 自营单：confirmed → 电商默认已付款，其余已下单
	_ = db.Exec(`
		UPDATE self_orders
		SET status = 'paid', pay_status = 'paid',
		    paid_at = COALESCE(paid_at, ordered_at, created_at)
		WHERE status = 'confirmed' AND source_channel = 'kdzs'
	`).Error
	_ = db.Exec(`
		UPDATE self_orders SET status = 'paid'
		WHERE status = 'confirmed' AND pay_status = 'paid'
	`).Error
	_ = db.Exec(`UPDATE self_orders SET status = 'ordered' WHERE status = 'confirmed'`).Error
	// 电商历史单补标已付款（创建时尚无付款态）
	_ = db.Exec(`
		UPDATE self_orders
		SET pay_status = 'paid',
		    paid_at = COALESCE(paid_at, ordered_at, created_at)
		WHERE source_channel = 'kdzs'
		  AND pay_status <> 'paid'
		  AND status NOT IN ('draft', 'cancelled')
	`).Error
	_ = db.Exec(`
		UPDATE self_orders SET status = 'paid'
		WHERE source_channel = 'kdzs' AND pay_status = 'paid' AND status = 'ordered'
	`).Error
	// 自营无到货环节：历史「已发货」视作已完成
	_ = db.Exec(`
		UPDATE self_orders
		SET status = 'completed',
		    completed_at = COALESCE(completed_at, shipped_at, updated_at)
		WHERE status = 'shipped'
	`).Error
	return ensureIndexes(db)
}

func ensureIndexes(db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "postgres":
		return db.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS idx_distributors_tenant_code ON distributors (tenant_id, code);
			CREATE INDEX IF NOT EXISTS idx_distributors_tenant_category ON distributors (tenant_id, category_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_distributor_categories_tenant_name ON distributor_categories (tenant_id, name);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_prices_tenant_sku_distributor_addr ON sku_distributor_prices (tenant_id, sku_id, distributor_id, ship_from_address_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_do_tenant_no ON dist_orders (tenant_id, dist_no);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_shipment_tenant_no ON dist_shipments (tenant_id, shipment_no);
			CREATE INDEX IF NOT EXISTS idx_do_ref_so ON dist_orders (tenant_id, ref_so_id);
			CREATE INDEX IF NOT EXISTS idx_do_ref_trace ON dist_orders (tenant_id, ref_trace_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_self_orders_tenant_no ON self_orders (tenant_id, so_no);
			CREATE INDEX IF NOT EXISTS idx_self_orders_ref_so ON self_orders (tenant_id, ref_so_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_self_shipments_tenant_no ON self_shipments (tenant_id, shipment_no);
		`).Error
	default:
		return nil
	}
}
