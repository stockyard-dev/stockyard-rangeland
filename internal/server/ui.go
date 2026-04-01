package server

import "net/http"

const uiHTML = `<!DOCTYPE html><html lang="en"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Rangeland — Stockyard</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital,wght@0,400;0,700;1,400&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
<style>:root{
  --bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;
  --rust:#c45d2c;--rust-light:#e8753a;--rust-dark:#8b3d1a;
  --leather:#a0845c;--leather-light:#c4a87a;
  --cream:#f0e6d3;--cream-dim:#bfb5a3;--cream-muted:#7a7060;
  --gold:#d4a843;--green:#5ba86e;--red:#c0392b;
  --font-serif:'Libre Baskerville',Georgia,serif;
  --font-mono:'JetBrains Mono',monospace;
}
*{margin:0;padding:0;box-sizing:border-box}
body{background:var(--bg);color:var(--cream);font-family:var(--font-serif);min-height:100vh}
a{color:var(--rust-light);text-decoration:none}a:hover{color:var(--gold)}
.hdr{background:var(--bg2);border-bottom:2px solid var(--rust-dark);padding:.9rem 1.8rem;display:flex;align-items:center;justify-content:space-between}
.hdr-left{display:flex;align-items:center;gap:1rem}
.hdr-brand{font-family:var(--font-mono);font-size:.75rem;color:var(--leather);letter-spacing:3px;text-transform:uppercase}
.hdr-title{font-family:var(--font-mono);font-size:1.1rem;color:var(--cream);letter-spacing:1px}
.badge{font-family:var(--font-mono);font-size:.6rem;padding:.2rem .6rem;letter-spacing:1px;text-transform:uppercase;border:1px solid}
.badge-free{color:var(--green);border-color:var(--green)}
.main{max-width:1000px;margin:0 auto;padding:2rem 1.5rem}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(130px,1fr));gap:1rem;margin-bottom:1.5rem}
.card{background:var(--bg2);border:1px solid var(--bg3);padding:1rem 1.2rem}
.card-val{font-family:var(--font-mono);font-size:1.6rem;font-weight:700;color:var(--cream);display:block}
.card-lbl{font-family:var(--font-mono);font-size:.58rem;letter-spacing:2px;text-transform:uppercase;color:var(--leather);margin-top:.2rem}
.section{margin-bottom:2rem}
.section-title{font-family:var(--font-mono);font-size:.68rem;letter-spacing:3px;text-transform:uppercase;color:var(--rust-light);margin-bottom:.8rem;padding-bottom:.5rem;border-bottom:1px solid var(--bg3)}
.empty{color:var(--cream-muted);text-align:center;padding:2rem;font-style:italic}
.btn{font-family:var(--font-mono);font-size:.7rem;padding:.3rem .8rem;border:1px solid var(--leather);background:transparent;color:var(--cream);cursor:pointer;transition:all .2s}
.btn:hover{border-color:var(--rust-light);color:var(--rust-light)}
.btn-rust{border-color:var(--rust);color:var(--rust-light)}.btn-rust:hover{background:var(--rust);color:var(--cream)}
.lbl{font-family:var(--font-mono);font-size:.62rem;letter-spacing:1px;text-transform:uppercase;color:var(--leather)}
input,select{font-family:var(--font-mono);font-size:.78rem;background:var(--bg3);border:1px solid var(--bg3);color:var(--cream);padding:.4rem .7rem;outline:none}
input:focus,select:focus{border-color:var(--leather)}
.row{display:flex;gap:.8rem;align-items:flex-end;flex-wrap:wrap;margin-bottom:1rem}
.field{display:flex;flex-direction:column;gap:.3rem}
pre{background:var(--bg3);padding:.8rem 1rem;font-family:var(--font-mono);font-size:.72rem;color:var(--cream-dim);overflow-x:auto}
.chart{height:120px;display:flex;align-items:flex-end;gap:2px;margin-bottom:1.5rem;padding:.5rem;background:var(--bg2);border:1px solid var(--bg3)}
.chart-bar{background:var(--rust);min-width:4px;flex:1;border-radius:1px 1px 0 0;position:relative;cursor:pointer;transition:background .2s}
.chart-bar:hover{background:var(--rust-light)}
.chart-bar .tip{display:none;position:absolute;bottom:calc(100% + 4px);left:50%;transform:translateX(-50%);background:var(--bg3);padding:.2rem .5rem;font-family:var(--font-mono);font-size:.58rem;white-space:nowrap;color:var(--cream);border:1px solid var(--leather);z-index:9}
.chart-bar:hover .tip{display:block}
.grid2{display:grid;grid-template-columns:1fr 1fr;gap:1.5rem}
.bar-row{display:flex;align-items:center;gap:.5rem;margin-bottom:.4rem;font-family:var(--font-mono);font-size:.72rem}
.bar-name{flex:1;color:var(--cream-dim);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.bar-count{color:var(--cream-muted);min-width:40px;text-align:right}
.bar-bg{flex:2;height:6px;background:var(--bg3);border-radius:3px;overflow:hidden}
.bar-fill{height:100%;background:var(--rust);border-radius:3px}
.period-sel{display:flex;gap:.3rem}
.period-btn{font-family:var(--font-mono);font-size:.62rem;padding:.2rem .6rem;border:1px solid var(--bg3);background:transparent;color:var(--cream-muted);cursor:pointer;letter-spacing:1px}
.period-btn.active{border-color:var(--rust);color:var(--rust-light)}
@media(max-width:700px){.grid2{grid-template-columns:1fr}}
</style></head><body>
<div class="hdr">
  <div class="hdr-left">
    <svg viewBox="0 0 64 64" width="22" height="22" fill="none"><rect x="8" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="28" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="48" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="8" y="27" width="48" height="7" rx="2.5" fill="#c4a87a"/></svg>
    <span class="hdr-brand">Stockyard</span>
    <span class="hdr-title">Rangeland</span>
  </div>
  <div style="display:flex;gap:.8rem;align-items:center">
    <span class="badge badge-free">Free</span>
    <a href="/api/status" class="lbl" style="color:var(--leather)">API</a>
  </div>
</div>
<div class="main">

<div id="no-sites" style="display:none">
  <div class="section">
    <div class="section-title">Add Your First Site</div>
    <div class="row">
      <div class="field"><span class="lbl">Domain</span><input id="c-domain" placeholder="example.com" style="width:220px"></div>
      <button class="btn btn-rust" onclick="addSite()">Add Site</button>
    </div>
    <div id="c-result"></div>
  </div>
</div>

<div id="dashboard" style="display:none">
  <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem;flex-wrap:wrap;gap:.5rem">
    <select id="site-select" onchange="switchSite()" style="min-width:200px"></select>
    <div class="period-sel">
      <button class="period-btn" onclick="setPeriod(1)">24h</button>
      <button class="period-btn active" onclick="setPeriod(7)">7d</button>
      <button class="period-btn" onclick="setPeriod(30)">30d</button>
      <button class="period-btn" onclick="setPeriod(90)">90d</button>
    </div>
  </div>

  <div class="cards">
    <div class="card"><span class="card-val" id="s-realtime">—</span><span class="card-lbl">Now</span></div>
    <div class="card"><span class="card-val" id="s-views">—</span><span class="card-lbl">Pageviews</span></div>
    <div class="card"><span class="card-val" id="s-visitors">—</span><span class="card-lbl">Visitors</span></div>
    <div class="card"><span class="card-val" id="s-vpv">—</span><span class="card-lbl">Views/Visitor</span></div>
    <div class="card"><span class="card-val" id="s-bounce">—</span><span class="card-lbl">Bounce %</span></div>
  </div>

  <div class="chart" id="chart"></div>

  <div class="grid2">
    <div class="section"><div class="section-title">Top Pages</div><div id="top-pages"></div></div>
    <div class="section"><div class="section-title">Referrers</div><div id="top-refs"></div></div>
    <div class="section"><div class="section-title">Browsers</div><div id="top-browsers"></div></div>
    <div class="section"><div class="section-title">Devices</div><div id="top-devices"></div></div>
  </div>

  <div class="section" style="margin-top:1rem">
    <div class="section-title">Tracking Snippet</div>
    <pre id="snippet-code"></pre>
  </div>
</div>

</div>
<script>
let sites=[],curSite=null,period=7;

async function init(){
  const r=await fetch('/api/sites');const d=await r.json();
  sites=d.sites||[];
  if(!sites.length){document.getElementById('no-sites').style.display='block';document.getElementById('dashboard').style.display='none';return;}
  document.getElementById('no-sites').style.display='none';document.getElementById('dashboard').style.display='block';
  const sel=document.getElementById('site-select');
  sel.innerHTML=sites.map(s=>'<option value="'+s.id+'">'+esc(s.domain)+'</option>').join('');
  curSite=sites[0].id;
  refresh();
}

function switchSite(){curSite=document.getElementById('site-select').value;refresh();}
function setPeriod(d){period=d;document.querySelectorAll('.period-btn').forEach((b,i)=>b.classList.toggle('active',[1,7,30,90][i]===d));refresh();}

async function refresh(){
  if(!curSite)return;
  try{
    const[ov,ts,pg,rf,br,dv,rt]=await Promise.all([
      fetch('/api/sites/'+curSite+'/overview?days='+period).then(r=>r.json()),
      fetch('/api/sites/'+curSite+'/timeseries?days='+period).then(r=>r.json()),
      fetch('/api/sites/'+curSite+'/pages?days='+period).then(r=>r.json()),
      fetch('/api/sites/'+curSite+'/referrers?days='+period).then(r=>r.json()),
      fetch('/api/sites/'+curSite+'/browsers?days='+period).then(r=>r.json()),
      fetch('/api/sites/'+curSite+'/devices?days='+period).then(r=>r.json()),
      fetch('/api/sites/'+curSite+'/realtime').then(r=>r.json()),
    ]);
    document.getElementById('s-realtime').textContent=rt.visitors||0;
    document.getElementById('s-views').textContent=fmt(ov.pageviews||0);
    document.getElementById('s-visitors').textContent=fmt(ov.visitors||0);
    document.getElementById('s-vpv').textContent=ov.views_per_visitor||'0';
    document.getElementById('s-bounce').textContent=(ov.bounce_rate||'0')+'%';
    renderChart(ts.data||[]);
    renderBars('top-pages',pg.pages);
    renderBars('top-refs',rf.referrers);
    renderBars('top-browsers',br.browsers);
    renderBars('top-devices',dv.devices);
    const site=sites.find(s=>s.id===curSite);
    document.getElementById('snippet-code').textContent='<script defer data-domain="'+site.domain+'" src="'+location.origin+'/js/script.js"><\/script>';
  }catch(e){console.error(e);}
}

function renderChart(data){
  const el=document.getElementById('chart');
  if(!data.length){el.innerHTML='<div class="empty" style="width:100%;display:flex;align-items:center;justify-content:center">No data yet</div>';return;}
  const max=Math.max(...data.map(d=>d.views),1);
  el.innerHTML=data.map(d=>{
    const h=Math.max(2,d.views/max*100);
    return '<div class="chart-bar" style="height:'+h+'%"><div class="tip">'+d.date+'<br>'+d.views+' views / '+d.visitors+' visitors</div></div>';
  }).join('');
}

function renderBars(id,items){
  const el=document.getElementById(id);
  if(!items||!items.length){el.innerHTML='<div class="empty">No data</div>';return;}
  const max=items[0].count;
  el.innerHTML=items.slice(0,8).map(i=>
    '<div class="bar-row"><span class="bar-name">'+esc(i.name||'(direct)')+'</span><div class="bar-bg"><div class="bar-fill" style="width:'+Math.max(2,i.count/max*100)+'%"></div></div><span class="bar-count">'+fmt(i.count)+'</span></div>'
  ).join('');
}

async function addSite(){
  const domain=document.getElementById('c-domain').value.trim();
  if(!domain)return;
  const r=await fetch('/api/sites',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({domain})});
  const d=await r.json();
  if(r.ok){document.getElementById('c-result').innerHTML='<pre style="margin-top:.5rem">'+esc(d.snippet)+'</pre>';init();}
  else{document.getElementById('c-result').innerHTML='<span style="color:var(--red)">'+esc(d.error)+'</span>';}
}

function fmt(n){if(n>=1e6)return(n/1e6).toFixed(1)+'M';if(n>=1e3)return(n/1e3).toFixed(1)+'K';return n;}
function esc(s){const d=document.createElement('div');d.textContent=s||'';return d.innerHTML;}

init();
setInterval(refresh,15000);
</script></body></html>`

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(uiHTML))
}
