# UE 临时票据客户端接入文档

## 1. 接入目标

客户端通过 uscene 后端获取 UE/码库临时票据，然后使用临时票据完成码库登录。

客户端不直接调用需要 UE 服务端凭证的接口，不接触：

~~~text
AccessSecret
UE AccessToken
服务端私钥
AES Key
UE 上游接口地址
Redis Key
~~~

完整流程：

~~~text
客户端登录 uscene
    |
    v
请求 uscene 获取临时票据
    |
    +-- 成功：立即调用 LoginTmpTicketAsync
    |
    +-- -13：提示用户绑定 UE 账号
    |
    +-- -14：提示用户绑定或注册
                    |
                    +-- 绑定成功：查询绑定状态后重新获取票据
                    +-- 注册成功：重新获取票据
~~~

## 2. 接口基础信息

测试环境基础地址由客户端环境配置决定，例如：

~~~text
https://api-test.uscene.cc
~~~

所有接口路径：

~~~text
POST /api/v1/ue/tmp-ticket
POST /api/v1/ue/register
GET  /api/v1/ue/tmp-ticket/bind/status
~~~

## 3. 登录认证

当前 uscene API 使用以下 Authorization 格式：

~~~http
Authorization: supermap <uscene_access_token>
~~~

不是：

~~~http
Authorization: Bearer <uscene_access_token>
~~~

请求头建议统一携带：

~~~http
Authorization: supermap <uscene_access_token>
Content-Type: application/json
Language: zh-cn
App-Version: <客户端版本>
Platform: android 或 ios
~~~

接口必须使用当前登录用户的 Access Token。客户端不需要，也不允许传递 user_id。

## 4. 获取临时票据

### 4.1 请求

~~~http
POST /api/v1/ue/tmp-ticket
Authorization: supermap <uscene_access_token>
Content-Type: application/json
~~~

请求体：

~~~json
{
  "request_id": "1e6e6cc7-61d8-49ab-8a25-6ca3a4e95e7d",
  "project_id": "token",
  "client_hardid": "客户端生成并持久化的设备标识"
}
~~~

字段说明：

| 字段 | 必填 | 说明 |
|---|---:|---|
| request_id | 是 | 每次流程唯一 UUID，用于幂等和问题追踪 |
| project_id | 是 | 当前项目 ID，第一阶段建议使用 token |
| client_hardid | 是 | 设备稳定标识，同一次登录流程必须保持一致 |
| user_id | 否 | 禁止传递，后端从登录态获取 |
| client_ip | 否 | 禁止传递，后端从 ALB/Ingress 获取 |

### 4.2 成功响应

~~~json
{
  "code": 200,
  "data": {
    "flow_id": "8e9c3f79-1d09-4b52-8e5d-8a44b63fd1a5",
    "result": 1,
    "device_id": "UE临时票据",
    "expires_in": 120,
    "project_id": "token"
  }
}
~~~

拿到 device_id 后必须立即调用码库 LoginTmpTicketAsync。不要等待用户下一次操作，不要把票据保存到本地长期存储。

### 4.3 未绑定响应：-13

~~~json
{
  "code": 200,
  "data": {
    "flow_id": "流程ID",
    "result": -13,
    "action": "bind",
    "project_id": "token",
    "message": "UE账号未绑定"
  }
}
~~~

处理方式：

1. 打开 UE 绑定已有账号页面；
2. 绑定完成后调用绑定状态接口；
3. 状态为 bound 后重新调用临时票据接口；
4. 重新请求时生成新的 request_id。

### 4.4 未绑定响应：-14

~~~json
{
  "code": 200,
  "data": {
    "flow_id": "流程ID",
    "result": -14,
    "action": "bind_or_register",
    "project_id": "token",
    "message": "UE账号未绑定，可绑定或注册"
  }
}
~~~

客户端可以显示：

~~~text
绑定已有 UE 账号
注册新的 UE 账号
取消
~~~

## 5. 查询绑定状态

### 5.1 请求

~~~http
GET /api/v1/ue/tmp-ticket/bind/status?flow_id=<flow_id>
Authorization: supermap <uscene_access_token>
~~~

### 5.2 绑定中

