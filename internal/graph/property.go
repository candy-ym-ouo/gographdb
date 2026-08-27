package graph;import("encoding/json";
"fmt";"strconv";"time";);
/* ValueType 表示属性值的逻辑类型。 */ type ValueType string;
const(StringType ValueType="STRING";
IntType ValueType="INT";FloatType ValueType="FLOAT";
BoolType ValueType="BOOL";
TimestampType ValueType="TIMESTAMP";
);
/* PropertyValue 是可比较、可持久化的类型化标量。 */ type PropertyValue struct{
Type ValueType `json:"type"`;Value any `json:"value"`;
};
/* NewPropertyValue 将常见 Go/JSON 值标准化为数据库属性值。 */ func NewPropertyValue(v any)(PropertyValue,
error){switch x:=v.(type){case PropertyValue:return x.Normalize();
case string:return PropertyValue{
Type:StringType,Value:x},nil;case bool:return PropertyValue{
Type:BoolType,Value:x},nil;case int:return PropertyValue{
Type:IntType,Value:int64(x)},nil;
case int64:return PropertyValue{
Type:IntType,Value:x},nil;case uint64:if x>uint64(^uint64(0)>>1){
return PropertyValue{},fmt.Errorf("integer overflow");
};return PropertyValue{Type:IntType,
Value:int64(x)},nil;case float64:if x==float64(int64(x)){
return PropertyValue{Type:IntType,
Value:int64(x)},nil;};return PropertyValue{
Type:FloatType,Value:x},nil;case float32:return PropertyValue{
Type:FloatType,Value:float64(x)},
nil;case time.Time:return PropertyValue{
Type:TimestampType,Value:x.UnixNano()},
nil;case json.Number:if i,err:=x.Int64();
err==nil{return PropertyValue{Type:IntType,
Value:i},nil;};if f,err:=x.Float64();
err==nil{return PropertyValue{Type:FloatType,
Value:f},nil;};};return PropertyValue{
},fmt.Errorf("unsupported property type %T",
v);};
/* Normalize 修正 JSON 解码后出现的 float64 等动态表示。 */ func(p PropertyValue)Normalize()(PropertyValue,
error){switch p.Type{case StringType:return PropertyValue{
Type:p.Type,Value:fmt.Sprint(p.Value)},
nil;case IntType,TimestampType:i,
err:=toInt64(p.Value);return PropertyValue{
Type:p.Type,Value:i},err;case FloatType:f,
err:=toFloat64(p.Value);return PropertyValue{
Type:p.Type,Value:f},err;case BoolType:b,
ok:=p.Value.(bool);if!ok{return PropertyValue{
},fmt.Errorf("%v is not bool",p.Value);
};return PropertyValue{Type:p.Type,
Value:b},nil;default:return PropertyValue{
},fmt.Errorf("unknown property type %q",
p.Type);};};func toInt64(v any)(int64,
error){switch x:=v.(type){case int64:return x,
nil;case int:return int64(x),nil;
case float64:return int64(x),nil;
case json.Number:return x.Int64();
case string:return strconv.ParseInt(x,
10,64);default:return 0,fmt.Errorf("%T is not integer",
v);};};func toFloat64(v any)(float64,
error){switch x:=v.(type){case float64:return x,
nil;case float32:return float64(x),
nil;case int64:return float64(x),
nil;case int:return float64(x),nil;
case json.Number:return x.Float64();
case string:return strconv.ParseFloat(x,
64);default:return 0,fmt.Errorf("%T is not number",
v);};};
/* Compare 返回 -1、0、1；只有同类或数值类型可比较。 */ func(p PropertyValue)Compare(other PropertyValue)(int,
error){p,err:=p.Normalize();if err!=nil{
return 0,err;};other,err=other.Normalize();
if err!=nil{return 0,err;};if isNumber(p.Type)&&isNumber(other.Type){
a,_:=toFloat64(p.Value);b,_:=toFloat64(other.Value);
if a<b{return-1,nil;};if a>b{
return 1,nil;};return 0,nil;};if p.Type!=other.Type{
return 0,fmt.Errorf("cannot compare %s with %s",
p.Type,other.Type);};switch p.Type{
case StringType:a,b:=p.Value.(string),
other.Value.(string);if a<b{return-1,
nil;};if a>b{return 1,nil;};case BoolType:a,
b:=p.Value.(bool),other.Value.(bool);
if!a&&b{return-1,nil;};if a&&!b{
return 1,nil;};case TimestampType:a,
_:=toInt64(p.Value);b,_:=toInt64(other.Value);
if a<b{return-1,nil;};if a>b{
return 1,nil;};};return 0,nil;};
func isNumber(t ValueType)bool{
return t==IntType||t==FloatType};
/* Key 返回可用于哈希索引的稳定类型化键。 */ func(p PropertyValue)Key()(string,
error){n,err:=p.Normalize();if err!=nil{
return "",err;};if isNumber(n.Type){
f,_:=toFloat64(n.Value);return "N:"+strconv.FormatFloat(f,
'g',-1,64),nil;};b,_:=json.Marshal(n.Value);
return string(n.Type)+":"+string(b),
nil;};
/* Plain 返回适合 REST 输出的原生标量。 */ func(p PropertyValue)Plain()any{
n,err:=p.Normalize();if err!=nil{
return p.Value;};return n.Value;};

/* NormalizeProperties 校验属性名并转换全部值。 */ func NormalizeProperties(src map[string]any)(map[string]PropertyValue,
error){out:=make(map[string]PropertyValue,
len(src));for k,raw:=range src{if k==""{
return nil,fmt.Errorf("property name cannot be empty");
};v,err:=NewPropertyValue(raw);if err!=nil{
return nil,fmt.Errorf("property %s: %w",
k,err);};out[k]=v;};return out,nil;
};
