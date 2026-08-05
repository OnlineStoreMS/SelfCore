package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"selfcore/internal/dto"
	"selfcore/internal/model"
	"selfcore/internal/repo"

	"gorm.io/gorm"
)

type TrackingService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewTrackingService(repos *repo.Repos) *TrackingService {
	return &TrackingService{repos: repos}
}

func (s *TrackingService) ForTenant(tenantID uint64) *TrackingService {
	return &TrackingService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *TrackingService) ensurePOTrackable(poID uint64) (*model.DistOrder, error) {
	po, err := s.repos.DistOrder.ForTenant(s.tenantID).GetByID(poID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if po.Status == model.DistStatusDraft || po.Status == model.DistStatusCancelled {
		return nil, ErrInvalidStatus
	}
	return po, nil
}

// --- Shipments ---

func (s *TrackingService) ListShipments(poID uint64) ([]dto.ShipmentDetail, error) {
	if _, err := s.repos.DistOrder.ForTenant(s.tenantID).GetByID(poID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	list, err := s.repos.Shipment.ForTenant(s.tenantID).ListByPO(poID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ShipmentDetail, 0, len(list))
	for i := range list {
		out = append(out, s.toShipmentDetail(&list[i]))
	}
	return out, nil
}

func (s *TrackingService) CreateShipment(poID uint64, in *dto.ShipmentInput) (*dto.ShipmentDetail, error) {
	if _, err := s.ensurePOTrackable(poID); err != nil {
		return nil, err
	}
	if len(in.Items) == 0 {
		return nil, ErrBadRequest
	}
	if strings.TrimSpace(in.TrackingNo) == "" {
		return nil, ErrBadRequest
	}
	no, err := s.repos.Shipment.ForTenant(s.tenantID).NextShipmentNo()
	if err != nil {
		return nil, err
	}
	sh := &model.DistShipment{
		DistOrderID: poID, ShipmentNo: no, Status: model.ShipmentStatusPending,
		CarrierCode: in.CarrierCode, CarrierName: in.CarrierName,
		TrackingNo: in.TrackingNo, ShipFromAddressID: in.ShipFromAddressID,
		ReceiverName: in.ReceiverName, ReceiverPhone: in.ReceiverPhone,
		ReceiverAddress: in.ReceiverAddress, Remark: in.Remark,
	}
	if d := parseDate(in.ExpectedArrivalDate); d != nil {
		sh.ExpectedArrivalDate = d
	}
	items, err := s.buildShipmentItems(poID, in.Items)
	if err != nil {
		return nil, err
	}
	if err := s.repos.Shipment.ForTenant(s.tenantID).Create(sh, items); err != nil {
		return nil, err
	}
	if err := s.syncShipmentStatus(poID); err != nil {
		return nil, err
	}
	detail := s.toShipmentDetail(sh)
	return &detail, nil
}

func (s *TrackingService) UpdateShipmentStatus(poID, shipmentID uint64, status string) (*dto.ShipmentDetail, error) {
	if _, err := s.ensurePOTrackable(poID); err != nil {
		return nil, err
	}
	sr := s.repos.Shipment.ForTenant(s.tenantID)
	sh, err := sr.GetByID(poID, shipmentID)
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
	if err := sr.Save(sh); err != nil {
		return nil, err
	}
	if err := s.syncShipmentStatus(poID); err != nil {
		return nil, err
	}
	detail := s.toShipmentDetail(sh)
	return &detail, nil
}

func (s *TrackingService) DeleteShipment(poID, shipmentID uint64) error {
	if _, err := s.ensurePOTrackable(poID); err != nil {
		return err
	}
	if err := s.repos.Shipment.ForTenant(s.tenantID).Delete(poID, shipmentID); err != nil {
		return err
	}
	return s.syncShipmentStatus(poID)
}

// SyncShipmentsFromOrders stubs OrderCore sync (not wired in SelfCore).
func (s *TrackingService) SyncShipmentsFromOrders(ctx context.Context, poID uint64, bearerToken string, in *dto.SyncShipmentsFromOrdersInput) (*dto.SyncShipmentsFromOrdersResult, error) {
	_ = ctx
	_ = poID
	_ = bearerToken
	_ = in
	return &dto.SyncShipmentsFromOrdersResult{}, nil
}

// --- Payments ---

func (s *TrackingService) ListReceipts(poID uint64) ([]dto.ReceiptDetail, error) {
	if _, err := s.repos.DistOrder.ForTenant(s.tenantID).GetByID(poID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	list, err := s.repos.Payment.ForTenant(s.tenantID).ListByPO(poID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ReceiptDetail, 0, len(list))
	for i := range list {
		out = append(out, s.toReceiptDetail(&list[i]))
	}
	return out, nil
}

func (s *TrackingService) CreateReceipt(poID uint64, in *dto.ReceiptInput) (*dto.ReceiptDetail, error) {
	po, err := s.ensurePOTrackable(poID)
	if err != nil {
		return nil, err
	}
	pay := &model.DistReceipt{
		DistOrderID: poID, PayAmount: in.PayAmount,
		PayMethod: in.PayMethod, PayAccount: in.PayAccount,
		PayeeAccount: in.PayeeAccount, PayeeName: in.PayeeName,
		PayStatus: defaultPayRecordStatus(in.PayStatus),
		Remark: in.Remark,
	}
	if in.PaidAt != "" {
		if t := parseDateTime(in.PaidAt); t != nil {
			pay.PaidAt = t
		}
	} else if pay.PayStatus == model.DistPayStatusPaid {
		now := time.Now()
		pay.PaidAt = &now
	}
	if err := s.repos.Payment.ForTenant(s.tenantID).Create(pay); err != nil {
		return nil, err
	}
	if err := s.syncPayStatus(po); err != nil {
		return nil, err
	}
	detail := s.toReceiptDetail(pay)
	return &detail, nil
}

func (s *TrackingService) UpdateReceipt(poID, paymentID uint64, in *dto.ReceiptInput) (*dto.ReceiptDetail, error) {
	po, err := s.ensurePOTrackable(poID)
	if err != nil {
		return nil, err
	}
	pr := s.repos.Payment.ForTenant(s.tenantID)
	pay, err := pr.GetByID(poID, paymentID)
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
	if err := pr.Save(pay); err != nil {
		return nil, err
	}
	if err := s.syncPayStatus(po); err != nil {
		return nil, err
	}
	detail := s.toReceiptDetail(pay)
	return &detail, nil
}

func (s *TrackingService) DeleteReceipt(poID, paymentID uint64) error {
	po, err := s.ensurePOTrackable(poID)
	if err != nil {
		return err
	}
	atts, err := s.repos.Attachment.ForTenant(s.tenantID).ListByPO(poID)
	if err != nil {
		return err
	}
	for _, a := range atts {
		if a.PaymentID == paymentID {
			if err := s.repos.Attachment.ForTenant(s.tenantID).Delete(poID, a.ID); err != nil {
				return err
			}
		}
	}
	if err := s.repos.Payment.ForTenant(s.tenantID).Delete(poID, paymentID); err != nil {
		return err
	}
	return s.syncPayStatus(po)
}

// --- Attachments ---

func (s *TrackingService) ListAttachments(poID uint64) ([]dto.AttachmentDetail, error) {
	if _, err := s.repos.DistOrder.ForTenant(s.tenantID).GetByID(poID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	list, err := s.repos.Attachment.ForTenant(s.tenantID).ListByPO(poID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.AttachmentDetail, 0, len(list))
	for i := range list {
		out = append(out, s.toAttachmentDetail(&list[i]))
	}
	return out, nil
}

func (s *TrackingService) CreateAttachment(poID, uploadedBy uint64, in *dto.AttachmentInput) (*dto.AttachmentDetail, error) {
	if _, err := s.ensurePOTrackable(poID); err != nil {
		return nil, err
	}
	a := &model.DistAttachment{
		DistOrderID: poID, PaymentID: in.PaymentID, ShipmentID: in.ShipmentID,
		FileType: in.FileType, FileName: in.FileName,
		FileURL: in.FileURL, UploadedBy: uploadedBy, Remark: in.Remark,
	}
	if err := s.repos.Attachment.ForTenant(s.tenantID).Create(a); err != nil {
		return nil, err
	}
	detail := s.toAttachmentDetail(a)
	return &detail, nil
}

func (s *TrackingService) DeleteAttachment(poID, attachmentID uint64) error {
	if _, err := s.ensurePOTrackable(poID); err != nil {
		return err
	}
	return s.repos.Attachment.ForTenant(s.tenantID).Delete(poID, attachmentID)
}

// --- sync ---

func (s *TrackingService) syncPayStatus(po *model.DistOrder) error {
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	fresh, err := pr.GetByID(po.ID)
	if err != nil {
		return err
	}
	sum, err := s.repos.Payment.ForTenant(s.tenantID).SumPaid(fresh.ID)
	if err != nil {
		return err
	}
	switch {
	case sum <= 0:
		fresh.PayStatus = model.DistPayStatusUnpaid
	case sum+0.001 < fresh.TotalAmount:
		fresh.PayStatus = model.DistPayStatusPartial
	default:
		// 付款累计金额 >= 采购总额时自动标记已付清
		fresh.PayStatus = model.DistPayStatusPaid
		if fresh.Status == model.DistStatusConfirmed {
			fresh.Status = model.DistStatusPaid
		}
	}
	*po = *fresh
	return pr.Save(fresh)
}

func (s *TrackingService) syncShipmentStatus(poID uint64) error {
	pr := s.repos.DistOrder.ForTenant(s.tenantID)
	po, err := pr.GetWithItems(poID)
	if err != nil {
		return err
	}
	if po.Status == model.DistStatusCompleted || po.Status == model.DistStatusCancelled || po.Status == model.DistStatusDraft {
		return nil
	}
	list, err := s.repos.Shipment.ForTenant(s.tenantID).ListByPO(poID)
	if err != nil || len(list) == 0 {
		return err
	}
	shippedQty := map[uint64]int{}
	hasInTransit, hasShipped, allDelivered := false, false, true
	for _, sh := range list {
		switch sh.Status {
		case model.ShipmentStatusInTransit:
			hasInTransit = true
			hasShipped = true
		case model.ShipmentStatusShipped, model.ShipmentStatusPending:
			hasShipped = true
		case model.ShipmentStatusDelivered:
			hasShipped = true
		}
		if sh.Status != model.ShipmentStatusDelivered {
			allDelivered = false
		}
		for _, it := range sh.Items {
			shippedQty[it.DistOrderItemID] += it.Qty
		}
	}
	fullyShipped := true
	activeLines := 0
	for _, it := range po.Items {
		if it.Cancelled {
			continue
		}
		activeLines++
		if shippedQty[it.ID] < it.Qty {
			fullyShipped = false
			break
		}
	}
	if activeLines == 0 {
		fullyShipped = false
	}

	switch {
	case allDelivered && fullyShipped:
		if po.FulfillmentType == model.DistFulfillmentDropship {
			po.Status = model.DistStatusCompleted
		} else {
			po.Status = model.DistStatusPartialReceived
		}
	case hasInTransit || (fullyShipped && hasShipped):
		// 明细已全部发出：标已发货，不再标「部分发货」
		po.Status = model.DistStatusShipped
	case hasShipped:
		po.Status = model.DistStatusPartialShipped
	}
	return pr.Save(po)
}

func (s *TrackingService) buildShipmentItems(poID uint64, inputs []ShipmentItemInput) ([]model.DistShipmentItem, error) {
	po, err := s.repos.DistOrder.ForTenant(s.tenantID).GetWithItems(poID)
	if err != nil {
		return nil, err
	}
	itemMap := map[uint64]model.DistOrderItem{}
	for _, it := range po.Items {
		itemMap[it.ID] = it
	}
	shippedQty := map[uint64]int{}
	existing, err := s.repos.Shipment.ForTenant(s.tenantID).ListByPO(poID)
	if err != nil {
		return nil, err
	}
	for _, sh := range existing {
		for _, it := range sh.Items {
			shippedQty[it.DistOrderItemID] += it.Qty
		}
	}
	seen := map[uint64]struct{}{}
	items := make([]model.DistShipmentItem, 0, len(inputs))
	for _, in := range inputs {
		if _, dup := seen[in.DistOrderItemID]; dup {
			return nil, ErrBadRequest
		}
		seen[in.DistOrderItemID] = struct{}{}
		poItem, ok := itemMap[in.DistOrderItemID]
		if !ok {
			return nil, ErrNotFound
		}
		remain := poItem.Qty - shippedQty[in.DistOrderItemID]
		if in.Qty <= 0 || in.Qty > remain {
			return nil, ErrBadRequest
		}
		items = append(items, model.DistShipmentItem{
			DistOrderItemID: in.DistOrderItemID, Qty: in.Qty,
		})
	}
	return items, nil
}

type ShipmentItemInput = dto.ShipmentItemInput

func (s *TrackingService) toShipmentDetail(sh *model.DistShipment) dto.ShipmentDetail {
	d := dto.ShipmentDetail{
		ID: sh.ID, PoID: sh.DistOrderID, ShipmentNo: sh.ShipmentNo, Status: sh.Status,
		CarrierCode: sh.CarrierCode, CarrierName: sh.CarrierName, TrackingNo: sh.TrackingNo,
		ReceiverName: sh.ReceiverName, ReceiverPhone: sh.ReceiverPhone,
		ReceiverAddress: sh.ReceiverAddress, Remark: sh.Remark,
		CreatedAt: formatTime(sh.CreatedAt),
		Items: make([]dto.ShipmentItemDetail, 0, len(sh.Items)),
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
	po, _ := s.repos.DistOrder.ForTenant(s.tenantID).GetWithItems(sh.DistOrderID)
	skuByItem := map[uint64]uint64{}
	if po != nil {
		for _, it := range po.Items {
			skuByItem[it.ID] = it.SkuID
		}
	}
	for _, it := range sh.Items {
		d.Items = append(d.Items, dto.ShipmentItemDetail{
			ID: it.ID, DistOrderItemID: it.DistOrderItemID, SkuID: skuByItem[it.DistOrderItemID], Qty: it.Qty,
		})
	}
	return d
}

func (s *TrackingService) toReceiptDetail(p *model.DistReceipt) dto.ReceiptDetail {
	d := dto.ReceiptDetail{
		ID: p.ID, PoID: p.DistOrderID, PayAmount: p.PayAmount,
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

func (s *TrackingService) toAttachmentDetail(a *model.DistAttachment) dto.AttachmentDetail {
	return dto.AttachmentDetail{
		ID: a.ID, PoID: a.DistOrderID, PaymentID: a.PaymentID, ShipmentID: a.ShipmentID,
		FileType: a.FileType, FileName: a.FileName, FileURL: a.FileURL,
		UploadedBy: a.UploadedBy, Remark: a.Remark,
		CreatedAt: formatTime(a.CreatedAt),
	}
}

func isValidShipmentStatus(s string) bool {
	switch s {
	case model.ShipmentStatusPending, model.ShipmentStatusShipped,
		model.ShipmentStatusInTransit, model.ShipmentStatusDelivered, model.ShipmentStatusException:
		return true
	}
	return false
}

func defaultPayRecordStatus(s string) string {
	if s == "" {
		return model.DistPayStatusPaid
	}
	return s
}

func parseDateTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t
		}
		if t, err := time.Parse(layout, s); err == nil {
			local := t.In(time.Local)
			return &local
		}
	}
	return nil
}
