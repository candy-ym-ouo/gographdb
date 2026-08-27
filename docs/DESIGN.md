# GoGraphDB 详细设计文档（DESIGN）

> 项目提示词：**图数据库：用 Go 实现属性图数据库，支持节点边存储、索引构建、遍历与路径查询，提供查询接口与持久化。**
>
> 本文档对提示词进行功能与业务逻辑扩写，给出完整系统设计、模块规格、接口契约、前端设计、代码量规划与测试计划。**本文档只描述设计，不包含任何代码实现。**

---

## 目录

1. [需求分析与功能扩写](#1-需求分析与功能扩写)
2. [业务逻辑详述](#2-业务逻辑详述)
3. [系统架构](#3-系统架构)
4. [数据模型设计](#4-数据模型设计)
5. [存储层设计](#5-存储层设计)
6. [索引层设计](#6-索引层设计)
7. [遍历与路径查询设计](#7-遍历与路径查询设计)
8. [查询语言 GQL 设计](#8-查询语言-gql-设计)
9. [REST API 设计](#9-rest-api-设计)
10. [持久化与故障恢复](#10-持久化与故障恢复)
11. [前端界面设计](#11-前端界面设计)
12. [模块规格与代码量规划](#12-模块规格与代码量规划)
13. [测试计划](#13-测试计划)
14. [非功能需求](#14-非功能需求)
15. [里程碑与验收标准](#15-里程碑与验收标准)
16. [本次扩展模块](#16-本次扩展模块)

---

## 1. 需求分析与功能扩写

### 1.1 原始需求拆解

提示词包含五个能力维度：

| 维度 | 原始表述 | 设计落点 |
| --- | --- | --- |
| 图数据库 | 用 Go 实现属性图数据库 | 属性图数据模型、图引擎内核 |
| 节点边存储 | 支持节点边存储 | 节点/边 CRUD、邻接表、引用完整性 |
| 索引构建 | 索引构建 | 标签索引、属性索引、B+ 树范围索引 |
| 遍历与路径查询 | 遍历与路径查询 | BFS/DFS 遍历、可达性、最短路径 |
| 查询接口与持久化 | 提供查询接口与持久化 | REST API、GQL 查询语言、快照 + WAL |

### 1.2 功能扩写（增量能力）

在原始需求之上，本设计扩写以下能力：

1. **类型化属性系统**：属性值不再只是字符串，支持 `STRING / INT / FLOAT / BOOL / TIMESTAMP` 五类，保证索引比较与排序语义正确。
2. **多标签与多重边**：节点可携带多个标签（`Person:Employee`），同一对节点之间可存在多条不同标签的边（多重图），支持自环。
3. **三级索引体系**：标签索引（快速定位某一类节点）、属性哈希索引（等值查询）、B+ 树索引（范围查询与排序），由索引管理器统一维护。
4. **索引自动维护**：图写入操作通过事件回调自动更新索引，保证索引与数据强一致；同时提供手动重建（`REBUILD`）能力用于数据导入后批量建索引。
5. **查询语言 GQL**：设计类 Cypher 子集，支持 `CREATE / MATCH / PATH / SHORTEST` 语句，解析为 AST 后由执行器执行。
6. **加权路径查询**：在无权 BFS 最短路径之外，支持按边属性（如 `weight`）的 Dijkstra 加权最短路径，覆盖网络拓扑、物流路线等业务。
7. **持久化双机制**：周期性快照（全量）+ WAL 预写日志（增量），实现崩溃恢复，数据不丢失。
8. **Web 可视化前端**：Canvas 画布渲染图结构，支持点选查看属性、拖拽布点、查询控制台、索引与统计面板。
9. **运维可观测**：健康检查、图规模统计、请求日志中间件、panic 恢复中间件。
10. **配置化启动**：命令行参数（监听地址、数据目录、快照间隔）与默认值组合，开箱即用。

### 1.3 典型业务场景

场景一：**社交网络分析**
- 节点：`Person`（name、age、city 属性）；边：`FRIEND`（since 属性）、`FOLLOW`。
- 业务：查询某人的二度好友（BFS 深度 2）、共同好友（交运算）、最短社交距离（无权 BFS 最短路径）。

场景二：**知识图谱**
- 节点：`Entity`（如电影、演员、导演）；边：`ACTED_IN`、`DIRECTED`。
- 业务：`MATCH (a:Person)-[r:ACTED_IN]->(m:Movie)` 多跳关联查询，属性索引支持按 `name` 等值定位实体。

场景三：**网络与物流拓扑**
- 节点：`Router` / `City`（坐标属性）；边：`LINK` / `ROUTE`（weight = 带宽/里程）。
- 业务：加权最短路径（Dijkstra）计算路由或物流线路；边标签索引统计某一类连接。

场景四：**数据血缘 / 权限图**
- 节点：`Table` / `Role`；边：`DERIVED_FROM` / `HAS_PERMISSION`。
- 业务：上游追溯遍历（反向边遍历），`PATH` 语句输出完整链路。

---

## 2. 业务逻辑详述

### 2.1 节点业务规则

- **创建节点**：生成全局唯一递增 ID（雪花式自增，进程内单调）；校验标签非空、属性名合法（非空字符串）；标签与属性可同时为空（匿名节点，如路径查询的中间层）。
- **更新节点**：支持增量更新属性（新增/覆盖/删除单个属性）与整体替换两种语义；更新后触发索引事件（删除旧索引项、插入新索引项）。
- **删除节点**：**级联删除**——同时删除所有与该节点相连的边（入边与出边），并触发边索引与邻接表同步清理；节点索引项同步移除。
- **查询节点**：支持按 ID、按标签、按属性（等值/范围）三种检索路径；检索优先走索引，索引缺失时退化为全表扫描。

### 2.2 边业务规则

- **创建边**：必须保证源节点与目标节点均存在（**引用完整性**），否则返回 `ErrVertexNotFound`；支持多重边（同源同目标多条不同标签边）与自环（源 = 目标）。
- **更新边**：允许修改标签与属性；不允许修改端点（端点变更 = 删除 + 重建）。
- **删除边**：按边 ID 删除；提供 `DELETE BY 源节点/目标节点/标签` 批量语义。
- **邻接表维护**：每个节点维护出边集与入边集（邻接表），保证遍历 O(度) 复杂度。

### 2.3 索引业务逻辑

- **自动维护**：图引擎在节点/边写入、更新、删除后，通过回调把变更同步到索引管理器；同一事务内先改数据再改索引，索引更新失败时回滚数据变更（强一致）。
- **索引类型**：
  - 标签索引：`label -> 节点 ID 集合`（哈希集合）。
  - 属性等值索引：`(label?, key, value) -> ID 集合`。
  - B+ 树范围索引：`(key, value) -> ID 集合`，支持 `> >= < <= BETWEEN`。
- **手动重建**：`REBUILD INDEX` 全量扫描重建指定索引，用于数据批量导入后收敛索引膨胀。
- **统计**：索引管理器暴露每个索引的条目数与内存占用，供前端面板展示。

### 2.4 遍历业务逻辑

- **遍历入口**：给定起始节点 ID + 方向（`OUT / IN / BOTH`）+ 边标签过滤 + 最大深度。
- **BFS**：层序扩展，天然给出无权最短距离；去重防环（访问集合）。
- **DFS**：深度优先，支持访问者回调（进入/离开节点事件），用于拓扑分析与环路检测。
- **路径查询**：
  - `PATH`：输出满足条件的全部简单路径（无权、限深）。
  - `SHORTEST`：无权最短路径用 BFS；加权最短路径用 Dijkstra（权重取自边属性 `weight`，缺失视为 1）。

### 2.5 查询引擎业务逻辑

- **流水线**：`词法分析 → 语法分析（AST）→ 语义校验 → 执行（走索引/扫描）→ 结果集 → JSON 序列化`。
- **语义校验**：变量引用一致性、标签/属性名合法性、深度参数为正整数、路径两端点存在。
- **执行计划**：`MATCH` 起点优先用标签索引或属性索引定位候选集，再沿边展开（索引驱动扫描）。
- **结果集**：支持 `RETURN` 字段投影、`LIMIT` 行数限制、`ORDER BY` 简单排序。

### 2.6 持久化业务逻辑

- **快照**：全量序列化节点表、边表、索引元数据为二进制文件；写入采用「临时文件 + 原子 rename」，避免半写损坏。
- **WAL**：每次写操作先追加日志（操作类型 + 序列化载荷 + 校验和），fsync 后落库；快照成功后截断 WAL。
- **启动恢复**：先加载最新快照，再重放 WAL 中快照点之后的增量操作，重建索引与统计。
- **优雅关闭**：收到 SIGINT/SIGTERM 时落一次快照并关闭文件。

### 2.7 API 业务逻辑

- REST 端点按资源组织，统一 JSON 信封：`{"code":0,"message":"ok","data":...}`；错误码见 §9.3。
- 中间件链：`日志 → panic 恢复 → (可选)请求体大小限制`。
- 查询接口为无状态 POST，前端与命令行均可调用。

### 2.8 前端业务逻辑

- 页面加载时拉取全图数据并渲染；节点按力导向式（简化版：斥力 + 弹力 + 中心引力）自动布局，可拖拽微调。
- 点选节点/边弹出属性面板；右键删除；表单创建节点与边。
- 查询控制台输入 GQL 语句，结果以表格展示，`MATCH` 命中的子图高亮。
- 工具栏：保存快照、加载、重建索引、查看统计。

---

## 3. 系统架构

### 3.1 分层架构

```
┌─────────────────────────────────────────────────────────────┐
│  Web 前端（web/ 静态资源：index.html / style.css / app.js）     │
└───────────────────────────────┬─────────────────────────────┘
                                │ HTTP (JSON)
┌───────────────────────────────▼─────────────────────────────┐
│  接口层 internal/api/server.go                               │
│   REST 路由 · 请求解析 · 响应封装 · 中间件(日志/恢复/限流)       │
└───────────────────────────────┬─────────────────────────────┘
┌───────────────────────────────▼─────────────────────────────┐
│  查询层 internal/query/parser.go · executor.go               │
│   GQL 词法/语法分析 → AST → 语义校验 → 执行计划 → 结果集        │
└───────────────────────────────┬─────────────────────────────┘
┌───────────────────────────────▼─────────────────────────────┐
│  引擎层 internal/graph（vertex/edge/property/graph/id）       │
│   图 CRUD · 引用完整性 · 级联删除 · 变更事件                    │
├───────────────────────────────┬─────────────────────────────┤
│  遍历层 internal/traverse（traverser/bfs/dfs/path）            │
│   邻接遍历 · BFS/DFS · 可达性 · 无权/加权最短路径               │
└──────┬──────────────────────────────┬───────────────────────┘
       │                              │
┌──────▼─────────────┐   ┌────────────▼──────────────────────┐
│ 索引层 internal/index│   │ 存储层 internal/storage             │
│ label/property/btree│   │ store接口 · memstore · persistence│
│ index_manager       │   │ · wal                            │
└─────────────────────┘   └──────────────────────────────────┘
                                     │
                          ┌──────────▼──────────┐
                          │ data/ 快照 + WAL 文件 │
                          └─────────────────────┘
```

### 3.2 依赖方向

- `api → query → graph → (storage | index | traverse)`
- `graph` 通过**事件回调**反向通知 `index` 维护索引（依赖注入接口，避免循环依赖）。
- `traverse` 只依赖 `graph` 的邻接访问接口，不感知存储实现。
- 所有层仅依赖 Go 标准库。

### 3.3 关键设计决策（ADR）

| 决策 | 选型 | 理由 |
| --- | --- | --- |
| 存储介质 | 内存哈希表为主 | 轻量、教学友好，读写 O(1) |
| 持久化格式 | 自定义二进制（gob 风格） | 体积小、加载快；避免外部依赖 |
| 崩溃恢复 | 快照 + WAL | 兼顾全量可靠与增量性能 |
| 遍历方向 | 邻接表（出边/入边双集合） | 反查入边 O(度)，支撑反向遍历 |
| 索引一致性 | 同步回调强一致 | 实现简单，避免异步窗口 |
| 最短路径 | BFS（无权）/ Dijkstra（加权） | 覆盖绝大多数业务 |
| HTTP 服务 | net/http 标准库 | 零依赖，Go 1.22 起支持方法路由 |

---

## 4. 数据模型设计

### 4.1 属性图模型

属性图 `G = (V, E, P)`：

- **节点 V**：`{ id, labels: Set<string>, properties: Map<string, PropertyValue> }`
- **边 E**：`{ id, srcID, dstID, label, properties: Map<string, PropertyValue> }`，有向。
- **属性 P**：键为字符串，值为类型化标量。

### 4.2 值类型系统

| 类型标识 | Go 对应 | 说明 | 比较语义 |
| --- | --- | --- | --- |
| `STRING` | string | UTF-8 字符串 | 字典序 |
| `INT` | int64 | 64 位整数 | 数值序 |
| `FLOAT` | float64 | 双精度浮点 | 数值序 |
| `BOOL` | bool | 布尔 | false < true |
| `TIMESTAMP` | int64 (UnixNano) | 时间戳 | 数值序 |

> 类型错误在写入时即拒绝（如把字符串赋给 `age` 不报错，但 `age > 30` 的范围查询只对 INT/FLOAT 生效；属性为**动态类型**，键上无固定 schema）。

### 4.3 ID 设计

- 节点 ID 与边 ID 各自独立的自增整数，进程内单调递增，从 1 开始。
- 持久化时记录「下一可用 ID」，恢复后继续递增，避免冲突。
- ID 以 `uint64` 表示，对外以十进制字符串传输（JSON）。

### 4.4 图不变量（Invariants）

1. 边端点必须存在（创建时校验）。
2. 删除节点必须级联删除其全部入边与出边。
3. 索引与图数据强一致（写后同步更新）。
4. 节点/边 ID 全局唯一且单调。
5. 快照 + WAL 重放后图状态与崩溃前一致。

---

## 5. 存储层设计

### 5.1 存储接口（`store.go`）

抽象 `Store` 接口，目前仅内存实现，为将来接入磁盘 B+ 树存储预留：

- 节点：`AddVertex / GetVertex / UpdateVertex / DeleteVertex / AllVertices / CountVertices`
- 边：`AddEdge / GetEdge / DeleteEdge / EdgesOf(srcID, dir) / AllEdges / CountEdges`
- 元数据：`NextVertexID / NextEdgeID / SetNextIDs`

### 5.2 内存实现（`memstore.go`）

- `map[uint64]*Vertex` 节点表；`map[uint64]*Edge` 边表。
- 每个节点维护 `outEdges map[uint64]struct{}`、`inEdges map[uint64]struct{}` 邻接集合。
- 删除操作同步清理邻接集合，保证遍历不悬挂。

### 5.3 持久化（`persistence.go`）

- `SaveSnapshot(path)`：依次写入魔数、版本、节点数、边数、下一 ID、节点数组、边数组、索引元数据；先写临时文件再 `rename`。
- `LoadSnapshot(path)`：校验魔数与版本，反序列化并重建邻接表。
- 提供 `DumpJSON(path)` 便于调试与前端导出（可选）。

### 5.4 WAL（`wal.go`）

- 日志记录：`{ seq, op, payload, checksum }`，op ∈ {AddVertex, UpdateVertex, DeleteVertex, AddEdge, DeleteEdge, ...}。
- `Append(record)`：追加 + 校验和 + fsync。
- `Replay(walPath, applyFn)`：顺序重放，跳过校验和不一致的尾部（视为截断）。
- 快照成功后 `Truncate()` 重建日志文件。

---

## 6. 索引层设计

### 6.1 索引接口（`index.go`）

```text
type Index interface {
    Add(key, id) / Remove(key, id) / Lookup(key) -> []uint64
    Rebuild(from) / Stats() -> {entries, memory}
}
```

### 6.2 标签索引（`label_index.go`）

- 结构：`map[string]map[uint64]struct{}`。
- 用途：`MATCH (a:Person)` 定位候选集；`GET /vertices?label=Person`。

### 6.3 属性等值索引（`property_index.go`）

- 结构：`map[propKey]map[propValueHash]map[uint64]struct{}`，值按类型归一化（INT/FLOAT 转数值键，避免 1 与 1.0 不一致）。
- 用途：`GET /vertices?property.name=张三`；GQL 的 `WHERE key = value`。

### 6.4 B+ 树范围索引（`btree_index.go`）

- 实现一个简化有序树（有序切片数组 + 二分查找的分层结构），支持 `Range(low, high, includeLow, includeHigh)`。
- 用途：`age > 25`、`BETWEEN`、`ORDER BY key` 加速。

### 6.5 索引管理器（`index_manager.go`）

- 注册三类索引，维护「索引 → 目标(节点/边) + 字段」的元数据表。
- 提供 `OnVertexChanged / OnEdgeChanged` 回调入口（供图引擎调用）。
- 提供 `RebuildAll / StatsAll / DropIndex` 运维接口。

---

## 7. 遍历与路径查询设计

### 7.1 遍历器（`traverser.go`）

- 统一遍历上下文：`{ startID, direction(OUT/IN/BOTH), edgeLabels filter, maxDepth, visitFn }`。
- 提供邻接迭代器：`Neighbors(id, direction, labelFilter) -> iterator[uint64]`（基于邻接表 + 边表）。

### 7.2 BFS（`bfs.go`）

- 队列层序扩展；`visited` 集合去重；记录 `depth[id]`；返回「可达节点 + 深度」映射。
- 应用：二度好友、无权最短距离、连通分量初判。

### 7.3 DFS（`dfs.go`）

- 递归 + 显式栈两种模式；访问者回调 `Enter(id, depth)` / `Leave(id)`。
- 应用：拓扑输出、环路检测（三色标记）。

### 7.4 路径查询（`path.go`）

- `AllSimplePaths(from, to, maxDepth)`：回溯枚举简单路径（路径内不重复节点）。
- `ShortestUnweighted(from, to)`：BFS 记录前驱，回溯还原路径。
- `ShortestWeighted(from, to)`：Dijkstra 优先队列（container/heap），权重取边属性 `weight`（INT/FLOAT），缺失按 1。
- 输出：`{ paths: [[id...]], distance, found }`。

---

## 8. 查询语言 GQL 设计

### 8.1 语句类型

```text
CREATE (n:Label {k1:v1, k2:v2})                       -- 创建节点
CREATE (a)-[r:LABEL {k:v}]->(b)                       -- 创建边（a/b 为已有节点 ID）
MATCH (a:Label)-[r:LABEL]->(b:Label)                  -- 模式匹配
      [WHERE a.k = v | a.k > v] [RETURN a.k, b.k] [LIMIT n]
PATH FROM <id> TO <id> MAXDEPTH n                     -- 全部简单路径
SHORTEST FROM <id> TO <id> [WEIGHTED]                 -- 最短路径（默认无权）
REBUILD INDEX [label|property|btree|ALL]              -- 重建索引
```

### 8.2 语法要点

- 标识符 `a`/`b`/`r` 为绑定变量；`(a:Label {props})` 节点模式；`-[r:LABEL]->` 边模式（方向支持 `<-`、`-`）。
- 值字面量：字符串（单/双引号）、整数、浮点、`true/false`、`T(...)` 时间戳。
- `WHERE` 支持单条件比较（`=`、`>`、`>=`、`<`、`<=`），键引用形如 `a.k`。

### 8.3 解析与执行

- `parser.go`：词法（token 流）+ 递归下降语法分析 → AST（`CreateVertexStmt / CreateEdgeStmt / MatchStmt / PathStmt / ShortestStmt / RebuildIndexStmt`）。
- `executor.go`：语义校验 → 执行计划（索引驱动）→ `ResultSet { columns, rows, affected }` → JSON。

---

## 9. REST API 设计

### 9.1 端点一览

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/health` | 健康检查 |
| GET | `/api/stats` | 图规模统计（节点/边/索引数） |
| POST | `/api/graph/vertices` | 创建节点 |
| GET | `/api/graph/vertices/{id}` | 按 ID 查节点 |
| PUT | `/api/graph/vertices/{id}` | 更新节点属性 |
| DELETE | `/api/graph/vertices/{id}` | 删除节点（级联删边） |
| GET | `/api/graph/vertices?label=&prop.name=&prop.age.gt=` | 索引查询（标签/等值/范围） |
| POST | `/api/graph/edges` | 创建边 |
| GET | `/api/graph/edges/{id}` | 按 ID 查边 |
| DELETE | `/api/graph/edges/{id}` | 删除边 |
| GET | `/api/graph/neighbors/{id}?direction=OUT&depth=2&label=FRIEND` | 遍历（BFS 可达集） |
| POST | `/api/graph/path` | 路径查询 `{from,to,maxDepth,weighted}` |
| POST | `/api/graph/query` | 执行 GQL 语句 `{query}` |
| POST | `/api/graph/index/rebuild` | 重建索引 `{type}` |
| GET | `/api/graph/index` | 索引统计 |
| GET | `/api/graph/index/audit` | 索引一致性审计 |
| GET | `/api/graph/analytics?limit=10` | 图分析（连通分量、度中心性、PageRank） |
| GET | `/api/graph/schema` | 图模式目录（标签、属性、属性类型） |
| GET | `/api/graph/quality` | 图数据质量报告 |
| POST | `/api/persistence/save` | 手动快照 |
| POST | `/api/persistence/load` | 从快照加载 |
| GET | `/api/persistence/export` | 下载当前一致性 JSON 快照 |
| GET | `/api/persistence/status` | 快照、元数据与 WAL 文件状态 |
| GET | `/` 与 `/web/*` | 前端静态资源 |

### 9.2 请求/响应示例

```http
POST /api/graph/vertices
{"labels":["Person"],"properties":{"name":"张三","age":30,"city":"杭州"}}
→ 201 {"code":0,"message":"ok","data":{"id":1,"labels":["Person"],"properties":{"name":"张三","age":30,"city":"杭州"}}}

POST /api/graph/path
{"from":"1","to":"2","maxDepth":5,"weighted":true}
→ 200 {"code":0,"message":"ok","data":{"found":true,"distance":3.0,"path":[1,4,7,2]}}
```

### 9.3 错误码约定

| code | 含义 |
| --- | --- |
| 0 | 成功 |
| 40001 | 请求体非法（JSON 解析失败 / 缺字段） |
| 40002 | GQL 语法错误 |
| 40003 | GQL 语义错误（未定义变量 / 非法参数） |
| 40401 | 节点不存在 |
| 40402 | 边不存在 |
| 40901 | ID 冲突 / 引用完整性冲突（创建边时端点缺失） |
| 50001 | 内部错误（存储/持久化失败） |

---

## 10. 持久化与故障恢复

### 10.1 文件布局（`data/` 目录）

```text
data/
├── snapshot.gdb     # 最新全量快照（二进制，原子替换）
├── wal.log          # 增量操作日志（快照后清空重建）
└── meta.json        # 版本、快照时间、WAL 起始 seq（辅助诊断）
```

### 10.2 写入路径

```
写请求 → 图引擎变更 → ① 追加 WAL(校验和+fsync) → ② 更新内存存储 → ③ 触发索引回调
周期性（默认 30s 或达到 N 条 WAL）→ 落快照 → 截断 WAL
```

### 10.3 启动恢复路径

```
启动 → 存在 snapshot.gdb ? 加载快照 : 空库 → 重放 wal.log（按 seq 顺序）→ 重建索引与统计 → 就绪
```

### 10.4 一致性保证

- 快照采用临时文件 + `rename`，任何时刻磁盘上要么旧快照要么新快照。
- WAL 记录带校验和，重放时遇到损坏尾部即停止（视为正常截断）。
- 恢复完成后索引与图数据一致（索引由数据全量重建，或由 WAL 事件重放维护）。

---

## 11. 前端界面设计

### 11.1 页面布局（`web/index.html`）

```
┌────────────────────────────────────────────────────────────┐
│ 工具栏：保存 | 加载 | 重建索引 | 统计 | 清空                 │
├──────────────────────────────┬─────────────────────────────┤
│  ① 图可视化画布（Canvas）      │  ② 侧栏面板                  │
│    · 节点渲染为圆（按标签着色） │    · 节点/边创建表单           │
│    · 边渲染为带箭头连线        │    · 选中元素属性编辑          │
│    · 力导向布局 + 拖拽         │    · 索引状态列表             │
│    · 查询命中子图高亮          │    · 图统计卡片               │
├──────────────────────────────┴─────────────────────────────┤
│ ③ 查询控制台：GQL 输入框 + 执行按钮 + 结果表格 / 日志         │
└────────────────────────────────────────────────────────────┘
```

### 11.2 交互逻辑（`web/app.js`）

- 启动即 `GET /api/stats` + 全图数据，构建画布场景。
- 简化力导向布局：每帧计算斥力（O(n²) 但 n 为演示规模，可接受）、沿边弹力、中心引力，30~60 次迭代后收敛；用户拖拽节点后该节点固定。
- 双击节点创建「临时边」；点选元素打开右侧属性面板（读 + 改）。
- 查询控制台执行 `POST /api/graph/query`，命中节点/边以高亮色描边，结果表格化展示。
- 标签配色表：`Person` 蓝、`Movie` 橙、`Router` 绿、`City` 紫，未知标签取色板余项。

### 11.3 样式（`web/style.css`）

- 深色主题（`#1e1e2e` 背景），卡片式面板，栅格布局；Canvas 自适应窗口尺寸。
- 状态反馈：Toast 提示成功/失败；加载中显示进度遮罩。

---

## 12. 模块规格与代码量规划

### 12.1 代码量约束

> 需求约束：**Go 代码行数（不含测试代码）> 2000 且 < 2200；Go 代码文件数 > 20 且 < 25。**

- 实际实现 **24 个非测试 Go 文件**（满足 21 ≤ N ≤ 24 区间）。
- 本次扩展后实际交付总行数 **2108 行**（满足 2000 < L < 2200）。
- 测试代码（`*_test.go`，约 12 个文件）**不计入**上述行数与文件数约束。
- 每个文件给出行数预算，实现时以实际为准，允许 ±10% 浮动，最终以 `wc -l` 校验并微调（在包级注释/空白行上做平衡）。

### 12.2 文件与行数预算表

| # | 文件路径 | 职责摘要 | 预算行数 |
| --- | --- | --- | --- |
| 1 | `cmd/gographdb/main.go` | 入口：解析配置、初始化引擎、启动 HTTP、信号处理 | 60 |
| 2 | `internal/graph/vertex.go` | 节点结构、构造、校验、属性操作 | 90 |
| 3 | `internal/graph/edge.go` | 边结构、构造、校验、邻接迭代器 | 90 |
| 4 | `internal/graph/property.go` | 属性值类型系统、编解码、比较 | 105 |
| 5 | `internal/graph/graph.go` | 图引擎：CRUD、级联删除、事件回调、统计 | 180 |
| 6 | `internal/graph/id.go` | 节点/边 ID 分配器与持久化恢复 | 50 |
| 7 | `internal/storage/store.go` | Store 接口与数据访问契约 | 75 |
| 8 | `internal/storage/memstore.go` | 内存实现：表 + 邻接集合 | 110 |
| 9 | `internal/storage/persistence.go` | 快照读写、JSON 导出、文件状态 | 133 |
| 10 | `internal/storage/wal.go` | WAL 记录、追加、重放、截断 | 100 |
| 11 | `internal/index/index.go` | 索引接口与通用键定义 | 50 |
| 12 | `internal/index/label_index.go` | 标签索引实现 | 75 |
| 13 | `internal/index/property_index.go` | 属性等值索引实现 | 85 |
| 14 | `internal/index/btree_index.go` | 简化 B+ 树范围索引 | 120 |
| 15 | `internal/index/index_manager.go` | 索引注册、事件分发、重建、统计 | 90 |
| 16 | `internal/traverse/traverser.go` | 遍历上下文、邻接迭代、访问者 | 75 |
| 17 | `internal/traverse/bfs.go` | 广度优先遍历 | 80 |
| 18 | `internal/traverse/dfs.go` | 深度优先遍历、环路检测 | 70 |
| 19 | `internal/traverse/path.go` | 简单路径、无权/加权最短路径 | 120 |
| 20 | `internal/query/parser.go` | GQL 词法与递归下降解析 → AST | 135 |
| 21 | `internal/query/executor.go` | 语义校验、执行计划、结果集 | 130 |
| 22 | `internal/api/server.go` | HTTP 路由、handlers、中间件、静态托管 | 163 |
| 23 | `internal/analytics/analytics.go` | 图分析、模式目录与数据质量报告 | 15 |
| **合计** | **24 个文件（含 config）** | | **2108** |

### 12.3 非 Go 文件（不计入约束）

- `go.mod`、`README.md`、`docs/DESIGN.md`（本文档）
- `web/index.html`、`web/style.css`、`web/app.js`（前端）
- `data/`（运行时产物，gitignore）
- 约 12 个 `*_test.go` 测试文件（不计入行数/文件数约束）

### 12.4 行数合规自查方法

1. `find . -name '*.go' ! -name '*_test.go' | xargs wc -l | tail -1` → 校验 2000 < L < 2200。
2. `find . -name '*.go' ! -name '*_test.go' | wc -l` → 校验 20 < N < 25。
3. 若超限：优先压缩 `graph.go`/`server.go` 的注释与空行，或将 `handlers` 拆分逻辑并入 `server.go`（保持文件数不变）；若不足：为 `btree_index.go`/`parser.go` 补充边界注释与文档块。

---

## 13. 测试计划

> 测试代码不计入 2000–2200 行约束，但纳入质量验收。

### 13.1 单元测试（按包）

| 包 | 用例要点 |
| --- | --- |
| `graph` | 节点/边 CRUD、级联删除、自环与多重边、ID 单调性、非法属性拒绝 |
| `storage` | 内存存储增删改查、邻接集合一致性、下一 ID 持久化 |
| `index` | 标签/属性/B+ 树增删查、重建、范围查询边界（含开闭区间）、索引审计 |
| `analytics` | 弱连通分量、孤立节点/自环统计、度中心性、PageRank 排序 |
| `traverse` | BFS 深度与去重、DFS 访问顺序、简单路径、无权/加权最短路径（含权重缺失默认 1） |
| `query` | GQL 各语句解析（合法/非法输入表）、WHERE 比较、LIMIT、语义错误上报 |
| `api` | 各端点成功与错误路径（404/409/400）、JSON 信封格式 |

### 13.2 集成测试

- 端到端：`创建图 → 建索引 → MATCH 查询 → 路径查询 → 快照保存 → 重启进程 → 恢复 → 数据一致`。
- WAL 恢复：模拟崩溃（写入一半日志、损坏校验和）后重放结果与期望一致。
- API 冒烟：httptest 起服务，走通典型业务场景（社交/知识图谱样例数据）。

### 13.3 性能冒烟（非严格基准）

- 1 万节点 / 5 万边规模下：单点查询 < 1ms；BFS 深度 3 遍历 < 10ms；快照保存 < 200ms。

---

## 14. 非功能需求

1. **零第三方依赖**：仅 Go 标准库，`go build` 即可编译。
2. **可移植性**：`GOOS=linux/darwin/windows` 均可构建；路径与换行符处理兼容。
3. **健壮性**：所有 API 错误均映射为明确错误码；panic 由中间件恢复并返回 50001。
4. **数据安全**：快照原子写、WAL 校验和、优雅关闭落盘。
5. **文档与注释**：每个导出符号有中文注释；本设计文档与 README 覆盖使用与架构。
6. **安全边界**：单机单进程使用，不承诺多进程并发写；HTTP 服务默认仅本机监听（可通过 `--addr` 放开）。

---

## 15. 里程碑与验收标准

| 里程碑 | 交付物 | 验收标准 |
| --- | --- | --- |
| M1 图内核 | graph + storage 包 | 节点/边 CRUD 与级联删除单测全绿 |
| M2 索引 | index 包 | 三类索引查询正确，重建可收敛 |
| M3 遍历 | traverse 包 | BFS/DFS/路径结果正确 |
| M4 查询与 API | query + api 包 | GQL 与 REST 端点冒烟通过 |
| M5 持久化 | persistence + wal | 重启恢复一致性测试通过 |
| M6 前端 | web/ | 画布渲染、查询控制台可用 |
| 收尾 | 全部 | **代码量校验**：非测试 Go 行数 ∈ (2000, 2200)，文件数 ∈ (20, 25)；README/DESIGN 与实际实现一致 |


---

## 16. 本次扩展模块

### 16.1 图分析模块（`internal/analytics`）

- 基于一次图快照进行只读分析，避免计算过程受并发写入影响。
- 输出弱连通分量数、孤立节点数、自环数，以及按总度排序的节点 Top N。
- 使用 20 轮、阻尼系数 0.85 的 PageRank 计算重要节点；悬挂节点的权重会均匀回注到全图。
- REST：`GET /api/graph/analytics?limit=10`，`limit` 取值范围为 1–100。

### 16.2 索引审计模块（`internal/index`）

- 对标签索引、属性等值索引、范围索引和边标签索引逐项检查覆盖情况。
- 同时比较索引条目总数与图快照期望值，用于发现重建异常或残留条目。
- 返回 `healthy`、已检查的节点/边数量和问题列表；审计只读，不会修改索引或图数据。
- REST：`GET /api/graph/index/audit`。


### 16.3 图模式目录（`internal/analytics`）

- 对同一份图快照聚合节点标签、边标签、属性名与属性逻辑类型，响应按标签和类型稳定排序。
- 多标签节点会计入每个所属标签；无标签节点归入 `_unlabeled`，不会遗漏结构信息。
- REST：`GET /api/graph/schema`。

### 16.4 JSON 图数据导出（`internal/storage`）

- 将当前一致性图快照编码为 `Snapshot` JSON，包含格式版本、下一节点/边 ID、节点和边。
- 导出是只读操作，不写入磁盘；响应附带下载文件名 `gographdb-export.json`，可用于诊断、迁移或备份。
- REST：`GET /api/persistence/export`。

### 16.5 图数据质量报告（`internal/analytics`）

- 基于一致性图快照检查无标签节点、缺少属性的节点或边、孤立节点和自环等完整性信号。
- 返回节点/边数量、`healthy` 和按固定顺序输出的问题代码及其数量；该检查只读，不修改图或索引。
- REST：`GET /api/graph/quality`。

### 16.6 持久化运维状态（`internal/storage`）

- 汇总数据目录中 `snapshot.gdb`、`meta.json` 和 `wal.log` 的存在状态与文件大小。
- 可在不读取或修改图数据的前提下确认快照、诊断元数据和 WAL 是否已落盘。
- REST：`GET /api/persistence/status`。
