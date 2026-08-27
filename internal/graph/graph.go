package graph;import("errors";
"fmt";"sort";"sync";);var(ErrVertexNotFound=errors.New("vertex not found");
ErrEdgeNotFound=errors.New("edge not found");
ErrConflict=errors.New("graph conflict");
);
/* Store 描述图引擎需要的存储能力。 */ type Store interface{
AddVertex(*Vertex)error;GetVertex(uint64)(*Vertex,
bool);UpdateVertex(*Vertex)error;
DeleteVertex(uint64)error;
AllVertices()[]*Vertex;AddEdge(*Edge)error;
GetEdge(uint64)(*Edge,bool);
DeleteEdge(uint64)error;AllEdges()[]*Edge;
EdgesOf(uint64,Direction)[]*Edge;
Counts()(int,int);NextIDs()(uint64,
uint64);SetNextIDs(uint64,uint64);
Reset([]*Vertex,[]*Edge,uint64,
uint64)error;};
/* ChangeListener 接收同步索引更新通知。 */ type ChangeListener interface{
VertexChanged(old,current*Vertex);
EdgeChanged(old,current*Edge);};
/* WALWriter 记录变更，具体格式由存储层实现。 */ type WALWriter interface{
Log(op string,payload any)error;};

/* Graph 是并发安全的属性图服务。 */ type Graph struct{
mu sync.RWMutex;store Store;
vertices*IDAllocator;edges*IDAllocator;
listeners[]ChangeListener;wal WALWriter;
}; /* New 创建图引擎。 */ func New(store Store)*Graph{
v,e:=store.NextIDs();return&Graph{
store:store,vertices:NewIDAllocator(v),
edges:NewIDAllocator(e)};};
/* SetWAL 设置预写日志；恢复阶段可传 nil 暂停日志。 */ func(g*Graph)SetWAL(w WALWriter){
g.mu.Lock();g.wal=w;g.mu.Unlock()};

