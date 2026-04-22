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

## 生产镜像传输约定

- 生产部署时，如果服务器拉取 Docker Hub 镜像慢或超时，统一改为在本地构建镜像后传到服务器。
- 镜像传输必须使用压缩流，不再直接使用未压缩的 `docker save | ssh ... docker load`。
- 推荐命令：

```bash
docker build -t new-api-local:prod .
docker save new-api-local:prod | gzip -1 | ssh -F ops/ssh/config.local aheapi-prod "gunzip | docker load"
ssh -F ops/ssh/config.local aheapi-prod "cd /opt/new-api/app && docker compose -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml up -d --no-build"
```

- 如果需要验证镜像体积，可先查看：

```bash
docker images new-api-local:prod
docker save new-api-local:prod | gzip -1 | wc -c
```

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
set -a
. ./.env.prod
set +a
docker compose -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml up -d --no-build
```

- 如果只想重建应用容器、不要动数据库容器，使用：

```bash
cd /opt/new-api/app
set -a
. ./.env.prod
set +a
docker compose -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml up -d --no-build --no-deps --force-recreate new-api
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
set -a
. ./.env.prod
set +a
docker compose -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml \
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
