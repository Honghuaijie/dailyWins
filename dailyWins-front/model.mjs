export const dateKey = d => `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`;
export function monthDays(year,month){
 const first=new Date(year,month,1), offset=(first.getDay()+6)%7;
 const size=Math.ceil((offset+new Date(year,month+1,0).getDate())/7)*7;
 return Array.from({length:size},(_,i)=>new Date(year,month,1-offset+i));
}
export function dayInfo(tasks,date){const items=tasks.filter(t=>t.date===date);const completed=items.filter(t=>t.done).length;return {items,completed,achieved:completed>=3,extra:Math.max(0,items.length-3)};}
export function moveTask(tasks,id,date){
 const task=tasks.find(t=>t.id===id);
 if(!task||task.done)throw new Error('只能移动未完成任务');
 if(!/^\d{4}-\d{2}-\d{2}$/.test(date)||dateKey(new Date(date+'T12:00:00'))!==date)throw new Error('请选择有效日期');
 task.date=date;
}
export function seedTasks(today){
 const result=[];
 const add=(offset,name,done,note='')=>{const d=new Date(today);d.setDate(d.getDate()+offset);result.push({id:crypto.randomUUID(),date:dateKey(d),name,done,note});};
 add(0,'晨跑 3 公里',true,'不追求速度，出门就已经很棒了。今天沿着河边跑了一圈。');
 add(0,'读书 20 分钟',true,'《也许你该找个人聊聊》 · 从第 86 页继续');
 add(0,'完成 Daily Wins 项目封面',false,'试试奶油白和鼠尾草绿，留一点呼吸的空间。');
 add(0,'练字一页，让自己慢下来',false);
 add(-1,'下楼散步，晒一会儿太阳',true);add(-1,'练字 15 分钟',true);add(-1,'读完喜欢的一章',true);
 add(-3,'跑步 3 公里',true);add(-3,'整理书桌',false);
 add(-5,'阅读 20 分钟',true);add(-5,'完成项目封面',true);add(-5,'练字一页',true);add(-5,'给绿植浇水',true);
 add(-7,'做一顿喜欢的早餐',true);add(-7,'记录今天值得感谢的小事',false);
 add(-10,'傍晚散步',true);add(-10,'读书 20 分钟',true);add(-10,'收拾房间',true);
 add(1,'跑步，不用太快',false);add(1,'阅读新买的书并整理三个最有启发的观点',false,'把喜欢的句子记在笔记本里。');
 add(3,'完成下一版项目封面草图',false);add(3,'练字 20 分钟',false);
 return result;
}
