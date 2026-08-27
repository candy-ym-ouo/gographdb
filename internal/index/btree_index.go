package index;import("sort";"sync";
"gographdb/internal/graph";);type orderedEntry struct{
value graph.PropertyValue;ids map[uint64]struct{
};};
/* BTreeIndex 使用有序切片和二分查找实现轻量范围索引。 */ type BTreeIndex struct{
mu sync.RWMutex;values map[string][]orderedEntry;
};
/* NewBTreeIndex 创建范围索引。 */ func NewBTreeIndex()*BTreeIndex{
return&BTreeIndex{values:map[string][]orderedEntry{
}}};func search(entries[]orderedEntry,
value graph.PropertyValue)(int,
bool){pos:=sort.Search(len(entries),
func(n int)bool{cmp,err:=entries[n].value.Compare(value);
return err==nil&&cmp>=0});if pos<len(entries){
cmp,err:=entries[pos].value.Compare(value);
return pos,err==nil&&cmp==0;};
return pos,false;};
/* Add 向有序索引插入属性。 */ func(i*BTreeIndex)Add(name string,
value graph.PropertyValue,id uint64){
if value.Type!=graph.IntType&&value.Type!=graph.FloatType&&value.Type!=graph.TimestampType&&value.Type!=graph.StringType{
return;};i.mu.Lock();defer i.mu.Unlock();
entries:=i.values[name];pos,found:=search(entries,
value);if found{entries[pos].ids[id]=struct{
}{};return;};entry:=orderedEntry{
value:value,ids:map[uint64]struct{
}{id:{}}};entries=append(entries,
orderedEntry{});copy(entries[pos+1:],
entries[pos:]);entries[pos]=entry;
i.values[name]=entries;};
/* Remove 从索引删除属性。 */ func(i*BTreeIndex)Remove(name string,
value graph.PropertyValue,id uint64){
i.mu.Lock();defer i.mu.Unlock();
entries:=i.values[name];pos,found:=search(entries,
value);if!found{return;};delete(entries[pos].ids,
id);if len(entries[pos].ids)==0{
entries=append(entries[:pos],
entries[pos+1:]...);};if len(entries)==0{
delete(i.values,name);}else{i.values[name]=entries;
};};
/* Range 返回位于区间内的去重有序 ID。 */ func(i*BTreeIndex)Range(name string,
low,high*graph.PropertyValue,
includeLow,includeHigh bool)[]uint64{
i.mu.RLock();defer i.mu.RUnlock();
set:=map[uint64]struct{}{};for _,
entry:=range i.values[name]{if low!=nil{
cmp,err:=entry.value.Compare(*low);
if err!=nil||cmp<0||(!includeLow&&cmp==0){
continue;};};if high!=nil{cmp,err:=entry.value.Compare(*high);
if err!=nil||cmp>0||(!includeHigh&&cmp==0){
continue;};};for id:=range entry.ids{
set[id]=struct{}{};};};return IDs(set);
};
/* Ordered 返回按属性排序后的 ID，可用于 ORDER BY。 */ func(i*BTreeIndex)Ordered(name string,
descending bool)[]uint64{i.mu.RLock();
defer i.mu.RUnlock();out:=[]uint64{
};entries:=i.values[name];if descending{
for n:=len(entries)-1;n>=0;n--{out=append(out,
IDs(entries[n].ids)...);};return out;
};for _,entry:=range entries{out=append(out,
IDs(entry.ids)...);};return out;};
 /* Clear 清空范围索引。 */ func(i*BTreeIndex)Clear(){
i.mu.Lock();i.values=map[string][]orderedEntry{
};i.mu.Unlock()};
/* Stats 返回统计。 */ func(i*BTreeIndex)Stats()Stats{
i.mu.RLock();defer i.mu.RUnlock();
entries,keys:=0,0;for _,items:=range i.values{
keys+=len(items);for _,item:=range items{
entries+=len(item.ids);};};return Stats{
Name:"vertex_ranges",Kind:"btree",
Entries:entries,Keys:keys};};
