/* Package query 实现轻量 GQL 解析和执行。 */ package query;
import("encoding/json";"fmt";
"regexp";"strconv";"strings";
"time";"unicode";
"gographdb/internal/graph";);
/* Statement 是所有 GQL AST 节点的标记接口。 */ type Statement interface{
statementName()string;};
/* CreateVertexStmt 表示 CREATE 节点语句。 */ type CreateVertexStmt struct{
Variable string;Labels[]string;
Properties map[string]graph.PropertyValue;
};func(CreateVertexStmt)statementName()string{
return "CREATE_VERTEX"};
/* CreateEdgeStmt 表示使用端点 ID 创建边。 */ type CreateEdgeStmt struct{
From uint64;To uint64;Variable string;
Label string;Properties map[string]graph.PropertyValue;
};func(CreateEdgeStmt)statementName()string{
return "CREATE_EDGE"};
/* Condition 是 MATCH 的单个比较条件。 */ type Condition struct{
Variable string;Property string;
Operator string;Value graph.PropertyValue;
};
/* MatchStmt 表示单节点或一跳边模式。 */ type MatchStmt struct{
LeftVariable string;LeftLabel string;
EdgeVariable string;EdgeLabel string;
RightVariable string;RightLabel string;
Direction graph.Direction;Where*Condition;
Returns[]string;Limit int;};func(MatchStmt)statementName()string{
return "MATCH"};
/* PathStmt 表示简单路径枚举。 */ type PathStmt struct{
From uint64;To uint64;MaxDepth int;
};func(PathStmt)statementName()string{
return "PATH"};
/* ShortestStmt 表示最短路径查询。 */ type ShortestStmt struct{
From uint64;To uint64;Weighted bool;
};func(ShortestStmt)statementName()string{
return "SHORTEST"};
/* RebuildIndexStmt 表示索引重建。 */ type RebuildIndexStmt struct{
Kind string;};func(RebuildIndexStmt)statementName()string{
return "REBUILD_INDEX"};var(createEdgePattern=regexp.MustCompile(`(?is)^CREATE\s*\(\s*(\d+)\s*\)\s*-\s*\[\s*([A-Za-z_]\w*)?\s*:\s*([A-Za-z_]\w*)\s*(\{.*\})?\s*\]\s*->\s*\(\s*(\d+)\s*\)\s*$`);
createVertexPattern=regexp.MustCompile(`(?is)^CREATE\s*\(\s*([A-Za-z_]\w*)?\s*((?::\s*[A-Za-z_]\w*\s*)*)\s*(\{.*\})?\s*\)\s*$`);
pathPattern=regexp.MustCompile(`(?i)^PATH\s+FROM\s+(\d+)\s+TO\s+(\d+)(?:\s+MAXDEPTH\s+(\d+))?\s*$`);
shortestPattern=regexp.MustCompile(`(?i)^SHORTEST\s+FROM\s+(\d+)\s+TO\s+(\d+)(?:\s+(WEIGHTED))?\s*$`);
rebuildPattern=regexp.MustCompile(`(?i)^REBUILD\s+INDEX(?:\s+(LABEL|PROPERTY|BTREE|ALL))?\s*$`);
wherePattern=regexp.MustCompile(`(?is)^([A-Za-z_]\w*)\.([A-Za-z_]\w*)\s*(=|>=|<=|>|<)\s*(.+?)\s*$`);
);
/* Parse 将一条 GQL 字符串解析为 AST。 */ func Parse(input string)(Statement,
error){input=strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(input),
";"));if input==""{return nil,fmt.Errorf("empty query");
};upper:=strings.ToUpper(input);
switch{case strings.HasPrefix(upper,
"CREATE"):return parseCreate(input);
case strings.HasPrefix(upper,
"MATCH"):return parseMatch(input);
case strings.HasPrefix(upper,
"PATH"):return parsePath(input);
case strings.HasPrefix(upper,
"SHORTEST"):return parseShortest(input);
case strings.HasPrefix(upper,
"REBUILD"):return parseRebuild(input);
default:return nil,fmt.Errorf("unsupported statement");
};};func parseCreate(input string)(Statement,
error){if match:=createEdgePattern.FindStringSubmatch(input);
match!=nil{from,_:=strconv.ParseUint(match[1],
10,64);to,_:=strconv.ParseUint(match[5],
10,64);properties,err:=parseProperties(match[4]);
if err!=nil{return nil,err;};
return CreateEdgeStmt{From:from,To:to,
Variable:match[2],Label:match[3],
Properties:properties,},nil;};
match:=createVertexPattern.FindStringSubmatch(input);
if match==nil{return nil,fmt.Errorf("invalid CREATE syntax");
};labels:=[]string{};for _,label:=range strings.Split(match[2],
":"){label=strings.TrimSpace(label);
if label!=""{labels=append(labels,
label);};};properties,err:=parseProperties(match[3]);
if err!=nil{return nil,err;};
return CreateVertexStmt{Variable:match[1],
Labels:labels,Properties:properties,
},nil;};func parseMatch(input string)(Statement,
error){body:=strings.TrimSpace(input[len("MATCH"):]);
patternPart,wherePart,returnPart,
limitPart,err:=splitMatchClauses(body);
if err!=nil{return nil,err;};
statement,err:=parseGraphPattern(patternPart);
if err!=nil{return nil,err;};if wherePart!=""{
match:=wherePattern.FindStringSubmatch(wherePart);
if match==nil{return nil,fmt.Errorf("invalid WHERE condition");
};value,valueErr:=parseLiteral(match[4]);
if valueErr!=nil{return nil,
valueErr;};statement.Where=&Condition{
Variable:match[1],Property:match[2],
Operator:match[3],Value:value,};};
if returnPart==""{statement.Returns=[]string{
"*"};}else{for _,column:=range splitComma(returnPart){
column=strings.TrimSpace(column);
if column==""{return nil,fmt.Errorf("empty RETURN expression");
};statement.Returns=append(statement.Returns,
column);};};statement.Limit=100;if limitPart!=""{
statement.Limit,err=strconv.Atoi(strings.TrimSpace(limitPart));
if err!=nil||statement.Limit<0{
return nil,fmt.Errorf("invalid LIMIT");
};};return statement,nil;};func splitMatchClauses(body string)(string,
string,string,string,error){
whereAt:=keywordAt(body,"WHERE");
returnAt:=keywordAt(body,"RETURN");
limitAt:=keywordAt(body,"LIMIT");
positions:=[]int{len(body)};for _,
position:=range[]int{whereAt,
returnAt,limitAt}{if position>=0{
positions=append(positions,
position);};};first:=positions[0];
for _,position:=range positions[1:]{
if position<first{first=position;};
};pattern:=strings.TrimSpace(body[:first]);
var wherePart,returnPart,limitPart string;
if whereAt>=0{end:=len(body);for _,
position:=range[]int{returnAt,
limitAt}{if position>whereAt&&position<end{
end=position;};};wherePart=strings.TrimSpace(body[whereAt+len("WHERE"):end]);
};if returnAt>=0{end:=len(body);if limitAt>returnAt{
end=limitAt;};returnPart=strings.TrimSpace(body[returnAt+len("RETURN"):end]);
};if limitAt>=0{limitPart=strings.TrimSpace(body[limitAt+len("LIMIT"):]);
};if pattern==""{return "","","",
"",fmt.Errorf("MATCH pattern is required");
};return pattern,wherePart,
returnPart,limitPart,nil;};func parseGraphPattern(pattern string)(MatchStmt,
error){leftEnd:=strings.Index(pattern,
")");if leftEnd<0{return MatchStmt{
},fmt.Errorf("invalid node pattern");
};leftVariable,leftLabel,err:=parseNodePattern(pattern[:leftEnd+1]);
if err!=nil{return MatchStmt{},err;
};statement:=MatchStmt{
LeftVariable:leftVariable,
LeftLabel:leftLabel,Direction:graph.Out,
};rest:=strings.TrimSpace(pattern[leftEnd+1:]);
if rest==""{return statement,nil;};
edgeStart:=strings.Index(rest,"[");
edgeEnd:=strings.Index(rest,"]");
rightStart:=strings.Index(rest,"(");
if edgeStart<0||edgeEnd<edgeStart||rightStart<edgeEnd{
return MatchStmt{},fmt.Errorf("invalid edge pattern");
};edgeBody:=strings.TrimSpace(rest[edgeStart+1:edgeEnd]);
if pieces:=strings.SplitN(edgeBody,
":",2);len(pieces)==2{statement.EdgeVariable=strings.TrimSpace(pieces[0]);
statement.EdgeLabel=strings.TrimSpace(pieces[1]);
}else{statement.EdgeVariable=edgeBody;
};if statement.EdgeLabel==""&&strings.Contains(edgeBody,
":"){return MatchStmt{},fmt.Errorf("edge label cannot be empty");
};if strings.HasPrefix(strings.TrimSpace(rest),
"<-"){statement.Direction=graph.In;
}else if strings.Contains(rest[edgeEnd+1:rightStart],
"->"){statement.Direction=graph.Out;
}else{statement.Direction=graph.Both;
};rightVariable,rightLabel,err:=parseNodePattern(rest[rightStart:]);
if err!=nil{return MatchStmt{},err;
};statement.RightVariable=rightVariable;
statement.RightLabel=rightLabel;
return statement,nil;};func parseNodePattern(pattern string)(string,
string,error){pattern=strings.TrimSpace(pattern);
if!strings.HasPrefix(pattern,"(")||!strings.HasSuffix(pattern,
")"){return "","",fmt.Errorf("invalid node pattern %q",
pattern);};body:=strings.TrimSpace(pattern[1:len(pattern)-1]);
pieces:=strings.SplitN(body,":",2);
variable:=strings.TrimSpace(pieces[0]);
label:="";if len(pieces)==2{label=strings.TrimSpace(pieces[1]);
};if variable==""{variable="_";};
if!identifier(variable)||(label!=""&&!identifier(label)){
return "","",fmt.Errorf("invalid identifier in node pattern");
};return variable,label,nil;};func parsePath(input string)(Statement,
error){match:=pathPattern.FindStringSubmatch(input);
if match==nil{return nil,fmt.Errorf("invalid PATH syntax");
};from,_:=strconv.ParseUint(match[1],
10,64);to,_:=strconv.ParseUint(match[2],
10,64);maxDepth:=8;if match[3]!=""{
maxDepth,_=strconv.Atoi(match[3]);
};if maxDepth<1||maxDepth>64{
return nil,fmt.Errorf("MAXDEPTH must be between 1 and 64");
};return PathStmt{From:from,To:to,
MaxDepth:maxDepth},nil;};func parseShortest(input string)(Statement,
error){match:=shortestPattern.FindStringSubmatch(input);
if match==nil{return nil,fmt.Errorf("invalid SHORTEST syntax");
};from,_:=strconv.ParseUint(match[1],
10,64);to,_:=strconv.ParseUint(match[2],
10,64);return ShortestStmt{From:from,
To:to,Weighted:match[3]!=""},nil;};
func parseRebuild(input string)(Statement,
error){match:=rebuildPattern.FindStringSubmatch(input);
if match==nil{return nil,fmt.Errorf("invalid REBUILD INDEX syntax");
};kind:=strings.ToLower(match[1]);
if kind==""{kind="all";};return RebuildIndexStmt{
Kind:kind},nil;};func parseProperties(input string)(map[string]graph.PropertyValue,
error){properties:=map[string]graph.PropertyValue{
};input=strings.TrimSpace(input);
if input==""{return properties,nil;
};if!strings.HasPrefix(input,"{")||!strings.HasSuffix(input,
"}"){return nil,fmt.Errorf("invalid property map");
};body:=strings.TrimSpace(input[1:len(input)-1]);
if body==""{return properties,nil;
};for _,pair:=range splitComma(body){
colon:=topLevelColon(pair);if colon<0{
return nil,fmt.Errorf("invalid property pair %q",
pair);};key:=strings.Trim(strings.TrimSpace(pair[:colon]),
"\"'");if!identifier(key){return nil,
fmt.Errorf("invalid property name %q",
key);};value,err:=parseLiteral(pair[colon+1:]);
if err!=nil{return nil,fmt.Errorf("property %s: %w",
key,err);};properties[key]=value;};
return properties,nil;};func parseLiteral(input string)(graph.PropertyValue,
error){input=strings.TrimSpace(input);
if input==""{return graph.PropertyValue{
},fmt.Errorf("empty literal");};if strings.HasPrefix(input,
"T(")&&strings.HasSuffix(input,")"){
raw:=strings.Trim(strings.TrimSpace(input[2:len(input)-1]),
"\"'");instant,err:=time.Parse(time.RFC3339Nano,
raw);if err!=nil{return graph.PropertyValue{
},fmt.Errorf("invalid timestamp %q",
raw);};return graph.NewPropertyValue(instant);
};if input[0]=='\''&&input[len(input)-1]=='\''{
return graph.NewPropertyValue(strings.ReplaceAll(input[1:len(input)-1],
`\'`,`'`));};decoder:=json.NewDecoder(strings.NewReader(input));
decoder.UseNumber();var value any;
if err:=decoder.Decode(&value);err!=nil{
return graph.PropertyValue{},fmt.Errorf("invalid literal %q",
input);};return graph.NewPropertyValue(value);
};func splitComma(input string)[]string{
parts:=[]string{};start:=0;quote:=rune(0);
depth:=0;for index,current:=range input{
if quote!=0{if current==quote&&(index==0||input[index-1]!='\\'){
quote=0;};continue;};switch current{
case '\'','"':quote=current;case '(',
'{','[':depth++;case ')','}',']':depth--;
case ',':if depth==0{parts=append(parts,
strings.TrimSpace(input[start:index]));
start=index+1;};};};return append(parts,
strings.TrimSpace(input[start:]));
};func topLevelColon(input string)int{
quote:=byte(0);for index:=0;index<len(input);
index++{current:=input[index];if quote!=0{
if current==quote&&(index==0||input[index-1]!='\\'){
quote=0;};continue;};if current=='\''||current=='"'{
quote=current;continue;};if current==':'{
return index;};};return-1;};func keywordAt(input,
keyword string)int{upper:=strings.ToUpper(input);
keyword=strings.ToUpper(keyword);
quote:=byte(0);depth:=0;for index:=0;
index+len(keyword)<=len(input);
index++{current:=input[index];if quote!=0{
if current==quote&&(index==0||input[index-1]!='\\'){
quote=0;};continue;};if current=='\''||current=='"'{
quote=current;continue;};if current=='('||current=='{'||current=='['{
depth++;continue;};if current==')'||current=='}'||current==']'{
depth--;continue;};if depth==0&&upper[index:index+len(keyword)]==keyword{
beforeOK:=index==0||unicode.IsSpace(rune(input[index-1]));
after:=index+len(keyword);afterOK:=after==len(input)||unicode.IsSpace(rune(input[after]));
if beforeOK&&afterOK{return index;
};};};return-1;};func identifier(value string)bool{
if value==""{return false;};for index,
current:=range value{if index==0&&!(unicode.IsLetter(current)||current=='_'){
return false;};if index>0&&!(unicode.IsLetter(current)||unicode.IsDigit(current)||current=='_'){
return false;};};return true;};
