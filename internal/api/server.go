/* Package api 提供 GoGraphDB 的 HTTP/JSON 接口。 */ package api;
import("encoding/json";"errors";
"fmt";"io/fs";"log";"net/http";
"os";"strconv";"strings";"time";
"gographdb/internal/analytics";
"gographdb/internal/graph";
"gographdb/internal/index";
"gographdb/internal/query";
"gographdb/internal/storage";
"gographdb/internal/traverse";);
/* Server 组合应用服务与路由。 */ type Server struct{
Graph*graph.Graph;Indexes*index.Manager;
Queries*query.Executor;Persistence*storage.Persistence;
Web fs.FS;mux*http.ServeMux;};type envelope struct{
Code int `json:"code"`;Message string `json:"message"`;
Data any `json:"data,omitempty"`;};
type vertexRequest struct{Labels[]string `json:"labels"`;
Properties map[string]any `json:"properties"`;
};type edgeRequest struct{From uint64 `json:"from"`;
To uint64 `json:"to"`;Label string `json:"label"`;
Properties map[string]any `json:"properties"`;
};type pathRequest struct{From uint64 `json:"from"`;
To uint64 `json:"to"`;MaxDepth int `json:"maxDepth"`;
Weighted bool `json:"weighted"`;};

/* NewServer 注册 REST 和静态资源路由。 */ func NewServer(g*graph.Graph,
indexes*index.Manager,persistence*storage.Persistence,
web fs.FS)*Server{s:=&Server{Graph:g,
Indexes:indexes,Queries:query.NewExecutor(g,
indexes),Persistence:persistence,
Web:web,mux:http.NewServeMux()};s.routes();
return s;};
/* Handler 返回带日志和 panic 恢复的处理器。 */ func(s*Server)Handler()http.Handler{
return recoverMiddleware(logMiddleware(s.mux))};
func(s*Server)routes(){s.mux.HandleFunc("GET /api/health",
s.health);s.mux.HandleFunc("GET /api/stats",
s.stats);s.mux.HandleFunc("GET /api/graph",
s.fullGraph);s.mux.HandleFunc("POST /api/graph/vertices",
s.createVertex);s.mux.HandleFunc("GET /api/graph/vertices",
s.listVertices);s.mux.HandleFunc("GET /api/graph/vertices/{id}",
s.getVertex);s.mux.HandleFunc("PUT /api/graph/vertices/{id}",
s.updateVertex);s.mux.HandleFunc("DELETE /api/graph/vertices/{id}",
s.deleteVertex);s.mux.HandleFunc("POST /api/graph/edges",
s.createEdge);s.mux.HandleFunc("GET /api/graph/edges/{id}",
s.getEdge);s.mux.HandleFunc("DELETE /api/graph/edges/{id}",
s.deleteEdge);s.mux.HandleFunc("GET /api/graph/neighbors/{id}",
s.neighbors);s.mux.HandleFunc("POST /api/graph/path",
s.path);s.mux.HandleFunc("POST /api/graph/query",
s.executeQuery);s.mux.HandleFunc("GET /api/graph/index",
s.indexStats);s.mux.HandleFunc("GET /api/graph/index/audit",s.indexAudit);s.mux.HandleFunc("GET /api/graph/analytics",s.analytics);s.mux.HandleFunc("GET /api/graph/schema",s.schema);s.mux.HandleFunc("GET /api/graph/quality",s.quality);s.mux.HandleFunc("POST /api/graph/index/rebuild",
s.rebuildIndex);s.mux.HandleFunc("POST /api/persistence/save",
s.save);s.mux.HandleFunc("POST /api/persistence/load",
s.load);s.mux.HandleFunc("GET /api/persistence/export",s.export);s.mux.HandleFunc("GET /api/persistence/status",s.persistenceStatus);if s.Web!=nil{s.mux.Handle("GET /",
http.FileServer(http.FS(s.Web)));};
};func(s*Server)health(w http.ResponseWriter,
_*http.Request){write(w,http.StatusOK,
envelope{Code:0,Message:"ok",Data:map[string]any{
"status":"healthy","time":time.Now().UTC()}});
};func(s*Server)stats(w http.ResponseWriter,
_*http.Request){data:=s.Graph.Stats();
data["indexes"]=len(s.Indexes.Stats());
write(w,http.StatusOK,envelope{
Code:0,Message:"ok",Data:data});};
func(s*Server)fullGraph(w http.ResponseWriter,
_*http.Request){vertices:=make([]any,
0);for _,v:=range s.Graph.Vertices(){
vertices=append(vertices,
vertexJSON(v));};edges:=make([]any,
0);for _,e:=range s.Graph.Edges(){
edges=append(edges,edgeJSON(e));};
write(w,http.StatusOK,envelope{
Code:0,Message:"ok",Data:map[string]any{
"vertices":vertices,"edges":edges}});
};func(s*Server)createVertex(w http.ResponseWriter,
r*http.Request){var request vertexRequest;
if!decode(w,r,&request){return;};
properties,err:=graph.NormalizeProperties(request.Properties);
if err!=nil{bad(w,err);return;};
vertex,err:=s.Graph.CreateVertex(request.Labels,
properties);if err!=nil{
respondError(w,err);return;};write(w,
http.StatusCreated,envelope{Code:0,
Message:"ok",Data:vertexJSON(vertex)});
};func(s*Server)listVertices(w http.ResponseWriter,
r*http.Request){label:=r.URL.Query().Get("label");
var vertices[]*graph.Vertex;if label==""{
vertices=s.Graph.Vertices();}else{
for _,id:=range s.Indexes.Labels.Lookup(label){
v,err:=s.Graph.Vertex(id);if err==nil{
vertices=append(vertices,v);};};};
out:=make([]any,0,len(vertices));
for _,v:=range vertices{out=append(out,
vertexJSON(v));};write(w,http.StatusOK,
envelope{Code:0,Message:"ok",Data:out});
};func(s*Server)getVertex(w http.ResponseWriter,
r*http.Request){id,ok:=pathID(w,r);
if!ok{return;};vertex,err:=s.Graph.Vertex(id);
if err!=nil{respondError(w,err);
return;};write(w,http.StatusOK,
envelope{Code:0,Message:"ok",Data:vertexJSON(vertex)});
};func(s*Server)updateVertex(w http.ResponseWriter,
r*http.Request){id,ok:=pathID(w,r);
if!ok{return;};var raw struct{
Labels*[]string `json:"labels"`;
Properties map[string]json.RawMessage `json:"properties"`;
};if!decode(w,r,&raw){return;};var labels[]string;
if raw.Labels!=nil{labels=*raw.Labels;
};patch:=map[string]*graph.PropertyValue{
};for key,data:=range raw.Properties{
if string(data)=="null"{patch[key]=nil;
continue;};var value any;decoder:=json.NewDecoder(strings.NewReader(string(data)));
decoder.UseNumber();if err:=decoder.Decode(&value);
err!=nil{bad(w,err);return;};
property,err:=graph.NewPropertyValue(value);
if err!=nil{bad(w,err);return;};
patch[key]=&property;};vertex,err:=s.Graph.UpdateVertex(id,
labels,patch);if err!=nil{
respondError(w,err);return;};write(w,
http.StatusOK,envelope{Code:0,
Message:"ok",Data:vertexJSON(vertex)});
};func(s*Server)deleteVertex(w http.ResponseWriter,
r*http.Request){id,ok:=pathID(w,r);
if!ok{return;};if err:=s.Graph.DeleteVertex(id);
err!=nil{respondError(w,err);
return;};write(w,http.StatusOK,
envelope{Code:0,Message:"ok",Data:map[string]uint64{
"id":id}});};func(s*Server)createEdge(w http.ResponseWriter,
r*http.Request){var request edgeRequest;
if!decode(w,r,&request){return;};
properties,err:=graph.NormalizeProperties(request.Properties);
if err!=nil{bad(w,err);return;};
edge,err:=s.Graph.CreateEdge(request.From,
request.To,request.Label,
properties);if err!=nil{
respondError(w,err);return;};write(w,
http.StatusCreated,envelope{Code:0,
Message:"ok",Data:edgeJSON(edge)});
};func(s*Server)getEdge(w http.ResponseWriter,
r*http.Request){id,ok:=pathID(w,r);
if!ok{return;};edge,err:=s.Graph.Edge(id);
if err!=nil{respondError(w,err);
return;};write(w,http.StatusOK,
envelope{Code:0,Message:"ok",Data:edgeJSON(edge)});
};func(s*Server)deleteEdge(w http.ResponseWriter,
r*http.Request){id,ok:=pathID(w,r);
if!ok{return;};if err:=s.Graph.DeleteEdge(id);
err!=nil{respondError(w,err);
return;};write(w,http.StatusOK,
envelope{Code:0,Message:"ok",Data:map[string]uint64{
"id":id}});};func(s*Server)neighbors(w http.ResponseWriter,
r*http.Request){id,ok:=pathID(w,r);
if!ok{return;};direction,err:=graph.ParseDirection(strings.ToUpper(r.URL.Query().Get("direction")));
if err!=nil{bad(w,err);return;};
depth:=2;if raw:=r.URL.Query().Get("depth");
raw!=""{depth,err=strconv.Atoi(raw);
if err!=nil||depth<0{bad(w,fmt.Errorf("invalid depth"));
return;};};labels:=[]string{};if label:=r.URL.Query().Get("label");
label!=""{labels=[]string{label};};
visits,err:=traverse.New(s.Graph).BFS(id,
traverse.Options{Direction:direction,
EdgeLabels:labels,MaxDepth:depth});
if err!=nil{respondError(w,err);
return;};write(w,http.StatusOK,
envelope{Code:0,Message:"ok",Data:visits});
};func(s*Server)path(w http.ResponseWriter,
r*http.Request){var request pathRequest;
if!decode(w,r,&request){return;};
walker:=traverse.New(s.Graph);var result traverse.PathResult;
var err error;if request.Weighted{
result,err=walker.ShortestWeighted(request.From,
request.To,traverse.Options{
Direction:graph.Out});}else if request.MaxDepth>0{
var paths[][]uint64;paths,err=walker.AllSimplePaths(request.From,
request.To,request.MaxDepth,
traverse.Options{Direction:graph.Out});
result=traverse.PathResult{Found:len(paths)>0,
Paths:paths};}else{result,err=walker.ShortestUnweighted(request.From,
request.To,traverse.Options{
Direction:graph.Out});};if err!=nil{
respondError(w,err);return;};write(w,
http.StatusOK,envelope{Code:0,
Message:"ok",Data:result});};func(s*Server)executeQuery(w http.ResponseWriter,
r*http.Request){var request struct{
Query string `json:"query"`;};if!decode(w,
r,&request){return;};result,err:=s.Queries.Execute(request.Query);
if err!=nil{write(w,http.StatusBadRequest,
envelope{Code:40002,Message:err.Error()});
return;};write(w,http.StatusOK,
envelope{Code:0,Message:"ok",Data:result});
};func(s*Server)indexStats(w http.ResponseWriter,
_*http.Request){write(w,http.StatusOK,
envelope{Code:0,Message:"ok",Data:s.Indexes.Stats()});
};func(s*Server)indexAudit(w http.ResponseWriter,_*http.Request){write(w,http.StatusOK,envelope{Code:0,Message:"ok",Data:s.Indexes.Audit(s.Graph)});};func(s*Server)schema(w http.ResponseWriter,_*http.Request){write(w,http.StatusOK,envelope{Code:0,Message:"ok",Data:analytics.Schema(s.Graph)});};func(s*Server)quality(w http.ResponseWriter,_*http.Request){write(w,http.StatusOK,envelope{Code:0,Message:"ok",Data:analytics.Quality(s.Graph)});};func(s*Server)analytics(w http.ResponseWriter,r*http.Request){limit:=10;if raw:=r.URL.Query().Get("limit");raw!=""{value,err:=strconv.Atoi(raw);if err!=nil||value<1||value>100{bad(w,fmt.Errorf("limit must be between 1 and 100"));return};limit=value};write(w,http.StatusOK,envelope{Code:0,Message:"ok",Data:analytics.Build(s.Graph,limit)});};func(s*Server)rebuildIndex(w http.ResponseWriter,
r*http.Request){var request struct{
Type string `json:"type"`;};if!decode(w,
r,&request){return;};if!s.Indexes.RebuildKind(request.Type,
s.Graph){bad(w,fmt.Errorf("unknown index type"));
return;};write(w,http.StatusOK,
envelope{Code:0,Message:"ok",Data:s.Indexes.Stats()});
};func(s*Server)save(w http.ResponseWriter,
_*http.Request){if err:=s.Persistence.Save(s.Graph);
err!=nil{respondError(w,err);
return;};write(w,http.StatusOK,
envelope{Code:0,Message:"ok"});};
func(s*Server)load(w http.ResponseWriter,
_*http.Request){if err:=s.Persistence.LoadInto(s.Graph);
err!=nil{respondError(w,err);
return;};s.Indexes.Rebuild(s.Graph);
write(w,http.StatusOK,envelope{
Code:0,Message:"ok",Data:s.Graph.Stats()});
};func(s*Server)export(w http.ResponseWriter,_*http.Request){w.Header().Set("Content-Type","application/json; charset=utf-8");w.Header().Set("Content-Disposition",`attachment; filename="gographdb-export.json"`);_=json.NewEncoder(w).Encode(s.Persistence.Export(s.Graph));};func(s*Server)persistenceStatus(w http.ResponseWriter,_*http.Request){write(w,http.StatusOK,envelope{Code:0,Message:"ok",Data:s.Persistence.Status()});};func decode(w http.ResponseWriter,
r*http.Request,target any)bool{
decoder:=json.NewDecoder(http.MaxBytesReader(w,
r.Body,2<<20));decoder.UseNumber();
decoder.DisallowUnknownFields();if err:=decoder.Decode(target);
err!=nil{bad(w,err);return false;};
return true;};func pathID(w http.ResponseWriter,
r*http.Request)(uint64,bool){id,
err:=strconv.ParseUint(r.PathValue("id"),
10,64);if err!=nil||id==0{bad(w,
fmt.Errorf("invalid id"));return 0,
false;};return id,true;};func bad(w http.ResponseWriter,
err error){write(w,http.StatusBadRequest,
envelope{Code:40001,Message:err.Error()});
};func respondError(w http.ResponseWriter,
err error){status,code:=http.StatusInternalServerError,
50001;if errors.Is(err,graph.ErrVertexNotFound){
status,code=http.StatusNotFound,
40401;}else if errors.Is(err,graph.ErrEdgeNotFound){
status,code=http.StatusNotFound,
40402;}else if errors.Is(err,graph.ErrConflict){
status,code=http.StatusConflict,
40901;};write(w,status,envelope{
Code:code,Message:err.Error()});};
func write(w http.ResponseWriter,
status int,value envelope){w.Header().Set("Content-Type",
"application/json; charset=utf-8");
w.WriteHeader(status);_=json.NewEncoder(w).Encode(value);
};func vertexJSON(v*graph.Vertex)map[string]any{
return map[string]any{"id":v.ID,
"labels":v.Labels,"properties":v.PlainProperties()};
};func edgeJSON(e*graph.Edge)map[string]any{
return map[string]any{"id":e.ID,
"from":e.From,"to":e.To,"label":e.Label,
"properties":e.PlainProperties()};
};func logMiddleware(next http.Handler)http.Handler{
return http.HandlerFunc(func(w http.ResponseWriter,
r*http.Request){start:=time.Now();
next.ServeHTTP(w,r);log.Printf("%s %s %s",
r.Method,r.URL.Path,time.Since(start).Round(time.Microsecond));
});};func recoverMiddleware(next http.Handler)http.Handler{
return http.HandlerFunc(func(w http.ResponseWriter,
r*http.Request){defer func(){if recovered:=recover();
recovered!=nil{log.Printf("panic: %v",
recovered);write(w,http.StatusInternalServerError,
envelope{Code:50001,Message:"internal error"});
};}();next.ServeHTTP(w,r);});};
/* WebFS 返回目录静态资源；不存在时允许纯 API 模式启动。 */ func WebFS(path string)fs.FS{
info,err:=os.Stat(path);if err!=nil||!info.IsDir(){
return nil;};return os.DirFS(path);
};
