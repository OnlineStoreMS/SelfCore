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
	SelfOrderPO       int64 `json:"selfOrderPO"`
	DropshipPO        int64 `json:"dropshipPO"`
	WholesalePO         int64 `json:"wholesalePO"`
	DraftPO           int64 `json:"draftPO"`
	ConfirmedPO         int64 `json:"confirmedPO"`
	UnpaidPO          int64 `json:"unpaidPO"`
	InTransitPO       int64 `json:"inTransitPO"`
	PartialReceivedPO int64 `json:"partialReceivedPO"`
	ActiveOffers      int64 `json:"activeOffers"`
	// 今日毛利：仅成本额 > 0 的订单；毛利润 = 有成本销售额 − 成本额
	TodayDropshipSaleAmount      float64 `json:"todayDropshipSaleAmount"`      // 有成本订单的销售额（tip 用）
	TodayDropshipWholesaleAmount float64 `json:"todayDropshipWholesaleAmount"` // 有成本订单的成本额
	TodayDropshipProfit          float64 `json:"todayDropshipProfit"`          // 今日毛利润（成本为 0 不计入）
	// 今日分销销售额：全部分销类型（直发用 sale_amount，批发用 total_amount）
	TodayDistSaleAmount float64 `json:"todayDistSaleAmount"`
	// 今日 / 近7日 / 本月自营销售额
	TodaySelfSaleAmount float64 `json:"todaySelfSaleAmount"`
	WeekSelfSaleAmount  float64 `json:"weekSelfSaleAmount"`
	MonthSelfSaleAmount float64 `json:"monthSelfSaleAmount"`
	// 本月分销销售额
	MonthDistSaleAmount float64 `json:"monthDistSaleAmount"`
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

// DashboardTrend 全类型趋势（自营+全部分销，按业务日）
type DashboardTrend struct {
	StartDate       string                `json:"startDate"`
	EndDate         string                `json:"endDate"`
	OrderCount      int64                 `json:"orderCount"`      // 区间订单量（自营+分销）
	SaleAmount      float64               `json:"saleAmount"`      // 区间销售额
	WholesaleAmount float64               `json:"wholesaleAmount"` // 区间成本额
	Profit          float64               `json:"profit"`          // 区间毛利润
	Points          []DashboardTrendPoint `json:"points"`
}

type DashboardTrendPoint struct {
	Date            string  `json:"date"`
	SelfOrderCount  int64   `json:"selfOrderCount"`  // 自营单量（图1）
	SelfSaleAmount  float64 `json:"selfSaleAmount"`  // 自营销售额（图1）
	OrderCount      int64   `json:"orderCount"`      // 全日订单量（自营+分销）
	SaleAmount      float64 `json:"saleAmount"`      // 全日销售额
	WholesaleAmount float64 `json:"wholesaleAmount"` // 全日成本额
	Profit          float64 `json:"profit"`          // 全日毛利润
}
