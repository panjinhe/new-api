# new-api 自动备份定时器

如果你已经在服务器上用 Docker 跑生产环境，建议把备份改成 systemd 定时任务，而不是只靠手工执行。

## 默认方案

下面这条命令会：

- 在 `/etc/systemd/system/` 下安装 `new-api-backup.service`
- 在 `/etc/systemd/system/` 下安装 `new-api-backup.timer`
- 设置每天凌晨自动备份
- 自动清理 14 天以前的旧备份

```bash
cd /opt/new-api/app
sudo ./scripts/install-prod-backup-timer.sh --app-dir /opt/new-api/app --env-name prod --retention-days 14 --on-calendar "*-*-* 04:20:00"
```

## 可调参数

- `--app-dir`：项目目录，默认 `/opt/new-api/app`
- `--env-name`：环境名，默认 `prod`
- `--retention-days`：备份保留天数，`0` 表示不自动清理
- `--on-calendar`：systemd 时间表达式，例如 `*-*-* 04:20:00`
- `--service-name`：定时任务名称，默认 `new-api-backup`

## 查看状态

```bash
systemctl status new-api-backup.timer --no-pager
systemctl list-timers --all --no-pager | grep new-api-backup
```

## 手工执行一次

```bash
cd /opt/new-api/app
./backup.sh --env-name prod --retention-days 14
```

## 卸载

```bash
sudo systemctl disable --now new-api-backup.timer
sudo rm -f /etc/systemd/system/new-api-backup.timer
sudo rm -f /etc/systemd/system/new-api-backup.service
sudo systemctl daemon-reload
```
