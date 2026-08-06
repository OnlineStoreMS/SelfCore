package shippingcore

import (
	"context"
	"log"
	"strings"
)

// Client 物流中心对接（本期预留，不阻塞发货主流程）。
type Client struct {
	baseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

type NotifyInput struct {
	SelfOrderNo    string
	RefSoID        uint64
	ExpressCompany string
	ExpressNo      string
}

func (c *Client) NotifyShipment(ctx context.Context, bearerToken string, in NotifyInput) error {
	if !c.Enabled() {
		return nil
	}
	log.Printf("[shippingcore] reserved notify selfOrder=%s refSo=%d express=%s/%s",
		in.SelfOrderNo, in.RefSoID, in.ExpressCompany, in.ExpressNo)
	return nil
}
