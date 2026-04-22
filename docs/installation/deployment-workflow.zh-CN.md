# new-api 本地开发 / 线上部署工作流

这套方案的目标只有一个：

- 线上服务器是唯一真实数据源
- 本地只负责开发代码和做临时验证
- 不做本地与线上数据库的双向同步

## 目录约定

仓库根目录下新增这几类目录：

- `data-dev/`：本地 Docker 开发库
- `logs-dev/`：本地 Docker 开发日志
- `data-prod/`：服务器生产数据目录
- `logs-prod/`：服务器生产日志目录
- `backups/`：备份目录
- `data-prod-snapshot/`：从线上拉回来的快照目录

对应文件：

- `docker-compose.dev.yml`
- `docker-compose.dev.postgres.yml`
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
docker compose -f docker-compose.dev.yml up -d --build
```

开发环境的特点：

- 监听端口是 `3001`
- 数据只写到 `data-dev/one-api.db`
- 不会碰线上 `data-prod/`
- 如果 `data-dev/one-api.db` 还不存在，而仓库根目录存在旧的 `one-api.db`，容器首次启动时会自动导入
- 如果仓库根目录存在旧的 `data/` 目录，容器首次启动时也会把里面的文件合并进 `data-dev/`，且不会覆盖已存在文件

如果你只是改小功能，也可以继续用本地 `go run main.go`。  
但在准备上线前，建议至少再用一次 `docker-compose.dev.yml` 做“接近生产”的验证。

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

生产部署命令：

```bash
cd /opt/new-api/app
./deploy.sh
```

如果你已经在服务器上，并且想在部署前顺便拉最新代码：

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

## Docker Hub 拉取失败时的应急发布

正常情况下，生产环境还是优先使用：

```bash
cd /opt/new-api/app
./deploy.sh
```

但如果服务器当前无法稳定访问 Docker Hub，导致 `docker compose up -d --build` 卡在拉取基础镜像这一步，可以使用一套已经验证过的应急方案：

- 本地编译前端
- 本地编译 Linux 二进制
- 直接替换线上容器里的 `/new-api`

这套方式适合当前项目，是因为前端产物会被嵌入 Go 二进制：

- `main.go` 里通过 `go:embed web/dist` 打包前端
- 所以只要本地先完成前端构建，再重新编译后端二进制，线上替换一个可执行文件就能同时更新前后端

### 适用场景

建议只在下面这些场景使用：

- 线上容器本身在正常运行
- 只是这次代码改动需要发布
- 服务器无法顺利从 Docker Hub 拉基础镜像
- 你不想因为镜像拉取问题阻塞上线

不建议把它作为长期默认发布方式。  
等网络恢复后，还是应该回到 `./deploy.sh` 这条标准路径。

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

### 服务器替换步骤

以下示例假设：

- 服务器项目目录：`/opt/new-api/app`
- 运行中的生产容器名：`new-api-prod`
- 服务器临时目录：`/opt/new-api/tmp`
- 二进制备份目录：`/opt/new-api/backups/bin`

先把本地编好的二进制传到服务器：

```bash
scp ./new-api-linux-amd64 root@your-server:/opt/new-api/tmp/new-api-linux-amd64
```

然后在服务器执行：

```bash
mkdir -p /opt/new-api/backups/bin /opt/new-api/tmp
TS=$(date +%Y%m%d-%H%M%S)

