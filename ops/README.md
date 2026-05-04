# Ops Notes

`ops/` 用来存放本地运维信息模板和 SSH 配置模板。

约定如下：

- `server.local.toml`：本地服务器清单，不提交到 git
- `server.local.toml.example`：示例模板，可提交到 git
- `ssh/config.local`：本地 SSH 别名配置，不提交到 git
- `ssh/config.local.example`：示例模板，可提交到 git

建议：

- 真实密钥只保留在本机，不写入仓库
- 真实环境变量继续只保留在服务器 `.env.prod`
- 如果以后新增服务器，也继续按这套命名扩展

## 生产部署默认约定

- 以后生产发版默认使用“本地构建 + 蓝绿短重叠 + Nginx upstream 切流 + 旧实例 drain”流程。
- `deploy-fast-prod.ps1` 只保留为备用/应急路径，不再作为日常默认发布方式。
- 不再把“本地构建镜像 + `docker save | gzip | ssh | docker load` + 服务器 `docker compose up -d --no-build`”作为日常发布方式。
- 原因很简单：
  - 整镜像传输太慢
  - 单容器原地重启会造成发布窗口内的 5xx 和流式中断
  - 当前生产已经完成首次 blue 演练，Nginx upstream 已切到蓝绿结构

### 默认发布步骤

```powershell
pwsh ./scripts/deploy-bluegreen-prod.ps1
```

- 这一步会自动：
  - 在 `web/dist` 已是最新时跳过前端构建
  - 编译 Linux `amd64` 二进制
  - 校验产物是 Linux `ELF`
  - 打包并同步当前 `HEAD` 源码
  - 上传二进制到 `aheapi-itdun`
  - 启动 idle color 容器并等待 `/api/status` 健康检查
  - 切换 Nginx upstream 到新 color
  - 公开地址 smoke test 通过后写入 `runtime-prod/active-color`
  - 按 drain 超时停止旧 color

常用变体：

```powershell
# 前端已确认无变化时，直接跳过前端构建
pwsh ./scripts/deploy-bluegreen-prod.ps1 -SkipFrontendBuild

# 已经手动构建过二进制，只发布现有产物
pwsh ./scripts/deploy-bluegreen-prod.ps1 -SkipBuild

# 只替换二进制，不同步源码包
pwsh ./scripts/deploy-bluegreen-prod.ps1 -SkipSourceSync
```

注意：首次从 legacy 切到 blue / green 时不能用 `-SkipSourceSync`，因为服务器需要同步蓝绿 compose、Nginx 模板和部署脚本。现在首次演练已完成，后续常规蓝绿切换可以按需使用 `-SkipSourceSync`，但默认仍建议同步当前 `HEAD`。

### 备用发布路径

如果蓝绿脚本本身需要维护，或者必须回到 legacy 单容器模式，才使用快速发布脚本：

```powershell
pwsh ./scripts/deploy-fast-prod.ps1
```

这条路径会替换 `new-api-prod` 并重启单容器，可能造成短暂中断；使用前必须明确接受这个影响。

生产环境禁止在服务器上执行 `./deploy.sh --env-name prod ...`；该命令会直接退出，避免误触发服务器端 Docker build。

## 蓝绿发布演练记录

### 当前蓝绿发布入口

首次蓝绿发布和后续蓝绿切换都使用本地脚本：

```powershell
pwsh ./scripts/deploy-bluegreen-prod.ps1
```

脚本会做这些事：

1. 本地构建 Linux `amd64` 二进制。
2. 从当前干净的 `HEAD` 打源码归档并上传到生产服务器。
3. 在服务器使用显式 `.env.prod` 解析 compose：

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml -f docker-compose.prod.bluegreen.yml
```

4. 启动 idle color，例如首次从 legacy 切换时启动 `new-api-blue`，监听 `127.0.0.1:3001`。
5. idle 实例健康后，修改 Nginx upstream 并 reload。
6. 公网 `/api/status` smoke test 通过后，写入 `runtime-prod/active-color`。
7. drain 并停止旧实例。

### 2026-05-05 首次低峰演练结果

- 发布版本：`3dbd2c588`
- 起始状态：
  - legacy `new-api-prod` 运行在 `127.0.0.1:3000`
  - `3001` / `3002` 未占用
  - 生产服务器资源充足：约 `79G` 磁盘可用、约 `2.9G` 内存 available
- 目标状态：
  - `new-api-blue` 运行在 `127.0.0.1:3001`
  - Nginx 站点统一 `proxy_pass http://new_api_bluegreen`
  - upstream 指向 `127.0.0.1:3001`
  - `runtime-prod/active-color` 为 `blue`
  - legacy `new-api-prod` 已按 `120s` drain 停止
