package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"selfcore/internal/dto"
	"selfcore/internal/model"
	"selfcore/internal/repo"

	"gorm.io/gorm"
)

type DistOrderService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewDistOrderService(repos *repo.Repos) *DistOrderService {
	return &DistOrderService{repos: repos}
}

func (s *DistOrderService) ForTenant(tenantID uint64) *DistOrderService {
	return &DistOrderService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *DistOrderService) List(f repo.DistOrderListFilter) ([]dto.DistOrderListItem, int64, error) {
	list, total, err := s.repos.DistOrder.ForTenant(s.tenantID).List(f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.DistOrderListItem, 0, len(list))
	sr := s.repos.Distributor.ForTenant(s.tenantID)
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	poIDs := make([]uint64, 0, len(list))
	for _, po := range list {
		poIDs = append(poIDs, po.ID)
	}
	specsByPO, _ := pr.ItemSpecsByDistOrderIDs(poIDs)
	for _, po := range list {
		item := dto.DistOrderListItem{
			ID: po.ID, DistNo: po.DistNo, DistributorID: po.DistributorID,
			Status: po.Status, PayStatus: po.PayStatus,
			FulfillmentType: po.FulfillmentType,
			TotalAmount: po.TotalAmount, SaleAmount: po.SaleAmount, Currency: po.Currency,
			RefSoID: po.RefSoID, RefTraceID: po.RefTraceID,
			CreatedAt: formatTime(po.CreatedAt),
		}
		if po.OrderedAt != nil {
			item.OrderedAt = formatTimePtr(po.OrderedAt)
		}
		if sup, err := sr.GetByID(po.DistributorID); err == nil {
			item.DistributorName = sup.Name
		}
		if n, err := pr.CountItems(po.ID); err == nil {
			item.ItemCount = int(n)
		}
		if specs := specsByPO[po.ID]; len(specs) > 0 {
			item.SkuSpecs = strings.Join(specs, "；")
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (s *DistOrderService) Get(id uint64) (*dto.DistOrderDetail, error) {
	po, err := s.repos.DistOrder.ForTenant(s.tenantID).GetWithItems(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.toDetail(po), nil
}

func (s *DistOrderService) Create(in *dto.DistOrderInput, buyerID uint64, buyerName string) (*dto.DistOrderDetail, error) {
	if _, err := s.repos.Distributor.ForTenant(s.tenantID).GetByID(in.DistributorID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	ft := defaultFulfillment(in.FulfillmentType)
	items, total, err := s.buildItems(in.DistributorID, ft, in.Items)
	if err != nil {
		return nil, err
	}
	fillItemSalesOrderRefs(items, in.RefSoID, in.RefTraceID)
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	const maxAttempts = 5
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		distNo, err := pr.NextDistNo()
		if err != nil {
			return nil, err
		}
		po := &model.DistOrder{
			DistNo: distNo, DistributorID: in.DistributorID,
			Status: model.DistStatusDraft, TotalAmount: total,
			SaleAmount: resolveSaleAmount(in.SaleAmount, items),
			Currency: defaultCurrency(in.Currency),
			FulfillmentType: ft,
			RefSoID: in.RefSoID, RefTraceID: strings.TrimSpace(in.RefTraceID),
			BuyerID: buyerID, BuyerName: buyerName,
			PayStatus: model.DistPayStatusUnpaid, Remark: in.Remark,
		}
		if d := parseDate(in.ExpectedArrivalDate); d != nil {
			po.ExpectedArrivalDate = d
		}
		if t := parseDateTime(in.OrderedAt); t != nil {
			po.OrderedAt = t
		} else {
			now := time.Now()
			po.OrderedAt = &now
		}
		lineItems := make([]model.DistOrderItem, len(items))
		copy(lineItems, items)
		for i := range lineItems {
			lineItems[i].ID = 0
			lineItems[i].DistOrderID = 0
		}
		if err := pr.Create(po, lineItems); err != nil {
			lastErr = err
			if isUniqueViolation(err) {
				continue
			}
			return nil, err
		}
		return s.Get(po.ID)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("创建分销订单失败：单号冲突")
}

func (s *DistOrderService) Update(id uint64, in *dto.DistOrderInput) (*dto.DistOrderDetail, error) {
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	po, err := pr.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if po.Status != model.DistStatusDraft {
		return nil, ErrImmutable
	}
	if _, err := s.repos.Distributor.ForTenant(s.tenantID).GetByID(in.DistributorID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	ft := defaultFulfillment(in.FulfillmentType)
	items, total, err := s.buildItems(in.DistributorID, ft, in.Items)
	if err != nil {
		return nil, err
	}
	fillItemSalesOrderRefs(items, in.RefSoID, in.RefTraceID)
	po.DistributorID = in.DistributorID
	po.TotalAmount = total
	po.SaleAmount = resolveSaleAmount(in.SaleAmount, items)
	po.Currency = defaultCurrency(in.Currency)
	po.FulfillmentType = ft
	po.WarehouseID = in.WarehouseID
	po.RefSoID = in.RefSoID
	po.RefTraceID = strings.TrimSpace(in.RefTraceID)
	po.Remark = in.Remark
	po.ExpectedArrivalDate = parseDate(in.ExpectedArrivalDate)
	if t := parseDateTime(in.OrderedAt); t != nil {
		po.OrderedAt = t
	}
	if err := pr.Save(po); err != nil {
		return nil, err
	}
	if err := pr.ReplaceItems(id, items); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// UpdateItemPrices 更新明细分销订单价并重算小计/采购总额。
// 代发单常自动提交并进入物流态后才补单价，故未付款且未完结即可改。
func (s *DistOrderService) UpdateItemPrices(id uint64, in *dto.UpdateDistOrderItemPricesInput) (*dto.DistOrderDetail, error) {
	if in == nil || len(in.Items) == 0 {
		return nil, ErrBadRequest
	}
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	po, err := pr.GetWithItems(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if po.Status == model.DistStatusCompleted || po.Status == model.DistStatusCancelled {
		return nil, ErrInvalidStatus
	}
	if po.PayStatus == model.DistPayStatusPaid || po.PayStatus == model.DistPayStatusPartial {
		return nil, fmt.Errorf("已付款单据不可修改分销订单价")
	}

	byID := make(map[uint64]*model.DistOrderItem, len(po.Items))
	for i := range po.Items {
		byID[po.Items[i].ID] = &po.Items[i]
	}
	for _, row := range in.Items {
		it, ok := byID[row.ItemID]
		if !ok {
			return nil, ErrNotFound
		}
		if it.Cancelled {
			continue
		}
		if row.UnitPrice < 0 {
			return nil, ErrBadRequest
		}
		it.UnitPrice = row.UnitPrice
		it.LineAmount = float64(it.Qty) * row.UnitPrice
		if err := pr.SaveItem(it); err != nil {
			return nil, err
		}
	}

	var totalActive float64
	for _, it := range po.Items {
		if it.Cancelled {
			continue
		}
		totalActive += it.LineAmount
	}
	po.TotalAmount = totalActive
	if err := pr.Save(po); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *DistOrderService) Delete(id uint64) error {
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	po, err := pr.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	// 已完成单据保留审计痕迹，其它状态允许删除（含关联物流/付款/入库）
	if po.Status == model.DistStatusCompleted {
		return ErrInvalidStatus
	}
	return pr.Delete(id)
}

func (s *DistOrderService) Submit(id uint64) (*dto.DistOrderDetail, error) {
	return s.transition(id, model.DistStatusDraft, model.DistStatusConfirmed, func(po *model.DistOrder) {
		if po.OrderedAt == nil {
			now := time.Now()
			po.OrderedAt = &now
		}
	})
}

func (s *DistOrderService) MarkPaid(id uint64) (*dto.DistOrderDetail, error) {
	return s.transition(id, model.DistStatusConfirmed, model.DistStatusPaid, func(po *model.DistOrder) {
		po.PayStatus = model.DistPayStatusPaid
	})
}

func (s *DistOrderService) Complete(id uint64) (*dto.DistOrderDetail, error) {
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	po, err := pr.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{
		model.DistStatusPaid: true, model.DistStatusPartialShipped: true,
		model.DistStatusShipped: true, model.DistStatusPartialReceived: true,
	}
	if !allowed[po.Status] {
		return nil, ErrInvalidStatus
	}
	now := time.Now()
	po.Status = model.DistStatusCompleted
	po.CompletedAt = &now
	if err := pr.Save(po); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *DistOrderService) Cancel(id uint64) (*dto.DistOrderDetail, error) {
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	po, err := pr.GetWithItems(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if po.Status != model.DistStatusDraft && po.Status != model.DistStatusConfirmed {
		return nil, ErrInvalidStatus
	}
	nowLabel := time.Now().Format("2006-01-02 15:04")
	for i := range po.Items {
		it := &po.Items[i]
		if it.Cancelled {
			continue
		}
		it.Cancelled = true
		tag := fmt.Sprintf("【已撤回 %s：整单取消】", nowLabel)
		if orderNo := strings.TrimSpace(it.RefOrderNo); orderNo != "" {
			tag = fmt.Sprintf("【已撤回 %s %s：整单取消】", orderNo, nowLabel)
		}
		if strings.TrimSpace(it.Remark) == "" {
			it.Remark = tag
		} else if !strings.Contains(it.Remark, "【已撤回") {
			it.Remark = tag + " " + it.Remark
		}
		if err := pr.SaveItem(it); err != nil {
			return nil, err
		}
	}
	po.SaleAmount = 0
	po.TotalAmount = 0
	po.RefSoID = 0
	po.RefTraceID = ""
	po.Status = model.DistStatusCancelled
	if err := pr.Save(po); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Merge 将多张草稿代发单合并到目标单：明细拼接、销售金额合计、关联单号合并；源单删除。
func (s *DistOrderService) Merge(in *dto.MergeDistOrdersInput) (*dto.MergeDistOrdersResult, error) {
	if in == nil || len(in.SourceDistOrderIDs) < 2 {
		return nil, ErrBadRequest
	}
	seen := map[uint64]struct{}{}
	ids := make([]uint64, 0, len(in.SourceDistOrderIDs))
	for _, id := range in.SourceDistOrderIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) < 2 {
		return nil, ErrBadRequest
	}
	targetID := in.TargetDistOrderID
	if targetID == 0 {
		targetID = ids[0]
	}
	if _, ok := seen[targetID]; !ok {
		return nil, ErrBadRequest
	}

	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	pos := make([]*model.DistOrder, 0, len(ids))
	var target *model.DistOrder
	for _, id := range ids {
		po, err := pr.GetWithItems(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		// 合并不限制订单状态；已付/部分付款仍不可合并
		if po.PayStatus == model.DistPayStatusPaid || po.PayStatus == model.DistPayStatusPartial {
			return nil, ErrInvalidStatus
		}
		if po.FulfillmentType != model.DistFulfillmentDropship {
			return nil, ErrBadRequest
		}
		pos = append(pos, po)
		if po.ID == targetID {
			target = po
		}
	}
	if target == nil {
		return nil, ErrNotFound
	}
	distributorID := target.DistributorID
	for _, po := range pos {
		if po.DistributorID != distributorID {
			return nil, ErrBadRequest
		}
	}

	mergedItems := make([]model.DistOrderItem, 0)
	oldItemIDs := make([]uint64, 0)
	traceParts := make([]string, 0)
	traceSeen := map[string]struct{}{}
	var saleTotal float64
	var totalAmount float64
	var refSoID uint64 = target.RefSoID

	addTrace := func(s string) {
		for _, p := range strings.Split(s, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, ok := traceSeen[p]; ok {
				continue
			}
			traceSeen[p] = struct{}{}
			traceParts = append(traceParts, p)
		}
	}

	// 目标单明细在前，其余按 id 顺序追加
	ordered := make([]*model.DistOrder, 0, len(pos))
	ordered = append(ordered, target)
	for _, po := range pos {
		if po.ID == target.ID {
			continue
		}
		ordered = append(ordered, po)
	}
	for _, po := range ordered {
		addTrace(po.RefTraceID)
		saleTotal += po.SaleAmount
		for _, it := range po.Items {
			if it.Cancelled {
				continue
			}
			oldItemIDs = append(oldItemIDs, it.ID)
			line := it
			line.ID = 0
			line.DistOrderID = 0
			line.CreatedAt = time.Time{}
			line.UpdatedAt = time.Time{}
			mergedItems = append(mergedItems, line)
			totalAmount += line.LineAmount
		}
		if refSoID == 0 && po.RefSoID > 0 {
			refSoID = po.RefSoID
		}
	}
	if len(mergedItems) == 0 {
		return nil, ErrBadRequest
	}

	// 合并后时间：取参与合并单中「最新」一条的采购时间、创建时间
	latest := latestPOByCreatedAt(pos)
	if latest != nil {
		target.CreatedAt = latest.CreatedAt
		if latest.OrderedAt != nil {
			t := *latest.OrderedAt
			target.OrderedAt = &t
		} else {
			target.OrderedAt = nil
		}
	}

	target.TotalAmount = totalAmount
	if saleTotal > 0 {
		target.SaleAmount = saleTotal
	}
	target.RefSoID = refSoID
	target.RefTraceID = strings.Join(traceParts, ",")
	if target.Remark == "" {
		target.Remark = fmt.Sprintf("合并代发 %d 单", len(ordered))
	} else if !strings.Contains(target.Remark, "合并") {
		target.Remark = target.Remark + fmt.Sprintf("（合并 %d 单）", len(ordered))
	}
	// 任一源单已下单，则合并后保留已下单状态
	for _, po := range ordered {
		if po.Status == model.DistStatusConfirmed {
			target.Status = model.DistStatusConfirmed
			break
		}
	}
	// 避免 Save 带出预加载的 Items 触发关联 upsert
	target.Items = nil
	if err := pr.Save(target); err != nil {
		return nil, err
	}
	// Save 默认不改 created_at，合并后强制对齐最新单时间
	if err := pr.UpdateHeaderTimes(target.ID, target.CreatedAt, target.OrderedAt); err != nil {
		return nil, err
	}
	if err := pr.ReplaceItems(target.ID, mergedItems); err != nil {
		return nil, err
	}
	if len(oldItemIDs) != len(mergedItems) {
		return nil, fmt.Errorf("合并明细数量不一致")
	}
	oldToNewItem := make(map[uint64]uint64, len(oldItemIDs))
	for i, oldID := range oldItemIDs {
		if mergedItems[i].ID == 0 {
			return nil, fmt.Errorf("合并后明细未生成 ID")
		}
		oldToNewItem[oldID] = mergedItems[i].ID
	}

	// 源单物流/付款/附件先挂到目标单，再删源单，避免物流被 Delete 级联清掉
	for _, po := range ordered {
		if po.ID == target.ID {
			continue
		}
		if err := pr.ReassignPOSideData(po.ID, target.ID); err != nil {
			return nil, err
		}
	}
	if err := s.repos.Shipment.ForTenant(s.tenantID).RemapItemDistOrderItemIDs(oldToNewItem); err != nil {
		return nil, err
	}

	mergedFrom := make([]string, 0, len(ordered)-1)
	for _, po := range ordered {
		if po.ID == target.ID {
			continue
		}
		mergedFrom = append(mergedFrom, po.DistNo)
		if err := pr.Delete(po.ID); err != nil {
			return nil, err
		}
	}

	_ = NewTrackingService(s.repos).ForTenant(s.tenantID).syncShipmentStatus(target.ID)

	detail, err := s.Get(target.ID)
	if err != nil {
		return nil, err
	}
	return &dto.MergeDistOrdersResult{
		DistOrderDetail: detail,
		MergedFromDistNos:     mergedFrom,
	}, nil
}

// DetachSalesOrder 从代发单撤回某笔销售单：对应明细标为已撤回（保留划线痕迹），更新备注与关联单号。
// 已付款/部分发货也可解绑（不冲销付款）；若全部明细已撤回且仍为草稿/已下单，则整单取消。
func (s *DistOrderService) DetachSalesOrder(in *dto.DetachSalesOrderInput) (*dto.DistOrderDetail, error) {
	if in == nil {
		return nil, ErrBadRequest
	}
	distNo := strings.TrimSpace(in.DistNo)
	orderNo := strings.TrimSpace(in.OrderNo)
	if distNo == "" || (orderNo == "" && in.SoID == 0) {
		return nil, ErrBadRequest
	}
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	po, err := pr.GetByDistNoWithItems(distNo)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if po.FulfillmentType != model.DistFulfillmentDropship {
		return nil, ErrBadRequest
	}
	// 解绑销售单：草稿/已下单/已付款/部分发货均可（仅划线明细并去掉关联，不冲销付款）。
	// 整单取消仍仅限草稿/已下单且全部明细已撤回。
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = "销售单撤回分配"
	}
	nowLabel := time.Now().Format("2006-01-02 15:04")
	matched := 0
	for i := range po.Items {
		it := &po.Items[i]
		if it.Cancelled {
			continue
		}
		hit := false
		if orderNo != "" && (it.RefOrderNo == orderNo || strings.Contains(it.Remark, orderNo)) {
			hit = true
		}
		if in.SoID > 0 && it.RefSoID == in.SoID {
			hit = true
		}
		if !hit {
			continue
		}
		it.Cancelled = true
		tag := fmt.Sprintf("【已撤回 %s：%s】", nowLabel, reason)
		if orderNo != "" && !strings.Contains(tag, orderNo) {
			tag = fmt.Sprintf("【已撤回 %s %s：%s】", orderNo, nowLabel, reason)
		}
		if strings.TrimSpace(it.Remark) == "" {
			it.Remark = tag
		} else if !strings.Contains(it.Remark, "【已撤回") {
			it.Remark = tag + " " + it.Remark
		}
		if err := pr.SaveItem(it); err != nil {
			return nil, err
		}
		matched++
	}
	// 整单已取消时仍允许补标（兼容历史数据），随后直接返回
	if po.Status == model.DistStatusCancelled {
		return s.Get(po.ID)
	}

	// 无 RefOrderNo 的历史明细：按「仍无法匹配」时，若 refTrace 含该单号也记入头备注
	if matched == 0 && orderNo != "" && !strings.Contains(po.RefTraceID, orderNo) {
		// 找不到明细也允许只更新头备注，避免卡住撤回
	}

	// 从关联单号中移除
	if orderNo != "" {
		parts := strings.Split(po.RefTraceID, ",")
		kept := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" || p == orderNo {
				continue
			}
			kept = append(kept, p)
		}
		po.RefTraceID = strings.Join(kept, ",")
	}
	if in.SoID > 0 && po.RefSoID == in.SoID {
		po.RefSoID = 0
		// 若还有其它未撤回明细带 RefSoID，取第一个
		for _, it := range po.Items {
			if !it.Cancelled && it.RefSoID > 0 {
				po.RefSoID = it.RefSoID
				break
			}
		}
	}

	var saleActive float64
	var totalActive float64
	activeLines := 0
	for _, it := range po.Items {
		if it.Cancelled {
			continue
		}
		activeLines++
		saleActive += it.SaleAmount
		totalActive += it.LineAmount
	}
	po.SaleAmount = saleActive
	po.TotalAmount = totalActive

	note := fmt.Sprintf("%s 撤回销售单 %s", nowLabel, orderNo)
	if orderNo == "" {
		note = fmt.Sprintf("%s 撤回销售单 #%d", nowLabel, in.SoID)
	}
	note = note + "（" + reason + "）"
	if strings.TrimSpace(po.Remark) == "" {
		po.Remark = note
	} else if !strings.Contains(po.Remark, note) {
		po.Remark = po.Remark + "；" + note
	}

	cancelWhole := activeLines == 0 && (po.Status == model.DistStatusDraft || po.Status == model.DistStatusConfirmed)
	if cancelWhole {
		po.Status = model.DistStatusCancelled
		po.RefSoID = 0
		// RefTraceID 已在逐单 detach 中清空；兜底再清一次
		po.RefTraceID = ""
	}
	if err := pr.Save(po); err != nil {
		return nil, err
	}
	return s.Get(po.ID)
}

func (s *DistOrderService) transition(id uint64, from, to string, apply func(*model.DistOrder)) (*dto.DistOrderDetail, error) {
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	po, err := pr.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if po.Status != from {
		return nil, ErrInvalidStatus
	}
	po.Status = to
	if apply != nil {
		apply(po)
	}
	if err := pr.Save(po); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// fillItemSalesOrderRefs 手工建单常只填单头关联；明细缺省时从单头补全，供物流同步/展示使用。
func fillItemSalesOrderRefs(items []model.DistOrderItem, refSoID uint64, refTraceID string) {
	orderNo := strings.TrimSpace(refTraceID)
	// 多单合并时 refTrace 可能是逗号分隔；单头 refSoID 仅对应首单，仍优先补有单头 id 的明细
	if refSoID == 0 && orderNo == "" {
		return
	}
	if strings.Contains(orderNo, ",") {
		// 多单场景：仅当明细已有 ref 时不覆盖；单头 orderNo 不宜整段写入每行
		orderNo = ""
	}
	for i := range items {
		if items[i].RefSoID == 0 && refSoID > 0 {
			items[i].RefSoID = refSoID
		}
		if strings.TrimSpace(items[i].RefOrderNo) == "" && orderNo != "" {
			items[i].RefOrderNo = orderNo
		}
	}
}

func (s *DistOrderService) buildItems(distributorID uint64, fulfillmentType string, inputs []dto.DistOrderItemInput) ([]model.DistOrderItem, float64, error) {
	or := s.repos.Offer.ForTenant(s.tenantID)
	dropship := fulfillmentType == model.DistFulfillmentDropship
	items := make([]model.DistOrderItem, 0, len(inputs))
	var total float64
	for _, in := range inputs {
		if in.Qty <= 0 {
			return nil, 0, ErrBadRequest
		}
		if !dropship && in.SkuID == 0 && in.OfferID == 0 {
			return nil, 0, errors.New("请选择 SKU 或批发价")
		}
		item := model.DistOrderItem{
			SkuID: in.SkuID, OfferID: in.OfferID,
			ProductName:     strings.TrimSpace(in.ProductName),
			SkuCode:         strings.TrimSpace(in.SkuCode),
			SkuSpecs:        strings.TrimSpace(in.SkuSpecs),
			PicURL:          strings.TrimSpace(in.PicURL),
			DistributorSkuCode: strings.TrimSpace(in.DistributorSkuCode),
			Qty:             in.Qty,
			SaleUnitPrice:   in.SaleUnitPrice,
			SaleAmount:      in.SaleAmount,
			RefSoID:         in.RefSoID,
			RefOrderNo:      strings.TrimSpace(in.RefOrderNo),
			Remark:          in.Remark,
		}
		if item.SaleAmount <= 0 && item.SaleUnitPrice > 0 {
			item.SaleAmount = item.SaleUnitPrice * float64(item.Qty)
		}
		if in.OfferID > 0 {
			offer, err := or.GetByID(in.OfferID)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, ErrNotFound
			}
			if err != nil {
				return nil, 0, err
			}
			if offer.DistributorID != distributorID {
				return nil, 0, errors.New("批发价与分销订单分销商不一致")
			}
			if item.SkuID == 0 {
				item.SkuID = offer.SkuID
			} else if item.SkuID != offer.SkuID {
				return nil, 0, errors.New("SKU 与批发价不匹配")
			}
			if item.DistributorSkuCode == "" {
				item.DistributorSkuCode = offer.DistributorSkuCode
			}
			if in.UnitPrice <= 0 {
				item.UnitPrice = offer.WholesalePrice
			} else {
				item.UnitPrice = in.UnitPrice
			}
		} else {
			// 代发草稿允许单价为 0（OMS 推送后在 SupplyCore 补价）
			if in.UnitPrice < 0 || (!dropship && in.UnitPrice <= 0) {
				return nil, 0, errors.New("请填写单价或选择批发价")
			}
			item.UnitPrice = in.UnitPrice
			// 未指定对方货号时：按分销商+SKU 取报价上的对方货号（不是平台 SKU）
			if item.DistributorSkuCode == "" && item.SkuID > 0 {
				if offers, err := or.ListBySku(item.SkuID, true); err == nil {
					for _, o := range offers {
						if o.DistributorID == distributorID && strings.TrimSpace(o.DistributorSkuCode) != "" {
							item.DistributorSkuCode = o.DistributorSkuCode
							break
						}
					}
				}
			}
		}
		item.LineAmount = float64(item.Qty) * item.UnitPrice
		total += item.LineAmount
		items = append(items, item)
	}
	return items, total, nil
}

func (s *DistOrderService) toDetail(po *model.DistOrder) *dto.DistOrderDetail {
	detail := &dto.DistOrderDetail{
		ID: po.ID, DistNo: po.DistNo, DistributorID: po.DistributorID,
		Status: po.Status, TotalAmount: po.TotalAmount, SaleAmount: po.SaleAmount, Currency: po.Currency,
		WarehouseID: po.WarehouseID, FulfillmentType: po.FulfillmentType,
		RefSoID: po.RefSoID, RefTraceID: po.RefTraceID, BuyerID: po.BuyerID, BuyerName: po.BuyerName,
		PayStatus: po.PayStatus, Remark: po.Remark,
		CreatedAt: formatTime(po.CreatedAt),
		Items: make([]dto.DistOrderItemDetail, 0, len(po.Items)),
	}
	if po.ExpectedArrivalDate != nil {
		detail.ExpectedArrivalDate = po.ExpectedArrivalDate.Format("2006-01-02")
	}
	if po.OrderedAt != nil {
		detail.OrderedAt = formatTimePtr(po.OrderedAt)
	}
	if po.CompletedAt != nil {
		detail.CompletedAt = formatTimePtr(po.CompletedAt)
	}
	if sup, err := s.repos.Distributor.ForTenant(s.tenantID).GetByID(po.DistributorID); err == nil {
		detail.DistributorName = sup.Name
		detail.DistributorCode = sup.Code
	}
	for _, it := range po.Items {
		detail.Items = append(detail.Items, dto.DistOrderItemDetail{
			ID: it.ID, SkuID: it.SkuID, OfferID: it.OfferID,
			ProductName: it.ProductName, SkuCode: it.SkuCode, SkuSpecs: it.SkuSpecs, PicURL: it.PicURL,
			DistributorSkuCode: it.DistributorSkuCode, Qty: it.Qty,
			SaleUnitPrice: it.SaleUnitPrice, SaleAmount: it.SaleAmount,
			UnitPrice: it.UnitPrice, LineAmount: it.LineAmount,
			ReceivedQty: it.ReceivedQty, RefSoID: it.RefSoID, RefOrderNo: it.RefOrderNo,
			Cancelled: it.Cancelled, Remark: it.Remark,
		})
	}
	return detail
}

func defaultCurrency(c string) string {
	if c == "" {
		return "CNY"
	}
	return c
}

func resolveSaleAmount(explicit float64, items []model.DistOrderItem) float64 {
	if explicit > 0 {
		return explicit
	}
	var sum float64
	for _, it := range items {
		sum += it.SaleAmount
	}
	return sum
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") || strings.Contains(msg, "23505")
}

// latestPOByCreatedAt 取创建时间最新的单据；相同时取 id 更大者。
func latestPOByCreatedAt(pos []*model.DistOrder) *model.DistOrder {
	var latest *model.DistOrder
	for _, po := range pos {
		if po == nil {
			continue
		}
		if latest == nil {
			latest = po
			continue
		}
		if po.CreatedAt.After(latest.CreatedAt) || (po.CreatedAt.Equal(latest.CreatedAt) && po.ID > latest.ID) {
			latest = po
		}
	}
	return latest
}

func defaultFulfillment(t string) string {
	if t == "" {
		return model.DistFulfillmentWholesale
	}
	return t
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil
	}
	return &t
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}
