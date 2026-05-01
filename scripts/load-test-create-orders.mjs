#!/usr/bin/env node
import crypto from 'node:crypto'

const config = {
  baseUrl: process.env.BASE_URL || 'http://localhost:8090',
  secretKey: process.env.UPAY_SECRET_KEY || '',
  total: Number(process.env.TOTAL_ORDERS || 50),
  concurrency: Number(process.env.CONCURRENCY || 10),
  amount: Number(process.env.ORDER_AMOUNT || 10),
  type: process.env.ORDER_TYPE || 'USDT-TRC20',
  notifyUrl: process.env.NOTIFY_URL || 'https://example.com/notify',
  redirectUrl: process.env.REDIRECT_URL || 'https://example.com/return',
  prefix: process.env.ORDER_PREFIX || `LOAD-${Date.now()}`,
}

function assertConfig() {
  if (!config.secretKey) {
    throw new Error('UPAY_SECRET_KEY is required')
  }
  if (!Number.isInteger(config.total) || config.total <= 0) {
    throw new Error('TOTAL_ORDERS must be a positive integer')
  }
  if (!Number.isInteger(config.concurrency) || config.concurrency <= 0) {
    throw new Error('CONCURRENCY must be a positive integer')
  }
  if (config.amount < 0.01) {
    throw new Error('ORDER_AMOUNT must be at least 0.01')
  }
}

function signatureFor(payload) {
  const pairs = [
    `type=${payload.type}`,
    `amount=${payload.amount}`,
    `notify_url=${payload.notify_url}`,
    `order_id=${payload.order_id}`,
    `redirect_url=${payload.redirect_url}`,
  ]
  pairs.sort()
  return crypto
    .createHash('md5')
    .update(`${pairs.join('&')}${config.secretKey}`)
    .digest('hex')
}

async function createOrder(index) {
  const orderId = `${config.prefix}-${String(index + 1).padStart(5, '0')}`
  const payload = {
    type: config.type,
    order_id: orderId,
    amount: config.amount,
    notify_url: config.notifyUrl,
    redirect_url: config.redirectUrl,
  }
  payload.signature = signatureFor(payload)

  const startedAt = performance.now()
  const response = await fetch(`${config.baseUrl}/api/create_order`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  const latencyMs = performance.now() - startedAt
  const body = await response.json().catch(() => ({}))

  if (!response.ok || body.status_code !== 200) {
    return {
      ok: false,
      orderId,
      status: response.status,
      latencyMs,
      message: body.message || body.error || 'request failed',
    }
  }

  return {
    ok: true,
    orderId,
    status: response.status,
    latencyMs,
    tradeId: body.data?.trade_id,
    actualAmount: body.data?.actual_amount,
    token: body.data?.token,
  }
}

async function runPool() {
  const results = []
  let nextIndex = 0

  async function worker() {
    while (nextIndex < config.total) {
      const current = nextIndex
      nextIndex += 1
      results[current] = await createOrder(current)
    }
  }

  const workerCount = Math.min(config.concurrency, config.total)
  await Promise.all(Array.from({ length: workerCount }, () => worker()))
  return results
}

function percentile(values, p) {
  if (values.length === 0) {
    return 0
  }
  const sorted = [...values].sort((a, b) => a - b)
  const index = Math.min(sorted.length - 1, Math.ceil((p / 100) * sorted.length) - 1)
  return sorted[index]
}

function summarize(results, elapsedMs) {
  const successes = results.filter((result) => result.ok)
  const failures = results.filter((result) => !result.ok)
  const latencies = results.map((result) => result.latencyMs)
  const actualAmountCounts = new Map()
  const duplicateActualAmounts = []
  const capacityExhaustedFailures = failures.filter((result) =>
    String(result.message || '').includes('递增金额次数超过最大次数'),
  )

  for (const result of successes) {
    const key = `${result.token}:${result.actualAmount}`
    const count = actualAmountCounts.get(key) || 0
    actualAmountCounts.set(key, count + 1)
  }

  for (const [key, count] of actualAmountCounts.entries()) {
    if (count > 1) {
      duplicateActualAmounts.push({ key, count })
    }
  }

  return {
    baseUrl: config.baseUrl,
    prefix: config.prefix,
    total: config.total,
    concurrency: config.concurrency,
    success: successes.length,
    failure: failures.length,
    elapsedMs: Math.round(elapsedMs),
    rps: Number((results.length / (elapsedMs / 1000)).toFixed(2)),
    latency: {
      minMs: Math.round(Math.min(...latencies)),
      p50Ms: Math.round(percentile(latencies, 50)),
      p95Ms: Math.round(percentile(latencies, 95)),
      maxMs: Math.round(Math.max(...latencies)),
    },
    uniqueActualAmounts: actualAmountCounts.size,
    duplicateActualAmounts,
    capacityExhaustedFailures: capacityExhaustedFailures.length,
    capacityHint: capacityExhaustedFailures.length > 0
      ? '同一钱包、同一基础金额的待支付订单已耗尽偏移容量。请等待订单过期、换 ORDER_AMOUNT、增加钱包地址，或评估是否需要调整 IncrementalMaximumNumber。'
      : '',
    firstFailures: failures.slice(0, 5),
  }
}

async function main() {
  assertConfig()
  const startedAt = performance.now()
  const results = await runPool()
  const elapsedMs = performance.now() - startedAt
  const summary = summarize(results, elapsedMs)

  console.log(JSON.stringify(summary, null, 2))

  if (summary.failure > 0) {
    process.exitCode = 1
  }
  if (summary.duplicateActualAmounts.length > 0) {
    process.exitCode = 1
  }
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