docker cp new-api-prod:/new-api /opt/new-api/backups/bin/new-api-$TS
chmod 755 /opt/new-api/tmp/new-api-linux-amd64
cp /opt/new-api/tmp/new-api-linux-amd64 /opt/new-api/app/new-api-linux-amd64
docker cp /opt/new-api/tmp/new-api-linux-amd64 new-api-prod:/new-api
docker commit new-api-prod new-api-local:prod >/dev/null
docker restart new-api-prod
```

最后检查：

```bash
docker inspect -f '{{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{end}}' new-api-prod
wget -q -O - http://127.0.0.1:3000/api/status
```

### 这套方式的特点

- 不依赖服务器重新拉 Docker 基础镜像
- 不会动 `data-prod/`、`logs-prod/` 和数据库内容
- 可以很快把代码变更发布到当前运行中的容器
- 二进制会额外备份一份，方便你手工回滚

### 需要注意的地方

- 这是“应急热更新”，不是完整的镜像重建
- 容器里的程序更新了，但 Dockerfile 构建链路并没有被重新验证
- 如果你只替换容器内二进制，没有同步服务器源码目录，后续再执行 `./deploy.sh` 时可能把改动覆盖掉
- 所以更稳妥的做法是：至少把对应源码也同步到 `/opt/new-api/app`，保证下次标准部署时源码和线上程序一致

### 应急发布后的收口建议

等服务器网络恢复、可以正常访问 Docker Hub 后，建议补做一次标准部署：

```bash
cd /opt/new-api/app
./deploy.sh
```

这样可以把：

- Docker 镜像
- 容器内程序
- 服务器源码目录

重新收敛到同一状态，避免以后排查问题时出现“源码是一版、容器里跑的是另一版”的情况。

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

对应说明见：

- [nginx-https.zh-CN.md](/E:/new-api/docs/installation/nginx-https.zh-CN.md)

这里额外提醒一条容易忽略的上线项：

- 如果你的客户端会走 `/v1/responses`，尤其是 Codex Desktop 或 CCSwitch，Nginx 不能保留默认约 `1m` 的请求体限制
- 否则请求会在反向代理层直接失败，表现为 `413 Payload Too Large`
- 项目提供的 Nginx 模板已经带了 `client_max_body_size 100m;`，正式上线时建议保留

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

- SQLite 数据库快照
- `data-prod/` 里的其他文件打包
- 当前 `.env.prod`
- 当前 `docker-compose.prod.yml`
- 一份元数据说明

备份目录示例：

```text
backups/prod/20260421-210000/
```

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
data-prod-snapshot/<timestamp>/
```

如果你想把这份线上快照作为本地临时调试库，可以手工复制：

```bash
cp data-prod-snapshot/<timestamp>/one-api.db data-dev/one-api.db
```

这样做的好处是：

- 本地可以基于线上真实配置排查问题
- 不会把本地测试数据反写回生产

## 固定工作流

推荐你以后一直按这个节奏走：

1. 本地改代码。
2. 本地用 `docker-compose.dev.yml` 验证。
3. 提交到 git 并 push 到自己的 fork。
4. 服务器 `git pull --ff-only`。
5. 服务器执行 `./deploy.sh`。
6. 如果要排查线上问题，再用 `pull-prod-snapshot.sh` 拉一次快照到本地。

## 不推荐的做法

下面这些做法建议避免：

- 本地和线上同时写同一份 SQLite 数据库
- 用 git 同步数据库文件
- 让生产数据只存在 Docker 容器内部
- 在服务器直接改代码，再手工回填到本地

## 恢复思路

如果你要回滚某次部署，可以这样做：

1. 停掉生产容器
2. 把某个备份目录里的 `one-api.db` 复制回 `data-prod/one-api.db`
3. 如果需要，也把 `data-extra.tar.gz` 里的文件解回 `data-prod/`
4. 重新执行 `./deploy.sh --skip-backup`

## 说明

这套工作流是按你当前场景定的：

- 个人维护
- 一套主线上环境
- 先继续使用 SQLite
- 需要从线上单向拉快照到本地

如果后面你要升级成：

- 测试环境 + 正式环境双环境
- PostgreSQL
- 自动化 CI/CD

可以在这套骨架上继续扩展，不用推倒重来。

如果你现在就准备把生产库从 SQLite 切到 PostgreSQL，可以直接看：

- [postgres-migration.zh-CN.md](/E:/new-api/docs/installation/postgres-migration.zh-CN.md)
