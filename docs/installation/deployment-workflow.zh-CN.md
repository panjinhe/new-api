# new-api 本地开发 / 线上部署工作流

这套方案的目标只有一个：

- 线上服务器是唯一真实数据源
- 本地只负责开发代码和做临时验证
- 不做本地与线上数据库的双向同步

## 目录约定

仓库根目录下新增这几类目录：

- `data-dev/`：本地 Docker 开发数据目录
- `logs-dev/`：本地 Docker 开发日志
- `data-prod/`：服务器生产数据目录
- `logs-prod/`：服务器生产日志目录
- `backups/`：备份目录
- `prod-backup-snapshot/`：从线上拉回来的备份副本目录

对应文件：

- `docker-compose.dev.postgres.yml`
- `docker-compose.dev.yml`
- `docker-compose.prod.yml`
- `docker-compose.prod.postgres.yml`
- `.env.dev`
- `.env.prod`
- `deploy.sh`
- `backup.sh`
- `pull-prod-snapshot.sh`

## 第一次迁移到服务器

这份工作流默认你使用这套固定生产路径：

- 服务器项目目录：`/opt/new-api/app`
- 生产数据目录：`/opt/new-api/app/data-prod`
- 生产日志目录：`/opt/new-api/app/logs-prod`
- 生产备份目录：`/opt/new-api/app/backups`

如果你现在的数据还在仓库根目录的 `one-api.db`，以及旧的 `data/` 目录里：

1. 把代码推到你的 fork。
2. 在服务器克隆仓库到固定目录，例如 `/opt/new-api/app`。
3. 复制 `.env.prod.example` 为 `.env.prod`，填好 `SESSION_SECRET`、`CRYPTO_SECRET`、域名等。
4. 运行 `./deploy.sh`。

`deploy.sh` 在生产环境下会自动做两件迁移辅助工作：

- 如果 `data-prod/one-api.db` 还不存在，但仓库根目录存在 `one-api.db`，会自动复制进去。
- 如果旧的 `data/` 目录存在，会把里面的文件复制到 `data-prod/`，并保留已有文件。

这样你第一次从“旧目录布局”切到“prod/dev 分离布局”时，不用手工搬每个文件。

另外，现在生产容器本身也支持“首启自动导入旧数据”：

- 如果 `data-prod/one-api.db` 还不存在
- 但仓库根目录存在旧的 `one-api.db`

那么即使你直接执行：

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

容器首次启动时也会自动把旧数据库导入到 `data-prod/one-api.db`。  
旧的 `data/` 目录也会在首启时合并进 `data-prod/`，且不会覆盖已存在文件。

## 本地开发

先准备开发环境变量：

```bash
cp .env.dev.example .env.dev
```

启动本地 Docker 开发环境：

```bash
docker compose -f docker-compose.dev.yml -f docker-compose.dev.postgres.yml up -d --build
```

开发环境的特点：

- 监听端口是 `3000`
- PostgreSQL 数据写到 `postgres-dev/`
- 业务文件仍写到 `data-dev/`
- 不会碰线上 `data-prod/`

本地开发建议固定只走 `docker compose -f docker-compose.dev.yml -f docker-compose.dev.postgres.yml`，不要再和 `go run main.go` 混用。  
这样可以避免：

- 端口占用混乱
- 前端构建产物已经更新，但本地进程还在跑旧的嵌入包
- 你以为自己在验证 Docker，实际上访问到的是宿主机进程

如果之前本地跑过 `go run main.go`，切回 Docker 前先停掉本机 `3000` 端口上的旧进程，再执行上面的 compose 命令。

## 线上部署

先准备生产环境变量：

```bash
cd /opt/new-api/app
cp .env.prod.example .env.prod
nano .env.prod
```

建议至少把这几项填掉：

```env
SESSION_SECRET=换成足够长的随机字符串
CRYPTO_SECRET=换成另一串随机字符串
FRONTEND_BASE_URL=https://your-domain.example.com
NODE_NAME=new-api-prod
TZ=Asia/Shanghai
ERROR_LOG_ENABLED=true
BATCH_UPDATE_ENABLED=true
MEMORY_CACHE_ENABLED=true
TRUSTED_REDIRECT_DOMAINS=your-domain.example.com
```

对你当前这套环境，下面这条命令不再作为日常默认发版方式，改为废弃保留项：

```bash
cd /opt/new-api/app
./deploy.sh
```

如果服务器上的 `/opt/new-api/app` 本身就是一个 git 工作树，并且你想在重建前顺便拉最新代码：

```bash
cd /opt/new-api/app
./deploy.sh --git-pull
```

生产脚本会按这个顺序执行：

