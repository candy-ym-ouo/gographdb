基于 Go 实现的属性图数据库 Web 项目，一款包含 REST API 与 Canvas 可视化界面的轻量图数据服务，支持节点和边管理、路径查询、索引、图分析与快照持久化。

# GoGraphDB 评测说明

## 环境要求

- Go 1.22+
- Docker Desktop（用于构建评测镜像）

项目仅使用 Go 标准库；`web/` 是无需 Node.js 构建的静态前端资源。

## 构建、测试与运行

```bash
# 编译全部 Go 包
go build ./...

# 执行测试与静态检查
go test ./...
go vet ./...

# 或使用项目命令
make test
make build

# 启动服务
./bin/gographdb -addr 127.0.0.1:8080 -data ./data -snapshot-interval 0
```

启动后访问 `http://127.0.0.1:8080`，健康检查接口为 `GET /api/health`。

## Docker 多架构验证

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh gographdb-benzhi-arm64 linux/arm64
docker run --rm gographdb-benzhi-arm64:latest go build ./...

./build_benzhi_docker.sh gographdb-benzhi-amd64 linux/amd64
docker run --rm gographdb-benzhi-amd64:latest go build ./...
```
