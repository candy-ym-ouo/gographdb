package index;import("strings";
"sync";"gographdb/internal/graph";
);
/* Manager 汇总并同步维护所有索引。 */ type Manager struct{
mu sync.RWMutex;Labels*LabelIndex;
Properties*PropertyIndex;Ranges*BTreeIndex;
edgeLabels map[string]map[uint64]struct{
};};
/* NewManager 创建索引管理器。 */ func NewManager()*Manager{
return&Manager{Labels:NewLabelIndex(),
Properties:NewPropertyIndex(),
Ranges:NewBTreeIndex(),edgeLabels:map[string]map[uint64]struct{
}{}};};
/* VertexChanged 实现 graph.ChangeListener。 */ func(m*Manager)VertexChanged(old,
current*graph.Vertex){if old!=nil{
m.removeVertex(old);};if current!=nil{
m.addVertex(current);};};
/* EdgeChanged 同步边标签索引。 */ func(m*Manager)EdgeChanged(old,
current*graph.Edge){m.mu.Lock();
defer m.mu.Unlock();if old!=nil{
set:=m.edgeLabels[old.Label];
delete(set,old.ID);if len(set)==0{
delete(m.edgeLabels,old.Label);};};
if current!=nil{set:=m.edgeLabels[current.Label];
if set==nil{set=map[uint64]struct{
}{};m.edgeLabels[current.Label]=set;
};set[current.ID]=struct{}{};};};
func(m*Manager)addVertex(v*graph.Vertex){
for _,label:=range v.Labels{m.Labels.Add(label,
v.ID);};for name,value:=range v.Properties{
m.Properties.Add(name,value,v.ID);
m.Ranges.Add(name,value,v.ID);};};
func(m*Manager)removeVertex(v*graph.Vertex){
for _,label:=range v.Labels{m.Labels.Remove(label,
v.ID);};for name,value:=range v.Properties{
m.Properties.Remove(name,value,v.ID);
m.Ranges.Remove(name,value,v.ID);};
};
/* Rebuild 从图数据全量重建索引。 */ func(m*Manager)Rebuild(g*graph.Graph){
m.Labels.Clear();m.Properties.Clear();
m.Ranges.Clear();m.mu.Lock();m.edgeLabels=map[string]map[uint64]struct{
}{};m.mu.Unlock();for _,v:=range g.Vertices(){
m.addVertex(v);};for _,e:=range g.Edges(){
m.EdgeChanged(nil,e);};};
/* VertexIDs 根据标签和可选等值属性取交集。 */ func(m*Manager)VertexIDs(label,
property string,value*graph.PropertyValue)[]uint64{
var result map[uint64]struct{};if label!=""{
result=cloneSet(sliceSet(m.Labels.Lookup(label)));
};if property!=""&&value!=nil{set:=sliceSet(m.Properties.Lookup(property,
*value));if result==nil{result=set;
}else{for id:=range result{if _,ok:=set[id];
!ok{delete(result,id);};};};};if result==nil{
return nil;};return IDs(result);};

/* EdgeIDs 返回指定标签的边。 */ func(m*Manager)EdgeIDs(label string)[]uint64{
m.mu.RLock();defer m.mu.RUnlock();
return IDs(m.edgeLabels[label]);};

/* Stats 返回所有索引统计。 */ func(m*Manager)Stats()[]Stats{
stats:=[]Stats{m.Labels.Stats(),m.Properties.Stats(),
m.Ranges.Stats()};m.mu.RLock();
defer m.mu.RUnlock();entries:=0;
for _,set:=range m.edgeLabels{
entries+=len(set);};return append(stats,
Stats{Name:"edge_labels",Kind:"label",
Entries:entries,Keys:len(m.edgeLabels)});
};
/* RebuildKind 兼容运维接口；当前索引规模小，统一全量重建。 */ func(m*Manager)RebuildKind(kind string,
g*graph.Graph)bool{switch strings.ToLower(kind){
case "","all","label","property",
"btree":m.Rebuild(g);return true;
default:return false;};};func sliceSet(ids[]uint64)map[uint64]struct{
}{set:=make(map[uint64]struct{},
len(ids));for _,id:=range ids{set[id]=struct{
}{};};return set;};

/* AuditReport 描述索引和图快照的一致性检查结果。 */ type AuditReport struct{Healthy bool `json:"healthy"`;Vertices int `json:"vertices"`;Edges int `json:"edges"`;Problems []string `json:"problems,omitempty"`;};
/* Audit 校验索引条目是否覆盖当前图数据，并检查条目总数是否存在残留。 */ func(m*Manager)Audit(g*graph.Graph)AuditReport{report:=AuditReport{Healthy:true};vertices,edges,_,_:=g.Snapshot();report.Vertices=len(vertices);report.Edges=len(edges);labels,properties,ranges:=0,0,0;for _,v:=range vertices{labels+=len(v.Labels);for _,label:=range v.Labels{if !hasID(m.Labels.Lookup(label),v.ID){report.Problems=append(report.Problems,"missing label index: "+label);}};for name,value:=range v.Properties{properties++;if !hasID(m.Properties.Lookup(name,value),v.ID){report.Problems=append(report.Problems,"missing property index: "+name);};if rangeable(value){ranges++;if !hasID(m.Ranges.Range(name,&value,&value,true,true),v.ID){report.Problems=append(report.Problems,"missing range index: "+name);}}}};for _,e:=range edges{if !hasID(m.EdgeIDs(e.Label),e.ID){report.Problems=append(report.Problems,"missing edge label index: "+e.Label);}};stats:=m.Stats();if stats[0].Entries!=labels||stats[1].Entries!=properties||stats[2].Entries!=ranges||stats[3].Entries!=len(edges){report.Problems=append(report.Problems,"index entry count differs from graph data")};report.Healthy=len(report.Problems)==0;return report;};func hasID(ids[]uint64,want uint64)bool{for _,id:=range ids{if id==want{return true}};return false};func rangeable(value graph.PropertyValue)bool{switch value.Type{case graph.IntType,graph.FloatType,graph.TimestampType,graph.StringType:return true};return false};
