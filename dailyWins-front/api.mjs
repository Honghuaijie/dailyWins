export class APIError extends Error {
 constructor(message,status,code){super(message);this.status=status;this.code=code;}
}
export async function api(path,body){
 const controller=new AbortController();const timer=setTimeout(()=>controller.abort(),20000);
 try{
  const response=await fetch(path,{method:body===undefined?'GET':'POST',credentials:'same-origin',headers:body===undefined?{}:{'Content-Type':'application/json','X-Daily-Wins':'1'},body:body===undefined?undefined:JSON.stringify(body),signal:controller.signal});
  let result;try{result=await response.json();}catch{throw new APIError('服务暂时不可用，请稍后重试',response.status);}
  if(!response.ok||result.code!==0)throw new APIError(result.message||'保存失败，请重试',response.status,result.code);
  return result.data;
 }catch(error){if(error instanceof APIError)throw error;throw new APIError('网络连接失败，输入已保留，请重试',0);}
 finally{clearTimeout(timer);}
}

export const taskAPI=(action,payload)=>api(`/v1/user/tasks/${action}`,payload);