- 验收结果：
  - `new-api-blue` healthy
  - `new-api-postgres` healthy
  - `nginx -t` 通过
  - `https://aheapi.com/api/status` 正常
  - 日志未发现迁移冲突、panic 或数据库认证失败

### 首次演练中踩到的坑

第一次自动脚本在 Nginx 切流阶段失败，原因是旧脚本把备份文件放在 `/etc/nginx/sites-enabled/` 里：

```text
/etc/nginx/sites-enabled/new-api.conf.bluegreen-backup-...
/etc/nginx/sites-enabled/pbroe-redirect.conf.bluegreen-backup-...
```

Nginx 会加载 `sites-enabled` 下的这些备份文件，导致：

```text
duplicate listen options for [::]:443
nginx: configuration file /etc/nginx/nginx.conf test failed
```

处理方式：

1. 不删除备份，只移动到不会被 Nginx include 的目录：

```bash
mkdir -p /etc/nginx/bluegreen-backups/20260504-180346
mv /etc/nginx/sites-enabled/*bluegreen-backup-20260504-180346 /etc/nginx/bluegreen-backups/20260504-180346/
nginx -t
```

2. 确认当时实际状态：
   - Nginx 站点已经被 patch 为 `proxy_pass http://new_api_bluegreen`
   - upstream 已回滚指向 `127.0.0.1:3000`
   - legacy `new-api-prod` 仍然 healthy
   - 新 `new-api-blue` 已启动并 healthy

3. 手动完成剩余切流：

```bash
cat >/etc/nginx/conf.d/new-api-bluegreen-upstream.conf <<'EOF'
upstream new_api_bluegreen {
    server 127.0.0.1:3001;
    keepalive 32;
}
EOF
nginx -t && nginx -s reload
curl -fsS -m 10 https://aheapi.com/api/status >/dev/null
printf 'blue\n' >/opt/new-api/app/runtime-prod/active-color
docker stop --time 120 new-api-prod
```

脚本已修复：Nginx 站点备份现在保存到 `/etc/nginx/bluegreen-backups/<timestamp>/`，不会再留在 `sites-enabled`。

#### 高频错误：本地 PowerShell 抢先展开远端 shell 变量

这次演练后续手动修复时又踩了一次老坑：在本地 PowerShell 里用双引号执行远端 `ssh` 命令，导致本应在服务器上展开的 `$backup_dir`、`$f`、`$(cat ...)` 被本地 PowerShell 先处理，最终远端拿到的是残缺命令。

典型错误现象：

