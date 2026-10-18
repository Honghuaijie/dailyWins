const {chromium}=require('playwright');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const url=process.env.E2E_URL||'http://localhost:5173';
const username=process.env.E2E_USERNAME,password=process.env.E2E_PASSWORD;
if(!username||!password)throw Error('请为独立测试账号设置 E2E_USERNAME 和 E2E_PASSWORD');
(async()=>{
 const browser=await chromium.launch({channel:'msedge',headless:true});
 try{
 const context=await browser.newContext({viewport:{width:1440,height:900},timezoneId:'Asia/Shanghai'});
 const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 page.on('dialog',d=>d.accept());
 const legacy=JSON.stringify([{id:'legacy-local-only',date:'2020-01-01',name:'不要上传',note:'原始浏览器记录',done:true}]);
 await page.addInitScript(raw=>{if(!localStorage.getItem('daily-wins:v1'))localStorage.setItem('daily-wins:v1',raw);},legacy);
 await page.goto(url);await page.locator('#login-form').waitFor();
 fs.mkdirSync('tests/results',{recursive:true});await page.screenshot({path:'tests/results/login.png',fullPage:true});
 async function login(){await page.locator('#username').fill(username);await page.locator('#password').fill(password);await page.locator('#login-submit').click();await page.locator('#main').waitFor({state:'visible'});await page.locator('#load-banner').waitFor({state:'hidden'});}
 async function saved(){await page.waitForFunction(()=>document.querySelector('#save-state').textContent.includes('已保存到云端'));}
 async function add(name,done=false){await page.locator(done?'.done-entry':'.add-button').click();await page.locator('#new-name').fill(name);await page.locator('#new-note').fill('测试备注 <script> 不执行');await page.locator('#new-form [type=submit]').click();await saved();}
 await login();assert.equal(await page.locator('.task').count(),0,'独立测试账号应为空，避免更改真实记录');
 assert.equal(await page.locator('#legacy-notice').isVisible(),true);
 const today=await page.locator('.day.selected').getAttribute('data-date');const future=new Date(today+'T12:00:00Z');future.setUTCDate(future.getUTCDate()+90);const targetDate=future.toISOString().slice(0,10);
 for(const name of ['阅读一章','散步十分钟','整理桌面'])await add(name,true);
 assert.equal(await page.locator('.day.selected .win-icon').count(),1);
 await add('未完成任务');assert.match(await page.locator('.day.selected .more').innerText(),/还有 1 项/);
 await page.getByRole('checkbox',{name:'取消完成：整理桌面',exact:true}).click();await saved();assert.equal(await page.locator('.day.selected .win-icon').count(),0);
 await page.getByRole('checkbox',{name:'完成：整理桌面',exact:true}).click();await saved();
 await page.getByRole('button',{name:'编辑：整理桌面',exact:true}).click();await page.getByRole('button',{name:'删除',exact:true}).click();await saved();assert.equal(await page.locator('.day.selected .win-icon').count(),0);
 await page.getByRole('button',{name:'撤销',exact:true}).click();await saved();assert.equal(await page.locator('.day.selected .win-icon').count(),1);
 await page.getByRole('button',{name:'编辑：未完成任务',exact:true}).click();await page.getByRole('textbox',{name:'编辑任务名称'}).fill('移动后仍是同一条任务');await saved();
 const before=await (await page.request.get(url+'/v1/user/tasks/list?from='+today+'&to='+today)).json();const movedID=before.data.items.find(t=>t.name==='移动后仍是同一条任务').id;
 await page.getByRole('button',{name:'移到其他日期',exact:true}).click();await page.locator('#move-date').fill(targetDate);await page.getByRole('button',{name:'移动',exact:true}).click();await saved();
 const moved=await (await page.request.get(url+'/v1/user/tasks/list?from='+targetDate+'&to='+targetDate)).json();assert.equal(moved.data.items[0].id,movedID);assert.equal(moved.data.items[0].note,'测试备注 <script> 不执行');
 await page.reload();await page.locator('#main').waitFor({state:'visible'});await page.locator('#load-banner').waitFor({state:'hidden'});assert.equal(await page.locator('.task').count(),3);
 const empty=page.locator('.day:not(.outside)').filter({hasNot:page.locator('.mini-task')}).first();const emptyDate=await empty.getAttribute('data-date');await empty.click();
 await page.locator('.add-button').click();await page.locator('#new-name').fill('跨日期草稿');await page.locator(`[data-date="${today}"]`).click();await page.locator(`[data-date="${emptyDate}"]`).click();assert.equal(await page.locator('#new-name').inputValue(),'跨日期草稿');await page.locator('#new-form [type=submit]').click();await saved();
 await page.getByRole('button',{name:'编辑：跨日期草稿',exact:true}).click();
 await page.route('**/v1/user/tasks/update',r=>r.abort());await page.getByRole('textbox',{name:'编辑备注'}).fill('断网也要保留');await page.locator('#retry').waitFor({state:'visible'});
 await page.locator(`[data-date="${today}"]`).click();await page.locator(`[data-date="${emptyDate}"]`).click();assert.equal(await page.locator('.task-note').innerText(),'断网也要保留');
 await page.unroute('**/v1/user/tasks/update');await page.locator('#retry').click();await saved();
 // 空名称也须作为草稿保留，不能恢复成旧文字并假称保存成功。
 await page.getByRole('button',{name:'编辑：跨日期草稿',exact:true}).click();await page.getByRole('textbox',{name:'编辑任务名称'}).fill('');await page.locator('#retry').waitFor({state:'visible'});await page.locator(`[data-date="${today}"]`).click();await page.locator(`[data-date="${emptyDate}"]`).click();await page.locator('.edit-button').click();assert.equal(await page.getByRole('textbox',{name:'编辑任务名称'}).inputValue(),'');await page.getByRole('textbox',{name:'编辑任务名称'}).fill('跨日期草稿');await saved();
 // 会话失效后的编辑，重新登录仍继续同一份未保存请求。
 await page.request.post(url+'/v1/user/auth/logout',{headers:{'X-Daily-Wins':'1'},data:{}});
 await page.getByRole('textbox',{name:'编辑备注'}).fill('重新登录后继续保存');await page.locator('#login').waitFor({state:'visible'});await login();await saved();
 await page.locator(`[data-date="${emptyDate}"]`).click();assert.equal(await page.locator('.task-note').innerText(),'重新登录后继续保存');
 // 月历加载失败明确反馈，重试恢复。
 await page.route('**/v1/user/tasks/list?**',r=>r.abort());await page.locator('#next').click();await page.locator('#reload-month').waitFor({state:'visible'});await page.unroute('**/v1/user/tasks/list?**');await page.locator('#reload-month').click();await page.locator('#load-banner').waitFor({state:'hidden'});await page.locator('#today').click();await page.locator('#load-banner').waitFor({state:'hidden'});
 // 服务端名称规范化不会导致无限保存。
 await page.getByRole('button',{name:'编辑：阅读一章',exact:true}).click();await page.getByRole('textbox',{name:'编辑任务名称'}).fill('阅读一章 ');await saved();
 // 另一设备更新后的409保留输入，导出草稿再重新加载，不覆盖服务器新版本。
 const snapshot=await (await page.request.get(url+'/v1/user/tasks/list?from='+today+'&to='+today)).json();const remoteTask=snapshot.data.items.find(t=>t.name==='阅读一章');
 const conflictUpdate=await page.request.post(url+'/v1/user/tasks/update',{headers:{'X-Daily-Wins':'1'},data:{id:remoteTask.id,requestId:require('node:crypto').randomUUID(),version:remoteTask.version,name:'另一设备更新',note:remoteTask.note,done:true}});assert.equal(conflictUpdate.status(),200);
 await page.getByRole('textbox',{name:'编辑任务名称'}).fill('我的未保存版本');await page.locator('#recover').waitFor({state:'visible'});assert.equal(await page.getByRole('textbox',{name:'编辑任务名称'}).inputValue(),'我的未保存版本');
 const downloadPromise=page.waitForEvent('download');await page.locator('#recover').click();const download=await downloadPromise;const backup=JSON.parse(fs.readFileSync(await download.path(),'utf8'));assert.ok(backup.tasks.some(t=>t.name==='我的未保存版本'));
 await page.waitForLoadState();await page.locator('#load-banner').waitFor({state:'hidden'});await page.getByRole('button',{name:'编辑：另一设备更新',exact:true}).waitFor();
 for(const [width,height] of [[1440,900],[1366,768],[1920,1080],[1024,768]]){await page.setViewportSize({width,height});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);await page.screenshot({path:`tests/results/cloud-${width}.png`,fullPage:true});}
 assert.equal(await page.evaluate(()=>localStorage.getItem('daily-wins:v1')),legacy);
 await page.locator('#logout').click();await page.locator('#login').waitFor({state:'visible'});const me=await page.request.get(url+'/v1/user/auth/me');assert.equal(me.status(),401);
 assert.deepEqual(errors,[]);console.log('PASS: login, cookie, legacy preservation, CRUD, completion, move identity, delete/undo, reload, drafts, network retry, empty-name protection, session expiry/relogin, month retry, name normalization, conflict/export, layouts, logout, no JS errors');
 }finally{await browser.close();}
})().catch(e=>{console.error(e);process.exit(1)});
