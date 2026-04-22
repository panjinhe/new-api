# new-api 生产环境 Nginx 与 HTTPS 配置

这份文档按下面这套固定路径来写：

- 服务器项目目录：`/opt/new-api/app`
- 站点域名：`your-domain.example.com`
- new-api Docker 生产端口：`127.0.0.1:3000`
- Nginx 配置文件：`/etc/nginx/sites-available/new-api.conf`

如果你后面没有特殊原因，建议一直保持这套结构。

## 为什么这样配

生产环境推荐用：

- Docker 只监听本机 `127.0.0.1:3000`
- Nginx 负责对外暴露 `80/443`
- 域名访问走 HTTPS

这样好处是：

- 应用本身不直接裸露在公网
- HTTPS、重定向、证书续期都放到 Nginx 层处理
- 以后换域名、加限流、加访问控制更方便

## 生产环境标准目录

推荐服务器目录：

```text
/opt/new-api/app
```

应用代码结构：

```text
/opt/new-api/app/.env.prod
/opt/new-api/app/docker-compose.prod.yml
/opt/new-api/app/data-prod/
/opt/new-api/app/logs-prod/
/opt/new-api/app/backups/
```

## 先启动应用

先把代码部署好，再配 Nginx。

```bash
cd /opt/new-api/app
cp .env.prod.example .env.prod
nano .env.prod
./deploy.sh
```

`.env.prod` 至少要改：

- `SESSION_SECRET`
- `CRYPTO_SECRET`
- `FRONTEND_BASE_URL=https://your-domain.example.com`
- `TRUSTED_REDIRECT_DOMAINS=your-domain.example.com`

## 安装 Nginx

Debian / Ubuntu：

```bash
sudo apt update
sudo apt install -y nginx
```

## 写入站点配置

项目里已经提供模板文件：

- [new-api.conf.example](/E:/new-api/deploy/nginx/new-api.conf.example)

复制到服务器：

```bash
sudo cp /opt/new-api/app/deploy/nginx/new-api.conf.example /etc/nginx/sites-available/new-api.conf
```

编辑这几个值：

- `server_name your-domain.example.com;`
- `ssl_certificate`
- `ssl_certificate_key`

这个模板里已经包含一条很重要的配置：

- `client_max_body_size 100m;`

这条建议保留。  
原因是 Codex Desktop、CCSwitch 以及走 `/v1/responses` 的调用，请求体通常会比普通接口大得多；如果 Nginx 还停留在默认的约 `1m` 限制，请求会先被 Nginx 拒绝，表现为：

- `413 Payload Too Large`
- 日志里出现 `client intended to send too large body`
- 出错地址通常是 `POST /v1/responses`

然后启用站点：

```bash
sudo ln -sf /etc/nginx/sites-available/new-api.conf /etc/nginx/sites-enabled/new-api.conf
sudo nginx -t
sudo systemctl reload nginx
```

## 申请 HTTPS 证书

推荐用 Certbot。

Debian / Ubuntu：

```bash
sudo apt install -y certbot python3-certbot-nginx
```

确保域名已经解析到这台服务器，然后执行：

```bash
sudo certbot --nginx -d your-domain.example.com
```

成功后，Certbot 通常会自动改写 Nginx 里的证书路径，并配置自动续期。

可以检查自动续期：

```bash
sudo systemctl status certbot.timer
```

或者手动试跑：

```bash
sudo certbot renew --dry-run
```

## 防火墙

如果服务器开了防火墙，至少要放行：

- `80/tcp`
- `443/tcp`

UFW 示例：

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw reload
```

## 生产上线后的验证

先看容器：

```bash
cd /opt/new-api/app
docker compose -f docker-compose.prod.yml ps
```

再看本机健康检查：

```bash
curl http://127.0.0.1:3000/api/status
```

最后看公网域名：

```bash
curl -I https://your-domain.example.com
```

你应该能看到：

- 容器状态正常
- 本机 `127.0.0.1:3000` 返回成功
- 域名是 HTTPS

如果你要额外验证“大请求不会被 Nginx 拦住”，可以再补一条检查思路：

- 正常情况下，大请求应该进入应用层，哪怕最后返回 `401`、`400` 或业务错误，也不应该再是 Nginx 的 `413`

## 推荐的正式上线命令

以后服务器上固定就用这几条：

部署更新：

```bash
cd /opt/new-api/app
git pull --ff-only
./deploy.sh
```

手工备份：

```bash
cd /opt/new-api/app
./backup.sh --env-name prod
```

查看日志：

```bash
cd /opt/new-api/app
docker compose -f docker-compose.prod.yml logs -f
```

## 常见说明

### 1. 为什么生产 compose 绑定 `127.0.0.1:3000`

因为这表示：

- 容器只接受服务器本机访问
- 公网访问必须经过 Nginx

这比直接暴露 `0.0.0.0:3000` 更适合长期运行。

### 2. 如果我暂时还没配 HTTPS

可以先临时改成 HTTP 访问，但长期运行不建议。  
只要涉及登录、令牌、管理后台，HTTPS 都应该尽快配好。

### 3. 证书路径为什么先写死示例

这是为了让你一眼能看懂 Nginx 怎么接进来。  
真正上线时，最常见的做法就是让 Certbot 自动生成并接管这些路径。

### 4. 如果 Codex / Responses 接口报 `413 Payload Too Large`

优先检查 Nginx，而不是先怀疑渠道或容器。

常见特征：

- 浏览器或客户端提示 `413 Payload Too Large`
- Nginx `error.log` 出现 `client intended to send too large body`
- `access.log` 里是 `POST /v1/responses` 返回 `413`

排查顺序建议：

1. 先确认 `new-api` 容器本身是正常的：

```bash
curl http://127.0.0.1:3000/api/status
```

2. 再检查 Nginx 生效配置里有没有 `client_max_body_size`：

```bash
sudo nginx -T | grep -n "client_max_body_size"
```

3. 如果你不是用 `/etc/nginx/sites-available/new-api.conf`，而是直接把站点写在 `/etc/nginx/nginx.conf`，那也要确保真正处理 `443 ssl` 的那个 `server` 块，或者 `http {}` 全局层级里，已经设置了例如：

```nginx
client_max_body_size 20m;
```

4. 修改后执行：

```bash
sudo nginx -t
sudo systemctl reload nginx
```

经验上，`20m` 已经能覆盖当前 Codex Desktop 的常见请求；如果你希望少折腾，直接沿用项目模板里的 `100m` 也可以。
