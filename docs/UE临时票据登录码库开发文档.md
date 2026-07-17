# UE 临时票据登录码库开发文档

## 1. 文档目的

本文档用于指导 uscene-api-main 接入 UE/码库临时票据登录能力。

目标流程：

1. 客户端向 uscene 后端申请临时票据；
2. uscene 后端根据项目配置、用户绑定状态和设备标识调用码库接口；
3. 如果用户未绑定 UE 账号，客户端根据返回码选择绑定或注册；
4. 注册或绑定完成后，客户端重新申请临时票据；
5. 客户端使用临时票据调用码库 LoginTmpTicketAsync 完成登录。

本文档基于：

- UeService.cs：UE 端已有的临时票据和注册逻辑；
- TmpTicketProjectOpt.cs：临时票据项目配置；
- 当前 uscene-api-main 的 UE SDK、路由、中间件和数据库模型。

本文档同时作为后端开发方案、接口契约和联调依据。当前实现已包含 uscene 路由、Redis 临时票据存储和可配置的 UE 上游适配层。UE 方已提供临时票据相关地址，测试环境配置如下；请求字段和加密协议仍需以 UE 方联调结果为最终准则。

~~~toml
[ue.tmp_ticket]
enabled = true
ticket_ttl_seconds = 120
flow_ttl_seconds = 300
get_ticket_url = "http://client2.ue.shumaoheng.com/svr/ThirdSvr/GetTmpTicket"
register_url = "http://client2.ue.shumaoheng.com/svr/UserSvr/Reg"
register_zs_url = "http://client2.ue.shumaoheng.com/svr/ThirdSvr/RegZsUser"
~~~

## 2. 当前 uscene 已有能力

### 2.1 已有 UE SDK 调用

当前 SDK 位于：

~~~text
pkg/ueapisdk/ue_api_v2/ue_api_v2.go
~~~

SDK 的基础地址由以下逻辑初始化：

~~~go
ue_api_v2.Initialization(n.Config.UE.UeApiUrl)
~~~

### 2.2 当前真正调用的码库接口

| 码库接口 | SDK 方法 | 当前调用位置 | 用途 |
|---|---|---|---|
| svr/AuthSvr/GetConfig | ue_api_v2.GetConfig | pkg/nucl/ue.go 的 LoginCore | 获取服务端公钥、AES 密钥等配置 |
| svr/AuthSvr/Login | ue_api_v2.Login | pkg/nucl/ue.go 的 LoginCore | 使用 AccessKeyId/AccessSecret 登录码库 |
| svr/AuthSvr/RefreshToken | ue_api_v2.RefreshToken | pkg/nucl/ue.go 的 RefreshCore | 刷新 UE AccessToken 和 AES 密钥 |
| 回调验签/解密 | ue_api_v2.GetResponse | cmd/uscene/z_ue.go | 处理 UE 回调，不是主动 HTTP 接口 |

### 2.3 当前回调接口

当前路由包括：

~~~http
POST /api/v1/ue/notify
POST /api/v1/ue/secret/open
POST /api/v1/ue/secret/close
GET  /api/v1/ue/secret/status
GET  /api/v1/ue/bind/status
POST /api/v1/ue/secret/status
~~~

其中：

- /ue/notify：支付或业务回调；
- /ue/secret/open：UE 免密绑定成功回调；
- /ue/secret/close：UE 免密解绑回调；
- /ue/secret/status、/ue/bind/status：查询当前用户的 UE 绑定状态。

/ue/secret/open 和 /ue/secret/close 会读取以下请求头：

~~~text
accesskeyid
signbody
aesiv
~~~

请求体经过 UE 加密和签名，后端使用 UE 登录产生的 AES Key、服务端公钥进行验签解密。

### 2.4 SDK 中存在但当前没有实际调用的接口

以下方法目前只在 SDK 中定义，没有在业务代码中发现实际调用：