1. 检查 `.env.prod`
2. 创建 `data-prod/` 和 `logs-prod/`
3. 首次部署时迁移旧的 `one-api.db` 和 `data/`
4. 调用 `backup.sh` 先做备份
5. `docker compose -f docker-compose.prod.yml up -d --build --remove-orphans`
6. 检查 `http://127.0.0.1:3000/api/status`

## 默认发布流程：本地重建前端并热更新服务器

这部分改成默认流程，原因也很直接：

- 你现在的服务器到 Docker Hub 不稳定
- 线上容器平时本身就在正常运行
- 这个项目的前端会被嵌入 Go 二进制
- 所以“本地构建后直接发到服务器”更符合你现在的长期实际用法

这套方式适合当前项目，是因为前端产物会被嵌入 Go 二进制：

- `main.go` 里通过 `go:embed web/dist` 打包前端
- 所以只要本地先完成前端构建，再重新编译后端二进制，线上替换一个可执行文件就能同时更新前后端

### 适用场景

推荐你当前长期按下面这些场景理解它：

- 线上容器本身在正常运行
- 只是这次代码改动需要发布
- 服务器无法稳定从 Docker Hub 拉基础镜像
- 你希望把发布动作稳定收敛到“本地构建 -> 服务器热更新”

### 本地构建

先在本地仓库根目录执行：

```bash
cd /path/to/new-api/web
bun run build
```

然后回到仓库根目录，编译 Linux 版本二进制：

```bash
cd /path/to/new-api
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOEXPERIMENT=greenteagc \
go build -ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=$(git rev-parse --short HEAD)" \
  -o new-api-linux-amd64
```

如果你本机主要用 PowerShell，推荐直接使用仓库里的脚本，它会自动：

- 重建前端
- 交叉编译 Linux `amd64` 二进制
- 校验产物文件头必须为 `ELF`
- 在发现错误平台产物时直接失败，避免把 Windows `MZ` 可执行文件误发到 Linux 容器里

示例：

```powershell
pwsh ./scripts/build-linux-release.ps1
```

如果你已经提前完成前端构建，也可以跳过前端阶段：

```powershell
pwsh ./scripts/build-linux-release.ps1 -SkipFrontendBuild
```

如果你经常只改后端，可以让脚本在 `web/dist` 已是最新时自动跳过前端构建：

```powershell
pwsh ./scripts/build-linux-release.ps1 -AutoSkipFrontendBuild
```

### 同步源码到服务器

以下示例假设：

- 服务器项目目录：`/opt/new-api/app`
- 运行中的生产容器名：`new-api-prod`
- 服务器临时目录：`/opt/new-api/tmp`
- 二进制备份目录：`/opt/new-api/backups/bin`

先把当前源码打成一个归档包：

```bash
cd /path/to/new-api
SHA=$(git rev-parse --short HEAD)
git archive --format=tar.gz -o /tmp/new-api-deploy-$SHA.tar.gz HEAD
```

再把源码包和本地编好的二进制一起传到服务器：

```bash
scp /tmp/new-api-deploy-$SHA.tar.gz root@your-server:/opt/new-api/tmp/new-api-deploy-$SHA.tar.gz
scp ./new-api-linux-amd64 root@your-server:/opt/new-api/tmp/new-api-linux-amd64
```

先在服务器解开源码包，覆盖更新 `/opt/new-api/app`：

```bash
mkdir -p /opt/new-api/app /opt/new-api/tmp
tar -xzf /opt/new-api/tmp/new-api-deploy-$SHA.tar.gz -C /opt/new-api/app
```

### 替换运行中的程序

然后在服务器执行：

```bash
mkdir -p /opt/new-api/backups/bin /opt/new-api/tmp
TS=$(date +%Y%m%d-%H%M%S)

docker cp new-api-prod:/new-api /opt/new-api/backups/bin/new-api-$TS
chmod 755 /opt/new-api/tmp/new-api-linux-amd64
cp /opt/new-api/tmp/new-api-linux-amd64 /opt/new-api/app/new-api-linux-amd64
docker cp /opt/new-api/tmp/new-api-linux-amd64 new-api-prod:/new-api
docker commit new-api-prod new-api-local:prod >/dev/null
docker restart --time 120 new-api-prod
```

如果你是在本地 PowerShell 里远程执行这些命令，需要额外注意一件事：

- 不要把远端 shell 需要展开的 `$TS`、`$(date ...)` 直接写进 PowerShell 双引号字符串。
- 错误示例：

```powershell
ssh -F ops/ssh/config.local aheapi-prod "TS=$(date +%Y%m%d-%H%M%S); docker cp new-api-prod:/new-api /opt/new-api/backups/bin/new-api-$TS"
```

