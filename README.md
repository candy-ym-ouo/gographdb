# GoGraphDB

GoGraphDB 是一个仅使用 Go 标准库实现的轻量属性图数据库，支持节点/边 CRUD、标签与属性索引、BFS/DFS、简单路径、无权/加权最短路径、图分析、模式目录、索引审计、GQL 子集、REST API、快照 + WAL 恢复，以及 Canvas Web 可视化界面。

## 环境要求

- Go 1.22+
- 无第三方依赖

## 快速启动

```bash
make test
make build
./bin/gographdb --addr 127.0.0.1:8080 --data ./data
```

浏览器访问 `http://127.0.0.1:8080`。服务退出时会保存快照；默认每 30 秒自动保存，可通过 `--snapshot-interval 0` 关闭。

## 常用命令

```bash
make fmt       # 标准格式化（会改变设计文档要求的 wc 行数，不用于交付统计）
make test      # 单元测试与集成测试
make vet       # 静态检查
make build     # 构建当前平台二进制
make package   # 生成 dist/gographdb-<os>-<arch>.tar.gz
make clean
```

## REST 示例

```bash
curl -X POST http://127.0.0.1:8080/api/graph/vertices \
  -H 'Content-Type: application/json' \
  -d '{"labels":["Person"],"properties":{"name":"张三","age":30}}'

curl -X POST http://127.0.0.1:8080/api/graph/query \
  -H 'Content-Type: application/json' \
  -d '{"query":"MATCH (n:Person) WHERE n.age >= 18 RETURN n.id, n.name LIMIT 20"}'
```

主要端点：

- `GET /api/health`、`GET /api/stats`、`GET /api/graph`
- `/api/graph/vertices`、`/api/graph/edges`：图 CRUD
- `GET /api/graph/neighbors/{id}`：BFS 邻接遍历
- `POST /api/graph/path`：路径查询
- `POST /api/graph/query`：执行 GQL
- `/api/graph/index`：索引统计与重建
- `GET /api/graph/index/audit`：索引与图数据一致性审计
- `GET /api/graph/analytics?limit=10`：弱连通分量、度中心性、PageRank 图分析
- `GET /api/graph/schema`：节点/边标签、属性与类型的模式目录
- `GET /api/graph/quality`：标签、属性、孤立节点与自环的数据质量报告
- `/api/persistence/save`、`/api/persistence/load`：快照操作
- `GET /api/persistence/export`：下载当前一致性 JSON 图快照
- `GET /api/persistence/status`：快照、元数据与 WAL 文件的运维状态

## GQL 子集

```text
CREATE (n:Person {name:"张三", age:30})
CREATE (1)-[r:FRIEND {weight:1}]->(2)
MATCH (a:Person)-[r:FRIEND]->(b:Person) WHERE a.age >= 18 RETURN a.name, b.name LIMIT 20
PATH FROM 1 TO 5 MAXDEPTH 6
SHORTEST FROM 1 TO 5
SHORTEST FROM 1 TO 5 WEIGHTED
REBUILD INDEX ALL
```

## 持久化文件

运行后在数据目录生成：

- `snapshot.gdb`：原子替换的 gob 二进制快照
- `wal.log`：带序号和 CRC32 的 JSON 行预写日志
- `meta.json`：快照时间及规模信息

## 项目结构

```text
cmd/gographdb/       服务入口
internal/graph/      数据模型与图引擎
internal/storage/    内存存储、快照、WAL
internal/index/      标签、等值、范围索引
internal/traverse/   BFS、DFS、路径算法
internal/analytics/  图分析：组件、度中心性、PageRank
internal/query/      GQL 解析与执行
internal/api/        REST API 与中间件
web/                 Canvas 前端
```

详细规格见 `docs/DESIGN.md`。


## 扩展模块

### 图分析

```bash
curl 'http://127.0.0.1:8080/api/graph/analytics?limit=10'
```

返回节点/边规模、弱连通分量、孤立节点、自环数量，以及按度和 PageRank 排序的前 N 个节点。

### 索引审计

```bash
curl http://127.0.0.1:8080/api/graph/index/audit
```

返回 `healthy: true` 表示标签、属性、范围和边标签索引均覆盖当前图数据；否则 `problems` 会列出缺失或条目总数不一致的问题。

### 图模式目录

```bash
curl http://127.0.0.1:8080/api/graph/schema
```

按标签汇总节点和边的数量，并列出每个属性出现过的逻辑类型；无标签节点归入 `_unlabeled`。

### JSON 图数据导出

```bash
curl -OJ http://127.0.0.1:8080/api/persistence/export
```

下载当前一致性快照，格式与 `DumpJSON` 导出的 `Snapshot` JSON 相同，包含版本、下一节点/边 ID、节点和边数据。

### 图数据质量报告

```bash
curl http://127.0.0.1:8080/api/graph/quality
```

基于同一份图快照检查无标签节点、没有属性的节点或边、孤立节点和自环；`healthy` 为 `true` 时表示未发现上述完整性信号。

### 持久化运维状态

```bash
curl http://127.0.0.1:8080/api/persistence/status
```

返回数据目录以及 `snapshot.gdb`、`meta.json`、`wal.log` 是否存在和各自文件大小，便于确认快照与日志的落盘状态。
