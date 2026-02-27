# AIAgent 工具调用契约（内部）

## 1) 创建短链

- Method: `POST`
- URL: `/internal/shortlinks`

Request:

```json
{
  "requestId": "req-123",
  "operatorId": "u-001",
  "tenantId": "t-001",
  "longUrl": "https://example.com/article?id=1",
  "customAlias": "promo2026"
}
```

Response:

```json
{
  "shortUrl": "https://s.example.com/promo2026",
  "shortCode": "promo2026"
}
```

## 2) 修改短链

- Method: `PATCH`
- URL: `/internal/shortlinks/{shortCode}`

Request:

```json
{
  "requestId": "req-124",
  "idempotencyKey": "idem-xxx",
  "operatorId": "u-001",
  "tenantId": "t-001",
  "longUrl": "https://example.com/new-target",
  "expireAt": "2026-12-31T23:59:59+08:00",
  "status": "active"
}
```

Response:

```json
{
  "updated": true,
  "shortCode": "promo2026",
  "longUrl": "https://example.com/new-target",
  "expireAt": "2026-12-31T23:59:59+08:00"
}
```

说明：`longUrl` 与 `expireAt` 在更新接口中可单独传递或同时传递，至少需要一个。

## 3) 查询流量

- Method: `GET`
- URL: `/internal/traffic/shortlinks/{shortCode}/summary?from=2026-02-01&to=2026-02-07`

Response:

```json
{
  "shortCode": "promo2026",
  "from": "2026-02-01",
  "to": "2026-02-07",
  "pv": 1024,
  "uv": 788,
  "topReferrers": [
    {"referrer": "wechat", "pv": 420},
    {"referrer": "weibo", "pv": 318}
  ]
}
```