~~~json
{
  "code": 200,
  "data": {
    "flow_id": "流程ID",
    "status": "binding",
    "retry_ticket": false
  }
}
~~~

客户端可以每 2 秒查询一次，最多查询 60 次。超过 120 秒后应停止轮询并重新发起流程。

### 5.3 绑定完成

~~~json
{
  "code": 200,
  "data": {
    "flow_id": "流程ID",
    "status": "bound",
    "retry_ticket": true
  }
}
~~~

客户端收到 bound 后，重新调用：

~~~http
POST /api/v1/ue/tmp-ticket
~~~

### 5.4 其他状态

可能的状态：

~~~text
pending
binding
bound
failed
cancelled
expired
~~~

expired、failed、cancelled 都应结束当前流程，客户端重新生成 request_id 后重试。

## 6. 注册 UE 账号

当前后端实现依据 UeService.cs 中的 RegAsync，注册需要手机号。

### 6.1 请求

~~~http
POST /api/v1/ue/register
Authorization: supermap <uscene_access_token>
Content-Type: application/json
~~~

请求体：

~~~json
{
  "request_id": "新的UUID",
  "flow_id": "之前获取票据时返回的流程ID",
  "project_id": "token",
  "mobileno": "+8613800000000",
  "area_code": "86"
}
~~~

字段说明：

| 字段 | 必填 | 说明 |
|---|---:|---|
| request_id | 是 | 注册操作唯一 UUID |
| flow_id | 建议 | 与 -14 流程关联 |
| project_id | 无 flow_id 时必填 | 必须使用后端已配置项目 |
| mobileno | 是 | 手机号，建议带国家码 |
| area_code | 否 | 国家区号，例如 86 |

### 6.2 注册成功

~~~json
{
  "code": 200,
  "data": {
    "flow_id": "流程ID",
    "result": 1,
    "bound": true,
    "retry_ticket": true,
    "project_id": "token"
  }
}
~~~

注册成功后重新获取临时票据：

~~~http
POST /api/v1/ue/tmp-ticket
~~~

注册成功后的自动重试最多执行一次。

### 6.3 注册失败

如果码库返回业务失败，接口仍然可能返回：

~~~json
{
  "code": 200,
  "data": {
    "flow_id": "流程ID",
    "result": -1,
    "message": "UE返回的错误信息",
    "project_id": "token"
  }
}
~~~

客户端不应无限重试，应向用户展示可操作提示。

## 7. LoginTmpTicketAsync

获取到临时票据后，由客户端按照 UE 方 SDK 或接口文档调用 LoginTmpTicketAsync。

请求中至少要保持以下值一致：

~~~text
projectId = 获取票据时使用的 project_id
hardid    = 获取票据时使用的同一设备标识
deviceId  = uscene 返回的 device_id
~~~

示例结构：

~~~json
{
  "version": "UE接口版本",
  "sid": 81126,
  "clientid": 1,
  "lang": "zh-cn",
  "hardid": "同一个 client_hardid",
  "publicKey": "UE方提供的公钥",
  "deviceId": "uscene返回的临时票据",
  "projectId": "token"
}
~~~

注意：

- LoginTmpTicketAsync 的实际 URL、clientid、publicKey 来源以 UE 方协议为准；
- device_id 有效期默认按 120 秒处理；
- 临时票据可能只能使用一次；
- 登录失败后不要重复使用旧 device_id，应重新获取票据。

## 8. Android/iOS 推荐调用流程

### 8.1 JS 层

JS 只需要调用原生桥：

~~~javascript
const result = await PKGateway.getCasTmpTicket({
  requestId: crypto.randomUUID(),
  projectId: "token",
  clientHardid: deviceHardId
});
~~~

JS 不保存 AccessSecret、UE AccessToken 或私钥。

### 8.2 原生层

原生层负责：

1. 获取并持久化 clientHardid；
2. 调用 uscene 临时票据接口；
3. 处理 -13、-14；
4. 显示绑定/注册选择；
5. 调用绑定状态接口；
6. 调用 LoginTmpTicketAsync；
7. 将最终登录结果回传 JS。

Android 和 iOS 必须使用相同语义的 clientHardid 字段。

### 8.3 推荐状态机

