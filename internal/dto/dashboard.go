package dto

// DashboardStats 工作台聚合统计
type DashboardStats struct {
	Workbench      DashboardWorkbench       `json:"workbench"`
	Distributor       DashboardDistributorStats   `json:"distributor"`
	DistOrder  DashboardPOStats         `json:"purchaseOrder"`
	Cost           DashboardCostStats       `json:"cost"`
	TopDistributors   []DashboardDistributorRank  `json:"topDistributors"`
	RecentOrders   []DistOrderListItem  `json:"recentOrders"`
	StatusBreakdown []DashboardStatusCount  `json:"statusBreakdown"`
}

type DashboardWorkbench struct {
	DropshipPO        int64 `json:"dropshipPO"`
	WholesalePO         int64 `json:"wholesalePO"`
	DraftPO           int64 `json:"draftPO"`
	ConfirmedPO         int64 `json:"confirmedPO"`
	UnpaidPO          int64 `json:"unpaidPO"`
	InTransitPO       int64 `json:"inTransitPO"`
	PartialReceivedPO int64 `json:"partialReceivedPO"`
	ActiveOffers      int64 `json:"activeOffers"`
	// 今日代发毛利：仅 total_amount > 0；毛利润 = sale_amount - total_amount
	TodayDropshipSaleAmount     float64 `json:"todayDropshipSaleAmount"`
	TodayDropshipWholesaleAmount float64 `json:"todayDropshipWholesaleAmount"`
	TodayDropshipProfit         float64 `json:"todayDropshipProfit"`
}

type DashboardDistributorStats struct {
	Total              int64 `json:"total"`
	Active             int64 `json:"active"`
	OfferCount         int64 `json:"offerCount"`
	OrderedThisMonth   int64 `json:"orderedThisMonth"`
}

type DashboardPOStats struct {
	Total       int64 `json:"total"`
	Draft       int64 `json:"draft"`
	InProgress  int64 `json:"inProgress"`
	Completed   int64 `json:"completed"`
	Cancelled   int64 `json:"cancelled"`
	TodayCount  int64 `json:"todayCount"`
	WeekCount   int64 `json:"weekCount"`
	MonthCount  int64 `json:"monthCount"`
}

type DashboardCostStats struct {
	TodayAmount  float64 `json:"todayAmount"`
	WeekAmount   float64 `json:"weekAmount"`
	MonthAmount  float64 `json:"monthAmount"`
	UnpaidAmount float64 `json:"unpaidAmount"`
	YearAmount   float64 `json:"yearAmount"`
}

type DashboardDistributorRank struct {
	DistributorID   uint64  `json:"distributorId"`
	DistributorName string  `json:"distributorName"`
	OrderCount   int64   `json:"orderCount"`
	TotalAmount  float64 `json:"totalAmount"`
}

type DashboardStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// DashboardTrend 代发趋势（按采购业务日）
type DashboardTrend struct {
	StartDate      string                `json:"startDate"`
	EndDate        string                `json:"endDate"`
	OrderCount     int64                 `json:"orderCount"`
	SaleAmount     float64               `json:"saleAmount"`
	WholesaleAmount float64               `json:"wholesaleAmount"`
	Profit         float64               `json:"profit"`
	Points         []DashboardTrendPoint `json:"points"`
}

type DashboardTrendPoint struct {
	Date           string  `json:"date"`
	OrderCount     int64   `json:"orderCount"`
	SaleAmount     float64 `json:"saleAmount"`
	WholesaleAmount float64 `json:"wholesaleAmount"`
	Profit         float64 `json:"profit"`
}
