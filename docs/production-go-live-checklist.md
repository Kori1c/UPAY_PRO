# UPay Pro 发布前检查清单

更新时间：2026-04-23

这份清单用于把当前二开分支从“本地可用”推进到“更接近正式环境可上线”。它不是架构设计文档，而是一份面向部署和验收的执行清单。

## Ready

- 管理后台已切换到 Vue 3 + Vite，并完成当前主流程联调
- 推荐使用预构建 Docker 镜像：`docker compose pull && docker compose up -d`
- 如需本地源码构建，可使用 `docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build`
- 默认后台账号密码为 `admin / admin`
- 登录、订单管理、钱包管理、系统设置、支付页已完成当前版本联调
- 支付成功后的异步回调链路已在 Docker 环境完成实测
- `manual-complete-order` 已支持更稳的按 `trade_id` / `order_id` 查询与幂等处理
- Redis 配置保存后支持自动重载与失败回滚
- 后台已支持单管理员 Passkey 登录，支持注册凭证后禁用密码登录
- Passkey 本地测试需使用 `http://localhost:8090`，不要使用 `127.0.0.1`
- 应用提供 `/healthz` 探活接口，Docker healthcheck 已检查数据库与 Redis
- 后台提供 `/admin/api/operations/summary` 运营状态摘要，可快速查看订单、回调和钱包风险信号
- 已提供 `scripts/smoke-check.sh`，可快速验证健康检查、登录、后台统计与 Passkey 设置接口
- 已补充同金额订单小并发回归测试，覆盖 `actual_amount` 唯一偏移与落库数量
- 已提供 `scripts/load-test-create-orders.mjs`，可对本地 `/api/create_order` 做可重复并发压测

## Needs Monitoring

- TRON 扫链的外部接口仍依赖第三方配额与稳定性
- TronGrid 的 `429` 已做冷却止血，但仍建议继续观察日志
- 正式环境仍建议配置自己的 Tronscan / TronGrid / Etherscan Key
- 异步回调、Telegram、Bark 目前依赖配置是否完整，建议上线后首日重点盯日志
- Passkey 与正式域名强绑定，切换域名、反向代理或 HTTPS 配置后必须重新做登录验证
- 单钱包同基础金额的待支付订单数量受 `IncrementalMaximumNumber` 限制，压测或高峰期需关注容量耗尽错误

## Not Ready

- 尚未完成针对正式业务流量的长时间并发压测，目前已有本地短时压测脚本
- 尚未补齐完整的监控、告警和自动备份方案
- 尚未形成正式发布版本的回滚预案与值班操作手册
- Passkey 已完成代码与本地接口验证，但仍需要在目标浏览器和正式访问域名下完成一次人工验收

## 上线前必做

### 1. 基础环境

- 确认服务器时区、NTP 时间同步正常
- 确认部署机器磁盘空间足够，尤其是 `DBS/` 与 `logs/`
- 确认反向代理、HTTPS、域名解析已经可用
- 确认 Redis 与应用之间网络可达
- 本地或发布后运行一次 `./scripts/smoke-check.sh`
- 使用正式前先运行一次 `scripts/load-test-create-orders.mjs`，确认成功率、延迟和 `actual_amount` 唯一性

### 2. 系统配置

- 修改默认后台账号密码
- 检查 `AppUrl` 是否为正式访问域名
- 检查 `SecretKey` 是否已替换为正式值
- 检查订单过期时间是否符合业务要求
- 检查客服联系方式是否已填写
- 如启用 Passkey，确认使用正式 HTTPS 域名访问后台，且 `AppUrl` 与实际访问域名一致
- 注册至少一个 Passkey 后，再决定是否禁用密码登录
- 禁用密码登录前，确认 Passkey 能完成退出后的重新登录

### 3. API Key

- 配置 Tronscan Key
- 配置 TronGrid Key
- 配置 Etherscan Key
- 至少完成一次“创建订单 -> 打开支付页 -> 回调成功”的实链路验证

### 4. 钱包与汇率

- 每条启用中的钱包地址都要先校验格式正确
- 检查每个币种至少有一条可用钱包
- 检查手动汇率或自动汇率是否符合当前业务定价
- 对正式收款钱包先做一笔小额验证

### 5. 回调与通知

- 检查商户 `notify_url` 从容器网络内可达
- 如在 Docker 内调宿主机服务，使用 `host.docker.internal`
- 检查 Telegram / Bark 配置是否完整
- 检查回调成功后数据库中的 `call_back_confirm` 是否变为 `1`

### 6. Passkey 登录

- 使用正式访问域名打开后台，不要混用 IP、localhost 与正式域名
- 本地 Docker 验证统一使用 `http://localhost:8090/login`
- 登录后台后进入系统设置，注册一个 Passkey
- 退出登录后，点击“使用 Passkey 登录”，确认无需输入账号即可登录
- 注册成功后再测试禁用密码登录
- 禁用密码登录后，确认登录页不再显示账号/密码输入框
- 禁用密码登录后，确认系统不允许删除最后一个 Passkey
- 如更换正式域名或反向代理配置，重新注册并验证 Passkey

## 当前压测记录

- 2026-04-23 本地 Docker：`TOTAL_ORDERS=20 CONCURRENCY=5 ORDER_AMOUNT=10`，成功 20，失败 0，重复 `actual_amount` 0
- 2026-04-23 本地 Docker：`TOTAL_ORDERS=100 CONCURRENCY=20 ORDER_AMOUNT=10`，成功 80，失败 20，原因是上一轮同金额待支付订单仍占用偏移容量
- 2026-04-23 本地 Docker：`TOTAL_ORDERS=100 CONCURRENCY=20 ORDER_AMOUNT=11`，成功 100，失败 0，重复 `actual_amount` 0，P95 约 390ms

## 建议监控项

- 应用容器存活状态
- Redis 存活状态
- `/healthz` 是否返回 `status=ok`
- `/admin/api/operations/summary` 是否能返回 `status=ok` 或明确的 `warnings`
- 应用日志中的 `ERROR`、`panic`、`429`
- 回调失败次数异常增长
- 订单长期停留在“待支付”但未过期的数量
- SQLite 文件体积与磁盘占用增长

## 建议备份项

- `DBS/upay_pro.db`
- `logs/`
- 发布前或升级前运行 `./scripts/backup-runtime-data.sh`，备份产物默认写入 `backups/`
- 反向代理配置
- Docker Compose 与镜像版本信息
- 系统设置截图或导出记录

## 每次发布后建议验证

- 登录后台是否正常
- Passkey 登录是否正常
- `./scripts/smoke-check.sh` 是否通过
- 下单压测是否无失败、无重复 `actual_amount`
- 仪表盘数据是否能正常加载
- 新建测试订单是否成功
- 支付页是否可以打开
- 支付成功后是否能回调成功
- 订单状态是否会在后台刷新
- 退出登录是否正常
