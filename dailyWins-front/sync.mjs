const fields=['name','note','done'];
const changed=(a,b)=>!b||a.date!==b.date||fields.some(f=>a[f]!==b[f]);

// 一次仅发送一个请求。失败保留原请求 ID 与内容，直到确认提交结果。
export class TaskSync {
 constructor(send,onChange=()=>{}){this.send=send;this.onChange=onChange;this.tasks=[];this.saved=new Map();this.pending=null;this.running=null;this.error=null;this.revision=0;}
 setTasks(tasks){this.tasks=tasks;this.onChange();}
 merge(items,from,to,queryRevision=this.revision){
  // 查询开始之后有写入成功，则整个快照（包括缺失项）可能过时，交给调用者重新读取。
  if(queryRevision!==this.revision)return false;
  const local=new Map(this.tasks.map(t=>[t.id,t]));const incoming=new Set(items.map(t=>t.id));
  if(from&&to)for(const [id,s] of this.saved){
   if(this.pending?.payload.id!==id&&!s.deleted&&s.date>=from&&s.date<=to&&!incoming.has(id)&&local.has(id)&&!changed(local.get(id),s)){
    local.delete(id);this.saved.delete(id);
   }
  }
  for(const item of items){
   const t={...item,note:item.note||'',done:!!item.done,deleted:!!item.deleted};const old=this.saved.get(t.id);
   if(old&&t.version<old.version)continue;
   // 已加载的本地编辑、删除与飞行中操作始终优先，避免月历刷新覆盖输入。
   if(old&&(!local.has(t.id)||changed(local.get(t.id),old)))continue;
   if(this.pending?.payload.id===t.id)continue;
   this.saved.set(t.id,{...t});if(!t.deleted)local.set(t.id,{...t});
  }
  this.tasks=[...local.values()];this.onChange();return true;
 }
 next(){
  for(const t of this.tasks){
   const s=this.saved.get(t.id);const base={id:t.id,requestId:crypto.randomUUID()};
   if(!s)return {action:'create',payload:{...base,date:t.date,name:t.name,note:t.note,done:t.done}};
   base.version=s.version;
   if(s.deleted)return {action:'restore',payload:base};
   if(t.date!==s.date){
    if(s.done)return {action:'update',payload:{...base,name:t.name,note:t.note,done:false}};
    return {action:'move',payload:{...base,date:t.date}};
   }
   if(fields.some(f=>t[f]!==s[f]))return {action:'update',payload:{...base,name:t.name,note:t.note,done:t.done}};
  }
  const ids=new Set(this.tasks.map(t=>t.id));
  for(const s of this.saved.values())if(!s.deleted&&!ids.has(s.id))return {action:'delete',payload:{id:s.id,version:s.version,requestId:crypto.randomUUID()}};
  return null;
 }
 get dirty(){return !!this.pending||!!this.next();}
 flush(){
  if(this.running)return this.running;
  this.error=null;
  this.running=this.drain().finally(()=>{this.running=null;this.onChange();});
  this.onChange();return this.running;
 }
 async drain(){
  try{
   while(this.pending||(this.pending=this.next())){
    const {action,payload}=this.pending;
    const {task}=await this.send(action,payload);
    this.revision++;
    this.saved.set(task.id,{...task,note:task.note||'',done:!!task.done,deleted:!!task.deleted});
    const local=this.tasks.find(t=>t.id===task.id);
    if(local){
     // 仅规范化仍与发送快照相同的字段；保存期间的新输入不能被旧响应覆盖。
     for(const field of [...fields,'date'])if(Object.hasOwn(payload,field)&&local[field]===payload[field])local[field]=task[field];
     local.version=task.version;local.createdAt=task.createdAt;local.updatedAt=task.updatedAt;
    }
    this.pending=null;this.onChange();
   }
  }catch(error){if(error.status===400)this.pending=null;this.error=error;this.onChange();}
 }
}
