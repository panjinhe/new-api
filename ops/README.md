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
