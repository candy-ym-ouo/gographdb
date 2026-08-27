package traverse;import("container/heap";
"math";"gographdb/internal/graph";
);
/* PathResult 是统一路径查询结果。 */ type PathResult struct{
Found bool `json:"found"`;Distance float64 `json:"distance"`;
Path[]uint64 `json:"path,omitempty"`;
Paths[][]uint64 `json:"paths,omitempty"`;
};
/* AllSimplePaths 枚举深度限制内所有简单路径。 */ func(t*Traverser)AllSimplePaths(from,
to uint64,maxDepth int,options Options)([][]uint64,
error){if _,err:=t.Graph.Vertex(from);
err!=nil{return nil,err;};if _,err:=t.Graph.Vertex(to);
err!=nil{return nil,err;};options.MaxDepth=maxDepth;
paths:=[][]uint64{};path:=[]uint64{
from};used:=map[uint64]bool{from:true};
var walk func(uint64,int)error;
walk=func(id uint64,depth int)error{
if id==to{paths=append(paths,
append([]uint64(nil),path...));
return nil;};if depth>=maxDepth{
return nil;};neighbors,err:=t.Neighbors(id,
options);if err!=nil{return err;};
for _,next:=range sortedKeys(neighbors){
if used[next]{continue;};used[next]=true;
path=append(path,next);if err=walk(next,
depth+1);err!=nil{return err;};
path=path[:len(path)-1];delete(used,
next);};return nil;};return paths,
walk(from,0);};
/* ShortestUnweighted 用 BFS 求无权最短路径。 */ func(t*Traverser)ShortestUnweighted(from,
to uint64,options Options)(PathResult,
error){if _,err:=t.Graph.Vertex(from);
err!=nil{return PathResult{},err;};if _,err:=t.Graph.Vertex(to);
err!=nil{return PathResult{},err;};if from==to{return PathResult{
Found:true,Distance:0,Path:[]uint64{
from}},nil;};
queue:=[]uint64{from};visited:=map[uint64]bool{
from:true};previous:=map[uint64]uint64{
};for len(queue)>0{current:=queue[0];
queue=queue[1:];neighbors,err:=t.Neighbors(current,
options);if err!=nil{return PathResult{
},err;};for _,next:=range sortedKeys(neighbors){
if visited[next]{continue;};
visited[next]=true;previous[next]=current;
if next==to{path:=buildPath(previous,
from,to);return PathResult{Found:true,
Distance:float64(len(path)-1),Path:path},
nil;};queue=append(queue,next);};};
return PathResult{Found:false},nil;
};type distanceItem struct{id uint64;
distance float64;index int;};type priorityQueue[]*distanceItem;
func(p priorityQueue)Len()int{
return len(p)};func(p priorityQueue)Less(i,
j int)bool{return p[i].distance<p[j].distance};
func(p priorityQueue)Swap(i,j int){
p[i],p[j]=p[j],p[i];p[i].index=i;p[j].index=j};
func(p*priorityQueue)Push(x any){
item:=x.(*distanceItem);item.index=len(*p);
*p=append(*p,item);};func(p*priorityQueue)Pop()any{
old:=*p;n:=len(old);item:=old[n-1];
*p=old[:n-1];return item;};
/* ShortestWeighted 使用 Dijkstra 和边属性 weight 求最短路径。 */ func(t*Traverser)ShortestWeighted(from,
to uint64,options Options)(PathResult,
error){if _,err:=t.Graph.Vertex(from);
err!=nil{return PathResult{},err;};if _,err:=t.Graph.Vertex(to);
err!=nil{return PathResult{},err;};
distance:=map[uint64]float64{};for _,
v:=range t.Graph.Vertices(){
distance[v.ID]=math.Inf(1);};
distance[from]=0;previous:=map[uint64]uint64{
};queue:=priorityQueue{&distanceItem{
id:from,distance:0}};heap.Init(&queue);
visited:=map[uint64]bool{};for queue.Len()>0{
item:=heap.Pop(&queue).(*distanceItem);
if visited[item.id]{continue;};
visited[item.id]=true;if item.id==to{
return PathResult{Found:true,
Distance:item.distance,Path:buildPath(previous,
from,to)},nil;};neighbors,err:=t.Neighbors(item.id,
options);if err!=nil{return PathResult{
},err;};for next,edges:=range neighbors{
best:=math.Inf(1);for _,edge:=range edges{
weight,weightErr:=edge.Weight("weight");
if weightErr!=nil{return PathResult{
},weightErr;};if weight<0{return PathResult{
},graph.ErrConflict;};if weight<best{
best=weight;};};candidate:=item.distance+best;
if candidate<distance[next]{
distance[next]=candidate;previous[next]=item.id;
heap.Push(&queue,&distanceItem{id:next,
distance:candidate});};};};return PathResult{
Found:false},nil;};func buildPath(previous map[uint64]uint64,
from,to uint64)[]uint64{path:=[]uint64{
to};for path[len(path)-1]!=from{
parent,ok:=previous[path[len(path)-1]];
if!ok{return nil;};path=append(path,
parent);};for left,right:=0,len(path)-1;
left<right;left,right=left+1,right-1{
path[left],path[right]=path[right],
path[left];};return path;};
