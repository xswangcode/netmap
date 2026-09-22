# NetMap

NetMap 是一个基于 TCP 的内网 IP 映射工具。

用于解决：

> 客户端电脑无法直接访问目标服务，但跳板机电脑可以访问目标服务。

NetMap 在客户端电脑和跳板机电脑之间建立 TCP 转发通道，使客户端应用可以通过本地代理访问目标服务。

---

## 1. v0.1 功能

v0.1 支持：

* HTTP Proxy
* HTTPS Proxy（HTTP CONNECT）
* SOCKS5 TCP CONNECT
* Client 本地代理
* Client 规则匹配
* Client 直连 / Relay 转发
* Relay 目标白名单
* TCP 双向转发
* IP / IP:Port
* Domain / Domain:Port

### 不支持

v0.1 暂不支持：

* UDP
* SOCKS5 UDP
* SOCKS5 BIND
* VPN
* TUN / TAP
* NAT
* DNS Proxy
* TCP/UDP Tunnel
* 加密
* 压缩
* 连接池
* 多路复用
* P2P 穿透
* 自动发现
* Web 管理后台
* 用户管理
* 集群

---

# 2. 工作方式

NetMap 由两个角色组成：

```text
客户端电脑
    │
    │ NetMap Client
    │
    ▼
跳板机电脑
    │
    │ NetMap Relay
    │
    ▼
目标服务
```

客户端电脑运行 Client。

跳板机电脑运行 Relay。

例如：

```text
客户端电脑
127.0.0.1:18080
       │
       │
       ▼
跳板机电脑
192.168.0.29:19000
       │
       │
       ▼
10.82.51.231:8080
```

---

# 3. Client 和 Relay 的职责

## Client

Client 运行在客户端电脑。

负责：

1. 接收本地应用的代理请求
2. 识别 HTTP / HTTPS / SOCKS5
3. 获取目标 Host 和 Port
4. 根据 rules 判断是否使用 Relay
5. 命中规则 → 通过 Relay
6. 未命中规则 → 客户端电脑直接访问

---

## Relay

Relay 运行在跳板机电脑。

负责：

1. 接收 Client 连接
2. 接收 Client 的目标地址
3. 检查目标白名单
4. 连接目标服务
5. 进行 TCP 双向转发

Relay 不负责解析 HTTP、HTTPS、SOCKS5。

---

# 4. 规则逻辑

Client 使用 `rules` 判断目标是否需要通过 Relay。

例如：

```json
"rules": [
  {
    "target": "10.82.51.231",
    "proxy": true
  }
]
```

表示：

```text
10.82.51.231
    ↓
所有端口通过 Relay
```

例如访问：

```text
10.82.51.231:80
10.82.51.231:81
10.82.51.231:8080
10.82.51.231:8088
```

都会通过 Relay。

---

## 4.1 支持的规则格式

### IP

```text
10.82.51.231
```

表示该 IP 所有端口。

---

### IP + Port

```text
10.82.51.231:8080
```

只匹配：

```text
10.82.51.231:8080
```

---

### Domain

```text
bastion.voyah.cn
```

表示该域名所有端口。

---

### Domain + Port

```text
bastion.voyah.cn:443
```

只匹配：

```text
bastion.voyah.cn:443
```

---

# 5. Client 配置

配置文件：

```text
configs/client.json
```

示例：

```json
{
  "role": "client",
  "listen": {
    "host": "127.0.0.1",
    "port": 18080
  },
  "relay": {
    "host": "192.168.0.29",
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
      "target": "10.82.51.66",
      "proxy": true
    },
    {
      "target": "10.82.51.115",
      "proxy": true
    },
    {
      "target": "10.223.127.130",
      "proxy": true
    },
    {
      "target": "bastion.voyah.cn",
      "proxy": true
    }
  ]
}
```

---

# 6. Relay 配置

配置文件：

```text
configs/relay.json
```

示例：

```json
{
  "role": "relay",
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
      "10.223.127.130",
      "bastion.voyah.cn"
    ]
  }
}
```

---

# 7. Relay 白名单

Relay 的 `allowedTargets` 用于限制 Relay 可以访问哪些目标。

支持：

```text
10.82.51.231
10.82.51.231:8080
bastion.voyah.cn
bastion.voyah.cn:443
```

例如：

```json
"allowedTargets": [
  "10.82.51.231"
]
```

表示：

```text
10.82.51.231:80
10.82.51.231:81
10.82.51.231:8080
10.82.51.231:8088
```

全部允许。

如果配置：

```json
"allowedTargets": [
  "10.82.51.231:8080"
]
```

则只允许：

```text
10.82.51.231:8080
```

---

# 8. Client 和 Relay 的区别

两者作用不同。

## Client rules

决定：

> 这个请求要不要通过 Relay？

例如：

```text
10.82.51.231 → Relay
www.baidu.com → 直连
```

---

## Relay allowedTargets

决定：

> Relay 是否允许访问这个目标？

例如：

```text
Client 请求：

10.82.51.231:8080
        ↓
Client rules
        ↓
通过 Relay
        ↓
Relay allowedTargets
        ↓
允许
        ↓
10.82.51.231:8080
```

