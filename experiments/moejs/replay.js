// Synthetic HTTP responses at the transport boundary. Entire upstream bundles
// (including SDKs) execute unchanged; this is not a live website benchmark.
var nativeFetch = fetch;
var liveNetwork = false;
var requests = [];
var handlers = {};
var blobSerial = 0;
var gopeed = {
 info: {identity: 'monkeyWie@' + extensionName},
 host: {env:{version:'2.0.0-beta.3',os:'darwin',arch:'arm64'}},
 settings:{quality:extensionName==='youtube'?'1080p':80,qualityFallback:'best',fallbackToBest:true,cookie:''},
 logger:{debug(){},info(){},warn(){},error(){}},
 storage:{get(){return null},set(){}},
 events:Object.fromEntries(['onResolve','onStart','onError','onDone'].map(k=>[k,fn=>handlers[k]=fn])),
 runtime:{
  ffmpeg:{merge(){throw new Error('media merge is outside this replay')},supportsInputFactory:true,supportsProducerProgress:true},
  blob:{async createObjectURL(){return 'blob:replay/'+(++blobSerial)},async revokeObjectURL(){}},
  webview:{isAvailable(){return true},async open(){return {
   async goto(){},async execute(expression){
    if (expression !== '() => navigator.userAgent') throw new Error('unexpected WebView execution');
    return 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/18.0 Safari/605.1.15';
   },async close(){},async getCookies(){return []},async setCookie(){},async deleteCookie(){}
  }}}
 }};
class MessageError extends Error {}
var route = (raw, init) => {
 const url = new URL(String(raw));
 requests.push(url.pathname);
 const json = x => new Response(JSON.stringify(x),{status:200,headers:{'content-type':'application/json'}});
 if (url.pathname === '/x/web-interface/nav') return json({code:0,data:{wbi_img:{img_url:'https://i0.hdslb.com/bfs/wbi/7cd084941338484aae1ad9425b84077c.png',sub_url:'https://i0.hdslb.com/bfs/wbi/4932caff0ff746eab6f01bf08b70ac45.png'}}});
 if (url.pathname === '/x/web-interface/wbi/view' || url.pathname === '/x/web-interface/view') return json({code:0,data:{title:'真实扩展 Replay 测试',pages:Array.from({length:3},(_,i)=>({cid:100+i,part:'P'+(i+1)}))}});
 if (url.pathname === '/x/player/wbi/playurl' || url.pathname === '/x/player/playurl') return json({code:0,data:{quality:80,format:'dash',dash:{video:[{id:64,width:1280,height:720,baseUrl:'https://upos-test.bilivideo.com/video64'},{id:80,width:1920,height:1080,baseUrl:'https://upos-test.bilivideo.com/video80',backupUrl:['https://backup.example.com/video80']}],audio:[{id:30280,baseUrl:'https://upos-test.bilivideo.com/audio'}]}}});
 if (url.pathname === '/youtubei/v1/player') return json({responseContext:{serviceTrackingParams:[]},playabilityStatus:{status:'OK'},videoDetails:{videoId:'dQw4w9WgXcQ',title:'YouTube Replay 测试',lengthSeconds:'212',channelId:'UCfixture',shortDescription:'Replay fixture',viewCount:'100',author:'Fixture',isLiveContent:false},microformat:{playerMicroformatRenderer:{title:{simpleText:'YouTube Replay 测试'},description:{simpleText:'Replay'},category:'Music',publishDate:'2026-11-01',uploadDate:'2026-11-01',isFamilySafe:true}},streamingData:{expiresInSeconds:'3600',adaptiveFormats:[]}});
 if (url.pathname.endsWith('/guest/activate.json')) return json({guest_token:'fixture-guest'});
 if (url.hostname === 'pastebin.com' && url.pathname === '/raw/nz3ApKQM') return new Response('Bearer fixture-only', {status:200});
 if (url.pathname.includes('/TweetResultByRestId')) {
  const media = [{type:'photo',media_url_https:'https://pbs.twimg.com/media/replay.jpg'}, {type:'video',media_url_https:'https://pbs.twimg.com/media/replay.jpg',video_info:{duration_millis:10000,variants:[{bitrate:256000,content_type:'video/mp4',url:'https://video.twimg.com/640x360/low.mp4'},{bitrate:2048000,content_type:'video/mp4',url:'https://video.twimg.com/1920x1080/high.mp4'}]}}];
  return json({data:{tweetResult:{result:{__typename:'Tweet',views:{count:'100'},core:{user_results:{result:{legacy:{screen_name:'fixture',name:'Replay'}}}},legacy:{id_str:'1734145541599383880',full_text:'Replay',created_at:'Tue Dec 12 00:00:00 +0000 2023',entities:{media}}}}}});
 }
 if (url.pathname === '/sw.js_data' || url.pathname === '/youtubei/v1/config') return new Response('Offline session: exercise YouTube.js local fallback', {status:503});
 throw new Error('unmatched replay request: '+url);
};
fetch = async (input,init) => liveNetwork ? nativeFetch(input,init) : route(input?.url || input,init);
async function resolveOnce() {
 requests = [];
 blobSerial = 0;
 const url = globalThis.inputURL || {bilibili:'https://www.bilibili.com/video/BV1BJ4m1e7g8?p=1-3',youtube:'https://www.youtube.com/watch?v=dQw4w9WgXcQ',twitter:'https://x.com/bbcchinese/status/1734145541599383880'}[extensionName];
 const ctx = {req:{url}};
 await handlers.onResolve(ctx);
 if (!ctx.res?.files?.length) throw new Error('resolve returned no files');
 return JSON.stringify({res:ctx.res,requests});
}

async function lifecycleOnce() {
 const resolved = JSON.parse(await resolveOnce());
 let continued = 0;
 for (const file of resolved.res.files) {
  const req = file.req;
  req.putLabel = async (key,value) => { (req.labels ||= {})[key] = value; };
  req.setUrl = async url => { req.url = url; };
  const task = {meta:{req},status:'ready',setUrl:req.setUrl,async continue(){continued++}};
  if (handlers.onStart) await handlers.onStart({task});
  task.status = 'error';
  if (handlers.onError) await handlers.onError({task});
 }
 return JSON.stringify({res:resolved.res,continued,requests});
}

// Optional common buffered transport to separate SDK work from each runtime's
// Response/ReadableStream implementation. Both engines execute this same JS.
class BufferedResponse {
 constructor(text,options={}) {
  this._text = String(text); this.status = options.status ?? 200;
  this.ok = this.status >= 200 && this.status < 300;
  this.statusText = this.ok ? 'OK' : 'Error'; this.url = '';
  this.headers = new Headers(options.headers);
  const bytes = new TextEncoder().encode(this._text);
  this.body = {getReader(){let sent=false;return {async read(){if(sent)return {done:true};sent=true;return {done:false,value:bytes}},async cancel(){},releaseLock(){}}},async cancel(){}};
 }
 async text(){return this._text}
 async json(){return JSON.parse(this._text)}
 async arrayBuffer(){return new TextEncoder().encode(this._text).buffer}
 clone(){return new BufferedResponse(this._text,{status:this.status,headers:this.headers})}
}
