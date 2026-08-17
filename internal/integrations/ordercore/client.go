package ordercore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			// 发货回传快递助手可能超过 1 分钟
			Timeout: 180 * time.Second,
		},
	}
}

type OrderItemBrief struct {
	ID          uint64  `json:"id"`
	SkuID       uint64  `json:"skuId"`
	SkuCode     string  `json:"skuCode"`
	ProductName string  `json:"productName"`
	SkuSpecs    string  `json:"skuSpecs"`
	PicURL      string  `json:"picUrl"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	TotalAmount float64 `json:"totalAmount"`
}

type OrderAddressBrief struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Province string `json:"province"`
	City     string `json:"city"`
	District string `json:"district"`
	Address  string `json:"address"`
	FullText string `json:"fullText"`
}

type OrderShipmentItemBrief struct {
	OrderItemID uint64 `json:"orderItemId"`
	Qty         int    `json:"qty"`
}

type OrderShipmentBrief struct {
	ID             uint64                   `json:"id"`
	ShipmentNo     string                   `json:"shipmentNo"`
	ExpressCompany string                   `json:"expressCompany"`
	ExpressNo      string                   `json:"expressNo"`
	ShippedAt      *string                  `json:"shippedAt,omitempty"`
	Remark         string                   `json:"remark"`
	Items          []OrderShipmentItemBrief `json:"items,omitempty"`
}

type OrderBrief struct {
	ID                   uint64               `json:"id"`
	OrderNo              string               `json:"orderNo"`
	SourceChannel        string               `json:"sourceChannel"`
	Platform             string               `json:"platform"`
	PlatformOrderID      string               `json:"platformOrderId"`
	PlatformSysTid       string               `json:"platformSysTid"`
	ShopName             string               `json:"shopName"`
	BuyerName            string               `json:"buyerName"`
	BuyerNick            string               `json:"buyerNick"`
	BuyerPhone           string               `json:"buyerPhone"`
	Status               string               `json:"status"`
	ShipStatus           string               `json:"shipStatus"`
	AllocType            string               `json:"allocType"`
	TotalAmount          float64              `json:"totalAmount"`
	PayAmount            float64              `json:"payAmount"`
	Remark               string               `json:"remark"`
	SellerRemark         string               `json:"sellerRemark"`
	FenFaRemark          string               `json:"fenFaRemark"`
	PrinterRemark        string               `json:"printerRemark"`
	AllocRemark          string               `json:"allocRemark"`
	PlatformStatus       string               `json:"platformStatus"`
	PlatformStatusText   string               `json:"platformStatusText"`
	EcommerceStatus      string               `json:"ecommerceStatus"`
	EcommerceStatusText  string               `json:"ecommerceStatusText"`
	AfterSaleStatus      string               `json:"afterSaleStatus"`
	AfterSaleStatusText  string               `json:"afterSaleStatusText"`
	AgentType            int                  `json:"agentType"`
	PayTime              *string              `json:"payTime,omitempty"`
	OrderedAt            *string              `json:"orderedAt,omitempty"`
	ShippedAt            *string              `json:"shippedAt,omitempty"`
	CreatedAt            string               `json:"createdAt"`
	Address              *OrderAddressBrief   `json:"address,omitempty"`
	Shipments            []OrderShipmentBrief `json:"shipments,omitempty"`
	Items                []OrderItemBrief     `json:"items,omitempty"`
}

// OrderListQuery mirrors OrderCore admin list filters.
type OrderListQuery struct {
	SourceChannel string
	Status        string
	ShipStatus    string
	AllocType     string
	Keyword       string
	Platform      string
	SalesChannel  string
	OrderedAtStart string
	OrderedAtEnd   string
	ShippedAtStart string
	ShippedAtEnd   string
	PayTimeStart   string
	PayTimeEnd     string
	Page           int
	PageSize       int
}

// FormatReceiverAddress 拼接收件地址展示文案。
func FormatReceiverAddress(addr *OrderAddressBrief) string {
	if addr == nil {
		return ""
	}
	if strings.TrimSpace(addr.FullText) != "" {
		return strings.TrimSpace(addr.FullText)
	}
	parts := make([]string, 0, 4)
	for _, p := range []string{addr.Province, addr.City, addr.District, addr.Address} {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "")
}

type pagePayload[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

type apiBody struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) ListOrders(ctx context.Context, bearerToken string, q OrderListQuery) ([]OrderBrief, int64, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	vals := url.Values{}
	vals.Set("page", strconv.Itoa(q.Page))
	vals.Set("pageSize", strconv.Itoa(q.PageSize))
	set := func(k, v string) {
		v = strings.TrimSpace(v)
		if v != "" {
			vals.Set(k, v)
		}
	}
	set("sourceChannel", q.SourceChannel)
	set("status", q.Status)
	set("shipStatus", q.ShipStatus)
	set("allocType", q.AllocType)
	set("keyword", q.Keyword)
	set("platform", q.Platform)
	set("salesChannel", q.SalesChannel)
	set("orderedAtStart", q.OrderedAtStart)
	set("orderedAtEnd", q.OrderedAtEnd)
	set("shippedAtStart", q.ShippedAtStart)
	set("shippedAtEnd", q.ShippedAtEnd)
	set("payTimeStart", q.PayTimeStart)
	set("payTimeEnd", q.PayTimeEnd)

	var pageData pagePayload[OrderBrief]
	if err := c.getJSON(ctx, bearerToken, "/api/v1/admin/orders?"+vals.Encode(), &pageData); err != nil {
		return nil, 0, err
	}
	if pageData.List == nil {
		pageData.List = []OrderBrief{}
	}
	return pageData.List, pageData.Total, nil
}

func (c *Client) SearchOrders(ctx context.Context, bearerToken, keyword string, page, pageSize int) ([]OrderBrief, int64, error) {
	return c.ListOrders(ctx, bearerToken, OrderListQuery{
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	})
}

func (c *Client) GetOrder(ctx context.Context, bearerToken string, id uint64) (*OrderBrief, error) {
	if id == 0 {
		return nil, fmt.Errorf("order id required")
	}
	var out OrderBrief
	if err := c.getJSON(ctx, bearerToken, fmt.Sprintf("/api/v1/admin/orders/%d", id), &out); err != nil {
		return nil, err
	}
	if out.Items == nil {
		out.Items = []OrderItemBrief{}
	}
	if out.Shipments == nil {
		out.Shipments = []OrderShipmentBrief{}
	}
	return &out, nil
}

type ShipRequest struct {
	ExpressCompany string          `json:"expressCompany"`
	ExpressNo      string          `json:"expressNo"`
	Remark         string          `json:"remark"`
	Callback       *bool           `json:"callback,omitempty"`
	Items          []ShipItemInput `json:"items,omitempty"`
}

type ShipItemInput struct {
	OrderItemID uint64 `json:"orderItemId"`
	Qty         int    `json:"qty"`
}

type UpdatePaymentRequest struct {
	PayStatus    string  `json:"payStatus"`
	PayTime      *string `json:"payTime,omitempty"`
	ClearPayTime bool    `json:"clearPayTime,omitempty"`
}

// UpdatePayment 回写订单中心付款状态（手工单才会落库）。
func (c *Client) UpdatePayment(ctx context.Context, bearerToken string, orderID uint64, req UpdatePaymentRequest) error {
	if orderID == 0 {
		return fmt.Errorf("order id required")
	}
	body := map[string]any{
		"payStatus":    req.PayStatus,
		"clearPayTime": req.ClearPayTime,
	}
	if req.PayTime != nil {
		body["payTime"] = *req.PayTime
	}
	return c.putJSON(ctx, bearerToken, fmt.Sprintf("/api/v1/admin/orders/%d/payment", orderID), body, nil)
}

// ShipOrder 调用订单中心填写物流（电商订单可回传 StoreSyncAgent）。
func (c *Client) ShipOrder(ctx context.Context, bearerToken string, orderID uint64, req ShipRequest) (*OrderBrief, error) {
	if orderID == 0 {
		return nil, fmt.Errorf("order id required")
	}
	if strings.TrimSpace(req.ExpressNo) == "" {
		return nil, fmt.Errorf("expressNo required")
	}
	var out OrderBrief
	if err := c.postJSON(ctx, bearerToken, fmt.Sprintf("/api/v1/admin/orders/%d/ship", orderID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type DecryptOrdersResult struct {
	Items   []OrderBrief `json:"items"`
	Success int          `json:"success"`
}

// DecryptOrders 调用订单中心解密电商收件信息。
func (c *Client) DecryptOrders(ctx context.Context, bearerToken string, orderIDs []uint64) (*DecryptOrdersResult, error) {
	if len(orderIDs) == 0 {
		return nil, fmt.Errorf("orderIds required")
	}
	var out DecryptOrdersResult
	if err := c.postJSON(ctx, bearerToken, "/api/v1/admin/orders/decrypt", map[string]any{
		"orderIds": orderIDs,
	}, &out); err != nil {
		return nil, err
	}
	if out.Items == nil {
		out.Items = []OrderBrief{}
	}
	return &out, nil
}

func (c *Client) RelinkPurchaseOrder(ctx context.Context, bearerToken string, fromPoNos []string, toPoNo string) (int64, error) {
	if len(fromPoNos) == 0 {
		return 0, fmt.Errorf("fromPoNos required")
	}
	body := map[string]any{
		"fromPoNos": fromPoNos,
		"toPoNo":    strings.TrimSpace(toPoNo),
	}
	var out struct {
		Updated int64  `json:"updated"`
		ToPoNo  string `json:"toPoNo"`
	}
	if err := c.postJSON(ctx, bearerToken, "/api/v1/admin/orders/relink-purchase-order", body, &out); err != nil {
		return 0, err
	}
	return out.Updated, nil
}

// UnlinkDropshipPO 供应链解绑销售单后回写订单中心（清空采购单号，可选清分配）。
func (c *Client) UnlinkDropshipPO(ctx context.Context, bearerToken string, orderIDs []uint64, orderNos []string, clearAlloc bool, remark string) (int64, error) {
	if len(orderIDs) == 0 && len(orderNos) == 0 {
		return 0, fmt.Errorf("orderIds or orderNos required")
	}
	body := map[string]any{
		"orderIds":   orderIDs,
		"orderNos":   orderNos,
		"clearAlloc": clearAlloc,
		"remark":     strings.TrimSpace(remark),
	}
	var out struct {
		Updated int64 `json:"updated"`
	}
	if err := c.postJSON(ctx, bearerToken, "/api/v1/admin/orders/unlink-dropship-po", body, &out); err != nil {
		return 0, err
	}
	return out.Updated, nil
}

func (c *Client) getJSON(ctx context.Context, bearerToken, path string, out any) error {
	return c.doJSON(ctx, http.MethodGet, bearerToken, path, nil, out)
}

func (c *Client) postJSON(ctx context.Context, bearerToken, path string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPost, bearerToken, path, body, out)
}

func (c *Client) putJSON(ctx context.Context, bearerToken, path string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPut, bearerToken, path, body, out)
}

func (c *Client) doJSON(ctx context.Context, method, bearerToken, path string, body any, out any) error {
	reqURL := c.baseURL + path
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reader)
	if err != nil {
		return err
	}
	if bearerToken != "" {
		if !strings.HasPrefix(bearerToken, "Bearer ") {
			bearerToken = "Bearer " + bearerToken
		}
		req.Header.Set("Authorization", bearerToken)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ordercore request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ordercore http %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var wrapped apiBody
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return fmt.Errorf("ordercore decode: %w", err)
	}
	if wrapped.Code != 200 && wrapped.Code != 201 {
		msg := wrapped.Message
		if msg == "" {
			msg = "ordercore error"
		}
		return fmt.Errorf("%s", msg)
	}
	if out == nil || len(wrapped.Data) == 0 || string(wrapped.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(wrapped.Data, out); err != nil {
		return fmt.Errorf("ordercore data decode: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