- 这种写法里，`$(...)` 和 `$TS` 可能会先被本地 PowerShell 处理，导致远端命令异常。
- 更稳妥的写法有两种：

```powershell
ssh -F ops/ssh/config.local aheapi-prod 'TS=$(date +%Y%m%d-%H%M%S); docker cp new-api-prod:/new-api /opt/new-api/backups/bin/new-api-$TS'
```

```powershell
@'
set -e
mkdir -p /opt/new-api/backups/bin /opt/new-api/tmp
TS=$(date +%Y%m%d-%H%M%S)
docker cp new-api-prod:/new-api /opt/new-api/backups/bin/new-api-$TS
chmod 755 /opt/new-api/tmp/new-api-linux-amd64
cp /opt/new-api/tmp/new-api-linux-amd64 /opt/new-api/app/new-api-linux-amd64
docker cp /opt/new-api/tmp/new-api-linux-amd64 new-api-prod:/new-api
docker commit new-api-prod new-api-local:prod >/dev/null
docker restart --time 120 new-api-prod
'@ | ssh -F ops/ssh/config.local aheapi-prod bash -s
```

这里的 `--time 120` 要和生产 Compose 里的 `stop_grace_period: 120s` 保持一致。热更新流程不会重新创建容器，因此不会自动读取 Compose 的 `stop_grace_period`；显式传入 `--time` 可以让新版本的优雅停机逻辑在后续发布时有足够时间等待普通请求完成。

- 对当前项目，推荐长期固定成下面这个约定：
  - 短命令：`ssh '...'`
  - 多行脚本：`@'...'@ | ssh ... bash -s`
  - 避免：`ssh "...$(...)..."` 和 `ssh "...$VAR..."`

最后检查：

```bash
docker inspect -f '{{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{end}}' new-api-prod
wget -q -O - http://127.0.0.1:3000/api/status
```

### 这套方式的特点

- 不依赖服务器重新拉 Docker 基础镜像
- 不会动 `data-prod/`、`logs-prod/` 和数据库内容
- 可以很快把代码变更发布到当前运行中的容器
- 服务器源码目录也会一起更新，和当前运行版本更一致
- 二进制会额外备份一份，方便你手工回滚

### 需要注意的地方

- 这是默认发布方式，也是今后长期固定使用的生产发布方式
- 它不是完整的镜像重建
- 容器里的程序更新了，但 Dockerfile 构建链路并没有被重新验证
- 如果你只替换容器内二进制，不同步服务器源码目录，后续再执行 `./deploy.sh` 时可能把改动覆盖掉
- 所以默认流程里也把源码归档同步到了 `/opt/new-api/app`

### 已废弃：完整镜像重建收口

- 以下“标准镜像发布 / 完整镜像重建收口”流程已标记为废弃，不再作为日常上线方案：

```bash
cd /opt/new-api/app
./deploy.sh
```

- 这套方式的缺点是：

- 整体耗时明显更长
- 很依赖服务器侧镜像拉取和网络状态
- 对你当前这套部署环境来说，性价比不高

- 只有在你明确要验证完整 Docker 构建链路，或者要排查镜像层问题时，才临时使用它。

生产 compose 默认只绑定：

```text
127.0.0.1:3000:3000
```

这表示：

- 应用容器只监听服务器本机
- 外网访问应该通过 Nginx 反向代理进来

长期使用时，建议你继续配：

- Nginx 反向代理
- HTTPS 证书
- 合适的 `client_max_body_size`
- 可排查请求大小和上游耗时的 Nginx `access_log` 格式

对应说明见：

- [nginx-https.zh-CN.md](/E:/new-api/docs/installation/nginx-https.zh-CN.md)

这里额外提醒一条容易忽略的上线项：

- 如果你的客户端会走 `/v1/responses`，尤其是 Codex Desktop 或 CCSwitch，Nginx 不能保留默认约 `1m` 的请求体限制
- 否则请求会在反向代理层直接失败，表现为 `413 Payload Too Large`
- 项目提供的 Nginx 模板已经带了 `client_max_body_size 100m;`，正式上线时建议保留
- `log_format` 建议按 Nginx 文档写入 `/etc/nginx/nginx.conf` 的 `http {}` 层，便于后续从 access log 直接看到 `req_len`、`content_len` 和上游耗时

## 线上数据备份

手工备份：

```bash
./backup.sh --env-name prod
```

如果你希望备份后自动清理旧快照，可以加保留天数：

```bash
./backup.sh --env-name prod --retention-days 14
```

备份内容包括：