| SDK 方法 | 目标接口 | 当前状态 |
|---|---|---|
| PayCallValidate | svr/PurseSvr/PayCallValidate | 仅定义，未实际调用 |
| PayCallValidateV2 | svr/PurseSvr/PayCallValidateV2 | 仅定义，未实际调用 |
| CreateOrder | svr/PurseSvr/CreateOrder | 仅定义，业务调用已注释 |
| CreateOrderV2 | svr/PurseSvr/CreateOrderV2 | 仅定义，业务调用已注释 |
| UpdateTCodeStatus | svr/SicIndexSvr/UpdateTCodeStatus | 仅定义，未实际调用 |

### 2.5 已实现的临时票据接口

当前实现已新增以下 uscene 后端路由，并通过 SDK 适配器调用 UE 方提供的上游地址：

~~~text
POST /api/v1/ue/tmp-ticket
POST /api/v1/ue/register
GET  /api/v1/ue/tmp-ticket/bind/status
~~~

对应上游接口：

| uscene 内部用途 | UE 上游地址 |
|---|---|
| 获取临时票据 | `http://client2.ue.shumaoheng.com/svr/ThirdSvr/GetTmpTicket` |
| 手机注册 UE 账号 | `http://client2.ue.shumaoheng.com/svr/UserSvr/Reg` |
| ZS 项目注册 | `http://client2.ue.shumaoheng.com/svr/ThirdSvr/RegZsUser` |

`LoginTmpTicketAsync` 仍由客户端原生 SDK 调用，后端不代理该登录动作。

## 3. UeService.cs 中的实际业务逻辑

### 3.1 GetTmpTicket2Async 流程

GetTmpTicket2Async 的逻辑为：

1. 校验 clientHardid、clientIp、projectId；
2. 根据 nodeid 查询用户注册信息；
3. 根据 projectId 查询临时票据项目配置；
4. 如果项目要求绑定 UE 账号，查询本地绑定关系；
5. 未绑定时返回 -13 或 -14；
6. 已绑定时调用 GetTmpTicketAsync；
7. 如果项目不要求绑定，则调用 RegZsUserAsync 并直接返回 DeviceId。

### 3.2 GetTmpTicketAsync 的上游参数

源码实际构造的上游请求包括：

~~~text
Opentype     = 10
Nodecode     = 当前用户的 UE 开放节点编码
ClientHardId = 客户端设备唯一标识
ClientIp     = 客户端 IP
~~~

这和客户端流程文档中列出的 version、sid、clientid、projectId 不完全一致。后几个字段可能由 UE SDK 的基础请求、AccessToken 或服务端配置自动补齐。

最终实现必须以 UE 方的真实接口协议为准，不能只根据客户端文档猜测字段映射。

### 3.3 -13 和 -14 的实际产生位置

在 UeService.cs 中，-13、-14 是根据本地项目配置和绑定状态产生的：

~~~text
RequiredBindUeAccount = 1
当前用户未绑定 UE 账号
~~~

对应关系：

~~~text
AutoRegUeAccount > 0  -> 返回 -14
AutoRegUeAccount <= 0 -> 返回 -13
~~~

源码中的 GetTmpTicket2Async 本身不会在返回 -14 后自动注册，而是把“允许注册”的决定交给后续流程。

### 3.4 RegAsync 流程

RegAsync 不是无参数静默注册，需要：

~~~text
mobileno
areaCode
~~~

内部请求大致为：

~~~text
Opentype = 10
Openid   = 当前用户的 Nodecode
Mobileno = 用户手机号
~~~

注册成功后，源码会写入本地 UE 授权关系。

因此，文档中的 UeRegAsync 是否等同于源码中的 RegAsync，必须向 UE 方确认。如果 UE 实际提供的是不需要手机号的 UeRegAsync，则需要另行实现上游适配器。

### 3.5 RegZsUserAsync 流程

当项目配置为不要求绑定 UE 账号时，源码调用 RegZsUserAsync，参数包括：

