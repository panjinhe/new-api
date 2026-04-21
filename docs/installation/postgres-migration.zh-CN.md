# new-api 从 SQLite 迁移到 PostgreSQL

这份文档是给你当前这套项目直接用的，目标是：

- 保留现在 SQLite 数据
- 切到 PostgreSQL 作为长期生产数据库
- 迁移过程可回滚

## 迁移前结论

推荐你按这个方式迁：

1. 先保留原来的 SQLite 文件不删
2. 启动 PostgreSQL 版生产容器
3. 让 new-api 先在 PostgreSQL 上自动建表
4. 再把 SQLite 数据导入 PostgreSQL
5. 验证没问题后，正式长期运行 PostgreSQL

## 新增的文件

这次项目里已经加了这些 PostgreSQL 相关文件：

- [docker-compose.prod.postgres.yml](/E:/new-api/docker-compose.prod.postgres.yml)
- [docker-compose.dev.postgres.yml](/E:/new-api/docker-compose.dev.postgres.yml)
- [migrate_sqlite_to_postgres.go](/E:/new-api/scripts/migrate_sqlite_to_postgres.go)

## 生产环境变量

先编辑服务器上的 `.env.prod`：

```env
DATABASE_BACKEND=postgres
POSTGRES_USER=newapi
POSTGRES_PASSWORD=换成强密码
POSTGRES_DB=newapi
SESSION_SECRET=换成随机字符串
CRYPTO_SECRET=换成随机字符串
FRONTEND_BASE_URL=https://your-domain.example.com
TRUSTED_REDIRECT_DOMAINS=your-domain.example.com
```

## 第一步：先备份现有 SQLite

如果你当前还在 SQLite 上跑，先做一份备份：

```bash
cd /opt/new-api/app
./backup.sh --env-name prod --db sqlite
```

这样即使迁移失败，也随时能回到 SQLite。

## 第二步：启动 PostgreSQL 版生产环境

现在 `deploy.sh` 和 `backup.sh` 已经支持按 `.env.prod` 里的 `DATABASE_BACKEND` 自动切换。

所以你只需要：

```bash
cd /opt/new-api/app
./deploy.sh
```

当 `DATABASE_BACKEND=postgres` 时，脚本会自动使用：

- `docker-compose.prod.yml`
- `docker-compose.prod.postgres.yml`

这一步会做两件事：

- 启动 PostgreSQL 容器
- 启动 new-api，并让它在 PostgreSQL 上自动建表

## 第三步：停止应用写入，准备导数据

为了避免导入时有新写入，建议先停掉应用容器，只保留 PostgreSQL：

```bash
cd /opt/new-api/app
docker compose -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml stop new-api
```

## 第四步：把 SQLite 数据导入 PostgreSQL

运行迁移工具：

```bash
cd /opt/new-api/app
go run ./scripts/migrate_sqlite_to_postgres.go \
  --sqlite ./data-prod/one-api.db \
  --postgres "postgresql://newapi:你的密码@127.0.0.1:5432/newapi?sslmode=disable"
```

注意：

- `--sqlite` 指向你原来的 SQLite 文件
- PostgreSQL DSN 要和 `.env.prod` 里的 `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` 一致

这个工具会：

- 按表读取 SQLite 数据
- 先清空 PostgreSQL 目标表
- 再把数据写进去
- 最后重置 PostgreSQL 自增序列

## 第五步：重启应用

导入完成后，重新拉起应用：

```bash
cd /opt/new-api/app
docker compose -f docker-compose.prod.yml -f docker-compose.prod.postgres.yml up -d new-api
```

或者直接再跑一次：

```bash
cd /opt/new-api/app
./deploy.sh
```

## 第六步：验证

建议你至少检查这些内容：

1. 登录后台是否正常
2. 渠道数量是否一致
3. 模型配置是否还在
4. 用户、令牌、订阅、兑换码是否都还在
5. 最近日志是否正常写入

本机健康检查：

```bash
curl http://127.0.0.1:3000/api/status
```

## PostgreSQL 备份

现在如果 `DATABASE_BACKEND=postgres`，你可以直接这样备份：

```bash
cd /opt/new-api/app
./backup.sh --env-name prod
```

脚本会自动导出：

- `postgres.dump`
- `.env.prod.backup`
- compose 文件副本
- 其他数据目录内容

## 回滚到 SQLite

如果迁移后发现有问题，回滚步骤是：

1. 把 `.env.prod` 里的 `DATABASE_BACKEND` 改回 `sqlite`
2. 保留原来的 `data-prod/one-api.db`
3. 重新部署

```bash
cd /opt/new-api/app
./deploy.sh
```

因为 SQLite 原文件没有删，所以回滚是很直接的。

## 本地先演练

如果你想先在本地演练 PostgreSQL，可以用：

```bash
cp .env.dev.example .env.dev
```

把 `.env.dev` 改成：

```env
DATABASE_BACKEND=postgres
POSTGRES_USER=newapi
POSTGRES_PASSWORD=change-me
POSTGRES_DB=newapi_dev
```

然后启动：

```bash
docker compose -f docker-compose.dev.yml -f docker-compose.dev.postgres.yml up -d --build
```

本地 PostgreSQL 默认映射到：

```text
127.0.0.1:5433
```

这样你也可以先在本地验证迁移工具。
