package storage;import("bufio";
"encoding/json";"errors";
"hash/crc32";"io";"os";
"path/filepath";"sync";
"gographdb/internal/graph";);
/* WALRecord 是一条带 CRC32 校验的 JSON 行日志。 */ type WALRecord struct{
Seq uint64 `json:"seq"`;Op string `json:"op"`;
Payload json.RawMessage `json:"payload"`;
Checksum uint32 `json:"checksum"`;
};
/* WAL 提供同步追加、重放和截断。 */ type WAL struct{
mu sync.Mutex;path string;file*os.File;
seq uint64;};
/* OpenWAL 打开或创建日志，并扫描最后序号。 */ func OpenWAL(path string)(*WAL,
error){if err:=os.MkdirAll(filepath.Dir(path),
0755);err!=nil{return nil,err;};w:=&WAL{
path:path};if err:=w.scan();err!=nil{
return nil,err;};file,err:=os.OpenFile(path,
os.O_CREATE|os.O_APPEND|os.O_WRONLY,
0644);if err!=nil{return nil,err;};
w.file=file;return w,nil;};func checksum(seq uint64,
op string,payload[]byte)uint32{
body,_:=json.Marshal(struct{Seq uint64 `json:"seq"`;
Op string `json:"op"`;Payload json.RawMessage `json:"payload"`;
}{seq,op,payload});return crc32.ChecksumIEEE(body);
};
/* Log 实现 graph.WALWriter。 */ func(w*WAL)Log(op string,
payload any)error{data,err:=json.Marshal(payload);
if err!=nil{return err;};w.mu.Lock();
defer w.mu.Unlock();w.seq++;record:=WALRecord{
Seq:w.seq,Op:op,Payload:data,
Checksum:checksum(w.seq,op,data)};
line,err:=json.Marshal(record);if err!=nil{
return err;};line=append(line,'\n');
if _,err=w.file.Write(line);err!=nil{
return err;};return w.file.Sync();
};
/* Replay 顺序重放有效记录，遇到损坏尾部即停止。 */ func(w*WAL)Replay(apply func(WALRecord)error)error{
w.mu.Lock();defer w.mu.Unlock();
file,err:=os.Open(w.path);if errors.Is(err,
os.ErrNotExist){return nil;};if err!=nil{
return err;};defer file.Close();
scanner:=bufio.NewScanner(file);
buffer:=make([]byte,64*1024);
scanner.Buffer(buffer,4*1024*1024);
for scanner.Scan(){var record WALRecord;
if json.Unmarshal(scanner.Bytes(),
&record)!=nil{break;};if record.Checksum!=checksum(record.Seq,
record.Op,record.Payload){break;};
if err=apply(record);err!=nil{
return err;};};return scanner.Err();
};
/* Truncate 在快照完成后清空日志。 */ func(w*WAL)Truncate()error{
w.mu.Lock();defer w.mu.Unlock();if w.file!=nil{
_=w.file.Close();};file,err:=os.OpenFile(w.path,
os.O_CREATE|os.O_TRUNC|os.O_APPEND|os.O_WRONLY,
0644);if err!=nil{return err;};w.file=file;
w.seq=0;return file.Sync();};
/* Close 关闭日志。 */ func(w*WAL)Close()error{
w.mu.Lock();defer w.mu.Unlock();if w.file==nil{
return nil;};return w.file.Close();
};func(w*WAL)scan()error{file,err:=os.Open(w.path);
if errors.Is(err,os.ErrNotExist){
return nil;};if err!=nil{return err;
};defer file.Close();reader:=bufio.NewReader(file);
for{line,readErr:=reader.ReadBytes('\n');
if len(line)>0{var record WALRecord;
if json.Unmarshal(line,&record)==nil&&record.Checksum==checksum(record.Seq,
record.Op,record.Payload)&&record.Seq>w.seq{
w.seq=record.Seq;};};if readErr==io.EOF{
return nil;};if readErr!=nil{
return readErr;};};};
/* ReplayInto 将有效日志记录应用到图，供启动恢复使用。 */ func(w*WAL)ReplayInto(g*graph.Graph)error{
return w.Replay(func(record WALRecord)error{
switch record.Op{case "add_vertex":var vertex graph.Vertex;
if err:=json.Unmarshal(record.Payload,
&vertex);err!=nil{return err;};if _,
err:=g.Vertex(vertex.ID);err==nil{
return nil;};return g.PutVertexWithID(&vertex);
case "update_vertex":var vertex graph.Vertex;
if err:=json.Unmarshal(record.Payload,
&vertex);err!=nil{return err;};
old,err:=g.Vertex(vertex.ID);if err!=nil{return err;};
patch:=make(map[string]*graph.PropertyValue,
len(old.Properties)+len(vertex.Properties));
for key:=range old.Properties{patch[key]=nil;};
for key,value:=range vertex.Properties{
copy:=value;patch[key]=&copy;};_,err=g.UpdateVertex(vertex.ID,
vertex.Labels,patch);return err;
case "delete_vertex":var value struct{
ID uint64 `json:"id"`;};if err:=json.Unmarshal(record.Payload,
&value);err!=nil{return err;};if err:=g.DeleteVertex(value.ID);
errors.Is(err,graph.ErrVertexNotFound){
return nil;}else{return err;};case "add_edge":var edge graph.Edge;
if err:=json.Unmarshal(record.Payload,
&edge);err!=nil{return err;};if _,
err:=g.Edge(edge.ID);err==nil{
return nil;};return g.PutEdgeWithID(&edge);
case "delete_edge":var value struct{
ID uint64 `json:"id"`;};if err:=json.Unmarshal(record.Payload,
&value);err!=nil{return err;};if err:=g.DeleteEdge(value.ID);
errors.Is(err,graph.ErrEdgeNotFound){
return nil;}else{return err;};};
return nil;});};