- PostgreSQL dump（当前默认）
- `data-prod/` 里的其他文件打包
- 当前 `.env.prod`
- 当前 `docker-compose.prod.yml` 和 PostgreSQL compose overlay（如果存在）
- 当前可读取的 Nginx 关键配置，默认写入 `nginx/`
- 一份元数据说明

备份目录示例：

```text
backups/prod/20260421-210000/
```

Nginx 配置备份默认开启，会尽力保存：

- `/etc/nginx/nginx.conf`
- `/etc/nginx/sites-available/new-api.conf`
- `/etc/nginx/sites-enabled/new-api.conf`
- `nginx -T` 的完整生效配置输出，保存为 `nginx/nginx-T.txt`

如果当前用户没有权限读取某些 Nginx 文件，备份不会因此失败，但 `metadata.txt` 会记录 `nginx_backup_status=partial` 或 `skipped_no_readable_config`。如果某台机器不使用 Nginx，可以显式跳过：

```bash
BACKUP_NGINX_CONFIG=0 ./backup.sh --env-name prod
```

迁移到新服务器时，恢复应用配置和数据库后，还要同步检查 Nginx：

```bash
sudo nginx -t && sudo systemctl reload nginx
curl -ksS https://your-domain.example.com/api/status >/dev/null
tail -n 5 /var/log/nginx/access.log
```

如果日志里能看到 `req_len=`、`content_len=`、`rt=`、`uht=`、`urt=`，说明排障日志格式也已经恢复。

## 生产环境自动备份

项目里已经带了 systemd 定时器模板和安装脚本，适合你现在这台长期运行的 Linux 服务器。

推荐命令：

```bash
cd /opt/new-api/app
sudo ./scripts/install-prod-backup-timer.sh --app-dir /opt/new-api/app --env-name prod --retention-days 14 --on-calendar "*-*-* 04:20:00"
```

这会做三件事：

- 安装 `new-api-backup.service`
- 安装 `new-api-backup.timer`
- 立刻启用定时任务，并让它每天自动执行一次 `./backup.sh --env-name prod`

安装完成后可以检查：

```bash
systemctl status new-api-backup.timer --no-pager
systemctl list-timers --all --no-pager | grep new-api-backup
```

## 从线上拉快照到本地

当你需要“在本地看线上当前配置”时，不要直接双向同步线上库。  
正确做法是：只把线上备份单向拉回本地。

示例：

```bash
REMOTE=ubuntu@your-server REMOTE_APP_DIR=/opt/new-api/app ./pull-prod-snapshot.sh
```

它会在远程服务器先执行：

```bash
./backup.sh --env-name prod --print-path
```

然后把最新备份目录拉到本地：

```text
prod-backup-snapshot/<timestamp>/
```

这样做的好处是：

- 本地可以基于线上真实配置排查问题
- 不会把本地测试数据反写回生产

## 固定工作流

推荐你以后一直按这个节奏走：

1. 本地改代码。
2. 本地用 `docker compose -f docker-compose.dev.yml -f docker-compose.dev.postgres.yml` 验证。
3. 提交到 git 并 push 到自己的 fork。
4. 本地执行 `bun run build` 和 Linux 二进制编译。
5. 本地打源码归档并上传到服务器。
6. 把新二进制传到服务器，替换运行中的 `/new-api` 并重启容器。
7. 检查 `https://your-domain/api/status` 和首页是否正常。
8. 不需要再额外跑 `./deploy.sh` 做镜像收口，默认到这里就可以结束发布。
9. 如果要排查线上问题，再用 `pull-prod-snapshot.sh` 拉一次备份副本到本地。

## 不推荐的做法

下面这些做法建议避免：

- 本地和线上同时写同一份 SQLite 数据库
- 用 git 同步数据库文件
- 让生产数据只存在 Docker 容器内部
- 在服务器直接改代码，再手工回填到本地

## 恢复思路

如果你要回滚某次部署，可以这样做：

1. 停掉生产容器
2. 从某个备份目录恢复对应的 PostgreSQL dump
3. 如果需要，也把 `data-extra.tar.gz` 里的文件解回 `data-prod/`
4. 重新执行对应的启动/部署流程

## 说明

这套工作流是按你当前场景定的：

- 个人维护
- 一套主线上环境
- 当前本地与线上都以 PostgreSQL 为主
- 需要从线上单向拉快照到本地

如果后面你要升级成：

- 测试环境 + 正式环境双环境
- PostgreSQL
- 自动化 CI/CD

可以在这套骨架上继续扩展，不用推倒重来。

如果你现在就准备把生产库从 SQLite 切到 PostgreSQL，可以直接看：

- [postgres-migration.zh-CN.md](/E:/new-api/docs/installation/postgres-migration.zh-CN.md)
