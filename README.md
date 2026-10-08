# NetMap v0.3

NetMap 是一个基于 TCP 的轻量级网络代理与端口映射工具。

支持两种主要使用方式：

1. **代理模式**：应用通过本地 HTTP / HTTPS / SOCKS5 代理访问目标服务。
2. **TCP Mapping 模式**：将客户端电脑的本地 TCP 端口映射到远程目标服务。

NetMap 采用 **Client + Relay** 架构：

```text
代理模式：

应用
 │
 ▼
Client
127.0.0.1:18080
 │
 ├── 本机直连
 │
 └── Relay
      │
      ▼
   Target


TCP Mapping：

应用
 │
 ▼
127.0.0.1:15236
 │
 ▼
Client
 │
 ▼
Relay
 │
 ▼
10.82.51.115:5236
```

---

# 一、v0.3 功能

## 1. HTTP Proxy

支持普通 HTTP 请求代理。

例如：

```text
应用 → 127.0.0.1:18080 → Relay → 10.82.51.231:8080
```

---

## 2. HTTPS Proxy

支持 HTTP CONNECT 方式建立 HTTPS TCP 隧道。

例如：

```text
应用
 ↓
127.0.0.1:18080
 ↓
HTTP CONNECT
 ↓
Relay
 ↓
目标 HTTPS 服务
```

---

## 3. SOCKS5 Proxy

支持：

* SOCKS5
* NO AUTH
* TCP CONNECT
* IPv4
* IPv6
* Domain

例如：

```text
应用
 ↓
SOCKS5
 ↓
127.0.0.1:18080
 ↓
Client
 ↓
Relay
 ↓
Target
```

---

## 4. Client 代理规则

Client 可以根据目标地址决定：

* `proxy=true`：通过 Relay
* `proxy=false`：本机直连

支持：

```text
IP
IP:Port
Domain
Domain:Port
```

例如：

```json
{
  "target": "10.82.51.231",
  "proxy": true
}
```

表示：

```text
10.82.51.231
```

所有端口都通过 Relay。

---

```json
{
  "target": "10.82.51.231:8080",
  "proxy": true
}
```

表示只有：

```text
10.82.51.231:8080
```

通过 Relay。

没有匹配到规则时：

```text
本机直连
```

---

## 5. TCP Mapping

v0.3 新增 TCP Mapping。

可以把客户端电脑上的一个本地 TCP 端口映射到远程目标服务。

例如：

```text
客户端电脑

127.0.0.1:15236
       │
       ▼
     Client
       │
       ▼
Relay 192.168.0.44:19000
       │
       ▼
10.82.51.115:5236
```

应用只需要访问：

```text
127.0.0.1:15236
```

不需要直接访问：

```text
10.82.51.115:5236
```

---

## 6. Relay 目标白名单

Relay 使用 `allowedTargets` 限制可以访问的目标。

支持：

```text
IP
IP:Port
Domain
Domain:Port
```

例如：

```json
"allowedTargets": [
  "10.82.51.231",
  "10.82.51.115",
  "10.82.51.115:5236"
]
```

其中：

```text
10.82.51.231
```

表示该 IP 的所有端口允许访问。

而：

```text
10.82.51.115:5236
```

只允许访问该 IP 的 `5236` 端口。

如果同时存在：

```text
10.82.51.115
10.82.51.115:5236
```

则：

```text
10.82.51.115
```

优先，表示该 IP 的所有端口都允许。

---

# 二、Client 和 Relay

NetMap 分为两个角色。

## Client

运行在需要使用代理或 TCP Mapping 的客户端电脑。

负责：

* 接收本地代理请求
* 处理 HTTP / HTTPS / SOCKS5
* 根据规则决定直连或 Relay
* 建立 TCP Mapping
* 与 Relay 建立连接

---

## Relay

运行在可以访问目标网络的跳板机电脑。

负责：

* 接收 Client 连接
* 解析 NetMap 内部协议
* 校验目标白名单
* 连接目标 TCP 服务
* 转发 TCP 数据

Relay 不解析 HTTP、HTTPS、SOCKS5，只处理 NetMap 内部 TCP 协议。

---

# 三、配置文件

## 1. Client 配置

文件：

```text
configs/client.json
```

示例：

