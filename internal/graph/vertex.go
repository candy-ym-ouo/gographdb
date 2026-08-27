package graph;import("fmt";"sort";
);
/* Vertex 表示带多标签和类型化属性的节点。 */ type Vertex struct{
ID uint64 `json:"id"`;Labels[]string `json:"labels"`;
Properties map[string]PropertyValue `json:"properties"`;
};
/* NewVertex 创建并验证节点。 */ func NewVertex(id uint64,
labels[]string,properties map[string]PropertyValue)(*Vertex,
error){if id==0{return nil,fmt.Errorf("vertex id must be positive");
};seen:=map[string]struct{}{};
clean:=make([]string,0,len(labels));
for _,label:=range labels{if label==""{
return nil,fmt.Errorf("label cannot be empty");
};if _,ok:=seen[label];!ok{seen[label]=struct{
}{};clean=append(clean,label);};};
sort.Strings(clean);props:=make(map[string]PropertyValue,
len(properties));for k,value:=range properties{
if k==""{return nil,fmt.Errorf("property name cannot be empty");
};n,err:=value.Normalize();if err!=nil{
return nil,fmt.Errorf("property %s: %w",
k,err);};props[k]=n;};return&Vertex{
ID:id,Labels:clean,Properties:props},
nil;};
/* Clone 返回可安全交给调用方修改的副本。 */ func(v*Vertex)Clone()*Vertex{
if v==nil{return nil;};out:=&Vertex{
ID:v.ID,Labels:append([]string(nil),
v.Labels...),Properties:map[string]PropertyValue{
}};for k,p:=range v.Properties{out.Properties[k]=p;
};return out;};
/* HasLabel 判断节点是否包含标签。 */ func(v*Vertex)HasLabel(label string)bool{
for _,current:=range v.Labels{if current==label{
return true;};};return false;};
/* SetProperty 设置一个经过标准化的属性。 */ func(v*Vertex)SetProperty(key string,
value any)error{if key==""{return fmt.Errorf("property name cannot be empty");
};p,err:=NewPropertyValue(value);
if err!=nil{return err;};if v.Properties==nil{
v.Properties=map[string]PropertyValue{
};};v.Properties[key]=p;return nil;
};
/* DeleteProperty 删除属性，返回属性是否原本存在。 */ func(v*Vertex)DeleteProperty(key string)bool{
if _,ok:=v.Properties[key];!ok{
return false;};delete(v.Properties,
key);return true;};
/* PlainProperties 转成前端友好的标量映射。 */ func(v*Vertex)PlainProperties()map[string]any{
out:=make(map[string]any,len(v.Properties));
for k,p:=range v.Properties{out[k]=p.Plain();
};return out;};
