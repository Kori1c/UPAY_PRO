# 回调补发与失败原因展示设计

## 目标

补齐订单管理中的回调可观测性与人工干预能力：

- 订单列表不再由前端自行猜测回调状态
- 已支付订单能展示统一的回调状态与最近一次失败原因
- 后台可对未确认回调订单执行手动补发

## 设计范围

本次只覆盖后台订单列表、订单回调状态归一化、最近一次回调失败原因落库、补发回调接口与前端按钮。

本次不做：

- 回调历史表
- 回调批量补发
- 回调告警中心

## 后端设计

### 订单状态归一化

后端为订单列表增加统一字段：

- `callback_state`
- `callback_state_label`
- `callback_message`
- `can_retry_callback`

状态规则：

- `not_applicable`
  - 订单未支付，或没有 `notify_url`
- `confirmed`
  - `call_back_confirm == 1`
- `failed`
  - 已支付、存在 `notify_url`、`call_back_confirm != 1`、且 `callback_num > 0`
- `pending`
  - 已支付、存在 `notify_url`、`call_back_confirm != 1`、且 `callback_num == 0`

### 失败原因记录

在订单表增加 `CallbackMessage` 字段，保存最近一次回调失败原因。

写入规则：

- 回调失败时覆盖为最新错误
- 回调成功时清空

### 手动补发接口

新增后台接口：

- `POST /admin/api/orders/:id/retry-callback`

规则：

- 仅允许已支付订单补发
- `notify_url` 为空时拒绝补发
- 已确认订单默认不再补发
- 接口只负责触发后台异步补发，不等待完整 5 次重试跑完

## 前端设计

### 订单列表

前端不再自行根据 `callback_num` 和 `call_back_confirm` 推断状态，统一使用后端返回值。

列表增加：

- 回调状态标签
- 回调说明列
- 条件显示的“补发回调”操作按钮

### 展示规则

- 状态标签显示后端 `callback_state_label`
- 失败原因列显示 `callback_message`
- 过长内容使用省略 + tooltip

## 测试策略

- 后端单元测试覆盖状态归一化逻辑
- 后端接口测试覆盖补发接口的允许/拒绝场景
- 前端沿用现有构建与测试，重点验证 TypeScript 类型与构建通过