```json
{
  "role": "client",
  "log": {
    "level": "DEBUG"
  },
  "listen": {
    "host": "127.0.0.1",
    "port": 18080
  },
  "relay": {
    "host": "192.168.0.44",
    "port": 19000,
    "connectTimeoutSeconds": 10
  },
  "rules": [
    {
      "target": "10.82.51.231",
      "proxy": true
    },
    {
      "target": "10.82.51.33",
      "proxy": true
    },
    {
      "target": "10.82.51.115",
      "proxy": true
    }
  ],
  "mappings": [
    {
      "name": "DM数据库",
      "enabled": true,
      "listen": {
        "host": "127.0.0.1",
        "port": 15236
      },
      "target": {
        "host": "10.82.51.115",
        "port": 5236
      }
    }
  ]
}
```

### Client 配置说明

#### role

```json
"role": "client"
```

表示当前运行 NetMap Client。

---

#### log.level

```json
"log": {
  "level": "DEBUG"
}
```

支持：

```text
INFO
DEBUG
```

如果不配置，默认：

```text
INFO
```

开发测试时可以使用：

```text
DEBUG
```

正式运行建议：

```text
INFO
```

---

#### listen

```json
"listen": {
  "host": "127.0.0.1",
  "port": 18080
}
```

Client 本地代理监听地址。

默认使用：

```text
127.0.0.1:18080
```

应用配置代理时使用：

```text
HTTP Proxy: 127.0.0.1:18080
SOCKS5:     127.0.0.1:18080
```

---

#### relay

```json
"relay": {
  "host": "192.168.0.44",
  "port": 19000,
  "connectTimeoutSeconds": 10
}
```

配置 Relay 地址。

例如：

```text
Relay IP:   192.168.0.44
Relay Port: 19000
```

Client 必须能够访问 Relay 的 `19000` TCP 端口。

---

# 四、Client 代理规则

规则配置：

```json
"rules": [
  {
    "target": "10.82.51.231",
    "proxy": true
  }
]
```

## IP

```json
{
  "target": "10.82.51.231",
  "proxy": true
}
```

表示：

```text
10.82.51.231:* → Relay
```

---

## IP + Port

```json
{
  "target": "10.82.51.231:8080",
  "proxy": true
}
```

表示：

```text
10.82.51.231:8080 → Relay
```

其他端口不匹配该规则。

---

## Domain

```json
{
  "target": "example.com",
  "proxy": true
}
```

表示：

```text
example.com:* → Relay
```

---

## Domain + Port

```json
{
  "target": "example.com:443",
  "proxy": true
}
```

表示：

```text
example.com:443 → Relay
```

---

## proxy=false

例如：

```json
{
  "target": "10.82.51.231",
  "proxy": false
}
```

表示：

```text
10.82.51.231:* → 本机直连
```

---

## 没有匹配规则

如果没有匹配到任何规则：

```text
本机直连
```

因此 NetMap Client 默认不是全局代理。

---

# 五、TCP Mapping 配置

Mapping 配置：

```json
"mappings": [
  {
    "name": "DM数据库",
    "enabled": true,
    "listen": {
      "host": "127.0.0.1",
      "port": 15236
    },
    "target": {
      "host": "10.82.51.115",
      "port": 5236
    }
  }
]
```

含义：

```text
本地：
127.0.0.1:15236

映射到：

远程：
10.82.51.115:5236
```

应用连接：

```text
127.0.0.1:15236
```

即可。

---

## enabled

启用：

```json
"enabled": true
```

禁用：

```json
"enabled": false
```

禁用后不会监听对应端口。

---

## name

```json
"name": "DM数据库"
```

Mapping 名称。

名称不能重复。

---

## listen

```json
"listen": {
  "host": "127.0.0.1",
  "port": 15236
}
```

表示 Client 本地监听地址。

---

## target

```json
"target": {
  "host": "10.82.51.115",
  "port": 5236
}
```

表示最终需要访问的目标服务。

---

## Mapping 注意事项

Mapping 的目标仍然需要通过 Relay 的：

```text
allowedTargets
```

检查。

例如 Mapping：

```text
10.82.51.115:5236
```

那么 Relay 至少需要允许：

```json
"allowedTargets": [
  "10.82.51.115:5236"
]
```

或者直接允许：

```json
"allowedTargets": [
  "10.82.51.115"
]
```

否则 Mapping 会连接失败。

---

# 六、Relay 配置

文件：

```text
configs/relay.json
```

示例：