~~~text
Idle
  -> RequestingTicket
  -> NeedBind
  -> Binding
  -> Registering
  -> RetryingTicket
  -> TicketIssued
  -> LoginTmpTicket
  -> Succeeded / Failed / Cancelled / Expired
~~~

限制：

- 同一时间只允许一个进行中的流程；
- 用户重复点击时复用当前 flow_id；
- 绑定或注册成功后只重试一次；
- 总流程建议 5 分钟超时；
- 单次 HTTP 请求建议 10 至 15 秒超时。

## 9. HTTP 错误处理

### 9.1 参数或登录错误

~~~json
{
  "code": 400,
  "err_msg": "参数错误"
}
~~~

或：

~~~json
{
  "code": 401,
  "err_msg": "Login authorization error"
}
~~~

客户端检查登录态和参数后再重试。

### 9.2 请求过快

~~~json
{
  "code": 429,
  "err_msg": "Request too fast"
}
~~~

客户端不要立即连续重试，应等待后再发起。

### 9.3 服务不可用

~~~json
{
  "code": 503,
  "err_msg": "服务不可用"
}
~~~

可能原因：

- UE Token 失效；
- UE 临时票据 URL 未配置；
- Redis 不可用；
- UE Gateway 超时；
- UE 返回格式不符合当前适配器。

## 10. curl 示例

以下示例只用于验证 uscene 后端接口，不能直接替代 LoginTmpTicketAsync。

获取临时票据：

~~~bash
curl -sS -X POST \
  'https://api-test.uscene.cc/api/v1/ue/tmp-ticket' \
  -H 'Authorization: supermap <USCENE_ACCESS_TOKEN>' \
  -H 'Content-Type: application/json' \
  -d '{
    "request_id": "7a8b3e35-5e9f-4e52-b7c6-924b8c17f5d1",
    "project_id": "token",
    "client_hardid": "test-device-hardid"
  }'
~~~

查询绑定状态：

~~~bash
curl -sS \
  'https://api-test.uscene.cc/api/v1/ue/tmp-ticket/bind/status?flow_id=<FLOW_ID>' \
  -H 'Authorization: supermap <USCENE_ACCESS_TOKEN>'
~~~

注册 UE 账号：

~~~bash
curl -sS -X POST \
  'https://api-test.uscene.cc/api/v1/ue/register' \
  -H 'Authorization: supermap <USCENE_ACCESS_TOKEN>' \
  -H 'Content-Type: application/json' \
  -d '{
    "request_id": "3c65d6e1-6e3a-4d9c-9fbe-9f8edb4b0e22",
    "flow_id": "<FLOW_ID>",
    "project_id": "token",
    "mobileno": "+8613800000000",
    "area_code": "86"
  }'
~~~

## 11. 客户端接入验收标准

1. 已登录 uscene 用户可以成功请求临时票据；
2. request_id 每次流程唯一；
3. client_hardid 在获取票据和码库登录时保持一致；
4. result=1 后立即调用 LoginTmpTicketAsync；
5. 正确处理 -13 和 -14；
6. 绑定成功后可以查询 bound；
7. 注册成功后可以重新获取票据；
8. 不重复使用过期票据；
9. 不在日志中打印 device_id；
10. 不向客户端暴露 UE AccessSecret、私钥或 Redis 信息；
11. 网络超时、取消和 WebView 销毁后能够结束当前流程；
12. 不出现无限注册或无限重试。

## 12. 当前接入前置条件

后端部署前必须配置 UE 临时票据上游地址：

~~~toml
[ue.tmp_ticket]
enabled = true
ticket_ttl_seconds = 120
flow_ttl_seconds = 300
get_ticket_url = "<UE方提供的GetTmpTicketAsync完整地址>"
register_url = "<UE方提供的RegAsync或UeRegAsync完整地址>"
register_zs_url = "<UE方提供的RegZsUserAsync完整地址>"

[[ue.tmp_ticket.projects]]
project_id = "token"
token_type = ""
auto_reg_ue_account = true
required_bind_ue_account = true
~~~

如果没有配置真实 UE 上游地址，uscene 接口会返回服务不可用，不会猜测或拼接未知接口路径。
