import test from 'node:test';
import assert from 'node:assert/strict';
import {TaskSync} from '../sync.mjs';

const task=()=>({id:crypto.randomUUID(),date:'2026-10-17',name:'第一版',note:'',done:false});
test('response loss retries the same request without duplicate creation',async()=>{
 const calls=[];let fail=true;
 const sync=new TaskSync(async(action,payload)=>{calls.push({...payload});if(fail){fail=false;throw Error('断网');}return {task:{...payload,version:1}};});
 const t=task();sync.setTasks([t]);await sync.flush();assert.ok(sync.error);assert.equal(sync.tasks[0].name,'第一版');
 await sync.flush();assert.equal(calls[0].requestId,calls[1].requestId);assert.equal(sync.dirty,false);
});
test('typing while saving is serialized and old response cannot overwrite newer input',async()=>{
 let resolve;const calls=[];let server={};
 const sync=new TaskSync(async(action,payload)=>{calls.push({...payload});if(calls.length===1)await new Promise(r=>resolve=r);server={...server,...payload,version:calls.length};return {task:server};});
 const t=task();sync.setTasks([t]);const saving=sync.flush();await Promise.resolve();t.name='第二版';sync.setTasks([t]);resolve();await saving;
 assert.equal(calls.length,2);assert.equal(calls[1].version,1);assert.equal(calls[1].name,'第二版');assert.equal(sync.tasks[0].name,'第二版');assert.equal(sync.dirty,false);
});
test('delete then undo restores the same task ID',async()=>{
 const calls=[];const t={...task(),version:1};
 const sync=new TaskSync(async(action,payload)=>{calls.push(action);return {task:{...t,version:calls.length+1,deleted:action==='delete'}};});
 sync.merge([t]);sync.setTasks([]);await sync.flush();sync.setTasks([t]);await sync.flush();
 assert.deepEqual(calls,['delete','restore']);assert.equal(sync.tasks[0].id,t.id);
});
test('failed month loading does not clear drafts and merge preserves edits',()=>{
 const sync=new TaskSync(()=>{});const t={...task(),version:1};sync.merge([t]);sync.tasks[0].name='未保存';sync.merge([{...t,name:'服务器'}]);assert.equal(sync.tasks[0].name,'未保存');
});
test('server-normalized names settle without repeated updates',async()=>{
 const t={...task(),version:1};let calls=0;
 const sync=new TaskSync(async(action,payload)=>{calls++;if(calls>2)throw Error('重复写入');return {task:{...t,...payload,name:payload.name.trim(),version:calls+1}};});
 sync.merge([t]);sync.tasks[0].name='新的名字 ';await sync.flush();assert.equal(sync.error,null);assert.equal(calls,1);assert.equal(sync.tasks[0].name,'新的名字');assert.equal(sync.dirty,false);
});
test('a late month snapshot cannot undo a successful mutation or remove a new task',async()=>{
 const t={...task(),version:1};
 const sync=new TaskSync(async(action,payload)=>({task:{...t,...payload,version:2}}));sync.merge([t]);const queryRevision=sync.revision;
 sync.tasks[0].name='新内容';await sync.flush();assert.equal(sync.merge([t],t.date,t.date,queryRevision),false);assert.equal(sync.tasks[0].name,'新内容');
 assert.equal(sync.merge([],t.date,t.date,queryRevision),false);assert.equal(sync.tasks.length,1);
});
test('moving then completing is persisted in a legal order',async()=>{
 const original={...task(),version:1};let server={...original};const calls=[];
 const sync=new TaskSync(async(action,payload)=>{calls.push(action);if(action==='move'&&server.done)throw Error('已完成不能移动');server={...server,...payload,version:server.version+1};return {task:server};});
 sync.merge([original]);sync.tasks[0].date='2027-01-01';sync.tasks[0].done=true;await sync.flush();assert.equal(sync.error,null);assert.deepEqual(calls,['move','update']);assert.equal(server.done,true);assert.equal(server.date,'2027-01-01');
});
test('uncomplete, move and recomplete coalesced edits still move safely',async()=>{
 const original={...task(),done:true,version:1};let server={...original};const calls=[];
 const sync=new TaskSync(async(action,payload)=>{calls.push(action);if(action==='move'&&server.done)throw Error('已完成不能移动');server={...server,...payload,version:server.version+1};return {task:server};});
 sync.merge([original]);sync.tasks[0].date='2027-01-01';await sync.flush();assert.equal(sync.error,null);assert.deepEqual(calls,['update','move','update']);assert.equal(server.done,true);
});
test('month refresh during a pending delete must preserve an undo',async()=>{
 const original={...task(),version:1};let release;const calls=[];
 const sync=new TaskSync(async(action,payload)=>{calls.push(action);if(action==='delete')await new Promise(r=>release=r);return {task:{...original,version:calls.length+1,deleted:action==='delete'}};});
 sync.merge([original]);sync.setTasks([]);const saving=sync.flush();await Promise.resolve();
 sync.setTasks([{...original}]);sync.merge([],original.date,original.date,sync.revision);release();await saving;
 assert.equal(sync.tasks.length,1);assert.deepEqual(calls,['delete','restore']);assert.equal(sync.dirty,false);
});
