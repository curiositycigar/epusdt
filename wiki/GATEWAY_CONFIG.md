# gateway.yaml 配置说明

本文档说明纯网关模式下 `gateway.yaml` 的用途、字段结构、同步行为和使用建议。

适用场景：

- 不使用 Epusdt 内置后台页面
- 不通过 `/admin/api/v1/*` 管理商户、钱包、链和设置
- 由运维或业务系统通过文件方式维护网关配置

相关 `.env` 配置：

```env
gateway_pure_mode=true
gateway_config=gateway.yaml
gateway_payment_url_template=https://your-app.example/pay/{trade_id}
```

说明：

- `gateway_pure_mode=true`
  - 关闭安装向导
  - 不注册后台管理 API
  - 不挂载内置后台和收银台页面
- `gateway_config`
  - 指定 `gateway.yaml` 路径
  - 相对路径相对于 `.env` 所在目录解析
- `gateway_payment_url_template`
  - 可选
  - 用于生成 `create-transaction` 返回的 `payment_url`
  - 支持 `{trade_id}` 占位符

## 总体结构

最小完整示例：

```yaml
merchants:
  - name: app-a
    pid: "1000"
    secret_key: "replace-with-random-secret"
    notify_url: "https://merchant.example/notify"
    ip_whitelist:
      - "127.0.0.1/32"
    enabled: true

wallets:
  - network: tron
    address: TTestTronAddress001
    remark: primary tron wallet
    source: manual
    enabled: true

chains:
  - network: tron
    display_name: TRON
    enabled: true
    min_confirmations: 1
    scan_interval_sec: 5

chain_tokens:
  - network: tron
    symbol: USDT
    contract_address: TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t
    decimals: 6
    enabled: true
    min_amount: 0

rpc_nodes:
  - network: tron
    url: https://api.trongrid.io
    type: http
    weight: 1
    enabled: true
    purpose: general
    status: unknown

settings:
  - group: system
    key: system.amount_precision
    value: "2"
    type: int
  - group: rate
    key: rate.forced_rate_list
    value: '{"cny":{"usdt":0.14635}}'
    type: json

notification_channels: []
```

## merchants

用于定义商户身份，也就是 `pid + secret_key`。

字段：

- `name`
  - 商户名称，仅用于识别和展示
- `pid`
  - 商户 PID
  - 下单请求和回调验签都会用到
  - 必填，建议全局唯一
- `secret_key`
  - 商户签名密钥
  - 必填
- `notify_url`
  - 默认回调地址
  - 当前订单创建时真正使用的还是请求里的 `notify_url`
  - 这个字段更适合留作管理信息或后续扩展
- `ip_whitelist`
  - IP 白名单数组
  - 会在启动同步时转成逗号分隔字符串
  - 支持单 IP 和 CIDR
- `enabled`
  - 是否启用
  - `true` 映射为启用
  - `false` 映射为禁用

示例：

```yaml
merchants:
  - name: app-a
    pid: "1000"
    secret_key: "secret-app-a"
    ip_whitelist:
      - "203.0.113.10/32"
      - "203.0.113.11/32"
    enabled: true
  - name: app-b
    pid: "1001"
    secret_key: "secret-app-b"
    enabled: true
```

建议：

- `secret_key` 使用高强度随机值
- `pid` 建议固定，不要频繁变更
- 如果有多个上游系统，每个系统单独分配一个 `pid`

## wallets

用于定义收款地址池。

字段：

- `network`
  - 网络名
  - 如 `tron`、`solana`、`ethereum`、`binance`、`polygon`、`plasma`
- `address`
  - 收款地址
  - 必填
- `remark`
  - 备注
- `source`
  - 地址来源
  - 可选值：`manual`、`import`
  - 不填默认 `manual`
- `enabled`
  - 是否启用

示例：

```yaml
wallets:
  - network: tron
    address: TXXXX...
    remark: tron-main-1
    enabled: true
  - network: solana
    address: 9abc...
    remark: sol-main-1
    enabled: true
```

说明：

- EVM 地址会按系统原逻辑标准化为小写
- 下单时只会从启用状态的钱包中分配地址

## chains

用于定义链是否启用，以及监听相关参数。

字段：

- `network`
  - 网络标识
  - 必填
- `display_name`
  - 展示名称
- `enabled`
  - 是否启用该链
- `min_confirmations`
  - 最小确认数
- `scan_interval_sec`
  - 扫描间隔秒数
- `extra`
  - 扩展字段，字符串形式

示例：

```yaml
chains:
  - network: tron
    display_name: TRON
    enabled: true
    min_confirmations: 1
    scan_interval_sec: 5
  - network: binance
    display_name: BSC
    enabled: true
    min_confirmations: 3
    scan_interval_sec: 5
```

说明：

- 链禁用后，下单和监听都会受影响
- 如果某条链没有启用，就算钱包存在，也不能创建该链订单

## chain_tokens

用于定义链上支持的币种、合约和精度。

字段：

- `network`
  - 所属网络
- `symbol`
  - 币种符号，如 `USDT`、`USDC`、`TRX`、`SOL`
- `contract_address`
  - 合约地址
  - 原生币可留空
- `decimals`
  - 精度