```text
bash: -c: line 1: syntax error near unexpected token `do'
Cannot find path 'E:\opt\new-api\app\runtime-prod\active-color'
```

错误写法：

```powershell
ssh -F ops/ssh/config.local aheapi-itdun "backup_dir=/etc/nginx/bluegreen-backups/ts; for f in /etc/nginx/sites-enabled/*backup*; do mv "$f" "$backup_dir/"; done"
ssh -F ops/ssh/config.local aheapi-itdun "echo 'active='$(cat /opt/new-api/app/runtime-prod/active-color)"
```

正确写法：

```powershell
ssh -F ops/ssh/config.local aheapi-itdun 'backup_dir=/etc/nginx/bluegreen-backups/ts; for f in /etc/nginx/sites-enabled/*backup*; do mv "$f" "$backup_dir/"; done'
ssh -F ops/ssh/config.local aheapi-itdun 'printf "active="; cat /opt/new-api/app/runtime-prod/active-color'
```

强制约定：凡是远端命令里出现 `$变量`、`$(...)`、循环、here-doc、awk 这类 shell 语法，默认使用单引号包整段远端命令，或使用 PowerShell here-string 管道到 `ssh ... bash -s`。不要再用双引号。

### 蓝绿发布前检查清单

本地：

```powershell
git status --short --branch
go test ./model ./common
pwsh ./scripts/deploy-bluegreen-prod.ps1 -DryRun -SkipFrontendBuild
```

要求：

- 工作区必须干净；首次蓝绿发布不能使用 `-SkipSourceSync`，因为服务器需要同步 `docker-compose.prod.bluegreen.yml` 和脚本。
- dry-run 输出的版本号必须是准备发布的 `HEAD`。
- dry-run 必须显示 compose 命令包含 `--env-file .env.prod`。

生产服务器只读检查：

```bash
cd /opt/new-api/app
docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml config | grep 'SQL_DSN:'
ss -ltnp | grep -E ':(3000|3001|3002)\b' || true
free -h
df -h /
nginx -t
curl -fsS -m 8 https://aheapi.com/api/status >/dev/null
```

要求：

- 当前 active 实例 healthy。
- PostgreSQL healthy。
- `3001` 或 `3002` 至少一个空闲。
- `SQL_DSN` 不能出现 `change-me`。
- Nginx 配置测试通过。
- 公网健康检查通过。

### 蓝绿发布后检查清单

```bash
cat /opt/new-api/app/runtime-prod/active-color
cat /etc/nginx/conf.d/new-api-bluegreen-upstream.conf
docker ps -a --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
curl -fsS -m 10 https://aheapi.com/api/status >/dev/null
nginx -t
docker logs --since 5m new-api-blue 2>&1 | grep -Ei 'panic|fatal|failed|error|duplicate|pg_class' | tail -80 || true
```

重点确认：

- active color 和 upstream 端口一致。
- 新 active 容器 healthy。
- 旧容器已停止，或处于预期 drain 状态。
- Nginx 配置仍然通过。
- 最近日志没有迁移冲突、数据库认证失败、panic。

### 手动回滚方式

如果切到 blue 后公网 smoke fail，但 legacy 仍在：

```bash
cat >/etc/nginx/conf.d/new-api-bluegreen-upstream.conf <<'EOF'
upstream new_api_bluegreen {
    server 127.0.0.1:3000;
    keepalive 32;
}
EOF
nginx -t && nginx -s reload
printf 'legacy\n' >/opt/new-api/app/runtime-prod/active-color
```

如果 legacy 已停，但上一个 color 仍在，例如要从 blue 回 green，改 upstream 到对应端口并 reload：

```bash
# green 为 3002，blue 为 3001
cat >/etc/nginx/conf.d/new-api-bluegreen-upstream.conf <<'EOF'
upstream new_api_bluegreen {
    server 127.0.0.1:3002;
    keepalive 32;
}
EOF
nginx -t && nginx -s reload
printf 'green\n' >/opt/new-api/app/runtime-prod/active-color
```

### PowerShell 远程执行注意事项

- 这是高频错误；如果你在本地用 PowerShell 调 `ssh` 执行远端 shell 命令，不要把本来应该留给远端展开的 `$VAR`、`$(...)` 直接写进双引号字符串。
- 判断规则很简单：远端命令只要包含 `$`，默认不要用双引号。
- 典型错误写法：

```powershell
ssh -F ops/ssh/config.local aheapi-prod "TS=$(date +%Y%m%d-%H%M%S); echo $TS"
```

- 这种写法里：
  - `$(...)` 很可能先被本地 PowerShell 当成子表达式处理
  - `$TS` 也可能被本地 PowerShell 提前展开
  - 最终导致远端没有按预期执行，或者命令直接报错
- 推荐固定使用下面两种方式：

```powershell
ssh -F ops/ssh/config.local aheapi-prod 'TS=$(date +%Y%m%d-%H%M%S); echo $TS'
```

```powershell
@'
set -e
TS=$(date +%Y%m%d-%H%M%S)
echo "$TS"
'@ | ssh -F ops/ssh/config.local aheapi-prod bash -s
```

- 建议长期约定：
  - 短命令用单引号包整段远端 shell
  - 多行脚本统一用 here-string 管道到 `ssh ... bash -s`
  - 不要默认写 `ssh "...$(...)..."` 这种形式

### 已废弃：标准镜像发布

- 以下流程标记为废弃，不再作为默认发布方式：

```bash
docker build -t new-api-local:prod .
docker save new-api-local:prod | gzip -1 | ssh -F ops/ssh/config.local aheapi-prod "gunzip | docker load"
ssh -F ops/ssh/config.local aheapi-prod "cd /opt/new-api/app && docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml up -d --no-build"
```

- 仅在你明确需要验证完整 Docker 构建链路时，才临时使用这套。

## 生产部署避坑

- 在服务器手动执行 `docker compose` 之前，必须先加载 `.env.prod`，否则 Compose 不会自动使用其中的变量值去展开 `docker-compose.prod*.yml` 里的 `${...}` 表达式。
- 本项目的 PostgreSQL 组合配置依赖这些变量：

```bash
SQL_DSN=postgresql://${POSTGRES_USER:-newapi}:${POSTGRES_PASSWORD:-change-me}@postgres:5432/${POSTGRES_DB:-newapi}?sslmode=disable
```

- 如果没有先加载 `.env.prod`，`POSTGRES_PASSWORD` 很容易被错误展开成默认值 `change-me`，从而让 `new-api` 容器出现数据库认证失败并反复重启。
- 这类错误的典型现象：
  - `new-api-prod` 容器持续 `Restarting`
  - 日志里出现 `password authentication failed for user "newapi"`
  - `docker inspect new-api-prod` 里能看到错误的 `SQL_DSN=...:change-me@postgres...`
- 服务器手动部署时，正确顺序必须是：

```bash
cd /opt/new-api/app
docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml up -d --no-build
```

- 如果只想重建应用容器、不要动数据库容器，使用：

```bash
cd /opt/new-api/app
docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml up -d --no-build --no-deps --force-recreate new-api
```

- 部署后务必检查：

```bash
docker ps
curl http://127.0.0.1:3000/api/status
docker logs --tail 100 new-api-prod
docker inspect new-api-prod --format '{{json .Config.Env}}'
```

## Codex 渠道排障经验

### 症状：本地能用，服务器页面新填的 Codex 渠道不能用

- 典型现象：
  - 同一份 Codex JSON 凭证，在本地页面添加后可以正常用。
  - 在阿里云部署的线上页面添加后，渠道测试或实际调用失败。
  - 已有 `Codex-1/3/4/...` 正常，只有新建的 `Codex-pro20x-1` 不正常。

### 根因

- 真正发请求的是后端，不是浏览器本身。
  - 本地页面成功：实际出网的是本地 `new-api` 后端。
  - 服务器页面失败：实际出网的是阿里云上的 `new-api` 后端。
- 这次故障的核心差异不是凭证，而是线上该渠道记录的 `setting.proxy` 为空。
  - 正常的 Codex 渠道：
    - `proxy = http://host.docker.internal:7890`
  - 异常的 `Codex-pro20x-1`：
    - `proxy = ""`
