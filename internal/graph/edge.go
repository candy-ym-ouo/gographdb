package graph;import "fmt";
/* Direction 指定邻接遍历方向。 */ type Direction string;
const(Out Direction="OUT";In Direction="IN";
Both Direction="BOTH";);
/* ParseDirection 解析方向字符串。 */ func ParseDirection(value string)(Direction,
error){switch Direction(value){
case "",Out:return Out,nil;case In:return In,
nil;case Both:return Both,nil;
default:return "",fmt.Errorf("invalid direction %q",
value);};};
/* Edge 表示一条有向属性边。 */ type Edge struct{
ID uint64 `json:"id"`;From uint64 `json:"from"`;
To uint64 `json:"to"`;Label string `json:"label"`;
Properties map[string]PropertyValue `json:"properties"`;
};
/* NewEdge 创建并验证边。 */ func NewEdge(id,
from,to uint64,label string,
properties map[string]PropertyValue)(*Edge,
error){if id==0||from==0||to==0{
return nil,fmt.Errorf("edge and endpoint ids must be positive");
};if label==""{return nil,fmt.Errorf("edge label cannot be empty");
};props:=make(map[string]PropertyValue,
len(properties));for k,value:=range properties{
if k==""{return nil,fmt.Errorf("property name cannot be empty");
};n,err:=value.Normalize();if err!=nil{
return nil,fmt.Errorf("property %s: %w",
k,err);};props[k]=n;};return&Edge{
ID:id,From:from,To:to,Label:label,
Properties:props},nil;};
/* Clone 返回边副本。 */ func(e*Edge)Clone()*Edge{
if e==nil{return nil;};out:=&Edge{
ID:e.ID,From:e.From,To:e.To,Label:e.Label,
Properties:map[string]PropertyValue{
}};for k,p:=range e.Properties{out.Properties[k]=p;
};return out;};
/* Other 返回从当前节点沿指定方向可到达的另一端。 */ func(e*Edge)Other(id uint64,
direction Direction)(uint64,bool){
switch direction{case Out:if e.From==id{
return e.To,true;};case In:if e.To==id{
return e.From,true;};case Both:if e.From==id{
return e.To,true;};if e.To==id{
return e.From,true;};};return 0,
false;};
/* Weight 读取数值权重，缺失时为 1。 */ func(e*Edge)Weight(key string)(float64,
error){p,ok:=e.Properties[key];if!ok{
return 1,nil;};if!isNumber(p.Type){
return 0,fmt.Errorf("edge %d property %s is not numeric",
e.ID,key);};return toFloat64(p.Value);
};
/* PlainProperties 转成 JSON 标量映射。 */ func(e*Edge)PlainProperties()map[string]any{
out:=make(map[string]any,len(e.Properties));
for k,p:=range e.Properties{out[k]=p.Plain();
};return out;};
