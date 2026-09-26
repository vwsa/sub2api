# 服务器操作手册

## 服务器

- SSH：`cherry@fshd.store`
- 公网地址：`https://sub2api.fshd.store`
- 部署目录：`/opt/sub2api`
- Docker Compose：`/opt/sub2api/docker-compose.yml`
- 数据目录：由线上 Compose 配置挂载；操作前不得删除数据卷或数据目录
- Docker 管理：当前用户需要使用 `sudo docker` / `sudo docker compose`

## 性能约束

服务器内存约 1 GiB，Swap 约 544 MiB。不要在服务器上执行完整前端构建：`vue-tsc` 可能耗尽内存并导致 SSH、Docker 服务响应变慢。

推荐流程：

1. 开发机顺序构建前端和后端，禁止同时启动多个 Go/Vite 构建。
2. Go 构建使用 `GOMAXPROCS=2 GOFLAGS=-p=1`，避免 16 GiB 开发机出现并行编译内存竞争。
3. 服务器不编译源码；只接收预编译二进制、运行时资源和入口脚本。
4. 有本地 Docker daemon 时，开发机生成镜像后使用 `docker save | gzip | ssh ... 'gunzip | sudo docker load'`。
5. 本地 Docker daemon 不可用时，在服务器基于当前运行镜像构建纯运行时替换层；远端 `docker build` 必须限制内存，且不能包含 Go/前端编译步骤。

## 当前低内存部署流程

### 1. 开发机构建

```bash
# 前端：构建产物写入 backend/internal/web/dist
cd frontend
pnpm install --frozen-lockfile
pnpm run build

# 后端：只开两个 Go 执行线程，包编译并行度固定为 1
cd ../backend
env GOMAXPROCS=2 GOFLAGS=-p=1 \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -tags embed \
  -ldflags='-s -w -X main.Version=<version> -X main.Commit=<commit> -X main.Date=<date> -X main.BuildType=release' \
  -trimpath -o /tmp/sub2api-<version>-<commit> ./cmd/server

/tmp/sub2api-<version>-<commit> --version
sha256sum /tmp/sub2api-<version>-<commit>
```

运行时发布包包含：

- 预编译的 `sub2api`；
- `backend/resources`；
- `deploy/docker-entrypoint.sh`。

### 2. 部署前备份

必须同时备份当前运行镜像和 Compose：

```bash
cd /opt/sub2api
sudo docker tag <current-image> <current-image>-backup-<date>
sudo cp docker-compose.yml docker-compose.yml.backup-<date>
```

### 3. 服务器构建纯运行时替换层

本地 Docker 不可用时，将运行时发布包上传到服务器临时目录，并使用以下 Dockerfile。基础镜像必须引用第 2 步创建的备份标签，保证运行时依赖、用户、入口和健康检查与当前线上一致。

```dockerfile
FROM <current-image>-backup-<date>
USER root
COPY --chown=sub2api:sub2api sub2api /app/sub2api
COPY --chown=sub2api:sub2api resources/ /app/resources/
COPY --chown=root:root docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod 0755 /app/sub2api /app/docker-entrypoint.sh
```

服务器只复制文件，不运行 `go build`、`pnpm` 或 `vue-tsc`：

```bash
sudo docker build \
  --memory=384m \
  --memory-swap=768m \
  -t sub2api:<release-tag> \
  /tmp/sub2api-release-<commit>

sudo docker run --rm sub2api:<release-tag> --version
sudo docker tag sub2api:<release-tag> sub2api:device-management-current
```

### 4. 切换应用容器

```bash
cd /opt/sub2api
# 仅修改 sub2api 服务的 image，禁止改写 .env、数据目录或数据库服务。
sudo docker compose config --quiet
sudo docker compose up -d --no-deps sub2api
sudo docker compose ps sub2api
```

确认发布稳定后，可以删除上传临时目录并清理 Builder 缓存；不得删除回滚镜像：

```bash
rm -rf /tmp/sub2api-release-<commit>
sudo docker builder prune -f
```

## 部署验证

```bash
sudo docker compose ps
sudo docker inspect sub2api --format '{{.State.Health.Status}}'
sudo docker logs --tail 100 sub2api
curl -fsS -I https://sub2api.fshd.store/
sudo docker exec sub2api /app/sub2api --version
```

健康状态必须为 `healthy`，公网首页应返回 HTTP `200`。应用内健康检查使用容器 Compose healthcheck；公网 `/health` 不一定是公开路由。

## 回滚

```bash
cd /opt/sub2api
sudo cp docker-compose.yml.backup-<date> docker-compose.yml
sudo docker compose config --quiet
sudo docker compose up -d --no-deps sub2api
sudo docker compose ps sub2api
```

## 注意事项

- 不要修改或删除 `/opt/sub2api/.env`；其中包含数据库密码、JWT 密钥和 TOTP 密钥。
- 不要执行 `docker compose down -v`。
- 不要删除应用数据、PostgreSQL 数据或 Redis 数据目录。
- 线上 Compose 当前使用 `sub2api:device-management-current`，该镜像由开发机构建后上传。
- 部署前必须保留旧镜像标签和 Compose 备份。
- 服务器上的 `/home/cherry/sub2api-src` 可能包含未提交的历史修改，不应直接当作发布源覆盖线上环境。
