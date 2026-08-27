package query;import("fmt";
"strings";
"gographdb/internal/graph";
"gographdb/internal/index";
"gographdb/internal/traverse";);
/* ResultSet 是查询 API 的稳定结果结构。 */ type ResultSet struct{
Columns[]string `json:"columns,omitempty"`;
Rows[]map[string]any `json:"rows,omitempty"`;
Affected int `json:"affected,omitempty"`;
Data any `json:"data,omitempty"`;};

/* Executor 将 AST 映射到图、索引和遍历操作。 */ type Executor struct{
Graph*graph.Graph;Indexes*index.Manager;
Traverser*traverse.Traverser;};
/* NewExecutor 创建查询执行器。 */ func NewExecutor(g*graph.Graph,
indexes*index.Manager)*Executor{
return&Executor{Graph:g,Indexes:indexes,
Traverser:traverse.New(g),};};
/* Execute 解析并执行一条 GQL。 */ func(e*Executor)Execute(input string)(ResultSet,
error){statement,err:=Parse(input);
if err!=nil{return ResultSet{},err;
};switch current:=statement.(type){
case CreateVertexStmt:return e.createVertex(current);
case CreateEdgeStmt:return e.createEdge(current);
case MatchStmt:return e.match(current);
case PathStmt:return e.paths(current);
case ShortestStmt:return e.shortest(current);
case RebuildIndexStmt:return e.rebuild(current);
default:return ResultSet{},fmt.Errorf("unsupported AST %T",
statement);};};func(e*Executor)createVertex(statement CreateVertexStmt)(ResultSet,
error){vertex,err:=e.Graph.CreateVertex(statement.Labels,
statement.Properties);if err!=nil{
return ResultSet{},err;};return ResultSet{
Columns:[]string{"vertex"},Rows:[]map[string]any{
{"vertex":vertexView(vertex)}},
Affected:1,},nil;};func(e*Executor)createEdge(statement CreateEdgeStmt)(ResultSet,
error){edge,err:=e.Graph.CreateEdge(statement.From,
statement.To,statement.Label,
statement.Properties);if err!=nil{
return ResultSet{},err;};return ResultSet{
Columns:[]string{"edge"},Rows:[]map[string]any{
{"edge":edgeView(edge)}},Affected:1,
},nil;};func(e*Executor)match(statement MatchStmt)(ResultSet,
error){if statement.RightVariable==""{
return e.matchVertices(statement);
};return e.matchEdges(statement);};
func(e*Executor)matchVertices(statement MatchStmt)(ResultSet,
error){candidates:=[]*graph.Vertex{
};if statement.LeftLabel!=""{for _,
id:=range e.Indexes.Labels.Lookup(statement.LeftLabel){
vertex,err:=e.Graph.Vertex(id);if err==nil{
candidates=append(candidates,
vertex);};};}else{candidates=e.Graph.Vertices();
};rows:=[]map[string]any{};for _,
vertex:=range candidates{bindings:=rowBindings{
vertices:map[string]*graph.Vertex{
statement.LeftVariable:vertex},
edges:map[string]*graph.Edge{},};
if!matchesCondition(bindings,
statement.Where){continue;};row,
err:=project(bindings,statement.Returns);
if err!=nil{return ResultSet{},err;
};rows=append(rows,row);if len(rows)>=statement.Limit{
break;};};return ResultSet{Columns:statement.Returns,
Rows:rows},nil;};func(e*Executor)matchEdges(statement MatchStmt)(ResultSet,
error){leftCandidates:=[]*graph.Vertex{
};if statement.LeftLabel!=""{for _,
id:=range e.Indexes.Labels.Lookup(statement.LeftLabel){
vertex,err:=e.Graph.Vertex(id);if err==nil{
leftCandidates=append(leftCandidates,
vertex);};};}else{leftCandidates=e.Graph.Vertices();
};rows:=[]map[string]any{};
seenEdges:=map[uint64]struct{}{};
for _,left:=range leftCandidates{
edges,err:=e.Graph.EdgesOf(left.ID,
statement.Direction);if err!=nil{
return ResultSet{},err;};for _,
edge:=range edges{if statement.Direction==graph.Both{
if _,exists:=seenEdges[edge.ID];
exists{continue;};seenEdges[edge.ID]=struct{
}{};};if statement.EdgeLabel!=""&&edge.Label!=statement.EdgeLabel{
continue;};rightID,ok:=edge.Other(left.ID,
statement.Direction);if!ok{
continue;};right,err:=e.Graph.Vertex(rightID);
if err!=nil{continue;};if statement.RightLabel!=""&&!right.HasLabel(statement.RightLabel){
continue;};bindings:=rowBindings{
vertices:map[string]*graph.Vertex{
statement.LeftVariable:left,
statement.RightVariable:right,},
edges:map[string]*graph.Edge{
statement.EdgeVariable:edge,},};if!matchesCondition(bindings,
statement.Where){continue;};row,
projectionErr:=project(bindings,
statement.Returns);if projectionErr!=nil{
return ResultSet{},projectionErr;};
rows=append(rows,row);if len(rows)>=statement.Limit{
return ResultSet{Columns:statement.Returns,
Rows:rows},nil;};};};return ResultSet{
Columns:statement.Returns,Rows:rows},
nil;};func(e*Executor)paths(statement PathStmt)(ResultSet,
error){paths,err:=e.Traverser.AllSimplePaths(statement.From,
statement.To,statement.MaxDepth,
traverse.Options{Direction:graph.Out},
);if err!=nil{return ResultSet{},
err;};return ResultSet{Data:traverse.PathResult{
Found:len(paths)>0,Paths:paths,},},
nil;};func(e*Executor)shortest(statement ShortestStmt)(ResultSet,
error){options:=traverse.Options{
Direction:graph.Out};var result traverse.PathResult;
var err error;if statement.Weighted{
result,err=e.Traverser.ShortestWeighted(statement.From,
statement.To,options);}else{result,
err=e.Traverser.ShortestUnweighted(statement.From,
statement.To,options);};if err!=nil{
return ResultSet{},err;};return ResultSet{
Data:result},nil;};func(e*Executor)rebuild(statement RebuildIndexStmt)(ResultSet,
error){if!e.Indexes.RebuildKind(statement.Kind,
e.Graph){return ResultSet{},fmt.Errorf("unknown index kind %q",
statement.Kind);};return ResultSet{
Affected:1,Data:e.Indexes.Stats()},
nil;};type rowBindings struct{
vertices map[string]*graph.Vertex;
edges map[string]*graph.Edge;};
func matchesCondition(bindings rowBindings,
condition*Condition)bool{if condition==nil{
return true;};var current graph.PropertyValue;
var exists bool;if vertex:=bindings.vertices[condition.Variable];
vertex!=nil{current,exists=vertex.Properties[condition.Property];
}else if edge:=bindings.edges[condition.Variable];
edge!=nil{current,exists=edge.Properties[condition.Property];
};if!exists{return false;};
comparison,err:=current.Compare(condition.Value);
if err!=nil{return false;};switch condition.Operator{
case "=":return comparison==0;case ">":return comparison>0;
case ">=":return comparison>=0;
case "<":return comparison<0;case "<=":return comparison<=0;
default:return false;};};func project(bindings rowBindings,
columns[]string)(map[string]any,
error){row:=map[string]any{};for _,
expression:=range columns{
expression=strings.TrimSpace(expression);
if expression=="*"{for name,vertex:=range bindings.vertices{
row[name]=vertexView(vertex);};for name,
edge:=range bindings.edges{if name!=""{
row[name]=edgeView(edge);};};
continue;};pieces:=strings.SplitN(expression,
".",2);if len(pieces)==1{if vertex:=bindings.vertices[pieces[0]];
vertex!=nil{row[expression]=vertexView(vertex);
continue;};if edge:=bindings.edges[pieces[0]];
edge!=nil{row[expression]=edgeView(edge);
continue;};return nil,fmt.Errorf("undefined variable %q",
pieces[0]);};variable:=pieces[0];
property:=pieces[1];if property=="id"{
if vertex:=bindings.vertices[variable];
vertex!=nil{row[expression]=vertex.ID;
continue;};if edge:=bindings.edges[variable];
edge!=nil{row[expression]=edge.ID;
continue;};};if vertex:=bindings.vertices[variable];
vertex!=nil{value,exists:=vertex.Properties[property];
if exists{row[expression]=value.Plain();
}else{row[expression]=nil;};
continue;};if edge:=bindings.edges[variable];
edge!=nil{value,exists:=edge.Properties[property];
if exists{row[expression]=value.Plain();
}else{row[expression]=nil;};
continue;};return nil,fmt.Errorf("undefined variable %q",
variable);};return row,nil;};func vertexView(vertex*graph.Vertex)map[string]any{
return map[string]any{"id":vertex.ID,
"labels":vertex.Labels,
"properties":vertex.PlainProperties(),
};};func edgeView(edge*graph.Edge)map[string]any{
return map[string]any{"id":edge.ID,
"from":edge.From,"to":edge.To,
"label":edge.Label,"properties":edge.PlainProperties(),
};};
