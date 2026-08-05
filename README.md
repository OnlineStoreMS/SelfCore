# SelfCore — OSMS 自营中心

独立应用，与 [UserCore](../UserCore)（IAM）、[ProductCore](../ProductCore)（PIM）、[SupplyCore](../SupplyCore)（供应链）并列部署。覆盖分销商、SKU 批发价、分销订单（代发/批发）及收款/发货跟踪。

| 组件 | 端口 | 说明 |
|------|------|------|
| API | **8103** | Go + Gin + GORM |
| Web | **5187** | Vue 3 + Element Plus |
| Docker 镜像 | `selfcore-api`、`selfcore-web` | 见 [deploy](../deploy) |
| UserCore app | `selfcore` | 权限 `self:read` / `self:write` |
| 对象存储 | MinIO bucket `selfcore` 或本地 `./data/uploads` | |
| 端口约定 | [deploy/docs/PORTS.md](../deploy/docs/PORTS.md) | |

## 当前能力（MVP）

### 分销商（Distributors）
- 分销商分类 CRUD
- 分销商 CRUD；地址、收款账户、收款二维码

### SKU 批发价
- `sku-prices` CRUD；`GET /skus/{id}/wholesale-options`
- 对接 ProductCore 商品/SKU 搜索

### 分销订单（Dist orders）
- 代发 / 批发订单：创建、合并、拆销售单、改价、同步采购价
- 状态流转：提交 → 标记已付 → 完成 / 取消
- 发货单、收款记录、附件上传（含移动端扫码传图）

### 自营订单（OrderCore 代理）
- 订单搜索、详情、解密、发货（`/orders/*`）

### 工作台
- Dashboard 统计与趋势

## 快速开始

```bash
cp configs/config.example.yaml configs/config.yaml
# jwt_secret 必须与 UserCore 一致

go run ./cmd/api -config configs/config.yaml

cd web && npm i && npm run dev
```

浏览器访问 http://localhost:5187 ，从 UserCore 应用中心进入（需已注册 `selfcore` 应用）。

### 数据库

默认 PostgreSQL（与平台其他 Core 一致）。首次可走平台脚本：

```bash
cd ~/projects/deploy
./scripts/init-external-db.sh      # 含 selfcore
./scripts/init-external-minio.sh   # 含 selfcore bucket
```

本地 smoke 可用 SQLite：在 `configs/config.yaml` 中设置：

```yaml
database:
  driver: sqlite
  sqlite_path: "./data/selfcore.db"
```

## UserCore 注册

应用中心应出现 **自营中心**（app key `selfcore`，入口 Web `:5187`）。平台配置见 `deploy/configs/usercore.yaml` 的 `apps.selfcore_url`。

权限码：

- `self:read` — 查看分销商、批发价、分销订单
- `self:write` — 编辑与状态操作

## 主要 API（JWT Admin）

前缀 `/api/v1/admin`（另有公开 `GET /health`、移动端 `/api/v1/mobile/photo-upload/:token`）。

| 域 | 路径 |
|----|------|
| 工作台 | `GET /dashboard/stats`、`/dashboard/trend` |
| 自营订单 | `GET /orders/search`、`GET /orders/:id`、`POST /orders/decrypt`、`POST /orders/:id/ship` |
| 分销商分类 | `/distributor-categories` |
| 分销商 | `/distributors`、`.../addresses`、`.../payment-accounts`、`.../payment-qrs` |
| 批发价 | `/sku-prices`、`GET /skus/:id/wholesale-options` |
| 商品代理 | `/product-skus/search`、`/products/search`、`/products/:id/skus` |
| 分销订单 | `/dist-orders`（含 merge / submit / mark-paid / complete / cancel 等） |
| 发货/收款/附件 | `/dist-orders/:id/shipments|receipts|attachments` |
| 上传 | `POST /upload`、photo-upload sessions |

## 环境变量（前端）

| 变量 | 默认 |
|------|------|
| `VITE_PORTAL_URL` | `http://localhost:5174` |
| `VITE_API_GATEWAY` | 未设置则直连 8103 |

## 平台编排

`deploy` 已接入：`selfcore-api` / `selfcore-web`、MinIO bucket `selfcore`、`SELFCORE_DB_*`、`make build-api-selfcore` / `build-web-selfcore`。
