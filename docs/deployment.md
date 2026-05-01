# UPay Pro 部署方式

本文档提供一个不依赖特定服务器、域名和密钥的通用部署方案。

## 1. 环境要求

- Linux 服务器
- Docker 24+
- Docker Compose
- 可用域名
- HTTPS 证书

建议配置：

- 2 CPU
- 2 GB 内存
- 20 GB 以上磁盘

## 2. 获取代码

```bash
git clone <your-private-repo-url>
cd UPAY_PRO
```

## 3. 目录说明

运行后会使用以下目录保存数据：

- `DBS/`：SQLite 数据库
- `logs/`：应用日志
- `backups/`：手动备份产物

这些目录不要删除，也不要直接提交回 Git。

## 4. 启动方式

### 方式 A：Docker Compose

推荐使用仓库自带的 `docker-compose.yml`。

```bash
docker compose up -d --build
```

查看状态：

```bash
docker compose ps
docker compose logs -f app
```

停止服务：

```bash
docker compose down
```

彻底清理：

```bash
docker compose down -v
```

## 5. 反向代理

容器默认监听：

- `127.0.0.1:8090` 或服务器 `8090` 端口

建议通过 Nginx 反向代理到正式域名，例如：

```nginx
server {
    listen 80;
    server_name pay.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name pay.example.com;

    ssl_certificate /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

## 6. 首次上线后必须做的事

1. 打开后台：`https://pay.example.com/login`
2. 使用默认账号密码登录：`admin / admin`
3. 立即修改默认密码
4. 配置系统设置：
   - 页面名称
   - 页面地址
   - 通信密钥
   - 订单过期时间
   - 客服联系方式
5. 配置钱包地址和汇率
6. 配置链上 API Key：
   - Tronscan
   - TronGrid
   - Etherscan
7. 配置通知能力：
   - Telegram
   - Bark
8. 用真实商户环境完成一次完整链路验证：
   - 创建订单
   - 打开支付页
   - 支付成功
   - 商户收到回调

## 7. Passkey 部署注意事项

- Passkey 必须在正式域名和 HTTPS 下使用
- `AppUrl` 必须与真实访问域名一致
- 如更换域名，需要重新注册 Passkey
- 建议先注册至少一个 Passkey，再禁用密码登录

## 8. 健康检查

应用提供：

```http
GET /healthz
```

预期返回：

```json
{
  "code": 0,
  "data": {
    "db": "ok",
    "redis": "ok",
    "status": "ok"
  }
}
```

本地或线上发布后建议执行：

```bash
./scripts/smoke-check.sh
```

## 9. 升级方式

### Docker Compose 升级

```bash
git pull
docker compose up -d --build
```

升级前建议先备份：

```bash
./scripts/backup-runtime-data.sh
```

## 10. 备份与恢复

手动备份：

```bash
./scripts/backup-runtime-data.sh
```

建议至少备份：

- `DBS/upay_pro.db`
- `logs/`
- 当前镜像版本
- Nginx 配置

## 11. 发布前验证清单

- `go test ./...`
- `cd frontend && npm test`
- `cd frontend && npm run build`
- `./scripts/smoke-check.sh`
- 创建一笔测试订单并完成支付
- 确认商户回调成功