~~~text
ClientIp
ClientHardId
Token      = 项目配置 TokenType
Addr       = 当前用户 Nodecode
Nodename   = 当前用户节点名称
Avatarurl  = null
~~~

所以 zs 项目可能不需要客户端单独调用注册接口，而是由获取临时票据的后端流程直接完成注册并返回 DeviceId。

## 4. 推荐的 uscene 方案

### 4.1 总体架构

~~~text
客户端 JS
    |
    v
Android/iOS 原生桥
    |
    v
uscene-api-main
    |
    +-- 用户登录态校验
    +-- 项目配置校验
    +-- 本地 UE 绑定状态查询
    +-- Redis 流程锁
    +-- UE SDK 临时票据/注册调用
    |
    v
码库 Gateway
~~~

客户端不直接持有 AccessSecret，也不直接调用需要服务端凭证的接口。

### 4.2 第一阶段后端接口数量

第一阶段建议：

1. 新增获取临时票据接口；
2. 新增 UE 注册接口，仅用于 RegAsync 或 UE 方确认的 UeRegAsync；
3. 复用现有绑定回调接口；
4. 复用现有绑定状态接口，必要时增加 flow_id 查询能力。

LoginTmpTicketAsync 暂时由客户端调用，后端不新增代理接口。

## 5. 后端接口契约

### 5.1 获取临时票据

~~~http
POST /api/v1/ue/tmp-ticket
Authorization: supermap <uscene access token>
Content-Type: application/json
~~~

请求：

~~~json
{
  "request_id": "uuid",
  "project_id": "token",
  "client_hardid": "device-hardid"
}
~~~

字段：

| 字段 | 必填 | 来源 | 说明 |
|---|---:|---|---|
| request_id | 是 | 客户端 | 请求幂等键，建议 UUID |
| project_id | 是 | 客户端 | 只允许配置白名单中的项目 |
| client_hardid | 是 | 客户端原生层 | 设备稳定标识 |
| user_id | 否 | 后端登录态 | 禁止客户端传入，以登录态为准 |
| nodeid | 否 | 后端登录态 | 后端内部查询，不暴露给客户端 |
| client_ip | 否 | 反向代理 | 后端从可信代理头获取，客户端不能自行传入 |

成功响应：

~~~json
{
  "code": 200,
  "data": {
    "flow_id": "flow-uuid",
    "result": 1,
    "device_id": "temporary-ticket",
    "expires_in": 120,
    "project_id": "token"
  }
}
~~~

未绑定且禁止注册：

~~~json
{
  "code": -13,
  "err_msg": "UE账号未绑定",
  "flow_id": "flow-uuid",
  "action": "bind"
}
~~~

未绑定但允许注册：

~~~json
{
  "code": -14,
  "err_msg": "UE账号未绑定，可绑定或注册",
  "flow_id": "flow-uuid",
  "action": "bind_or_register"
}
~~~

要求：

- device_id 只在成功时返回；
- device_id 不写日志、不写数据库、不长期缓存；
- 票据有效期按 UE 返回值处理，默认按 120 秒处理；
- 同一 request_id 在有效期内应返回同一结果；
- 同一用户、项目、设备同时只能存在一个进行中的流程。

### 5.2 UE 注册

~~~http
POST /api/v1/ue/register
Authorization: supermap <uscene access token>
Content-Type: application/json
~~~

如果实际使用源码中的 RegAsync，请求字段为：

~~~json
{
  "request_id": "uuid",
  "flow_id": "flow-uuid",
  "mobileno": "+8613800000000",
  "area_code": "86"
}
~~~

后端从登录态获取：

~~~text
user_id
nodeid
nodecode
~~~

成功响应：

~~~json
{
  "code": 200,
  "data": {
    "flow_id": "flow-uuid",
    "result": 1,
    "bound": true,
    "retry_ticket": true
  }
}
~~~

客户端收到成功后，重新调用 /ue/tmp-ticket，最多重试一次。

