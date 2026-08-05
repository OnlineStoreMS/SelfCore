package repo

import "gorm.io/gorm"

type Repos struct {
	Distributor         *DistributorRepo
	DistributorCategory *DistributorCategoryRepo
	Offer               *PriceRepo
	DistOrder           *DistOrderRepo
	Shipment            *ShipmentRepo
	Payment             *ReceiptRepo
	Attachment          *AttachmentRepo
	Dashboard           *DashboardRepo
}

func New(db *gorm.DB) *Repos {
	return &Repos{
		Distributor:         NewDistributorRepo(db),
		DistributorCategory: NewDistributorCategoryRepo(db),
		Offer:               NewPriceRepo(db),
		DistOrder:           NewDistOrderRepo(db),
		Shipment:            NewShipmentRepo(db),
		Payment:             NewReceiptRepo(db),
		Attachment:          NewAttachmentRepo(db),
		Dashboard:           NewDashboardRepo(db),
	}
}