- `enabled`
  - 是否启用
- `min_amount`
  - 最小金额

示例：

```yaml
chain_tokens:
  - network: tron
    symbol: USDT
    contract_address: TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t
    decimals: 6
    enabled: true
  - network: tron
    symbol: TRX
    contract_address: ""
    decimals: 6
    enabled: true
```

说明：

- 监听器通过这里的配置识别要监听的 token
- 原生币通常使用空合约地址

## rpc_nodes

用于定义各链的 RPC 节点。

字段：

- `network`
  - 网络名
- `url`
  - RPC 地址
- `type`
  - `http` 或 `ws`
- `weight`
  - 权重
- `api_key`
  - 节点服务商 API Key，可选
- `enabled`
  - 是否启用
- `purpose`
  - 用途
  - 可选值：
    - `general`
    - `manual_verify`
    - `both`
- `status`
  - 初始状态
  - 一般建议 `unknown`

示例：

```yaml
rpc_nodes:
  - network: tron
    url: https://api.trongrid.io
    type: http
    weight: 1
    enabled: true
    purpose: general
    status: unknown
  - network: binance
    url: wss://bsc.drpc.org
    type: ws
    weight: 1
    enabled: true
    purpose: general
    status: unknown
```

建议：

- 每条链至少准备一个可用节点
- `manual_verify` 适合放高质量、低频使用的补单节点
- 普通监听优先用 `general`

## settings

用于同步 `settings` 表中的键值配置。

字段：

- `group`
  - 设置分组
  - 常见值：`rate`、`epay`、`brand`、`okpay`、`system`
- `key`
  - 设置 key
- `value`
  - 设置值，始终按字符串写入
- `type`
  - 类型声明
  - 常见值：`string`、`int`、`bool`、`json`

示例：

```yaml
settings:
  - group: system
    key: system.amount_precision
    value: "2"
    type: int
  - group: rate
    key: rate.forced_rate_list
    value: '{"cny":{"usdt":0.14635}}'
    type: json
  - group: epay
    key: epay.default_network
    value: tron
    type: string
  - group: epay
    key: epay.default_token
    value: usdt
    type: string
  - group: epay
    key: epay.default_currency
    value: cny
    type: string
```

常用 key：

- `rate.forced_rate_list`
  - 强制汇率
- `epay.default_network`
- `epay.default_token`
- `epay.default_currency`
- `okpay.enabled`
- `okpay.shop_id`
- `okpay.shop_token`
- `okpay.api_url`

说明：

- `settings` 适合放运行期会被业务逻辑读取的参数
- `order_expiration_time` 这类仍然主要来自 `.env`，不是这里

## notification_channels

用于同步通知渠道。

字段：

- `type`
  - 类型，如 `telegram`
- `name`
  - 名称
- `config`
  - JSON 对象
- `events`
  - 事件订阅映射
- `enabled`
  - 是否启用

示例：

```yaml
notification_channels:
  - type: telegram
    name: ops-bot
    config:
      bot_token: "123:ABC"
      chat_id: 123456789
      proxy: ""
    events:
      pay_success: true
      order_expired: true
      daily_report: false
    enabled: true
```

说明：

- 这是网关自身的通知渠道，不是商户回调
- 商户支付回调仍然是订单级 `notify_url`

## 同步规则

`gateway.yaml` 会在服务启动时同步到数据库。

当前同步规则：

- `merchants`
  - 按 `pid` 查找并更新
  - 不存在则创建
- `wallets`
  - 按 `network + address` 查找并更新
  - 不存在则创建
- `chains`
  - 按 `network` 查找并更新
  - 不存在则创建
- `chain_tokens`
  - 按 `network + symbol` 查找并更新
  - 不存在则创建
- `rpc_nodes`
  - 按 `network + url` 查找并更新
  - 不存在则创建
- `settings`
  - 按 `key` upsert
- `notification_channels`
  - 按 `type + name` 查找并更新
  - 不存在则创建

注意：

- 当前是“增量同步”，不会自动删除数据库里但文件中已经移除的项
- 如果你希望彻底清理旧数据，建议手动处理数据库

## payment_url 行为

纯网关模式下，`create-transaction` 返回值中的 `payment_url` 行为如下：

- 如果未配置 `gateway_payment_url_template`
  - 返回空字符串
- 如果配置了模板
  - 使用模板渲染
  - 例如：

```env
gateway_payment_url_template=https://your-app.example/pay/{trade_id}
```

返回：

```text
https://your-app.example/pay/abc123
```

## 推荐实践

- 每个业务系统单独使用一个 `pid`
- `secret_key` 使用随机强密钥，不要复用
- 同一条链尽量配置多个 RPC 节点
- 不使用的链和币种显式设为 `enabled: false`
- `rate.forced_rate_list` 建议在对账要求高时显式配置
- `gateway.yaml` 建议纳入私有配置仓库或 Secret 管理系统，不要直接公开提交

## 与数据库的关系

纯网关模式不是“去数据库化”。

数据库仍然用于保存：

- 订单
- 子订单
- 回调重试状态
- 交易锁
- provider order
- 运行时扫描状态

`gateway.yaml` 只是用来替代后台页面维护“静态配置”。
