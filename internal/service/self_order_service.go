package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"selfcore/internal/dto"
	"selfcore/internal/integrations/ordercore"
	"selfcore/internal/integrations/shippingcore"
	"selfcore/internal/integrations/warehousecore"
	"selfcore/internal/model"
	"selfcore/internal/repo"

	"gorm.io/gorm"
)

type SelfOrderService struct {
	repos    *repo.Repos
	tenantID uint64
	wh       *warehousecore.Client
	oc       *ordercore.Client
	ship     *shippingcore.Client
}

func NewSelfOrderService(repos *repo.Repos, wh *warehousecore.Client, oc *ordercore.Client, ship *shippingcore.Client) *SelfOrderService {
	return &SelfOrderService{repos: repos, wh: wh, oc: oc, ship: ship}
}

func (s *SelfOrderService) ForTenant(tenantID uint64) *SelfOrderService {
	return &SelfOrderService{
		repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID),
		wh: s.wh, oc: s.oc, ship: s.ship,
	}
}

func (s *SelfOrderService) CountStatusFacets(f repo.SelfOrderListFilter) (*repo.SelfOrderStatusCounts, error) {
	return s.repos.SelfOrder.ForTenant(s.tenantID).CountStatusFacets(f)
}

func (s *SelfOrderService) List(f repo.SelfOrderListFilter) ([]dto.SelfOrderListItem, int64, error) {
	list, total, err := s.repos.SelfOrder.ForTenant(s.tenantID).List(f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.SelfOrderListItem, 0, len(list))
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	ids := make([]uint64, 0, len(list))
	for _, o := range list {
		ids = append(ids, o.ID)
	}
	specsBySO, _ := r.ItemSpecsBySelfOrderIDs(ids)
	for _, o := range list {
		item := dto.SelfOrderListItem{
			ID: o.ID, SoNo: o.SoNo, Status: o.Status, WarehouseID: o.WarehouseID,
			RefSoID: o.RefSoID, RefTraceID: o.RefTraceID,
			SaleAmount: o.SaleAmount, CostAmount: o.CostAmount,
			PayStatus: firstNonEmpty(o.PayStatus, model.DistPayStatusUnpaid),
			BuyerName: o.BuyerName, BuyerPhone: o.BuyerPhone,
			SourceChannel: o.SourceChannel, Platform: o.Platform, ShopName: o.ShopName,
			ManualSourceName: o.ManualSourceName,
			BuyerRemark: o.BuyerRemark, SellerRemark: o.SellerRemark,
			FenFaRemark: o.FenFaRemark, PrinterRemark: o.PrinterRemark,
			StockDeducted: o.StockDeducted, StockError: o.StockError,
			CreatedAt: formatTime(o.CreatedAt),
		}
		if o.PaidAt != nil {
			item.PaidAt = formatTimePtr(o.PaidAt)
		}
		if o.OrderedAt != nil {
			item.OrderedAt = formatTimePtr(o.OrderedAt)
		}
		if o.ShippedAt != nil {
			item.ShippedAt = formatTimePtr(o.ShippedAt)
		}
		if n, err := r.CountItems(o.ID); err == nil {
			item.ItemCount = int(n)
		}
		if specs := specsBySO[o.ID]; len(specs) > 0 {
			item.SkuSpecs = strings.Join(specs, "；")
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (s *SelfOrderService) Get(id uint64) (*dto.SelfOrderDetail, error) {
	o, err := s.repos.SelfOrder.ForTenant(s.tenantID).GetWithItems(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.toDetail(o), nil
}

func (s *SelfOrderService) Create(ctx context.Context, bearerToken string, in *dto.SelfOrderInput) (*dto.SelfOrderDetail, error) {
	if in == nil || len(in.Items) == 0 {
		return nil, fmt.Errorf("自营单明细不能为空")
	}
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	if in.RefSoID > 0 {
		if existing, err := r.FindActiveByRefSoID(in.RefSoID); err == nil && existing != nil {
			return s.Get(existing.ID)
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	items := make([]model.SelfOrderItem, 0, len(in.Items))
	var saleTotal, costTotal float64
	for _, it := range in.Items {
		qty := it.Qty
		if qty <= 0 {
			qty = 1
		}
		line := model.SelfOrderItem{
			PimSkuID: it.PimSkuID, SkuCode: strings.TrimSpace(it.SkuCode),
			ProductName: it.ProductName, SkuSpecs: it.SkuSpecs, PicURL: it.PicURL,
			Qty: qty, SaleUnitPrice: it.SaleUnitPrice, SaleAmount: it.SaleAmount,
			InvSkuID: it.InvSkuID, InvSkuCode: strings.TrimSpace(it.InvSkuCode),
			CostUnitPrice: it.CostUnitPrice, CostAmount: it.CostAmount,
			RefSoID: it.RefSoID, RefOrderNo: strings.TrimSpace(it.RefOrderNo),
			Remark: it.Remark,
		}
		if line.RefSoID == 0 {
			line.RefSoID = in.RefSoID
		}
		if line.RefOrderNo == "" {
			line.RefOrderNo = strings.TrimSpace(in.RefTraceID)
		}
		if line.SaleAmount <= 0 && line.SaleUnitPrice > 0 {
			line.SaleAmount = round2(line.SaleUnitPrice * float64(qty))
		}
		// 尝试绑定库存 SKU + 成本
		if line.InvSkuID == 0 && s.wh != nil && s.wh.Enabled() {
			if invID, invCode, cost, err := s.wh.ResolveInvSku(ctx, bearerToken, line.PimSkuID, line.SkuCode); err == nil && invID > 0 {
				line.InvSkuID = invID
				if line.InvSkuCode == "" {
					line.InvSkuCode = invCode
				}
				if line.CostUnitPrice <= 0 && cost > 0 {
					line.CostUnitPrice = cost
				}
			}
		}
		if line.CostAmount <= 0 && line.CostUnitPrice > 0 {
			line.CostAmount = round2(line.CostUnitPrice * float64(qty))
		}
		saleTotal += line.SaleAmount
		costTotal += line.CostAmount
		items = append(items, line)
	}
	if in.SaleAmount > 0 {
		saleTotal = in.SaleAmount
	}

	warehouseID := in.WarehouseID
	if warehouseID == 0 && s.wh != nil && s.wh.Enabled() {
		if list, _, err := s.wh.ListWarehouses(ctx, bearerToken, "", 1, 50); err == nil {
			for _, w := range list {
				if w.IsDefault == 1 && w.Status == 1 {
					warehouseID = w.ID
					break
				}
			}
			if warehouseID == 0 {
				for _, w := range list {
					if w.Status == 1 {
						warehouseID = w.ID
						break
					}
				}
			}
		}
	}

	const maxAttempts = 5
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		soNo, err := r.NextSoNo()
		if err != nil {
			return nil, err
		}
		status, payStatus, paidAt := resolveSelfOrderCreatePay(in)
		o := &model.SelfOrder{
			SoNo: soNo, Status: status,
			WarehouseID: warehouseID,
			RefSoID: in.RefSoID, RefTraceID: strings.TrimSpace(in.RefTraceID),
			SaleAmount: round2(saleTotal), CostAmount: round2(costTotal),
			PayStatus: payStatus, PaidAt: paidAt,
			BuyerName: in.BuyerName, BuyerPhone: in.BuyerPhone, Address: in.Address,
			Remark: in.Remark,
			SourceChannel: strings.TrimSpace(in.SourceChannel),
			Platform:      strings.TrimSpace(in.Platform),
			ShopName:      strings.TrimSpace(in.ShopName),
			ManualSourceName: strings.TrimSpace(in.ManualSourceName),
			BuyerRemark:   strings.TrimSpace(in.BuyerRemark),
			SellerRemark:  strings.TrimSpace(in.SellerRemark),
			FenFaRemark:   strings.TrimSpace(in.FenFaRemark),
			PrinterRemark: strings.TrimSpace(in.PrinterRemark),
		}
		if o.ShopName == "" && o.ManualSourceName != "" {
			o.ShopName = o.ManualSourceName
		}
		if t := parseDateTime(in.OrderedAt); t != nil {
			o.OrderedAt = t
		} else if status != model.SelfOrderStatusDraft {
			now := time.Now()
			o.OrderedAt = &now
		}
		if o.PaidAt == nil && payStatus == model.DistPayStatusPaid {
			if o.OrderedAt != nil {
				o.PaidAt = o.OrderedAt
			} else {
				now := time.Now()
				o.PaidAt = &now
			}
		}
		lineItems := make([]model.SelfOrderItem, len(items))
		copy(lineItems, items)
		if err := r.Create(o, lineItems); err != nil {
			lastErr = err
			if isUniqueViolation(err) {
				continue
			}
			return nil, err
		}
		return s.Get(o.ID)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("创建自营单失败：单号冲突")
}

func (s *SelfOrderService) BindInvSku(itemID uint64, in *dto.BindInvSkuInput) (*dto.SelfOrderDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	it, err := r.GetItem(itemID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	oHead, err := r.GetByID(it.SelfOrderID)
	if err != nil {
		return nil, err
	}
	if !canEditSelfOrderCost(oHead.Status) {
		return nil, fmt.Errorf("已完成/已取消的自营单不可绑定库存 SKU")
	}
	it.InvSkuID = in.InvSkuID
	it.InvSkuCode = strings.TrimSpace(in.InvSkuCode)
	if in.CostUnitPrice > 0 {
		it.CostUnitPrice = in.CostUnitPrice
	}
	it.CostAmount = round2(it.CostUnitPrice * float64(it.Qty))
	if err := r.SaveItem(it); err != nil {
		return nil, err
	}
	return s.recalcOrderCost(it.SelfOrderID)
}

func (s *SelfOrderService) UpdateItemCost(itemID uint64, in *dto.UpdateItemCostInput) (*dto.SelfOrderDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	it, err := r.GetItem(itemID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	oHead, err := r.GetByID(it.SelfOrderID)
	if err != nil {
		return nil, err
	}
	if !canEditSelfOrderCost(oHead.Status) {
		return nil, fmt.Errorf("已完成/已取消的自营单不可修改成本价")
	}
	if in.CostUnitPrice < 0 {
		return nil, fmt.Errorf("成本单价不能为负")
	}
	it.CostUnitPrice = in.CostUnitPrice
	it.CostAmount = round2(it.CostUnitPrice * float64(it.Qty))
	if err := r.SaveItem(it); err != nil {
		return nil, err
	}
	return s.recalcOrderCost(it.SelfOrderID)
}

func (s *SelfOrderService) recalcOrderCost(selfOrderID uint64) (*dto.SelfOrderDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetWithItems(selfOrderID)
	if err != nil {
		return nil, err
	}
	var costTotal float64
	for _, line := range o.Items {
		costTotal += line.CostAmount
	}
	o.CostAmount = round2(costTotal)
	if err := r.Save(o); err != nil {
		return nil, err
	}
	return s.toDetail(o), nil
}

func (s *SelfOrderService) ListShipments(selfOrderID uint64) ([]dto.SelfShipmentDTO, error) {
	list, err := s.repos.SelfOrder.ForTenant(s.tenantID).ListShipments(selfOrderID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.SelfShipmentDTO, 0, len(list))
	for i := range list {
		out = append(out, s.toShipmentDTO(&list[i]))
	}
	return out, nil
}

func (s *SelfOrderService) GetShipment(selfOrderID, shipmentID uint64) (*dto.SelfShipmentDTO, error) {
	sh, err := s.repos.SelfOrder.ForTenant(s.tenantID).GetShipmentByID(selfOrderID, shipmentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d := s.toShipmentDTO(sh)
	return &d, nil
}

func (s *SelfOrderService) CreateShipment(ctx context.Context, bearerToken string, id uint64, in *dto.SelfShipmentCreateInput) (*dto.SelfShipmentDTO, error) {
	if in == nil || len(in.Items) == 0 {
		return nil, fmt.Errorf("请选择发货明细")
	}
	if strings.TrimSpace(in.TrackingNo) == "" {
		return nil, fmt.Errorf("请填写物流单号")
	}
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetWithItems(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status == model.SelfOrderStatusCancelled {
		return nil, fmt.Errorf("已取消的自营单不能发货")
	}
	if o.Status == model.SelfOrderStatusDraft {
		return nil, fmt.Errorf("草稿自营单请先提交下单再发货")
	}
	if o.RefSoID == 0 {
		return nil, fmt.Errorf("未关联销售单，无法回传订单中心")
	}

	carrierName := strings.TrimSpace(in.CarrierName)
	if carrierName == "" && strings.TrimSpace(in.CarrierCode) != "" {
		carrierName = strings.TrimSpace(in.CarrierCode)
	}
	if carrierName == "" {
		return nil, fmt.Errorf("请填写快递公司")
	}

	items, err := s.buildShipmentItems(o, in.Items)
	if err != nil {
		return nil, err
	}

	shipNo, err := r.NextShipmentNo()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	receiverName := strings.TrimSpace(in.ReceiverName)
	receiverPhone := strings.TrimSpace(in.ReceiverPhone)
	receiverAddr := strings.TrimSpace(in.ReceiverAddress)
	if receiverName == "" {
		receiverName = o.BuyerName
	}
	if receiverPhone == "" {
		receiverPhone = o.BuyerPhone
	}
	if receiverAddr == "" {
		receiverAddr = o.Address
	}

	sh := &model.SelfShipment{
		SelfOrderID:     o.ID,
		ShipmentNo:      shipNo,
		Status:          model.ShipmentStatusShipped,
		CarrierCode:     strings.TrimSpace(in.CarrierCode),
		CarrierName:     carrierName,
		TrackingNo:      strings.TrimSpace(in.TrackingNo),
		ShippedAt:       &now,
		ReceiverName:    receiverName,
		ReceiverPhone:   receiverPhone,
		ReceiverAddress: receiverAddr,
		Remark:          in.Remark,
	}
	if d := parseDate(in.ExpectedArrivalDate); d != nil {
		sh.ExpectedArrivalDate = d
	}
	if err := r.CreateShipment(sh, items); err != nil {
		return nil, err
	}
	sh.Items = items

	if err := s.syncSelfOrderShipStatus(o.ID); err != nil {
		return nil, err
	}

	doCallback := true
	if in.Callback != nil {
		doCallback = *in.Callback
	}
	if doCallback {
		if err := s.callbackOrderCore(ctx, bearerToken, o, sh); err != nil {
			detail := s.toShipmentDTO(sh)
			return &detail, nil
		}
	}

	if sh.CallbackOK {
		if err := s.deductStockForShipment(ctx, bearerToken, o, sh); err != nil {
			o.StockError = err.Error()
			_ = r.Save(o)
		}
	}

	if s.ship != nil && s.ship.Enabled() {
		_ = s.ship.NotifyShipment(ctx, bearerToken, shippingcore.NotifyInput{
			SelfOrderNo: o.SoNo, RefSoID: o.RefSoID,
			ExpressCompany: sh.CarrierName, ExpressNo: sh.TrackingNo,
		})
	}

	detail := s.toShipmentDTO(sh)
	return &detail, nil
}

func (s *SelfOrderService) UpdateShipmentStatus(selfOrderID, shipmentID uint64, status string) (*dto.SelfShipmentDTO, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetByID(selfOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status == model.SelfOrderStatusCancelled {
		return nil, fmt.Errorf("已取消的自营单不能修改物流")
	}
	sh, err := r.GetShipmentByID(selfOrderID, shipmentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !isValidShipmentStatus(status) {
		return nil, ErrBadRequest
	}
	now := time.Now()
	sh.Status = status
	switch status {
	case model.ShipmentStatusShipped:
		if sh.ShippedAt == nil {
			sh.ShippedAt = &now
		}
	case model.ShipmentStatusInTransit:
		if sh.ShippedAt == nil {
			sh.ShippedAt = &now
		}
	case model.ShipmentStatusDelivered:
		sh.DeliveredAt = &now
	}
	if err := r.SaveShipment(sh); err != nil {
		return nil, err
	}
	if err := s.syncSelfOrderShipStatus(selfOrderID); err != nil {
		return nil, err
	}
	detail := s.toShipmentDTO(sh)
	return &detail, nil
}

func (s *SelfOrderService) DeleteShipment(selfOrderID, shipmentID uint64) error {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	sh, err := r.GetShipmentByID(selfOrderID, shipmentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if sh.StockDeducted {
		return fmt.Errorf("已扣库不可删除")
	}
	if err := r.DeleteShipment(selfOrderID, shipmentID); err != nil {
		return err
	}
	return s.syncSelfOrderShipStatus(selfOrderID)
}

func (s *SelfOrderService) SyncShipmentsFromOrders(ctx context.Context, selfOrderID uint64, bearerToken string, in *dto.SyncShipmentsFromOrdersInput) (*dto.SyncShipmentsFromOrdersResult, error) {
	if s.oc == nil {
		return nil, fmt.Errorf("OrderCore 未配置")
	}
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetWithItems(selfOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status == model.SelfOrderStatusCancelled {
		return nil, fmt.Errorf("已取消的自营单不能同步物流")
	}

	type soGroup struct {
		refSoID uint64
		orderNo string
		items   []model.SelfOrderItem
	}
	groups := map[uint64]*soGroup{}
	filterSoID := uint64(0)
	if in != nil {
		filterSoID = in.RefSoID
	}
	for _, it := range o.Items {
		refSoID := it.RefSoID
		orderNo := strings.TrimSpace(it.RefOrderNo)
		if refSoID == 0 {
			refSoID = o.RefSoID
		}
		if orderNo == "" {
			trace := strings.TrimSpace(o.RefTraceID)
			if trace != "" && !strings.Contains(trace, ",") {
				orderNo = trace
			}
		}
		if refSoID == 0 {
			continue
		}
		if filterSoID > 0 && refSoID != filterSoID {
			continue
		}
		g := groups[refSoID]
		if g == nil {
			g = &soGroup{refSoID: refSoID, orderNo: orderNo}
			groups[refSoID] = g
		}
		g.items = append(g.items, it)
		if g.orderNo == "" && orderNo != "" {
			g.orderNo = orderNo
		}
	}

	out := &dto.SyncShipmentsFromOrdersResult{}
	if len(groups) == 0 {
		out.Skipped = 1
		out.Errors = append(out.Errors, "没有可同步的销售单明细")
		return out, nil
	}

	for _, g := range groups {
		order, gerr := s.oc.GetOrder(ctx, bearerToken, g.refSoID)
		if gerr != nil {
			out.Skipped++
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", coalesceOrderNo(g.orderNo, g.refSoID), gerr))
			continue
		}
		if g.orderNo == "" {
			g.orderNo = order.OrderNo
		}
		receiverName := strings.TrimSpace(order.BuyerName)
		receiverPhone := strings.TrimSpace(order.BuyerPhone)
		receiverAddr := ordercore.FormatReceiverAddress(order.Address)
		if order.Address != nil {
			if n := strings.TrimSpace(order.Address.Name); n != "" {
				receiverName = n
			}
			if p := strings.TrimSpace(order.Address.Phone); p != "" {
				receiverPhone = p
			}
		}

		type logistics struct {
			trackingNo string
			carrier    string
			shippedAt  *time.Time
			remark     string
		}
		logs := make([]logistics, 0)
		seenTrack := map[string]struct{}{}
		for _, osh := range order.Shipments {
			tn := strings.TrimSpace(osh.ExpressNo)
			if tn == "" {
				continue
			}
			if _, ok := seenTrack[tn]; ok {
				continue
			}
			seenTrack[tn] = struct{}{}
			var shippedAt *time.Time
			if osh.ShippedAt != nil && strings.TrimSpace(*osh.ShippedAt) != "" {
				if t := parseDateTime(*osh.ShippedAt); t != nil {
					shippedAt = t
				}
			}
			logs = append(logs, logistics{
				trackingNo: tn,
				carrier:    strings.TrimSpace(osh.ExpressCompany),
				shippedAt:  shippedAt,
				remark:     fmt.Sprintf("同步自订单 %s", order.OrderNo),
			})
		}
		if len(logs) == 0 && order.ShipStatus == "shipped" {
			logs = append(logs, logistics{
				trackingNo: fmt.Sprintf("SYNC-%s", order.OrderNo),
				carrier:    "订单中心已发货",
				remark:     fmt.Sprintf("同步自订单 %s（无快递单号）", order.OrderNo),
			})
		}
		if len(logs) == 0 {
			out.Skipped++
			continue
		}

		primary := logs[0]
		existing, ferr := r.FindShipmentByTrackingNo(selfOrderID, primary.trackingNo)
		if ferr != nil && !errors.Is(ferr, gorm.ErrRecordNotFound) {
			out.Skipped++
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", order.OrderNo, ferr))
			continue
		}
		if existing != nil {
			changed := false
			if strings.TrimSpace(existing.ReceiverName) == "" && receiverName != "" {
				existing.ReceiverName = receiverName
				changed = true
			}
			if strings.TrimSpace(existing.ReceiverPhone) == "" && receiverPhone != "" {
				existing.ReceiverPhone = receiverPhone
				changed = true
			}
			if strings.TrimSpace(existing.ReceiverAddress) == "" && receiverAddr != "" {
				existing.ReceiverAddress = receiverAddr
				changed = true
			}
			if strings.TrimSpace(existing.CarrierName) == "" && primary.carrier != "" {
				existing.CarrierName = primary.carrier
				changed = true
			}
			if existing.Status == model.ShipmentStatusPending {
				existing.Status = model.ShipmentStatusShipped
				now := time.Now()
				if primary.shippedAt != nil {
					existing.ShippedAt = primary.shippedAt
				} else {
					existing.ShippedAt = &now
				}
				changed = true
			}
			if changed {
				if err := r.SaveShipment(existing); err != nil {
					out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", order.OrderNo, err))
					out.Skipped++
					continue
				}
				out.Updated++
			} else {
				out.Skipped++
			}
			continue
		}

		shippedQty, _ := r.SumShippedQtyByItem(selfOrderID)
		remainInputs := make([]dto.SelfShipmentItemInput, 0, len(g.items))
		for _, it := range g.items {
			if it.ID == 0 {
				continue
			}
			remain := it.Qty - shippedQty[it.ID]
			if remain <= 0 {
				continue
			}
			remainInputs = append(remainInputs, dto.SelfShipmentItemInput{
				SelfOrderItemID: it.ID, Qty: remain,
			})
		}
		if len(remainInputs) == 0 {
			out.Skipped++
			continue
		}
		items, berr := s.buildShipmentItems(o, remainInputs)
		if berr != nil {
			out.Skipped++
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", order.OrderNo, berr))
			continue
		}
		no, nerr := r.NextShipmentNo()
		if nerr != nil {
			out.Skipped++
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", order.OrderNo, nerr))
			continue
		}
		now := time.Now()
		shippedAt := &now
		if primary.shippedAt != nil {
			shippedAt = primary.shippedAt
		}
		sh := &model.SelfShipment{
			SelfOrderID:     selfOrderID,
			ShipmentNo:      no,
			Status:          model.ShipmentStatusShipped,
			CarrierName:     primary.carrier,
			TrackingNo:      primary.trackingNo,
			ShippedAt:       shippedAt,
			ReceiverName:    receiverName,
			ReceiverPhone:   receiverPhone,
			ReceiverAddress: receiverAddr,
			Remark:          primary.remark,
		}
		if err := r.CreateShipment(sh, items); err != nil {
			out.Skipped++
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", order.OrderNo, err))
			continue
		}
		out.Created++

		for i := 1; i < len(logs); i++ {
			extra := logs[i]
			if ex, _ := r.FindShipmentByTrackingNo(selfOrderID, extra.trackingNo); ex != nil {
				continue
			}
			eno, _ := r.NextShipmentNo()
			esh := &model.SelfShipment{
				SelfOrderID:     selfOrderID,
				ShipmentNo:      eno,
				Status:          model.ShipmentStatusShipped,
				CarrierName:     extra.carrier,
				TrackingNo:      extra.trackingNo,
				ShippedAt:       shippedAt,
				ReceiverName:    receiverName,
				ReceiverPhone:   receiverPhone,
				ReceiverAddress: receiverAddr,
				Remark:          extra.remark + "（附加运单）",
			}
			if err := r.CreateShipment(esh, nil); err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("%s extra: %v", order.OrderNo, err))
				continue
			}
			out.Created++
		}
	}

	if err := s.syncSelfOrderShipStatus(selfOrderID); err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return out, nil
}

func (s *SelfOrderService) ListPayments(selfOrderID uint64) ([]dto.SelfPaymentDetail, error) {
	if _, err := s.repos.SelfOrder.ForTenant(s.tenantID).GetByID(selfOrderID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	list, err := s.repos.SelfOrder.ForTenant(s.tenantID).ListPayments(selfOrderID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.SelfPaymentDetail, 0, len(list))
	for i := range list {
		out = append(out, s.toPaymentDetail(&list[i]))
	}
	return out, nil
}

func (s *SelfOrderService) CreatePayment(ctx context.Context, bearerToken string, selfOrderID uint64, in *dto.SelfPaymentInput) (*dto.SelfPaymentDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetByID(selfOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status == model.SelfOrderStatusCancelled {
		return nil, fmt.Errorf("已取消的自营单不能记录付款")
	}
	if o.Status == model.SelfOrderStatusDraft {
		return nil, fmt.Errorf("草稿自营单请先提交下单再记录付款")
	}
	if isEcommerceSelfOrder(o) {
		return nil, fmt.Errorf("电商订单默认已付款，无需再记录付款")
	}
	pay := &model.SelfPayment{
		SelfOrderID:  selfOrderID,
		PayAmount:    in.PayAmount,
		PayMethod:    in.PayMethod,
		PayAccount:   in.PayAccount,
		PayeeAccount: in.PayeeAccount,
		PayeeName:    in.PayeeName,
		PayStatus:    defaultPayRecordStatus(in.PayStatus),
		Remark:       in.Remark,
	}
	if in.PaidAt != "" {
		if t := parseDateTime(in.PaidAt); t != nil {
			pay.PaidAt = t
		}
	} else if pay.PayStatus == model.DistPayStatusPaid {
		now := time.Now()
		pay.PaidAt = &now
	}
	if err := r.CreatePayment(pay); err != nil {
		return nil, err
	}
	if err := s.syncPayStatus(ctx, bearerToken, o); err != nil {
		return nil, err
	}
	detail := s.toPaymentDetail(pay)
	return &detail, nil
}

func (s *SelfOrderService) UpdatePayment(ctx context.Context, bearerToken string, selfOrderID, paymentID uint64, in *dto.SelfPaymentInput) (*dto.SelfPaymentDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetByID(selfOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status == model.SelfOrderStatusCancelled {
		return nil, fmt.Errorf("已取消的自营单不能修改付款")
	}
	if o.Status == model.SelfOrderStatusDraft {
		return nil, fmt.Errorf("草稿自营单请先提交下单再修改付款")
	}
	if isEcommerceSelfOrder(o) {
		return nil, fmt.Errorf("电商订单默认已付款，无需再修改付款")
	}
	pay, err := r.GetPayment(selfOrderID, paymentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	pay.PayAmount = in.PayAmount
	pay.PayMethod = in.PayMethod
	pay.PayAccount = in.PayAccount
	pay.PayeeAccount = in.PayeeAccount
	pay.PayeeName = in.PayeeName
	pay.PayStatus = defaultPayRecordStatus(in.PayStatus)
	pay.Remark = in.Remark
	if in.PaidAt != "" {
		pay.PaidAt = parseDateTime(in.PaidAt)
	}
	if err := r.SavePayment(pay); err != nil {
		return nil, err
	}
	if err := s.syncPayStatus(ctx, bearerToken, o); err != nil {
		return nil, err
	}
	detail := s.toPaymentDetail(pay)
	return &detail, nil
}

func (s *SelfOrderService) DeletePayment(ctx context.Context, bearerToken string, selfOrderID, paymentID uint64) error {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetByID(selfOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if isEcommerceSelfOrder(o) {
		return fmt.Errorf("电商订单默认已付款，无需删除付款记录")
	}
	atts, err := r.ListAttachments(selfOrderID)
	if err != nil {
		return err
	}
	for _, a := range atts {
		if a.PaymentID == paymentID {
			if err := r.DeleteAttachment(selfOrderID, a.ID); err != nil {
				return err
			}
		}
	}
	if err := r.DeletePayment(selfOrderID, paymentID); err != nil {
		return err
	}
	return s.syncPayStatus(ctx, bearerToken, o)
}

func (s *SelfOrderService) syncPayStatus(ctx context.Context, bearerToken string, o *model.SelfOrder) error {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	fresh, err := r.GetByID(o.ID)
	if err != nil {
		return err
	}
	sum, err := r.SumPaidPayments(fresh.ID)
	if err != nil {
		return err
	}
	earliest, err := r.EarliestPaidAt(fresh.ID)
	if err != nil {
		return err
	}
	switch {
	case sum <= 0:
		fresh.PayStatus = model.DistPayStatusUnpaid
		fresh.PaidAt = nil
	case sum+0.001 < fresh.SaleAmount:
		fresh.PayStatus = model.DistPayStatusPartial
		fresh.PaidAt = earliest
	default:
		fresh.PayStatus = model.DistPayStatusPaid
		fresh.PaidAt = earliest
		// 付款累计 >= 销售额且单据为已下单时，自动推进为已付款
		if fresh.Status == model.SelfOrderStatusOrdered || fresh.Status == model.SelfOrderStatusConfirmed {
			fresh.Status = model.SelfOrderStatusPaid
		}
	}
	if err := r.Save(fresh); err != nil {
		return err
	}
	*o = *fresh
	return s.callbackOrderPayment(ctx, bearerToken, fresh)
}

func (s *SelfOrderService) callbackOrderPayment(ctx context.Context, bearerToken string, o *model.SelfOrder) error {
	if s.oc == nil || o.RefSoID == 0 {
		return nil
	}
	req := ordercore.UpdatePaymentRequest{
		PayStatus: firstNonEmpty(o.PayStatus, model.DistPayStatusUnpaid),
	}
	if o.PaidAt != nil {
		t := formatTimePtr(o.PaidAt)
		req.PayTime = &t
	} else {
		req.ClearPayTime = true
	}
	if err := s.oc.UpdatePayment(ctx, bearerToken, o.RefSoID, req); err != nil {
		return fmt.Errorf("回写订单中心付款状态失败: %w", err)
	}
	return nil
}

func (s *SelfOrderService) toPaymentDetail(p *model.SelfPayment) dto.SelfPaymentDetail {
	d := dto.SelfPaymentDetail{
		ID: p.ID, SelfOrderID: p.SelfOrderID, PayAmount: p.PayAmount,
		PayMethod: p.PayMethod, PayAccount: p.PayAccount,
		PayeeAccount: p.PayeeAccount, PayeeName: p.PayeeName,
		PayStatus: p.PayStatus, Remark: p.Remark,
		CreatedAt: formatTime(p.CreatedAt),
	}
	if p.PaidAt != nil {
		d.PaidAt = formatTimePtr(p.PaidAt)
	}
	return d
}

func (s *SelfOrderService) ListAttachments(selfOrderID uint64) ([]dto.SelfAttachmentDTO, error) {
	if _, err := s.repos.SelfOrder.ForTenant(s.tenantID).GetByID(selfOrderID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	list, err := s.repos.SelfOrder.ForTenant(s.tenantID).ListAttachments(selfOrderID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.SelfAttachmentDTO, 0, len(list))
	for i := range list {
		out = append(out, s.toAttachmentDTO(&list[i]))
	}
	return out, nil
}

func (s *SelfOrderService) CreateAttachment(selfOrderID, uploadedBy uint64, in *dto.SelfAttachmentInput) (*dto.SelfAttachmentDTO, error) {
	o, err := s.repos.SelfOrder.ForTenant(s.tenantID).GetByID(selfOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status == model.SelfOrderStatusCancelled {
		return nil, fmt.Errorf("已取消的自营单不能上传附件")
	}
	a := &model.SelfAttachment{
		SelfOrderID: selfOrderID,
		PaymentID:   in.PaymentID,
		ShipmentID:  in.ShipmentID,
		FileType:    in.FileType,
		FileName:    in.FileName,
		FileURL:     in.FileURL,
		UploadedBy:  uploadedBy,
		Remark:      in.Remark,
	}
	if err := s.repos.SelfOrder.ForTenant(s.tenantID).CreateAttachment(a); err != nil {
		return nil, err
	}
	detail := s.toAttachmentDTO(a)
	return &detail, nil
}

func (s *SelfOrderService) DeleteAttachment(selfOrderID, attachmentID uint64) error {
	if _, err := s.repos.SelfOrder.ForTenant(s.tenantID).GetAttachment(selfOrderID, attachmentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.repos.SelfOrder.ForTenant(s.tenantID).DeleteAttachment(selfOrderID, attachmentID)
}

// Ship 便捷发货：CreateShipment 包装，发全部剩余明细。
func (s *SelfOrderService) Ship(ctx context.Context, bearerToken string, id uint64, in *dto.SelfShipInput) (*dto.SelfOrderDetail, error) {
	if in == nil || strings.TrimSpace(in.ExpressNo) == "" {
		return nil, fmt.Errorf("请填写物流单号")
	}
	if strings.TrimSpace(in.ExpressCompany) == "" {
		return nil, fmt.Errorf("请填写快递公司")
	}
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetWithItems(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status == model.SelfOrderStatusCancelled {
		return nil, fmt.Errorf("已取消的自营单不能发货")
	}
	if o.Status == model.SelfOrderStatusDraft {
		return nil, fmt.Errorf("草稿自营单请先提交下单再发货")
	}
	if o.Status == model.SelfOrderStatusCompleted {
		return nil, fmt.Errorf("已完成，不能重复发货")
	}

	shippedQty, err := r.SumShippedQtyByItem(id)
	if err != nil {
		return nil, err
	}
	var items []dto.SelfShipmentItemInput
	for _, it := range o.Items {
		remain := it.Qty - shippedQty[it.ID]
		if remain > 0 {
			items = append(items, dto.SelfShipmentItemInput{SelfOrderItemID: it.ID, Qty: remain})
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("没有可发货明细")
	}

	_, err = s.CreateShipment(ctx, bearerToken, id, &dto.SelfShipmentCreateInput{
		CarrierName: strings.TrimSpace(in.ExpressCompany),
		TrackingNo:  strings.TrimSpace(in.ExpressNo),
		Remark:      in.Remark,
		Callback:    in.Callback,
		Items:       items,
	})
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

// RetryCallback 对已登记物流重试订单中心回传；成功后尝试扣库。
func (s *SelfOrderService) RetryCallback(ctx context.Context, bearerToken string, id uint64, shipmentID uint64) (*dto.SelfOrderDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetWithItems(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.RefSoID == 0 {
		return nil, fmt.Errorf("未关联销售单，无法回传")
	}

	var sh *model.SelfShipment
	if shipmentID > 0 {
		sh, err = r.GetShipment(shipmentID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if sh.SelfOrderID != o.ID {
			return nil, fmt.Errorf("发货记录不属于该自营单")
		}
	} else {
		sh, err = r.FindLatestShipment(o.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("尚无发货记录，请先登记物流")
		}
		if err != nil {
			return nil, err
		}
	}
	if sh.CallbackOK {
		if !sh.StockDeducted {
			if err := s.deductStockForShipment(ctx, bearerToken, o, sh); err != nil {
				o.StockError = err.Error()
				_ = r.Save(o)
				return nil, err
			}
		}
		return s.Get(o.ID)
	}

	if err := s.callbackOrderCore(ctx, bearerToken, o, sh); err != nil {
		return nil, err
	}
	if err := s.deductStockForShipment(ctx, bearerToken, o, sh); err != nil {
		o.StockError = err.Error()
		_ = r.Save(o)
		detail, _ := s.Get(o.ID)
		if detail != nil {
			detail.StockError = err.Error()
		}
		return detail, nil
	}
	return s.Get(o.ID)
}

func (s *SelfOrderService) callbackOrderCore(ctx context.Context, bearerToken string, o *model.SelfOrder, sh *model.SelfShipment) error {
	if s.oc == nil {
		return fmt.Errorf("OrderCore 未配置")
	}
	_, err := s.oc.ShipOrder(ctx, bearerToken, o.RefSoID, ordercore.ShipRequest{
		ExpressCompany: sh.CarrierName,
		ExpressNo:      sh.TrackingNo,
		Remark:         sh.Remark,
		Callback:       true,
	})
	if err != nil {
		return fmt.Errorf("订单中心回传失败: %w", err)
	}
	sh.CallbackOK = true
	return s.repos.SelfOrder.ForTenant(s.tenantID).SaveShipment(sh)
}

func (s *SelfOrderService) RetryStockDeduct(ctx context.Context, bearerToken string, id uint64) (*dto.SelfOrderDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetWithItems(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.StockDeducted {
		return s.toDetail(o), nil
	}

	shipments, err := r.ListShipments(id)
	if err != nil {
		return nil, err
	}
	hasCallbackOK := false
	for i := range shipments {
		sh := &shipments[i]
		if !sh.CallbackOK {
			continue
		}
		hasCallbackOK = true
		if sh.StockDeducted {
			continue
		}
		if err := s.deductStockForShipment(ctx, bearerToken, o, sh); err != nil {
			o.StockError = err.Error()
			_ = r.Save(o)
			return nil, err
		}
	}
	if !hasCallbackOK {
		return nil, fmt.Errorf("订单中心尚未回传成功，请先重试回传")
	}
	return s.Get(o.ID)
}

func (s *SelfOrderService) Submit(id uint64) (*dto.SelfOrderDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetWithItems(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status != model.SelfOrderStatusDraft {
		return nil, ErrInvalidStatus
	}
	if len(o.Items) == 0 {
		return nil, fmt.Errorf("自营单明细不能为空")
	}
	for i, it := range o.Items {
		if it.CostUnitPrice <= 0 && it.CostAmount <= 0 {
			name := strings.TrimSpace(it.SkuSpecs)
			if name == "" {
				name = strings.TrimSpace(it.ProductName)
			}
			if name == "" {
				name = fmt.Sprintf("第%d行", i+1)
			}
			return nil, fmt.Errorf("请先填写成本价或绑定库存 SKU：%s", name)
		}
	}
	return s.transition(id, model.SelfOrderStatusDraft, model.SelfOrderStatusOrdered, func(so *model.SelfOrder) {
		if so.OrderedAt == nil {
			now := time.Now()
			so.OrderedAt = &now
		}
	})
}

func (s *SelfOrderService) MarkPaid(id uint64) (*dto.SelfOrderDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if isEcommerceSelfOrder(o) {
		return nil, fmt.Errorf("电商订单默认已付款，无需再标记付款")
	}
	return s.transition(id, model.SelfOrderStatusOrdered, model.SelfOrderStatusPaid, func(o *model.SelfOrder) {
		o.PayStatus = model.DistPayStatusPaid
		if o.PaidAt == nil {
			now := time.Now()
			o.PaidAt = &now
		}
	})
}

func (s *SelfOrderService) Complete(id uint64) (*dto.SelfOrderDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{
		model.SelfOrderStatusPaid: true, model.SelfOrderStatusPartialShipped: true,
		model.SelfOrderStatusShipped: true,
	}
	if !allowed[o.Status] {
		return nil, ErrInvalidStatus
	}
	now := time.Now()
	o.Status = model.SelfOrderStatusCompleted
	o.CompletedAt = &now
	if err := r.Save(o); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *SelfOrderService) transition(id uint64, from, to string, apply func(*model.SelfOrder)) (*dto.SelfOrderDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status != from {
		return nil, ErrInvalidStatus
	}
	o.Status = to
	if apply != nil {
		apply(o)
	}
	if err := r.Save(o); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *SelfOrderService) Delete(id uint64) error {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	// 已完成单据保留审计痕迹，其它状态允许删除（含关联物流/附件）
	if o.Status == model.SelfOrderStatusCompleted {
		return ErrInvalidStatus
	}
	return r.Delete(id)
}

func (s *SelfOrderService) Cancel(id uint64) (*dto.SelfOrderDetail, error) {
	return s.CancelWithReason(id, "")
}

// CancelWithReason 取消自营单（撤回分配等）；已取消幂等成功。
func (s *SelfOrderService) CancelWithReason(id uint64, reason string) (*dto.SelfOrderDetail, error) {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status == model.SelfOrderStatusCancelled {
		return s.Get(o.ID)
	}
	if o.Status == model.SelfOrderStatusShipped ||
		o.Status == model.SelfOrderStatusCompleted ||
		o.Status == model.SelfOrderStatusPartialShipped {
		return nil, fmt.Errorf("已发货单据不能取消")
	}
	o.Status = model.SelfOrderStatusCancelled
	reason = strings.TrimSpace(reason)
	if reason != "" {
		tag := fmt.Sprintf("【已取消：%s】", reason)
		if o.Remark == "" {
			o.Remark = tag
		} else if !strings.Contains(o.Remark, "【已取消") {
			o.Remark = tag + " " + o.Remark
		}
	}
	if err := r.Save(o); err != nil {
		return nil, err
	}
	return s.Get(o.ID)
}

// CancelByRefSoID 按销售单取消关联自营单（订单中心撤回分配）。
func (s *SelfOrderService) CancelByRefSoID(refSoID uint64, reason string) ([]dto.SelfOrderDetail, error) {
	if refSoID == 0 {
		return nil, fmt.Errorf("refSoId 无效")
	}
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	list, _, err := r.List(repo.SelfOrderListFilter{
		RefSoID:  refSoID,
		Page:     1,
		PageSize: 50,
	})
	if err != nil {
		return nil, err
	}
	out := make([]dto.SelfOrderDetail, 0, len(list))
	for _, o := range list {
		if o.Status == model.SelfOrderStatusCancelled {
			continue
		}
		d, err := s.CancelWithReason(o.ID, reason)
		if err != nil {
			return out, err
		}
		if d != nil {
			out = append(out, *d)
		}
	}
	return out, nil
}

func (s *SelfOrderService) deductStockForShipment(ctx context.Context, bearerToken string, o *model.SelfOrder, sh *model.SelfShipment) error {
	if sh.StockDeducted {
		return nil
	}
	if s.wh == nil || !s.wh.Enabled() {
		return fmt.Errorf("WarehouseCore 未配置")
	}
	if o.WarehouseID == 0 {
		return fmt.Errorf("未设置发货仓库，无法扣库存")
	}

	itemMap := map[uint64]model.SelfOrderItem{}
	for _, it := range o.Items {
		itemMap[it.ID] = it
	}
	if len(sh.Items) == 0 {
		fresh, err := s.repos.SelfOrder.ForTenant(s.tenantID).GetShipment(sh.ID)
		if err == nil && fresh != nil {
			sh.Items = fresh.Items
		}
	}

	items := make([]warehousecore.SaleOutboundItem, 0)
	for _, sit := range sh.Items {
		it, ok := itemMap[sit.SelfOrderItemID]
		if !ok {
			continue
		}
		if it.InvSkuID == 0 {
			return fmt.Errorf("明细未绑定库存SKU: %s", it.ProductName)
		}
		items = append(items, warehousecore.SaleOutboundItem{
			InvSkuID: it.InvSkuID,
			Qty:      float64(sit.Qty),
		})
	}
	if len(items) == 0 {
		return fmt.Errorf("无有效扣库明细")
	}
	err := s.wh.PostSaleOutbound(ctx, bearerToken, &warehousecore.SaleOutboundRequest{
		WarehouseID: o.WarehouseID,
		RefDocType:  "self_shipment",
		RefDocID:    sh.ID,
		RefDocNo:    sh.ShipmentNo,
		Remark:      fmt.Sprintf("自营发货扣库 %s", sh.ShipmentNo),
		Items:       items,
	})
	if err != nil {
		return err
	}
	sh.StockDeducted = true
	if err := s.repos.SelfOrder.ForTenant(s.tenantID).SaveShipment(sh); err != nil {
		return err
	}
	return s.syncOrderStockDeductedFlag(o.ID)
}

func (s *SelfOrderService) syncOrderStockDeductedFlag(selfOrderID uint64) error {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetWithItems(selfOrderID)
	if err != nil {
		return err
	}
	shipments, err := r.ListShipments(selfOrderID)
	if err != nil {
		return err
	}
	allShipmentsDeducted := len(shipments) > 0
	for _, sh := range shipments {
		if len(sh.Items) == 0 {
			continue
		}
		if !sh.StockDeducted {
			allShipmentsDeducted = false
			break
		}
	}
	shippedQty, err := r.SumShippedQtyByItem(selfOrderID)
	if err != nil {
		return err
	}
	fullyShipped := true
	for _, it := range o.Items {
		if shippedQty[it.ID] < it.Qty {
			fullyShipped = false
			break
		}
	}
	if allShipmentsDeducted && fullyShipped {
		o.StockDeducted = true
		o.StockError = ""
		return r.Save(o)
	}
	return nil
}

func (s *SelfOrderService) buildShipmentItems(o *model.SelfOrder, inputs []dto.SelfShipmentItemInput) ([]model.SelfShipmentItem, error) {
	itemMap := map[uint64]model.SelfOrderItem{}
	for _, it := range o.Items {
		itemMap[it.ID] = it
	}
	shippedQty, err := s.repos.SelfOrder.ForTenant(s.tenantID).SumShippedQtyByItem(o.ID)
	if err != nil {
		return nil, err
	}
	seen := map[uint64]struct{}{}
	items := make([]model.SelfShipmentItem, 0, len(inputs))
	for _, in := range inputs {
		if _, dup := seen[in.SelfOrderItemID]; dup {
			return nil, ErrBadRequest
		}
		seen[in.SelfOrderItemID] = struct{}{}
		orderItem, ok := itemMap[in.SelfOrderItemID]
		if !ok {
			return nil, ErrNotFound
		}
		remain := orderItem.Qty - shippedQty[in.SelfOrderItemID]
		if in.Qty <= 0 || in.Qty > remain {
			return nil, fmt.Errorf("明细 %s 可发数量不足（剩余 %d）", orderItem.ProductName, remain)
		}
		items = append(items, model.SelfShipmentItem{
			SelfOrderItemID: in.SelfOrderItemID, Qty: in.Qty,
		})
	}
	return items, nil
}

func (s *SelfOrderService) syncSelfOrderShipStatus(selfOrderID uint64) error {
	r := s.repos.SelfOrder.ForTenant(s.tenantID)
	o, err := r.GetWithItems(selfOrderID)
	if err != nil {
		return err
	}
	if o.Status == model.SelfOrderStatusCancelled || o.Status == model.SelfOrderStatusCompleted {
		return nil
	}
	list, err := r.ListShipments(selfOrderID)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return nil
	}
	shippedQty := map[uint64]int{}
	hasShipped := false
	for _, sh := range list {
		if sh.Status != model.ShipmentStatusPending {
			hasShipped = true
		}
		for _, it := range sh.Items {
			shippedQty[it.SelfOrderItemID] += it.Qty
		}
	}
	fullyShipped := true
	for _, it := range o.Items {
		if shippedQty[it.ID] < it.Qty {
			fullyShipped = false
			break
		}
	}
	switch {
	case fullyShipped && hasShipped:
		// 全部明细发完 → 已发货（可再点「完成」归档）
		o.Status = model.SelfOrderStatusShipped
	case hasShipped:
		o.Status = model.SelfOrderStatusPartialShipped
	}
	if hasShipped && o.ShippedAt == nil {
		now := time.Now()
		o.ShippedAt = &now
	}
	return r.Save(o)
}

func (s *SelfOrderService) toShipmentDTO(sh *model.SelfShipment) dto.SelfShipmentDTO {
	d := dto.SelfShipmentDTO{
		ID: sh.ID, SelfOrderID: sh.SelfOrderID, ShipmentNo: sh.ShipmentNo, Status: sh.Status,
		CarrierCode: sh.CarrierCode, CarrierName: sh.CarrierName, TrackingNo: sh.TrackingNo,
		CallbackOK: sh.CallbackOK, StockDeducted: sh.StockDeducted,
		ReceiverName: sh.ReceiverName, ReceiverPhone: sh.ReceiverPhone,
		ReceiverAddress: sh.ReceiverAddress, Remark: sh.Remark,
		CreatedAt: formatTime(sh.CreatedAt),
		Items: make([]dto.SelfShipmentItemDTO, 0, len(sh.Items)),
	}
	if sh.ShippedAt != nil {
		d.ShippedAt = formatTimePtr(sh.ShippedAt)
	}
	if sh.ExpectedArrivalDate != nil {
		d.ExpectedArrivalDate = sh.ExpectedArrivalDate.Format("2006-01-02")
	}
	if sh.DeliveredAt != nil {
		d.DeliveredAt = formatTimePtr(sh.DeliveredAt)
	}
	for _, it := range sh.Items {
		d.Items = append(d.Items, dto.SelfShipmentItemDTO{
			ID: it.ID, SelfOrderItemID: it.SelfOrderItemID, Qty: it.Qty,
		})
	}
	return d
}

func (s *SelfOrderService) toAttachmentDTO(a *model.SelfAttachment) dto.SelfAttachmentDTO {
	return dto.SelfAttachmentDTO{
		ID: a.ID, SelfOrderID: a.SelfOrderID, PaymentID: a.PaymentID, ShipmentID: a.ShipmentID,
		FileType: a.FileType, FileName: a.FileName, FileURL: a.FileURL,
		UploadedBy: a.UploadedBy, Remark: a.Remark,
		CreatedAt: formatTime(a.CreatedAt),
	}
}

func coalesceOrderNo(orderNo string, soID uint64) string {
	if strings.TrimSpace(orderNo) != "" {
		return orderNo
	}
	return fmt.Sprintf("#%d", soID)
}

func (s *SelfOrderService) toDetail(o *model.SelfOrder) *dto.SelfOrderDetail {
	d := &dto.SelfOrderDetail{
		ID: o.ID, SoNo: o.SoNo, Status: o.Status, WarehouseID: o.WarehouseID,
		RefSoID: o.RefSoID, RefTraceID: o.RefTraceID,
		SaleAmount: o.SaleAmount, CostAmount: o.CostAmount,
		PayStatus: firstNonEmpty(o.PayStatus, model.DistPayStatusUnpaid),
		BuyerName: o.BuyerName, BuyerPhone: o.BuyerPhone, Address: o.Address,
		Remark: o.Remark,
		SourceChannel: o.SourceChannel, Platform: o.Platform, ShopName: o.ShopName,
		ManualSourceName: o.ManualSourceName,
		BuyerRemark: o.BuyerRemark, SellerRemark: o.SellerRemark,
		FenFaRemark: o.FenFaRemark, PrinterRemark: o.PrinterRemark,
		StockDeducted: o.StockDeducted, StockError: o.StockError,
		CreatedAt: formatTime(o.CreatedAt), UpdatedAt: formatTime(o.UpdatedAt),
	}
	if o.PaidAt != nil {
		d.PaidAt = formatTimePtr(o.PaidAt)
	}
	if o.OrderedAt != nil {
		d.OrderedAt = formatTimePtr(o.OrderedAt)
	}
	if o.ShippedAt != nil {
		d.ShippedAt = formatTimePtr(o.ShippedAt)
	}
	if o.CompletedAt != nil {
		d.CompletedAt = formatTimePtr(o.CompletedAt)
	}
	for _, it := range o.Items {
		d.Items = append(d.Items, dto.SelfOrderItemDTO{
			ID: it.ID, PimSkuID: it.PimSkuID, SkuCode: it.SkuCode,
			ProductName: it.ProductName, SkuSpecs: it.SkuSpecs, PicURL: it.PicURL,
			Qty: it.Qty, SaleUnitPrice: it.SaleUnitPrice, SaleAmount: it.SaleAmount,
			InvSkuID: it.InvSkuID, InvSkuCode: it.InvSkuCode,
			CostUnitPrice: it.CostUnitPrice, CostAmount: it.CostAmount,
			RefSoID: it.RefSoID, RefOrderNo: it.RefOrderNo, Remark: it.Remark,
		})
	}
	if d.Items == nil {
		d.Items = []dto.SelfOrderItemDTO{}
	}
	return d
}

// resolveSelfOrderCreatePay 创建时确定单据状态与付款：
// - 电商(kdzs)或显式 payStatus=paid → 已付款
// - 手工单 → 已下单（未付），便于直接填物流发货；成本可在完成前补填
// - 其它有关联销售单 → 已下单（未付）
// - 无关联 → 草稿
func isEcommerceSelfOrder(o *model.SelfOrder) bool {
	if o == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(o.SourceChannel), "kdzs")
}

// canEditSelfOrderCost 完成/取消前允许改成本、绑库存 SKU。
func canEditSelfOrderCost(status string) bool {
	switch strings.TrimSpace(status) {
	case model.SelfOrderStatusCompleted, model.SelfOrderStatusCancelled:
		return false
	default:
		return true
	}
}

func resolveSelfOrderCreatePay(in *dto.SelfOrderInput) (status, payStatus string, paidAt *time.Time) {
	status = model.SelfOrderStatusOrdered
	payStatus = model.DistPayStatusUnpaid
	if in == nil {
		return status, payStatus, nil
	}
	channel := strings.ToLower(strings.TrimSpace(in.SourceChannel))
	explicitPaid := strings.EqualFold(strings.TrimSpace(in.PayStatus), model.DistPayStatusPaid)
	ecommercePaid := channel == "kdzs" || explicitPaid
	if ecommercePaid {
		status = model.SelfOrderStatusPaid
		payStatus = model.DistPayStatusPaid
		if t := parseDateTime(in.PaidAt); t != nil {
			paidAt = t
		} else if t := parseDateTime(in.OrderedAt); t != nil {
			paidAt = t
		}
		return status, payStatus, paidAt
	}
	if channel == "manual" {
		// 手工单默认已下单，可直接发货；成本完成前仍可改
		return model.SelfOrderStatusOrdered, model.DistPayStatusUnpaid, nil
	}
	if in.RefSoID == 0 {
		status = model.SelfOrderStatusDraft
	}
	return status, payStatus, nil
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