如果 UE 方确认存在不需要手机号的 UeRegAsync，应保留相同的 uscene 接口，但替换后端内部上游参数，不应让客户端感知 UE 两套协议差异。

### 5.3 绑定状态

现有接口：

~~~http
GET /api/v1/ue/bind/status
~~~

如果临时票据流程需要精确查询某一次绑定流程，增加：

~~~http
GET /api/v1/ue/bind/status?flow_id=flow-uuid
~~~

建议返回：

~~~json
{
  "code": 200,
  "data": {
    "flow_id": "flow-uuid",
    "status": "bound",
    "retry_ticket": true
  }
}
~~~

状态值：

~~~text
pending
bound
failed
cancelled
expired
~~~

### 5.4 UE 绑定回调

继续复用：

~~~http
POST /api/v1/ue/secret/open
~~~

当前回调逻辑：

1. 校验 accesskeyid；
2. 读取 signbody、aesiv；
3. 使用 UE 登录后的服务端公钥和 AES Key 验签解密；
4. 解析 SecretDto：

~~~json
{
  "opennodecode": "用户标识",
  "uenodecode": "UE节点编码"
}
~~~

5. 写入当前 uscene 的 UE 绑定表。

需要向 UE 方确认是否能在回调中增加：

~~~text
flow_id
request_id
project_id
~~~

如果不能增加，后端通过用户、设备和项目关联流程。

## 6. 项目配置设计

TmpTicketProjectOpt.cs 中的配置字段为：

~~~text
ProjectId
TokenType
AutoRegUeAccount
RequiredBindUeAccount
~~~

在 uscene 中建议配置为服务端配置，不允许客户端修改：

~~~toml
[ue.tmp_ticket]
enabled = true
ticket_ttl_seconds = 120
flow_ttl_seconds = 300
get_ticket_url = "http://client2.ue.shumaoheng.com/svr/ThirdSvr/GetTmpTicket"
register_url = "http://client2.ue.shumaoheng.com/svr/UserSvr/Reg"
register_zs_url = "http://client2.ue.shumaoheng.com/svr/ThirdSvr/RegZsUser"

[[ue.tmp_ticket.projects]]
project_id = "token"
token_type = ""
auto_reg_ue_account = true
required_bind_ue_account = true

# zs 项目需待 UE 方提供 TokenType 后再启用
~~~

未知 project_id 必须报错，不能像示例 C# 代码一样自动降级到 common，避免客户端传入未知项目后绕过项目策略。

## 7. 数据与状态设计

### 7.1 绑定关系

当前 uscene 已有 user_ue_secret，保存用户和 UE 节点的绑定关系。

如果第一阶段只支持 token 且绑定关系与现有免密绑定完全一致，可以复用该表。

如果后续同时支持 token、zs 或多个码库项目，建议增加独立的项目维度，至少包含：

~~~text
user_id
project_id
ue_nodecode
client_hardid_hash
status
created_at
updated_at
~~~

不能只用 user_id 判断所有项目的绑定状态。

### 7.2 Redis 流程状态

建议使用 Redis 保存短期流程：

~~~text
ue:tmp-ticket:flow:{flow_id}
ue:tmp-ticket:request:{user_id}:{project_id}:{request_id}
ue:tmp-ticket:lock:{user_id}:{project_id}:{client_hardid_hash}
~~~

流程状态：

~~~text
REQUESTED
TICKET_ISSUED
NEED_BIND
BINDING
REGISTERING
RETRYING
SUCCEEDED
FAILED
CANCELLED
EXPIRED
~~~

建议 TTL：

| 内容 | TTL |
|---|---:|
| 临时票据流程 | 5 分钟 |
| 请求幂等记录 | 5 分钟 |
| 设备/用户互斥锁 | 30 秒至 5 分钟 |

### 7.3 Redis 临时票据存储要求

获取到 UE 临时票据后，允许将票据短暂保存到 Redis，但不得写入数据库或长期缓存。

临时票据 Key：