/* AddListener 注册索引等同步监听器。 */ func(g*Graph)AddListener(listener ChangeListener){
g.mu.Lock();defer g.mu.Unlock();g.listeners=append(g.listeners,
listener);};
/* CreateVertex 创建节点。 */ func(g*Graph)CreateVertex(labels[]string,
properties map[string]PropertyValue)(*Vertex,
error){g.mu.Lock();defer g.mu.Unlock();
v,err:=NewVertex(g.vertices.Next(),
labels,properties);if err!=nil{
return nil,err;};if err=g.log("add_vertex",
v);err!=nil{return nil,err;};if err=g.store.AddVertex(v);
err!=nil{return nil,err;};g.store.SetNextIDs(g.vertices.Peek(),
g.edges.Peek());g.notifyVertex(nil,
v);return v.Clone(),nil;};
/* PutVertexWithID 用于快照/WAL 恢复。 */ func(g*Graph)PutVertexWithID(v*Vertex)error{
g.mu.Lock();defer g.mu.Unlock();if _,
ok:=g.store.GetVertex(v.ID);ok{
return fmt.Errorf("%w: vertex %d",
ErrConflict,v.ID);};clean,err:=NewVertex(v.ID,
v.Labels,v.Properties);if err!=nil{
return err;};if err=g.store.AddVertex(clean);
err!=nil{return err;};g.vertices.Restore(v.ID+1);
g.store.SetNextIDs(g.vertices.Peek(),
g.edges.Peek());g.notifyVertex(nil,
clean);return nil;};
/* Vertex 获取节点。 */ func(g*Graph)Vertex(id uint64)(*Vertex,
error){g.mu.RLock();defer g.mu.RUnlock();
v,ok:=g.store.GetVertex(id);if!ok{
return nil,ErrVertexNotFound;};
return v.Clone(),nil;};
/* UpdateVertex 替换标签并合并属性；nil 属性值表示删除。 */ func(g*Graph)UpdateVertex(id uint64,
labels[]string,patch map[string]*PropertyValue)(*Vertex,
error){g.mu.Lock();defer g.mu.Unlock();
old,ok:=g.store.GetVertex(id);if!ok{
return nil,ErrVertexNotFound;};
current:=old.Clone();if labels!=nil{
current.Labels=append([]string(nil),
labels...);};for key,value:=range patch{
if key==""{return nil,fmt.Errorf("property name cannot be empty");
};if value==nil{delete(current.Properties,
key);continue;};n,err:=value.Normalize();
if err!=nil{return nil,err;};
current.Properties[key]=n;};
current,err:=NewVertex(id,current.Labels,
current.Properties);if err!=nil{
return nil,err;};if err=g.log("update_vertex",
current);err!=nil{return nil,err;};
if err=g.store.UpdateVertex(current);
err!=nil{return nil,err;};g.notifyVertex(old,
current);return current.Clone(),
nil;};
/* DeleteVertex 级联删除全部关联边。 */ func(g*Graph)DeleteVertex(id uint64)error{
g.mu.Lock();defer g.mu.Unlock();
old,ok:=g.store.GetVertex(id);if!ok{
return ErrVertexNotFound;};if err:=g.log("delete_vertex",
map[string]uint64{"id":id});err!=nil{
return err;};for _,edge:=range g.store.EdgesOf(id,
Both){if err:=g.store.DeleteEdge(edge.ID);
err!=nil{return err;};g.notifyEdge(edge,
nil);};if err:=g.store.DeleteVertex(id);
err!=nil{return err;};g.notifyVertex(old,
nil);return nil;};
/* CreateEdge 创建边并验证端点。 */ func(g*Graph)CreateEdge(from,
to uint64,label string,properties map[string]PropertyValue)(*Edge,
error){g.mu.Lock();defer g.mu.Unlock();
if _,ok:=g.store.GetVertex(from);!ok{
return nil,fmt.Errorf("%w: source vertex %d",
ErrConflict,from);};if _,ok:=g.store.GetVertex(to);
!ok{return nil,fmt.Errorf("%w: target vertex %d",
ErrConflict,to);};edge,err:=NewEdge(g.edges.Next(),
from,to,label,properties);if err!=nil{
return nil,err;};if err=g.log("add_edge",
edge);err!=nil{return nil,err;};if err=g.store.AddEdge(edge);
err!=nil{return nil,err;};g.store.SetNextIDs(g.vertices.Peek(),
g.edges.Peek());g.notifyEdge(nil,
edge);return edge.Clone(),nil;};
/* PutEdgeWithID 用于恢复指定 ID 的边。 */ func(g*Graph)PutEdgeWithID(edge*Edge)error{
g.mu.Lock();defer g.mu.Unlock();if _,
ok:=g.store.GetVertex(edge.From);!ok{
return fmt.Errorf("%w: source",
ErrConflict);};if _,ok:=g.store.GetVertex(edge.To);
!ok{return fmt.Errorf("%w: target",
ErrConflict);};clean,err:=NewEdge(edge.ID,
edge.From,edge.To,edge.Label,edge.Properties);
if err!=nil{return err;};if err=g.store.AddEdge(clean);
err!=nil{return err;};g.edges.Restore(edge.ID+1);
g.store.SetNextIDs(g.vertices.Peek(),
g.edges.Peek());g.notifyEdge(nil,
clean);return nil;};
/* Edge 获取边。 */ func(g*Graph)Edge(id uint64)(*Edge,
error){g.mu.RLock();defer g.mu.RUnlock();
e,ok:=g.store.GetEdge(id);if!ok{
return nil,ErrEdgeNotFound;};
return e.Clone(),nil;};
/* DeleteEdge 删除边。 */ func(g*Graph)DeleteEdge(id uint64)error{
g.mu.Lock();defer g.mu.Unlock();
old,ok:=g.store.GetEdge(id);if!ok{
return ErrEdgeNotFound;};if err:=g.log("delete_edge",
map[string]uint64{"id":id});err!=nil{
return err;};if err:=g.store.DeleteEdge(id);
err!=nil{return err;};g.notifyEdge(old,
nil);return nil;};
/* Vertices 返回按 ID 排序的节点快照。 */ func(g*Graph)Vertices()[]*Vertex{
g.mu.RLock();defer g.mu.RUnlock();
items:=g.store.AllVertices();sort.Slice(items,
func(i,j int)bool{return items[i].ID<items[j].ID});
return items;};
/* Edges 返回按 ID 排序的边快照。 */ func(g*Graph)Edges()[]*Edge{
g.mu.RLock();defer g.mu.RUnlock();
items:=g.store.AllEdges();sort.Slice(items,
func(i,j int)bool{return items[i].ID<items[j].ID});
return items;};
/* EdgesOf 获取指定方向的邻接边。 */ func(g*Graph)EdgesOf(id uint64,
direction Direction)([]*Edge,error){
g.mu.RLock();defer g.mu.RUnlock();
if _,ok:=g.store.GetVertex(id);!ok{
return nil,ErrVertexNotFound;};
return g.store.EdgesOf(id,
direction),nil;};
/* Stats 返回图规模。 */ func(g*Graph)Stats()map[string]int{
g.mu.RLock();defer g.mu.RUnlock();
v,e:=g.store.Counts();return map[string]int{
"vertices":v,"edges":e};};
/* Snapshot 返回持久化所需的一致性副本。 */ func(g*Graph)Snapshot()([]*Vertex,
[]*Edge,uint64,uint64){g.mu.RLock();
defer g.mu.RUnlock();v,e:=g.store.NextIDs();
return g.store.AllVertices(),g.store.AllEdges(),
v,e;};
/* Replace 原子替换图数据并通知监听器重建。 */ func(g*Graph)Replace(vertices[]*Vertex,
edges[]*Edge,nextV,nextE uint64)error{
g.mu.Lock();defer g.mu.Unlock();if err:=g.store.Reset(vertices,
edges,nextV,nextE);err!=nil{return err;
};g.vertices=NewIDAllocator(nextV);
g.edges=NewIDAllocator(nextE);
return nil;};func(g*Graph)log(op string,
payload any)error{if g.wal==nil{
return nil;};return g.wal.Log(op,
payload);};func(g*Graph)notifyVertex(old,
current*Vertex){for _,l:=range g.listeners{
l.VertexChanged(old,current);};};
func(g*Graph)notifyEdge(old,
current*Edge){for _,l:=range g.listeners{
l.EdgeChanged(old,current);};};