```json
{
  "role": "relay",
  "log": {
    "level": "DEBUG"
  },
  "listen": {
    "host": "0.0.0.0",
    "port": 19000
  },
  "target": {
    "connectTimeoutSeconds": 10,
    "allowedTargets": [
      "10.82.51.231",
      "10.82.51.33",
      "10.82.51.66",
      "10.82.51.115",
      "10.82.79.139",
      "10.82.79.242",
      "10.82.79.159",
      "10.82.79.67",
      "10.82.79.108",
      "10.82.79.48",
      "10.82.79.182",
      "10.82.79.37",
      "10.82.79.181",
      "10.83.60.84",
      "10.83.60.85",
      "10.223.127.130"
    ]
  }
}
```

### Relay 配置说明

#### role

```json
"role": "relay"
```

表示当前运行 NetMap Relay。

---

#### listen

```json
"listen": {
  "host": "0.0.0.0",
  "port": 19000
}
```

Relay 监听：

```text
0.0.0.0:19000
```

因此客户端电脑需要能够访问：

```text
Relay_IP:19000
```

---

#### connectTimeoutSeconds

```json
"connectTimeoutSeconds": 10
```

Relay 连接目标服务的 TCP 超时时间，单位：

```text
秒
```

---

#### allowedTargets

```json
"allowedTargets": [
  "10.82.51.231",
  "10.82.51.115"
]
```

控制 Relay 最终允许访问哪些目标。

如果目标不在白名单中：

```text
Client → Relay
          ↓
      白名单检查
          ↓
        拒绝
```

---

# 七、如何启动

NetMap 根据配置文件中的：

```json
"role"
```

决定运行模式。

## 启动 Client

使用：

```text
configs/client.json
```

启动 NetMap。

Client 启动成功后，可以看到：

```text
client server listening addr=127.0.0.1:18080
```

如果配置了 Mapping，还会看到：

```text
mapping started name=DM数据库 listen=127.0.0.1:15236 target=10.82.51.115:5236
```

---

## 启动 Relay

使用：

```text
configs/relay.json
```

启动 NetMap。

Relay 启动成功后：

```text
relay: server listening on 0.0.0.0:19000
```

---

# 八、如何使用代理

## HTTP

将应用的 HTTP Proxy 设置为：

```text
127.0.0.1:18080
```

例如使用 curl：

```bash
curl -x http://127.0.0.1:18080 http://10.82.51.231:8080/
```

如果：

```text
10.82.51.231
```

命中了：

```json
{
  "target": "10.82.51.231",
  "proxy": true
}
```

请求路径为：

```text
curl
 ↓
127.0.0.1:18080
 ↓
Client
 ↓
Relay
 ↓
10.82.51.231:8080
```

---

## HTTPS

HTTPS 使用 HTTP CONNECT。

例如：

```bash
curl -x http://127.0.0.1:18080 https://10.82.51.231:443/
```

流程：

```text
curl
 ↓
HTTP CONNECT
 ↓
Client
 ↓
Relay
 ↓
Target
```

---

## SOCKS5

将应用 SOCKS5 代理设置为：

```text
127.0.0.1:18080
```

例如：

```bash
curl --socks5 127.0.0.1:18080 http://10.82.51.231:8080/
```

---

# 九、如何使用 TCP Mapping

假设配置：

```json
{
  "name": "DM数据库",
  "enabled": true,
  "listen": {
    "host": "127.0.0.1",
    "port": 15236
  },
  "target": {
    "host": "10.82.51.115",
    "port": 5236
  }
}
```

启动 Client 后，本机监听：

```text
127.0.0.1:15236
```

应用不需要配置 HTTP Proxy，也不需要配置 SOCKS5。

直接连接：

```text
127.0.0.1:15236
```

NetMap 会自动完成：

```text
应用
 ↓
127.0.0.1:15236
 ↓
Client
 ↓
Relay 192.168.0.44:19000
 ↓
10.82.51.115:5236
```

---

# 十、网络要求

Client 和 Relay 之间需要能够建立 TCP 连接。

例如：

```text
Client
  │
  │ TCP 19000
  ▼
Relay
```

需要保证：

```text
Client → Relay:19000
```

网络可达。

Relay 到目标服务也必须可达：

```text
Relay → Target:Port
```

例如：

```text
Relay → 10.82.51.115:5236
```

如果 Client 可以连接 Relay，但是 Relay 无法连接目标服务，代理或 Mapping 仍然无法正常工作。

---

# 十一、日志

日志等级：

```json
"log": {
  "level": "INFO"
}
```

或者：

```json
"log": {
  "level": "DEBUG"
}
```

