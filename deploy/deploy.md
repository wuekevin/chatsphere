# ChatSphere 部署指南

## 一、Docker Compose 部署（推荐开发/测试）

```bash
cd /path/to/chatsphere
docker compose -f deploy/docker-compose.yml up -d
```

首次启动后访问 `http://<IP>:8080`，系统自动跳转安装向导。完成安装后服务自动切换为正式模式。

---

## 二、宝塔面板部署（生产推荐）

### 1. 编译二进制

在本地或 CI 执行：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o chatsphere ./cmd/server
```

将 `chatsphere` 上传至服务器 `/opt/chatsphere/`。

### 2. 创建目录结构

```
/opt/chatsphere/
├── chatsphere        # 编译好的二进制
└── config.yaml       # 安装向导完成后自动生成
```

### 3. 安装为系统服务

```bash
cp deploy/chatsphere.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable chatsphere
systemctl start chatsphere
```

查看日志：

```bash
journalctl -u chatsphere -f
```

### 4. 宝塔 Nginx 反代配置

在宝塔面板 → 网站 → 添加站点，域名填写你的域名，然后进入「配置文件」，将 `deploy/nginx.conf` 的 `location` 块合并到生成的 server 块中。

关键点：
- WebSocket 路径 `/ws` 必须设置 `Upgrade` 和 `Connection` 头
- SSE 路径 `/api/setup/install` 必须关闭 `proxy_buffering`
- SSL 证书通过宝塔「SSL」面板申请 Let's Encrypt 即可

### 5. 访问安装向导

首次启动时 `config.yaml` 不存在，服务以安装模式运行。访问 `https://你的域名` 完成配置后，服务自动重启为正式模式，无需手动操作。

---

## 三、环境变量（可选覆盖）

| 变量 | 说明 | 默认 |
|------|------|------|
| `CHATSPHERE_CONFIG` | config.yaml 路径 | `./config.yaml` |
| `GIN_MODE` | `release` / `debug` | `release` |

---

## 四、数据库初始化

安装向导会自动执行 GORM AutoMigrate，创建以下表：

- `users`
- `channels`
- `messages`
- `reports`
- `direct_messages`
- `reputation_tiers`

无需手动建表。

---

## 五、升级

```bash
# 停止服务
systemctl stop chatsphere

# 替换二进制
cp chatsphere_new /opt/chatsphere/chatsphere

# 重启
systemctl start chatsphere
```

config.yaml 和数据库数据不受影响。
