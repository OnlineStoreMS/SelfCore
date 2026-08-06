package productcore

import (
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
			Timeout: 15 * time.Second,
		},
	}
}

type SkuSearchItem struct {
	ProductID     uint64            `json:"productId"`
	ProductName   string            `json:"productName"`
	MaterialCode  string            `json:"materialCode"`
	ProductSn     string            `json:"productSn"`
	ProductPic    string            `json:"productPic"`
	BrandName     string            `json:"brandName"`
	CategoryName  string            `json:"categoryName"`
	PublishStatus int8              `json:"publishStatus"`
	SkuID         uint64            `json:"skuId"`
	SkuCode       string            `json:"skuCode"`
	Specs         map[string]string `json:"specs"`
	SpecLabel     string            `json:"specLabel"`
	Price         float64           `json:"price"`
	Stock         int               `json:"stock"`
	Pic           string            `json:"pic"`
}

type ProductBrief struct {
	ID           uint64  `json:"id"`
	Name         string  `json:"name"`
	MaterialCode string  `json:"materialCode"`
	ProductSn    string  `json:"productSn"`
	Pic          string  `json:"pic"`
	BrandName    string  `json:"brandName"`
	CategoryName string  `json:"categoryName"`
	Price        float64 `json:"price"`
	Stock        int     `json:"stock"`
	SkuCount     int     `json:"skuCount"`
}

type ProductSkuItem struct {
	ID      uint64            `json:"id"`
	SkuCode string            `json:"skuCode"`
	Specs   map[string]string `json:"specs"`
	Price   float64           `json:"price"`
	Stock   int               `json:"stock"`
	Pic     string            `json:"pic"`
}

type ProductSkusPayload struct {
	ID           uint64           `json:"id"`
	Name         string           `json:"name"`
	MaterialCode string           `json:"materialCode"`
	Pic          string           `json:"pic,omitempty"`
	SkuCount     int              `json:"skuCount"`
	Skus         []ProductSkuItem `json:"skus"`
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

func (c *Client) SearchSkus(ctx context.Context, bearerToken, keyword string, page, pageSize int) ([]SkuSearchItem, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, 0, fmt.Errorf("keyword required")
	}
	page, pageSize = normalizePage(page, pageSize)
	q := url.Values{}
	q.Set("keyword", keyword)
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))
	var pageData pagePayload[SkuSearchItem]
	if err := c.getJSON(ctx, bearerToken, "/api/v1/admin/super-search?"+q.Encode(), &pageData); err != nil {
		return nil, 0, err
	}
	if pageData.List == nil {
		pageData.List = []SkuSearchItem{}
	}
	return pageData.List, pageData.Total, nil
}

func (c *Client) SearchProducts(ctx context.Context, bearerToken, keyword string, page, pageSize int) ([]ProductBrief, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	q := url.Values{}
	if kw := strings.TrimSpace(keyword); kw != "" {
		q.Set("keyword", kw)
	}
	q.Set("publishStatus", "1")
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))
	var pageData pagePayload[ProductBrief]
	if err := c.getJSON(ctx, bearerToken, "/api/v1/admin/products?"+q.Encode(), &pageData); err != nil {
		return nil, 0, err
	}
	if pageData.List == nil {
		pageData.List = []ProductBrief{}
	}
	return pageData.List, pageData.Total, nil
}

func (c *Client) GetProductSkus(ctx context.Context, bearerToken string, productID uint64) (*ProductSkusPayload, error) {
	if productID == 0 {
		return nil, fmt.Errorf("product id required")
	}
	var out ProductSkusPayload
	path := fmt.Sprintf("/api/v1/admin/products/%d/skus", productID)
	if err := c.getJSON(ctx, bearerToken, path, &out); err != nil {
		return nil, err
	}
	if out.Skus == nil {
		out.Skus = []ProductSkuItem{}
	}
	return &out, nil
}

func (c *Client) getJSON(ctx context.Context, bearerToken, path string, out any) error {
	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	if bearerToken != "" {
		if !strings.HasPrefix(bearerToken, "Bearer ") {
			bearerToken = "Bearer " + bearerToken
		}
		req.Header.Set("Authorization", bearerToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("productcore request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("productcore http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var wrapped apiBody
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return fmt.Errorf("productcore decode: %w", err)
	}
	if wrapped.Code != 200 {
		msg := wrapped.Message
		if msg == "" {
			msg = "productcore error"
		}
		return fmt.Errorf("%s", msg)
	}
	if err := json.Unmarshal(wrapped.Data, out); err != nil {
		return fmt.Errorf("productcore data decode: %w", err)
	}
	return nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return page, pageSize
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