- 当时线上运行的还是旧代码，尚未包含“编辑已有渠道并保存时，也自动回填默认代理”的修复。
  - 所以即使在页面里点了“编辑 -> 保存”，也不会把空代理补上。

### 如何确认

- 先查生产数据库中的渠道记录：

```bash
ssh -F ops/ssh/config.local aheapi-prod <<'EOF'
cd /opt/new-api/app
docker exec new-api-postgres psql -U newapi -d newapi -c \
"select id,name,type,\"group\",status,test_model,models,setting from channels where name like 'Codex-%' order by id;"
EOF
```

- 如果 `Codex-pro20x-1` 的 `setting.proxy` 为空，而其他 Codex 渠道不为空，就说明是代理缺失。

- 再确认线上源码是否已包含“更新渠道时回填代理”的修复：

```bash
ssh -F ops/ssh/config.local aheapi-prod <<'EOF'
cd /opt/new-api/app
sed -n '855,885p' controller/channel.go
EOF
```

- 如果这里还是旧逻辑，说明服务器还没部署到最新版本。

### 这次的实际修复方式

1. 将本地最新代码同步到服务器。
2. 因服务器拉 Docker Hub 超时，改为本地构建镜像并压缩传输到服务器。
3. 服务器使用 `--no-build --no-deps --force-recreate new-api` 重建应用容器。
4. 直接把生产库里 `Codex-pro20x-1` 的 `proxy` 补成和其他 Codex 一样的值：
   - `http://host.docker.internal:7890`
5. 重启 `new-api-prod`，等待健康检查通过。

