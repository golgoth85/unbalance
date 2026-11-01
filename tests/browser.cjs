// Solo API sintetiche: non contatta il NAS e non avvia trasferimenti reali.
const { chromium } = require('playwright');
const fs = require('fs');
const assert = require('assert/strict');
process.chdir(require('path').resolve(__dirname, '..'));
const base = process.env.UI_URL || 'http://127.0.0.1:17091';
const output = process.env.UI_OUTPUT;
if (!output || !output.startsWith('/temp/unbalance/')) throw Error('UI_OUTPUT deve essere sotto /temp/unbalance');
const now = '2026-11-01T12:00:00Z';
const long = 'Serie sintetica dal nome molto lungo per verificare la leggibilità su telefono';
const disks = ['disk3','disk4','disk5'].map((name,i)=>({id:i,name,path:'/mnt/'+name,device:'synthetic',type:'data',fsType:'xfs',free:1200000000000,size:16000000000000,serial:'TEST-SYNTHETIC-SERIAL-'+i,status:'DISK_OK',blocksFree:1e8,blocksTotal:1e9}));
const item={name:long,path:'Libri/'+long,location:'/mnt/disk4',size:3355000000000,transferSize:646000000000,requiredSpace:647000000000,estimated:true,blocksUsed:0};
const plan={id:'test-plan',started:now,ended:now,chosenFolders:['Libri'],ownerIssue:0,groupIssue:0,folderIssue:0,fileIssue:0,bytesToTransfer:item.transferSize,rsyncArgs:['-X'],vdisks:Object.fromEntries(disks.map(d=>[d.path,{path:d.path,currentFree:d.free,plannedFree:d.free-(d.name==='disk3'?item.requiredSpace:0),src:d.name==='disk4',dst:d.name==='disk3',bin:d.name==='disk3'?{size:item.transferSize,requiredSpace:item.requiredSpace,blocksUsed:0,items:[item]}:null}]))};
const operation={id:'test-copy',opKind:3,started:now,finished:now,bytesToTransfer:646e9,bytesTransferred:0,dryRun:true,rsyncArgs:['-avPR','-X','--dry-run'],rsyncStrArgs:'-avPR -X --dry-run',commands:[{id:'cmd',src:'/mnt/disk4',dst:'/mnt/disk3',entry:item.path,size:0,transferred:0,status:0,reason:''}],completed:0,speed:0,remaining:'0s',deltaTransfer:0,line:'Confronto '+long};
const failed={...operation,id:'failed-move',opKind:2,dryRun:false,commands:[{...operation.commands[0],status:6,estimated:true,reason:'Verifica fallita: sorgente preservata'}]};
const state={status:0,unraid:{numDisks:3,numProtected:3,synced:now,syncErrs:0,resync:0,resyncPos:0,state:'STARTED',size:48e12,free:3.6e12,disks,blockSize:4096},operation:null,history:{version:1,lastChecked:now,items:{[failed.id]:failed,[operation.id]:operation},order:[failed.id,operation.id]}};
const config={version:'test-resume-mobile',dryRun:true,notifyPlan:0,notifyTransfer:0,reservedAmount:1,reservedUnit:'Gb',rsyncArgs:['-X'],verbosity:0,refreshRate:1000,logLines:100,speedWindow:'90s',authEnabled:false,authUsername:'admin'};
(async()=>{
 const browser=await chromium.launch({headless:true});
 const results=[];
 try {
 for(const [name,width,height,mobile] of [['desktop',1440,900,false],['intermedio',1280,850,false],['mobile',390,844,true]]){
  const context=await browser.newContext({viewport:{width,height},isMobile:mobile,hasTouch:mobile});
  const errors=[];const commands=[];
  await context.route('**/*',async route=>{
   const url=new URL(route.request().url());
   if(url.origin!==base){return route.abort();}
   if(!url.pathname.startsWith('/api/'))return route.continue();
   let data;
   if(url.pathname==='/api/auth/status')data={enabled:false,configured:false,authenticated:true,username:'test',csrfToken:''};
   else if(url.pathname==='/api/config')data=config;
   else if(url.pathname==='/api/state')data=state;
   else if(url.pathname==='/api/plans')data=[];
   else if(url.pathname.startsWith('/api/tree/'))data={nodes:{libri:{id:'libri',label:'Libri',leaf:true,dir:true,parent:'root'}},order:['libri']};
   else if(url.pathname.startsWith('/api/locate/'))data=['/mnt/disk3','/mnt/disk4'];
   else if(url.pathname.startsWith('/api/size/'))data={total:3355e9,disks:{'/mnt/disk3':2600e9,'/mnt/disk4':3355e9}};
   else if(url.pathname==='/api/logs')data=['2026/11/01 12:00:00 '+long,'2026/11/01 12:00:01 warning: esempio sintetico'];
   else data=config;
   return route.fulfill({json:data});
  });
  await context.routeWebSocket('**/ws?*',ws=>ws.onMessage(message=>{
   const packet=JSON.parse(message);commands.push(packet);
   if(packet.topic==='scatter:plan:start'){
    ws.send(JSON.stringify({topic:'scatter:plan:started',payload:'Pianificazione sintetica'}));
    ws.send(JSON.stringify({topic:'scatter:plan:ended',payload:plan}));
   }else if(packet.topic==='scatter:copy'){
    ws.send(JSON.stringify({topic:'transfer:started',payload:operation}));
   }else throw Error('Comando inatteso nel test: '+packet.topic);
  }));
  const page=await context.newPage();
  page.on('pageerror',e=>{errors.push(e.message);console.error('PAGEERROR',e.message)});
  page.on('console',m=>{if(m.type()==='error')errors.push(m.text())});
  const capture=async label=>{
   await page.waitForTimeout(180);
   const geometry=await page.evaluate(()=>({w:innerWidth,scroll:document.documentElement.scrollWidth}));
   assert(geometry.scroll<=geometry.w+1,`${name}/${label}: scroll ${JSON.stringify(geometry)}`);
   await page.screenshot({path:output+'/'+name+'-'+label+'.png',fullPage:true});
  };
  await page.goto(base);await page.getByText('Source Disk',{exact:true}).waitFor();
  await page.getByRole('button').filter({hasText:'disk4'}).click();
  await page.getByRole('button',{name:'Seleziona Libri',exact:true}).click();
  const destination=page.getByRole('checkbox',{name:'Destinazione disk3'});
  if(mobile) await destination.tap(); else await destination.click();
  await capture('select');
  await page.getByRole('button',{name:'Next',exact:true}).click();
  await page.getByRole('button',{name:'Next',exact:true}).click();
  await page.getByText('Dati da trasferire:',{exact:false}).waitFor();
  await page.getByRole('button').filter({hasText:'disk3'}).click();
  await page.getByText('da trasferire)',{exact:false}).waitFor();
  await capture('validation');
  assert(await page.getByRole('checkbox').isChecked(),'dry run deve restare attivo nel test');
  await page.getByRole('button',{name:'COPY',exact:true}).click();
  await page.getByText('Dati simulati',{exact:true}).waitFor();
  await capture('operation');
  assert(!(await page.locator('body').innerText()).includes('NaN'),'NaN in UI');
  // A new page resets the synthetic busy state; history actions remain guarded.
  await page.goto(base);await page.getByText('HISTORY',{exact:true}).click();
  await page.getByRole('button').filter({hasText:'Scatter / Move'}).click();
  await page.getByText('Sorgente preservata: verifica non superata.',{exact:false}).waitFor();
  assert.equal(await page.getByText('rmsrc',{exact:true}).count(),0,'verifica fallita non deve offrire rimozione sorgente');
  await capture('protected-source');
  const row=page.getByRole('button').filter({hasText:'Scatter / Copy'});
  await row.focus();await page.keyboard.press('Enter');
  assert(await row.getAttribute('aria-pressed')==='true');
  await capture('history');
  await page.getByText('GATHER',{exact:true}).click();
  await page.getByRole('button',{name:'Seleziona Libri',exact:true}).click();
  await capture('gather');
  await page.getByText('SETTINGS',{exact:true}).click();
  await page.getByRole('link',{name:'Rsync Flags',exact:true}).click();
  await capture('settings');
  await page.getByText('LOG',{exact:true}).click();
  await page.getByRole('textbox',{name:'Cerca nei log'}).fill('warning');
  await capture('logs');
  await page.keyboard.press('Tab');
  assert(await page.evaluate(()=>document.activeElement!==document.body),'focus tastiera mancante');
  // Markup reale; rimossi soltanto PHP e script dipendenti dalla UI Unraid.
  const native=fs.readFileSync('meta/plugin/unbalanced.page','utf8').split('---')[1]
    .replace(/<\?php[\s\S]*?\?>/g,'').replace(/<\?=[\s\S]*?\?>/g,'')
    .replace(/<script[\s\S]*?<\/script>/g,'').replace(/on(?:Click|Change)="[^"]*"/gi,'');
  await page.setContent(`<meta name="viewport" content="width=device-width, initial-scale=1"><style>body{font:16px system-ui;margin:16px}input,select{padding:8px}</style>${native}`);
  await capture('unraid-template');
  results.push({name,width,height,isMobile:mobile,screenshots:9,errors,commands:commands.map(c=>c.topic)});
  await context.close();
  assert.equal(errors.length,0,JSON.stringify(errors));
 }
 fs.writeFileSync(output+'/results.json',JSON.stringify(results,null,2));
 console.log(JSON.stringify(results));
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
