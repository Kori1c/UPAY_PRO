# UPay Pro

UPay Pro 是一个基于 Go 构建的加密货币支付网关，提供订单创建、支付页展示、链上支付状态校验、异步回调、通知推送和管理后台。

当前仓库包含：

- Go + Gin 后端
- Vue 3 + Vite + TypeScript 管理后台
- SQLite 持久化
- Redis 作为金额占位与异步队列依赖
- Docker / Docker Compose 部署方式

## 功能概览

- 支持多链多币种：USDT-TRC20、TRX、USDT-Polygon、USDT-BSC、USDT-ERC20、USDT-ArbitrumOne、USDC-ERC20、USDC-Polygon、USDC-BSC、USDC-ArbitrumOne
- 支持支付页、订单状态轮询、订单过期处理
- 支持商户回调、Bark / Telegram 通知
- 支持后台钱包管理、订单管理、系统设置、API Key 设置
- 支持单管理员 Passkey 登录

## 技术栈

- 后端：Go、Gin、Gorm、SQLite
- 前端：Vue 3、Vite、TypeScript、Arco Design Vue、Pinia、Vue Router
- 队列与缓存：Redis、Asynq
- 部署：Docker、Docker Compose、Nginx 反向代理

## 快速开始

### 本地运行

1. 安装 Go、Node.js、Redis
2. 安装前端依赖
3. 启动后端和前端构建

```bash
cd frontend
npm ci
npm run build

cd ..
go run .
```

默认访问地址：

- 后台：`http://localhost:8090/login`
- 支付页：`http://localhost:8090/pay/checkout-counter/{trade_id}`

默认后台账号密码：

- `admin / admin`

首次启动后请立即修改默认密码，或直接注册 Passkey 后禁用密码登录。

### Docker Compose

```bash
docker compose up -d --build
```

常用命令：

```bash
docker compose logs -f app
docker compose down
docker compose down -v
./scripts/smoke-check.sh
./scripts/backup-runtime-data.sh
```

## 配置说明

系统核心配置可在后台完成，也可以通过初始环境变量提供默认值：

- `UPAY_HTTP_PORT`
- `UPAY_APP_URL`
- `UPAY_REDIS_HOST`
- `UPAY_REDIS_PORT`
- `UPAY_REDIS_PASSWORD`
- `UPAY_REDIS_DB`

说明：

- 当前仓库不会预置任何真实通信密钥、区块链 API Key、通知 Key
- API Key 请在部署后到后台单独填写
- 正式环境必须配置自己的域名、HTTPS、通信密钥和链上 API Key

## 部署文档

完整部署方式见：

- [docs/deployment.md](docs/deployment.md)
- [docs/production-go-live-checklist.md](docs/production-go-live-checklist.md)

## 测试与验证

后端测试：

```bash
go test ./...
```

前端测试：

```bash
cd frontend
npm test
```

前端构建：

```bash
cd frontend
npm run build
```

本地冒烟检查：

```bash
./scripts/smoke-check.sh
```

并发下单测试：

```bash
UPAY_SECRET_KEY=your-secret TOTAL_ORDERS=50 CONCURRENCY=10 ./scripts/load-test-create-orders.mjs
```

## 项目结构

```text
.
├── frontend/         # Vue 3 管理后台
├── web/              # HTTP 路由与接口
├── db/               # SQLite / Redis 配置
├── cron/             # 订单轮询、回调、定时任务
├── notification/     # Bark / Telegram 通知
├── scripts/          # 冒烟、备份、压测脚本
├── static/           # 支付页与构建产物目录
└── docs/             # 部署与上线文档
```

## API

创建订单：

```http
POST /api/create_order
Content-Type: application/json

{
  "type": "USDT-TRC20",
  "order_id": "ORDER123456",
  "amount": 100.0,
  "notify_url": "https://merchant.example.com/notify",
  "redirect_url": "https://merchant.example.com/return",
  "signature": "md5_signature"
}
```

查询订单状态：

```http
GET /pay/check-status/{trade_id}
```

支付页：

```http
GET /pay/checkout-counter/{trade_id}
```

更详细的接口说明见：

- [支付接口API文档.md](支付接口API文档.md)
