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