~~~text
ue:tmp-ticket:{user_id}:{project_id}:{client_hardid_hash}
~~~

其中：

- user_id 使用当前登录用户 ID；
- project_id 必须是服务端白名单项目；
- client_hardid 不得直接放入 Key，使用不可逆哈希值；
- Key 不得包含原始临时票据。

Value 建议结构：

~~~json
{
  "flow_id": "流程ID",
  "device_id": "临时票据",
  "issued_at": "2026-07-17T10:00:00Z",
  "expires_at": "2026-07-17T10:02:00Z",
  "project_id": "token"
}
~~~

写入要求：

~~~text
SET ue:tmp-ticket:{user_id}:{project_id}:{client_hardid_hash} <value> EX 120
~~~

规则：

1. 临时票据 Redis TTL 固定为 120 秒，实际有效期以 UE 返回值为准；
2. 如果 UE 返回更短的有效期，应使用更短的 TTL；
3. 同一用户、项目、设备只保留一个有效票据；
4. 新票据生成后覆盖旧票据；
5. Redis 自动过期后，后端不得继续使用旧票据；
6. 票据获取成功后立即返回客户端，并尽快调用 LoginTmpTicketAsync；
7. 票据原文不得出现在应用日志、错误消息、监控标签或链路追踪字段中；
8. Redis 访问异常时，接口应返回服务错误，不得把票据写入数据库作为降级方案；
9. Redis 应启用访问控制和传输加密，生产环境禁止使用无认证 Redis；
10. 票据存储完成后，接口响应中的 expires_in 应返回剩余有效秒数。

### 7.4 临时票据消费说明

如果客户端直接调用码库的 LoginTmpTicketAsync，uscene 后端无法知道客户端何时真正消费了票据。因此，第一阶段只能保证：

- 票据最多在 Redis 中保存 120 秒；
- 票据过期后自动删除；
- 同一设备的新票据覆盖旧票据。

如果后续需要“登录成功后立即删除票据”，必须改为以下任一方案：

1. 由 uscene 后端代理 LoginTmpTicketAsync，登录成功后删除 Redis 票据；
2. 增加一次性票据领取/消费接口，由后端在领取时删除或标记已消费。

在客户端直连 LoginTmpTicketAsync 的方案下，不应把 Redis 中的票据状态误认为码库侧的真实消费状态。

## 8. 客户端配合要求

客户端只调用 uscene 接口：

~~~text
POST /api/v1/ue/tmp-ticket
POST /api/v1/ue/register
GET  /api/v1/ue/bind/status
~~~

处理逻辑：

~~~text
result = 1
  -> 立即使用 device_id 调用 LoginTmpTicketAsync

result = -13
  -> 提示用户绑定 UE 账号

result = -14
  -> 提示绑定已有账号或注册新账号

注册/绑定成功
  -> 重新申请临时票据，最多一次
~~~

客户端不能提交：

~~~text
AccessSecret
UE AccessToken
服务端私钥
sid
UE 内部 Nodecode
任意 user_id
~~~

## 9. 安全要求

1. 临时票据接口必须要求 uscene 登录态；
2. 新增接口不能加入 DontLoginUrls；
3. /ue/secret/open、/ue/secret/close 仍需保留 UE 回调验签；
4. client_hardid 需要校验长度和字符集；
5. 日志只记录 client_hardid 的哈希值，不记录原值；
6. 不记录完整 device_id、AccessToken、AccessSecret、私钥和 AES Key；
7. 当前 SDK 和配置文件中存在静态 UE 测试凭据，正式开发前应迁移到 Kubernetes Secret，并轮换已有凭据；
8. 生产环境必须使用 HTTPS UE 地址；
9. 真实客户端 IP 只能来自可信 ALB/Ingress 转发头，不能信任客户端自传的 client_ip；
10. 对获取票据和注册接口增加用户、设备、IP 限流。

## 10. 需要 UE 方确认的协议

以下事项已影响联调，但不阻塞当前代码合入：

