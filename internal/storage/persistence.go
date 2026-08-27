package storage;import("bufio";
"encoding/gob";"encoding/json";
"fmt";"os";"path/filepath";"time";
"gographdb/internal/graph";);const snapshotVersion uint16=1;
var snapshotMagic=[8]byte{'G','O',
'G','R','A','P','H','1'};type diskSnapshot struct{
Magic[8]byte;Data Snapshot;};
/* Persistence 管理图快照路径。 */ type Persistence struct{
DataDir string};
/* NewPersistence 创建持久化服务。 */ func NewPersistence(dataDir string)*Persistence{
return&Persistence{DataDir:dataDir}};

type FileStatus struct{Exists bool `json:"exists"`;Size int64 `json:"size"`;};type PersistenceStatus struct{DataDir string `json:"dataDir"`;Snapshot FileStatus `json:"snapshot"`;Metadata FileStatus `json:"metadata"`;WAL FileStatus `json:"wal"`;};
// Status 返回快照、元数据和 WAL 文件的运维状态。
func(p*Persistence)Status()PersistenceStatus{file:=func(name string)FileStatus{info,err:=os.Stat(filepath.Join(p.DataDir,name));if err!=nil{return FileStatus{}};return FileStatus{true,info.Size()}};return PersistenceStatus{p.DataDir,file("snapshot.gdb"),file("meta.json"),file("wal.log")}}

/* SnapshotPath 返回默认快照文件。 */ func(p*Persistence)SnapshotPath()string{
return filepath.Join(p.DataDir,
"snapshot.gdb")};
/* Save 原子写入全量快照和诊断元数据。 */ func(p*Persistence)Save(g*graph.Graph)error{
if err:=os.MkdirAll(p.DataDir,0755);
err!=nil{return err;};vertices,
edges,nextV,nextE:=g.Snapshot();
payload:=diskSnapshot{Magic:snapshotMagic,
Data:Snapshot{Version:snapshotVersion,
NextVertexID:nextV,NextEdgeID:nextE,
Vertices:vertices,Edges:edges}};
tmp:=p.SnapshotPath()+".tmp";file,
err:=os.Create(tmp);if err!=nil{
return err;};writer:=bufio.NewWriter(file);
if err=gob.NewEncoder(writer).Encode(payload);
err==nil{err=writer.Flush();};if syncErr:=file.Sync();
err==nil{err=syncErr;};if closeErr:=file.Close();
err==nil{err=closeErr;};if err!=nil{
_=os.Remove(tmp);return err;};if err=os.Rename(tmp,
p.SnapshotPath());err!=nil{return err;
};meta:=map[string]any{"version":snapshotVersion,
"savedAt":time.Now().UTC(),
"vertices":len(vertices),"edges":len(edges)};
data,_:=json.MarshalIndent(meta,"",
"  ");return os.WriteFile(filepath.Join(p.DataDir,
"meta.json"),data,0644);};
/* Load 读取快照；文件不存在时返回 os.ErrNotExist。 */ func(p*Persistence)Load()(Snapshot,
error){file,err:=os.Open(p.SnapshotPath());
if err!=nil{return Snapshot{},err;
};defer file.Close();var disk diskSnapshot;
if err=gob.NewDecoder(bufio.NewReader(file)).Decode(&disk);
err!=nil{return Snapshot{},err;};
if disk.Magic!=snapshotMagic{
return Snapshot{},fmt.Errorf("invalid snapshot magic");
};if disk.Data.Version!=snapshotVersion{
return Snapshot{},fmt.Errorf("unsupported snapshot version %d",
disk.Data.Version);};return disk.Data,
nil;};
/* LoadInto 将快照装载到图中。 */ func(p*Persistence)LoadInto(g*graph.Graph)error{
snapshot,err:=p.Load();if err!=nil{
return err;};return g.Replace(snapshot.Vertices,
snapshot.Edges,snapshot.NextVertexID,
snapshot.NextEdgeID);};
/* Export 返回可直接编码为 JSON 的一致性快照。 */ func(p*Persistence)Export(g*graph.Graph)Snapshot{v,e,nv,ne:=g.Snapshot();return Snapshot{Version:snapshotVersion,NextVertexID:nv,NextEdgeID:ne,Vertices:v,Edges:e};};
/* DumpJSON 导出便于调试和迁移的 JSON。 */ func(p*Persistence)DumpJSON(g*graph.Graph,
path string)error{data,err:=json.MarshalIndent(p.Export(g),"","  ");if err!=nil{return err;};return os.WriteFile(path,data,0644);};
