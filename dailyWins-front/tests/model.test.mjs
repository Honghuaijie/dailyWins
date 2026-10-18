import {test} from 'node:test';
import assert from 'node:assert/strict';
import {dayInfo, moveTask, dateKey, monthDays} from '../model.mjs';
test('achievement always reflects current completed tasks',()=>{
 const tasks=[1,2,3,4].map(id=>({id,date:'2026-10-17',done:id<4}));
 assert.equal(dayInfo(tasks,'2026-10-17').achieved,true);
 tasks[0].done=false;
 assert.equal(dayInfo(tasks,'2026-10-17').achieved,false);
 assert.equal(dayInfo(tasks,'2026-10-17').extra,1);
 assert.equal(dayInfo(tasks,'2026-10-18').completed,0);
});
test('move preserves identity and note, excludes completed tasks',()=>{
 const tasks=[{id:'a',date:'2026-10-17',name:'阅读',note:'第十章',done:false},{id:'b',date:'2026-10-17',done:true}];
 moveTask(tasks,'a','2026-11-01');
 assert.deepEqual(tasks[0],{id:'a',date:'2026-11-01',name:'阅读',note:'第十章',done:false});
 assert.throws(()=>moveTask(tasks,'b','2026-11-01'));
 assert.throws(()=>moveTask(tasks,'a','2026-02-30'));
});
test('calendar supports leap years and year boundaries with complete weeks',()=>{
 const days=monthDays(2024,1);
 assert.equal(days.length%7,0);
 assert.equal(days.filter(d=>d.getMonth()===1).length,29);
 assert.equal(dateKey(monthDays(2026,0)[0]),'2025-12-29');
});