1. 已确认 GetTmpTicket、Reg、RegZsUser 测试地址；仍需确认生产地址是否不同；
2. 仍需确认 GetTmpTicket 的真实请求字段和字段大小写；
3. hardid 与 clientHardid 的含义和传值规则；
4. UeService.cs 中的 RegAsync 是否就是文档中的 UeRegAsync；
5. 是否支持不需要手机号的自动注册；
6. RegZsUserAsync 的完整字段和 TokenType；
7. clientid 的 Android/iOS 对应值；
8. LoginTmpTicketAsync 的完整地址、请求字段和返回结构；
9. publicKey 的来源和轮换方式；
10. 临时票据有效期及是否一次性使用；
11. 绑定页面、绑定完成回调和取消回调；
12. 回调是否支持携带 flow_id、request_id；
13. 测试环境和生产环境的 AccessKey、签名规则及 IP 白名单；
14. UE 用户、PK 用户和 uscene user_id 的关联规则。

## 11. 开发阶段

### 阶段一：协议确认

- 获取 UE 方接口文档和测试账号；
- 使用 curl 或 Postman 验证临时票据、注册、登录接口；
- 确认 hardid、clientHardid、publicKey 规则；
- 确认 -13/-14 和其他错误码。

### 阶段二：后端基础能力

- 增加临时票据项目配置；
- 增加 UE 临时票据上游适配器；
- 增加 Redis 流程和幂等锁；
- 增加用户绑定状态查询；
- 增加日志和敏感字段脱敏。

### 阶段三：后端接口

- 实现 /api/v1/ue/tmp-ticket；
- 实现 /api/v1/ue/register；
- 扩展 /api/v1/ue/bind/status 的流程查询能力；
- 复用并增强 /api/v1/ue/secret/open 的流程关联。

### 阶段四：客户端接入

- Android/iOS 原生桥接；
- 绑定/注册选择弹窗；
- 临时票据立即登录码库；
- 处理取消、超时、重复点击和 WebView 销毁。

### 阶段五：联调和上线

- 测试环境接口联调；
- Kubernetes Secret 配置；
- 部署后检查 UE 初始化日志；
- 检查临时票据和注册调用日志；
- 验证生产域名和回调白名单。

## 12. 测试用例

至少覆盖：

1. 已绑定用户获取票据成功；
2. 未绑定且禁止注册返回 -13；
3. 未绑定且允许注册返回 -14；
4. 手机号注册成功后重新获取票据；
5. zs 项目自动注册并返回票据；
6. 非法 project_id；
7. 缺少 client_hardid；
8. 当前用户未登录；
9. 同一 request_id 重复请求；
10. 同一用户、设备重复点击；
11. UE Gateway 超时；
12. UE AccessToken 失效；
13. 绑定回调验签失败；
14. 临时票据过期；
15. hardid 和 clientHardid 不一致；
16. Android/iOS 不同客户端类型；
17. Redis 不可用；
18. 回调重复发送。

## 13. 最终结论

当前 uscene-api-main 已经调用的码库接口只有：

~~~text
svr/AuthSvr/GetConfig
svr/AuthSvr/Login
svr/AuthSvr/RefreshToken
~~~

当前已经处理的 UE 回调包括：

~~~text
/api/v1/ue/notify
/api/v1/ue/secret/open
/api/v1/ue/secret/close
~~~

当前没有代理实现、仍由客户端或 UE SDK 负责的动作包括：

~~~text
LoginTmpTicketAsync
~~~

第一期推荐开发：

~~~text
新增：POST /api/v1/ue/tmp-ticket
新增：POST /api/v1/ue/register
复用：GET  /api/v1/ue/bind/status
复用：POST /api/v1/ue/secret/open
客户端直接调用：LoginTmpTicketAsync
~~~

当前后端已经代理 GetTmpTicket、Reg 和 RegZsUser；其中 RegAsync 与文档中的 UeRegAsync 是否为同一协议，仍需 UE 联调确认。