### INFO

适合正常运行。

主要记录：

* 服务启动
* 服务停止
* 连接成功
* 错误

### DEBUG

适合排查问题。

会额外记录：

* 规则匹配
* Relay 连接过程
* Mapping 连接过程
* SOCKS5 协议处理
* 白名单匹配
* TCP 转发信息

---

# 十二、常见问题

## 1. Client 启动失败

首先检查：

```text
127.0.0.1:18080
```

是否已经被其他程序占用。

Mapping 也需要检查对应本地端口是否被占用，例如：

```text
127.0.0.1:15236
```

---

## 2. Client 无法连接 Relay

检查：

```text
Client relay.host
Client relay.port
```

例如：

```json
"relay": {
  "host": "192.168.0.44",
  "port": 19000
}
```

并确认：

```text
Client → 192.168.0.44:19000
```

网络可达。

---

## 3. Relay 提示目标不允许

如果日志出现：

```text
target is not allowed
```

说明目标没有配置到：

```text
allowedTargets
```

例如需要访问：

```text
10.82.51.115:5236
```

可以配置：

```json
"allowedTargets": [
  "10.82.51.115:5236"
]
```

或者：

```json
"allowedTargets": [
  "10.82.51.115"
]
```

---

## 4. Relay 可以连接，但是目标服务连接失败

检查 Relay 到目标服务之间的网络。

例如：

```text
Relay
 ↓
10.82.51.115:5236
```

需要确认 Relay 机器本身能够访问目标地址和端口。

---

## 5. Mapping 连接失败

检查三个地方：

```text
1. Client Mapping 是否 enabled=true
2. 本地 Mapping 端口是否被占用
3. Relay allowedTargets 是否允许 Target
```

例如：

```text
127.0.0.1:15236
        ↓
10.82.51.115:5236
```

Relay 必须允许：

```text
10.82.51.115
```

或者：

```text
10.82.51.115:5236
```

---

# 十三、配置校验

NetMap 启动时会自动检查配置。

Client 会检查：

* `role`
* `listen`
* `relay`
* `rules`
* `mappings`

Relay 会检查：

* `role`
* `listen`
* `target`
* `allowedTargets`

Mapping 会检查：

* 名称不能为空
* 名称不能重复
* 本地监听地址有效
* 本地监听端口有效
* Mapping 监听地址不能和主 Proxy 监听地址重复
* 目标地址有效
* 目标端口有效

配置错误时 NetMap 会直接返回错误，不会继续启动。

---

# 十四、v0.3 目录

推荐发布目录：

```text
NetMap-v0.3/
├─ netmap.exe
├─ configs/
│  ├─ client.json
│  └─ relay.json
├─ assets/
│  └─ netmap.ico
├─ logs/
└─ README.md
```

其中：

```text
netmap.exe
```

为 NetMap 主程序。

```text
configs/client.json
```

为 Client 配置。

```text
configs/relay.json
```

为 Relay 配置。

```text
logs/
```

用于保存运行日志。

---

# 十五、v0.3 架构

```text
                    ┌─────────────────────┐
                    │      Target         │
                    │ 10.82.51.115:5236  │
                    └──────────▲──────────┘
                               │
                               │ TCP
                               │
                    ┌──────────┴──────────┐
                    │       Relay         │
                    │ 192.168.0.44:19000  │
                    │                     │
                    │  allowedTargets     │
                    └──────────▲──────────┘
                               │
                               │ NetMap TCP
                               │
                    ┌──────────┴──────────┐
                    │       Client        │
                    │ 127.0.0.1:18080     │
                    │                     │
                    │ rules               │
                    │ mappings            │
                    └──────────▲──────────┘
                               │
                ┌──────────────┴──────────────┐
                │                             │
          Proxy 请求                    TCP Mapping
                │                             │
        127.0.0.1:18080                127.0.0.1:15236
                │                             │
                └─────────────┬───────────────┘
                              │
                              ▼
                           应用
```

---

# 十六、版本

当前版本：

```text
NetMap v0.3
```

v0.3 相比之前版本主要增加：

```text
TCP Mapping
```

即：

```text
本地 TCP 端口
      ↓
Client
      ↓
Relay
      ↓
远程 TCP 服务
```

同时保留：

```text
HTTP Proxy
HTTPS CONNECT Proxy
SOCKS5 TCP CONNECT
Client Rules
Relay allowedTargets
Windows 客户端运行支持
日志
配置文件驱动
```