所以两边都需要配置。

---

# 9. 启动

## 9.1 启动 Relay

在跳板机电脑执行：

```powershell
.\netmap.exe -config .\configs\relay.json
```

正常情况下：

```text
relay server listening on 0.0.0.0:19000
```

---

## 9.2 启动 Client

在客户端电脑执行：

```powershell
.\netmap.exe -config .\configs\client.json
```

正常情况下：

```text
client server listening on 127.0.0.1:18080
```

---

# 10. 使用 HTTP Proxy

客户端应用配置 HTTP Proxy：

```text
Host: 127.0.0.1
Port: 18080
```

例如：

```text
http://10.82.51.231:8080/
```

请求流程：

```text
应用
 ↓
127.0.0.1:18080
 ↓
Client
 ↓
rules
 ↓
Relay
 ↓
10.82.51.231:8080
```

---

# 11. 使用 HTTPS Proxy

HTTPS 使用 HTTP CONNECT。

客户端 Proxy：

```text
Host: 127.0.0.1
Port: 18080
```

例如：

```text
https://bastion.voyah.cn
```

流程：

```text
应用
 ↓
HTTP CONNECT
 ↓
Client
 ↓
Relay
 ↓
目标 HTTPS 服务
```

CONNECT 建立成功以后，后续数据直接进行 TCP 双向转发。

NetMap 不解析 HTTPS 加密内容。

---

# 12. 使用 SOCKS5

客户端应用配置：

```text
SOCKS5 Host: 127.0.0.1
SOCKS5 Port: 18080
```

例如：

```text
10.82.51.231:8080
```

流程：

```text
应用
 ↓
SOCKS5
 ↓
Client
 ↓
rules
 ↓
Relay
 ↓
目标服务
```

---

# 13. curl 测试

## HTTP

```powershell
curl.exe -x http://127.0.0.1:18080 http://10.82.51.231:8080/
```

---

## HTTPS

```powershell
curl.exe -k -x http://127.0.0.1:18080 https://bastion.voyah.cn/
```

---

## SOCKS5

```powershell
curl.exe --socks5-hostname 127.0.0.1:18080 http://10.82.51.231:8080/
```

---

# 14. 网络要求

客户端电脑必须能够访问跳板机电脑：

```text
客户端电脑
      ↓
192.168.0.29:19000
```

可以测试：

```powershell
Test-NetConnection 192.168.0.29 -Port 19000
```

如果：

```text
TcpTestSucceeded : True
```

说明客户端可以连接 Relay。

另外：

```text
跳板机电脑
      ↓
目标服务
```

必须能够访问目标服务。

例如：

```text
跳板机电脑
      ↓
10.82.51.231:8080
```

---

# 15. 配置文件

项目默认配置：

```text
netmap/
├── netmap.exe
├── configs/
│   ├── client.json
│   └── relay.json
└── ...
```

启动时通过 `-config` 指定配置：

```powershell
.\netmap.exe -config .\configs\client.json
```

或者：

```powershell
.\netmap.exe -config .\configs\relay.json
```

配置文件与程序分离，修改 IP、端口、规则时不需要重新编译程序。

---

# 16. v0.1 架构

```text
                    NetMap v0.1

┌──────────────────────┐
│     客户端电脑        │
│                      │
│  Application         │
│       │              │
│       ▼              │
│  NetMap Client       │
│       │              │
│       │ rules        │
└───────┼──────────────┘
        │
        │ TCP
        │
        ▼
┌──────────────────────┐
│     跳板机电脑        │
│                      │
│  NetMap Relay        │
│       │              │
│       │ whitelist    │
│       ▼              │
└───────┼──────────────┘
        │
        │ TCP
        ▼
┌──────────────────────┐
│       Target         │
│                      │
│  IP / Domain:Port    │
└──────────────────────┘
```

---

# 17. v0.1 核心原则

NetMap v0.1 不做 VPN。

也不修改操作系统路由表。

也不接管所有网络流量。

它本质上是：

```text
应用
 ↓
本地代理
 ↓
规则判断
 ↓
TCP Relay
 ↓
目标服务
```

只有配置到 `rules` 中的目标才会通过 Relay。

没有命中规则的请求，直接从客户端电脑访问目标。

---

# 18. 当前版本限制

v0.1 是一个简单的 TCP 代理 / Relay 工具。

当前设计重点是：

```text
简单
稳定
配置明确
容易部署
容易调试
```

暂时不追求：

```text
VPN
复杂网络隧道
高性能多路复用
分布式
集群
Web 管理
```

后续功能根据实际使用需求逐步增加。

```

这份 README 和现在代码的设计是对应的，尤其把 **Client `rules` 和 Relay `allowedTargets` 的职责区别**写清楚了，后面你自己维护配置时不容易混淆。

下一步如果继续做，我建议就是把 **v0.1 的目录结构、配置文件和 README 定下来，然后打一个真正的 v0.1 release**，之后再单独开 `v0.2` 做 Windows 托盘。
```
