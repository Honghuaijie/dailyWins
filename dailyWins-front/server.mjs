import http from 'node:http';
import https from 'node:https';
import {readFile} from 'node:fs/promises';
const backend=new URL(process.env.API_TARGET||'http://127.0.0.1:10006');
if(!['http:','https:'].includes(backend.protocol)||backend.username||backend.password)throw Error('API_TARGET 必须为 http(s) 服务地址');
const files={'/':'index.html','/index.html':'index.html','/styles.css':'styles.css','/app.mjs':'app.mjs','/model.mjs':'model.mjs','/api.mjs':'api.mjs','/sync.mjs':'sync.mjs'};
const types={html:'text/html; charset=utf-8',css:'text/css; charset=utf-8',mjs:'text/javascript; charset=utf-8'};
http.createServer(async(req,res)=>{
 const requestURL=new URL(req.url,'http://localhost');const pathname=requestURL.pathname;
 if(pathname.startsWith('/v1/')){
  const transport=backend.protocol==='https:'?https:http;
  const target=new URL(backend);target.pathname=`${backend.pathname.replace(/\/+$/,'')}${pathname}`;target.search=requestURL.search;
  const upstream=transport.request(target,{method:req.method,headers:{...req.headers,host:target.host},timeout:25000},response=>{res.writeHead(response.statusCode,response.headers);response.pipe(res);});
  upstream.on('timeout',()=>upstream.destroy());
  upstream.on('error',()=>{if(!res.headersSent)res.writeHead(502,{'Content-Type':'application/json'});res.end(JSON.stringify({code:50200,message:'后端暂时不可用，请确认服务已启动；输入不会丢失'}));});
  req.pipe(upstream);return;
 }
 const file=files[pathname];if(!file){res.writeHead(404);res.end();return;}
 try{res.writeHead(200,{'Content-Type':types[file.split('.').pop()],'Cache-Control':'no-cache','X-Content-Type-Options':'nosniff','Content-Security-Policy':"default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"});res.end(await readFile(new URL(file,import.meta.url)));}catch{res.writeHead(500);res.end('Unable to read file');}
}).listen(Number(process.env.PORT||5173),process.env.HOST||'127.0.0.1',()=>console.log(`Daily Wins: http://localhost:${process.env.PORT||5173}`));
