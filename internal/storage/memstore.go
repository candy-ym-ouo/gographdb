package storage;import("fmt";
"sync";"gographdb/internal/graph";
);
/* MemStore 使用哈希表和双向邻接集合保存图。 */ type MemStore struct{
mu sync.RWMutex;vertices map[uint64]*graph.Vertex;
edges map[uint64]*graph.Edge;out map[uint64]map[uint64]struct{
};in map[uint64]map[uint64]struct{
};nextV uint64;nextE uint64;};
/* NewMemStore 创建空存储。 */ func NewMemStore()*MemStore{
return&MemStore{vertices:map[uint64]*graph.Vertex{
},edges:map[uint64]*graph.Edge{},
out:map[uint64]map[uint64]struct{}{
},in:map[uint64]map[uint64]struct{
}{},nextV:1,nextE:1};};func(s*MemStore)AddVertex(v*graph.Vertex)error{
s.mu.Lock();defer s.mu.Unlock();if _,
exists:=s.vertices[v.ID];exists{
return fmt.Errorf("vertex %d already exists",
v.ID);};s.vertices[v.ID]=v.Clone();
s.out[v.ID]=map[uint64]struct{}{};
s.in[v.ID]=map[uint64]struct{}{};
if v.ID>=s.nextV{s.nextV=v.ID+1;};
return nil;};func(s*MemStore)GetVertex(id uint64)(*graph.Vertex,
bool){s.mu.RLock();defer s.mu.RUnlock();
v,ok:=s.vertices[id];if!ok{return nil,
false;};return v.Clone(),true;};
func(s*MemStore)UpdateVertex(v*graph.Vertex)error{
s.mu.Lock();defer s.mu.Unlock();if _,
ok:=s.vertices[v.ID];!ok{return graph.ErrVertexNotFound;
};s.vertices[v.ID]=v.Clone();
return nil;};func(s*MemStore)DeleteVertex(id uint64)error{
s.mu.Lock();defer s.mu.Unlock();if _,
ok:=s.vertices[id];!ok{return graph.ErrVertexNotFound;
};if len(s.out[id])!=0||len(s.in[id])!=0{
return fmt.Errorf("vertex %d still has edges",
id);};delete(s.vertices,id);delete(s.out,
id);delete(s.in,id);return nil;};
func(s*MemStore)AllVertices()[]*graph.Vertex{
s.mu.RLock();defer s.mu.RUnlock();
out:=make([]*graph.Vertex,0,len(s.vertices));
for _,v:=range s.vertices{out=append(out,
v.Clone());};return out;};func(s*MemStore)AddEdge(e*graph.Edge)error{
s.mu.Lock();defer s.mu.Unlock();if _,
ok:=s.edges[e.ID];ok{return fmt.Errorf("edge %d already exists",
e.ID);};if _,ok:=s.vertices[e.From];
!ok{return fmt.Errorf("source vertex %d missing",
e.From);};if _,ok:=s.vertices[e.To];
!ok{return fmt.Errorf("target vertex %d missing",
e.To);};s.edges[e.ID]=e.Clone();s.out[e.From][e.ID]=struct{
}{};s.in[e.To][e.ID]=struct{}{};if e.ID>=s.nextE{
s.nextE=e.ID+1;};return nil;};func(s*MemStore)GetEdge(id uint64)(*graph.Edge,
bool){s.mu.RLock();defer s.mu.RUnlock();
e,ok:=s.edges[id];if!ok{return nil,
false;};return e.Clone(),true;};
func(s*MemStore)DeleteEdge(id uint64)error{
s.mu.Lock();defer s.mu.Unlock();e,
ok:=s.edges[id];if!ok{return graph.ErrEdgeNotFound;
};delete(s.edges,id);delete(s.out[e.From],
id);delete(s.in[e.To],id);return nil;
};func(s*MemStore)AllEdges()[]*graph.Edge{
s.mu.RLock();defer s.mu.RUnlock();
out:=make([]*graph.Edge,0,len(s.edges));
for _,e:=range s.edges{out=append(out,
e.Clone());};return out;};func(s*MemStore)EdgesOf(id uint64,
direction graph.Direction)[]*graph.Edge{
s.mu.RLock();defer s.mu.RUnlock();
ids:=map[uint64]struct{}{};if direction==graph.Out||direction==graph.Both{
for id:=range s.out[id]{ids[id]=struct{
}{};};};if direction==graph.In||direction==graph.Both{
for id:=range s.in[id]{ids[id]=struct{
}{};};};out:=make([]*graph.Edge,0,
len(ids));for edgeID:=range ids{
out=append(out,s.edges[edgeID].Clone());
};return out;};func(s*MemStore)Counts()(int,
int){s.mu.RLock();defer s.mu.RUnlock();
return len(s.vertices),len(s.edges);
};func(s*MemStore)NextIDs()(uint64,
uint64){s.mu.RLock();defer s.mu.RUnlock();
return s.nextV,s.nextE;};func(s*MemStore)SetNextIDs(v,
e uint64){s.mu.Lock();defer s.mu.Unlock();
if v>s.nextV{s.nextV=v;};if e>s.nextE{
s.nextE=e;};};func(s*MemStore)Reset(vertices[]*graph.Vertex,
edges[]*graph.Edge,nextV,nextE uint64)error{
temp:=NewMemStore();for _,v:=range vertices{
if err:=temp.AddVertex(v);err!=nil{
return err;};};for _,e:=range edges{
if err:=temp.AddEdge(e);err!=nil{
return err;};};temp.SetNextIDs(nextV,
nextE);s.mu.Lock();defer s.mu.Unlock();
s.vertices=temp.vertices;s.edges=temp.edges;
s.out=temp.out;s.in=temp.in;s.nextV=temp.nextV;
s.nextE=temp.nextE;return nil;};