### 直接修复命令

- 修改生产库里的 `proxy`：

```bash
ssh -F ops/ssh/config.local aheapi-prod <<'EOF'
cd /opt/new-api/app
docker exec new-api-postgres psql -U newapi -d newapi -c \
"update channels
 set setting = jsonb_set(
   coalesce(setting::jsonb, '{}'::jsonb),
   '{proxy}',
   '\"http://host.docker.internal:7890\"'::jsonb,
   true
 )::text
 where name='Codex-pro20x-1';"
EOF
```

- 重建应用容器：

```bash
ssh -F ops/ssh/config.local aheapi-prod <<'EOF'
cd /opt/new-api/app
docker compose --env-file .env.prod -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml \
  up -d --no-build --no-deps --force-recreate new-api
EOF
```

### 额外提醒

- Codex 渠道只支持 `/v1/responses`，不支持 `/v1/chat/completions`。
- 如果日志里出现：

```text
codex channel: /v1/chat/completions endpoint not supported
```

- 这不是渠道坏了，而是调用入口走错了协议。
- 所以排查 Codex 渠道时要分开看：
  - 渠道配置是否正确：看 `proxy / key / account_id / models / group`
  - 客户端调用方式是否正确：看是不是走了 `/v1/responses`

## Nginx 源站收口经验

> 历史记录：本节记录迁移前 `pbroe.com` 在阿里云源站上的 Nginx 收口方式。当前生产主域名已切换为 `aheapi.com`，`pbroe.com` 仅作为旧域名兼容与跳转入口保留。

### 为什么要做

- 只有安全组放行 `80/443` 还不够。
- 如果 Nginx 仍然接受 IP 直连、兜底 `server_name _;` 也正常转发，别人只要扫到源站 IP，就可以绕过域名直接打到站。
- 对生产站点来说，至少要先做到：
  - 拒绝 IP 直连
  - 只响应正式域名
  - 加一层基础限流
  - 隐藏 Nginx 版本和不必要的上游响应头

### 这次线上实际做了什么（旧阿里云 `pbroe.com`）

- 域名只保留：
  - `pbroe.com`
  - `www.pbroe.com`
- Nginx 增加默认兜底站点：
  - `80` 默认 `server` 直接 `return 444;`
  - `443` 默认 `server` 直接 `return 444;`
- 只有命中正式域名的请求才会：
  - `80 -> 443` 跳转
  - `443 -> 127.0.0.1:3000` 反代到 `new-api`
- 开启基础限流：
  - `limit_req_zone $binary_remote_addr zone=perip_general:10m rate=30r/s;`
  - `limit_conn_zone $binary_remote_addr zone=perip_conn:10m;`
  - `location /` 内：
    - `limit_req zone=perip_general burst=60 nodelay;`
    - `limit_conn perip_conn 30;`
- 隐藏版本与敏感响应头：
  - `server_tokens off;`
  - `proxy_hide_header X-New-Api-Version;`
  - `proxy_hide_header X-Oneapi-Request-Id;`

### 验收方式

- 校验 Nginx 配置：

```bash
sudo nginx -t && sudo systemctl reload nginx
```

- 验证正式域名仍然可用：

```bash
curl -I https://aheapi.com
```

- 验证 HTTP IP 直连被拒绝：

```bash
curl -I http://47.111.11.175
```

- 预期现象：
  - 域名返回 `200` 或业务正常响应
  - `http://47.111.11.175` 应该是空回复、连接被直接掐掉，或等价的拒绝行为

- 验证 HTTPS IP 直连被兜底站点丢弃：

```bash
curl -vkI https://47.111.11.175
```

- 预期现象：
  - 握手后拿不到业务页面
  - 不会再被转发到 `new-api`

### 仍然要注意的一点

- 现在只是把“HTTP 层的裸奔”先收住了，不代表源站 IP 已完全隐藏。
- 如果公网 DNS 直接把域名解析到源站 IP：
  - `pbroe.com`
  - `www.pbroe.com`
  - `aheapi.com`
  解析到源站 IP
- 那么源站 IP 依然能被公开看到。
- 如果后面要继续增强防护，正确方向是：
  - 接 CDN / WAF
  - 回源只允许 CDN / WAF 出口
  - 安全组和 Nginx 一起只信任代理层流量
