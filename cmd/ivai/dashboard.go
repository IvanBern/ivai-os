package main

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Ivai OS — Admin Dashboard</title>
<style>
  :root {
    --bg: #0d1117; --surface: #161b22; --border: #30363d;
    --text: #c9d1d9; --dim: #8b949e; --accent: #58a6ff;
    --green: #3fb950; --red: #f85149; --yellow: #d29922;
    --purple: #a371f7; --orange: #d2693a; --radius: 6px;
  }
  * { margin:0; padding:0; box-sizing:border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif; background:var(--bg); color:var(--text); height:100vh; display:flex; flex-direction:column; }
  header { background:var(--surface); border-bottom:1px solid var(--border); padding:12px 20px; display:flex; align-items:center; gap:16px; flex-shrink:0; }
  header h1 { font-size:18px; font-weight:600; }
  header .dot { width:8px; height:8px; border-radius:50%; background:var(--green); display:inline-block; margin-right:6px; }
  nav { display:flex; gap:4px; margin-left:24px; flex-wrap:wrap; }
  nav button { background:none; border:1px solid transparent; color:var(--dim); padding:6px 14px; border-radius:var(--radius); cursor:pointer; font-size:13px; transition:all .15s; white-space:nowrap; }
  nav button:hover { color:var(--text); background:var(--border); }
  nav button.active { color:var(--accent); border-color:var(--accent); background:rgba(88,166,255,0.1); }
  main { flex:1; overflow:auto; padding:20px; }
  .tab { display:none; }
  .tab.active { display:block; }

  .grid { display:grid; grid-template-columns:repeat(auto-fill, minmax(220px, 1fr)); gap:12px; margin-bottom:20px; }
  .card { background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:14px; }
  .card h3 { font-size:11px; text-transform:uppercase; letter-spacing:.5px; color:var(--dim); margin-bottom:6px; }
  .card .value { font-size:24px; font-weight:700; }
  .card .sub { font-size:11px; color:var(--dim); margin-top:3px; }

  .model-row { display:flex; justify-content:space-between; align-items:center; padding:7px 0; border-bottom:1px solid var(--border); }
  .model-row:last-child { border-bottom:none; }
  .badge { font-size:10px; padding:2px 8px; border-radius:10px; font-weight:600; text-transform:uppercase; }
  .badge-ok { background:rgba(63,185,80,0.15); color:var(--green); }
  .badge-err { background:rgba(248,81,73,0.15); color:var(--red); }
  .badge-warn { background:rgba(210,153,34,0.15); color:var(--yellow); }

  .console-input { display:flex; gap:8px; margin-bottom:14px; }
  .console-input input { flex:1; background:var(--surface); border:1px solid var(--border); color:var(--text); padding:10px 14px; border-radius:var(--radius); font-size:14px; outline:none; }
  .console-input input:focus { border-color:var(--accent); }
  .console-input button { background:var(--accent); color:#fff; border:none; padding:10px 20px; border-radius:var(--radius); cursor:pointer; font-weight:600; font-size:13px; }
  .console-input button:hover { opacity:.9; }
  .console-input button:disabled { opacity:.5; cursor:default; }

  .event-log { background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); max-height:55vh; overflow:auto; padding:12px; font-family:'SF Mono', 'Cascadia Code', 'Fira Code', monospace; font-size:12px; line-height:1.6; }
  .event { padding:4px 8px; border-left:3px solid transparent; margin-bottom:2px; }
  .event-start { border-color:var(--accent); }
  .event-thinking { border-color:var(--purple); color:var(--dim); }
  .event-tool_call { border-color:var(--yellow); }
  .event-tool_result { border-color:var(--orange); }
  .event-complete { border-color:var(--green); }
  .event-error { border-color:var(--red); }
  .evt-type { color:var(--dim); margin-right:8px; }
  .evt-label { font-weight:600; }

  table { width:100%; border-collapse:collapse; background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); overflow:hidden; }
  th { text-align:left; padding:9px 12px; font-size:11px; text-transform:uppercase; letter-spacing:.5px; color:var(--dim); background:var(--bg); border-bottom:1px solid var(--border); }
  td { padding:9px 12px; border-bottom:1px solid var(--border); font-size:13px; vertical-align:top; }
  tr:last-child td { border-bottom:none; }
  tr:hover td { background:rgba(88,166,255,0.04); }
  .role-badge { display:inline-block; padding:2px 8px; border-radius:4px; font-size:11px; font-weight:600; text-transform:uppercase; }
  .role-user { background:rgba(88,166,255,0.15); color:var(--accent); }
  .role-assistant { background:rgba(63,185,80,0.15); color:var(--green); }
  .role-tool { background:rgba(210,153,34,0.15); color:var(--yellow); }
  .role-system { background:rgba(139,148,158,0.15); color:var(--dim); }
  .msg-content { max-width:400px; white-space:pre-wrap; word-break:break-word; }

  .pagination { display:flex; gap:8px; align-items:center; margin-top:12px; justify-content:center; }
  .pagination button { background:var(--surface); border:1px solid var(--border); color:var(--text); padding:6px 14px; border-radius:var(--radius); cursor:pointer; font-size:13px; }
  .pagination button:disabled { opacity:.3; cursor:default; }
  .pagination span { color:var(--dim); font-size:13px; }

  .tool-card { background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:14px; margin-bottom:10px; }
  .tool-card h3 { font-size:14px; font-weight:600; margin-bottom:5px; }
  .tool-card .tool-desc { color:var(--dim); font-size:12px; margin-bottom:8px; }
  .tool-card .tool-params { font-size:12px; color:var(--dim); }
  .tool-card .tool-params code { background:var(--bg); padding:1px 6px; border-radius:3px; font-size:11px; }
  .param-req { color:var(--red); font-size:10px; }

  .prompt-box { background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:16px; font-family:'SF Mono', 'Cascadia Code', 'Fira Code', monospace; font-size:12px; line-height:1.6; white-space:pre-wrap; max-height:50vh; overflow:auto; }
  .bar-wrap { background:var(--bg); border-radius:3px; height:8px; margin-top:4px; overflow:hidden; }
  .bar-fill { height:100%; border-radius:3px; transition:width .5s; }
  .bar-green { background:var(--green); }
  .bar-red { background:var(--red); }

  .empty-state { text-align:center; padding:40px; color:var(--dim); }
  .section-title { font-size:13px; font-weight:600; color:var(--dim); margin:16px 0 8px 0; text-transform:uppercase; letter-spacing:.5px; }
  .expand-row { cursor:pointer; }
  .expand-row:hover td { background:rgba(88,166,255,0.08); }
  .expand-detail { display:none; background:var(--bg); }
  .expand-detail td { padding:12px 16px; }
  .expand-detail pre { margin:0; white-space:pre-wrap; font-size:12px; max-height:200px; overflow:auto; }
  .mem-subnav { display:flex; gap:4px; margin-bottom:14px; }
  .mem-subnav button { background:var(--surface); border:1px solid var(--border); color:var(--dim); padding:6px 14px; border-radius:var(--radius); cursor:pointer; font-size:12px; }
  .mem-subnav button.active { color:var(--accent); border-color:var(--accent); background:rgba(88,166,255,0.1); }

</style>
</head>
<body>
<header>
  <div><span class="dot"></span><strong>Ivai OS</strong> <span id="header-version"></span></div>
  <nav>
    <button class="active" data-tab="dashboard">Dashboard</button>
    <button data-tab="console">Task Console</button>
    <button data-tab="results">Task Results</button>
    <button data-tab="memory">Memory</button>
    <button data-tab="tools">Tools</button>
    <button data-tab="system">System</button>
    <button data-tab="swarm">Swarm</button>
  </nav>
  <div style="margin-left:auto;font-size:12px;color:var(--dim)" id="uptime"></div>
</header>
<main>
  <div id="tab-dashboard" class="tab active">
    <div class="grid" id="status-cards"></div>
    <div class="grid" id="task-summary"></div>
    <div class="card"><h3>LLM Providers</h3><div id="model-list">Loading...</div></div>
  </div>

  <div id="tab-console" class="tab">
    <div class="console-input">
      <input id="instruction" placeholder="Enter instruction..." />
      <button id="send-btn" onclick="sendTask()">&#9654; Send</button>
      <button id="stop-btn" style="background:var(--red)" onclick="stopTask()" disabled>&#9205; Stop</button>
    </div>
    <div class="event-log" id="event-log"><div class="empty-state">Send a task to see live events...</div></div>
  </div>

  <div id="tab-results" class="tab">
    <div id="results-header"></div>
    <table><thead><tr><th>ID</th><th>Instruction</th><th>Model</th><th>Status</th><th>Duration</th><th>Time</th></tr></thead>
    <tbody id="results-body"><tr><td colspan="6" class="empty-state">Loading...</td></tr></tbody></table>
    <div class="pagination" id="results-pager"></div>
  </div>

  <div id="tab-memory" class="tab">
    <table class="memory-table"><thead><tr><th>ID</th><th>Role</th><th>Content</th><th>Reasoning</th><th>Time</th></tr></thead>
    <tbody id="memory-body"><tr><td colspan="5" class="empty-state">Loading...</td></tr></tbody></table>
    <div class="pagination" id="memory-pager"></div>
  </div>

  <div id="tab-tools" class="tab"><div id="tools-list"></div></div>

  <div id="tab-system" class="tab">
    <div class="grid" id="system-cards"></div>
    <div class="section-title">System Prompt</div>
    <div class="prompt-box" id="system-prompt">Loading...</div>
  </div>

  <div id="tab-swarm" class="tab">
    <div class="grid" id="swarm-worker-cards"></div>
    <div id="swarm-worker-detail" style="display:none">
      <div style="display:flex;gap:8px;align-items:center;margin-bottom:10px">
        <button id="swarm-back-btn" style="background:var(--surface);border:1px solid var(--border);color:var(--text);padding:6px 14px;border-radius:var(--radius);cursor:pointer;font-size:13px">&larr; Back</button>
        <span class="section-title" style="margin:0">Worker: <span id="swarm-detail-name"></span></span>
        <button id="swarm-refresh-log" style="background:var(--surface);border:1px solid var(--border);color:var(--text);padding:6px 14px;border-radius:var(--radius);cursor:pointer;font-size:13px;margin-left:auto">&#8635; Refresh</button>
      </div>
      <div class="card"><div class="event-log" id="swarm-log-view" style="max-height:40vh">Loading logs...</div></div>
    </div>
    <div class="section-title" style="margin-top:16px">Dispatch Task</div>
    <div class="console-input" style="margin-bottom:8px">
      <input id="swarm-worker-input" placeholder="Worker (e.g. localhost:8081)" style="max-width:220px" />
      <input id="swarm-instruction-input" placeholder="Enter instruction..." />
      <button id="swarm-dispatch-btn" onclick="dispatchSwarmTask()">&#9654; Send</button>
    </div>
    <div id="swarm-dispatch-result" class="event-log" style="max-height:25vh;display:none"></div>
  </div>
</main>

<script>
document.querySelectorAll('nav button').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('nav button').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById('tab-' + btn.dataset.tab).classList.add('active');
    if (btn.dataset.tab === 'dashboard') loadStatus();
    if (btn.dataset.tab === 'results') loadResults();
    if (btn.dataset.tab === 'memory') loadMemory();
    if (btn.dataset.tab === 'tools') loadTools();
    if (btn.dataset.tab === 'system') loadSystem();
    if (btn.dataset.tab === 'swarm') loadSwarmWorkers();
  });
});

async function loadStatus() {
  try {
    const [sr, tr] = await Promise.all([fetch('/api/status').then(r=>r.json()), fetch('/api/task-results?limit=1').then(r=>r.json())]);
    document.getElementById('header-version').textContent = 'v' + sr.version;
    document.getElementById('status-cards').innerHTML =
      '<div class="card"><h3>Uptime</h3><div class="value">'+fmtDuration(sr.uptime_sec)+'</div><div class="sub">since start</div></div>'+
      '<div class="card"><h3>Go Runtime</h3><div class="value">'+sr.go_version+'</div><div class="sub">'+sr.goroutines+' goroutines &middot; '+sr.heap_alloc_mb.toFixed(1)+' MB</div></div>'+
      '<div class="card"><h3>System</h3><div class="value">'+sr.os+'/'+sr.arch+'</div><div class="sub">'+sr.num_cpu+' CPUs</div></div>'+
      '<div class="card"><h3>Active Model</h3><div class="value" style="font-size:18px">'+sr.active_model+'</div></div>';
    if (sr.build_date) document.getElementById('status-cards').innerHTML += '<div class="card"><h3>Build</h3><div class="value" style="font-size:14px">'+sr.commit+'</div><div class="sub">'+sr.build_date+'</div></div>';
    document.getElementById('uptime').textContent = 'Up ' + fmtDuration(sr.uptime_sec);

    let mh = '';
    sr.models.forEach(m => {
      const ok = m.available === 'true';
      mh += '<div class="model-row"><span>'+m.id+' <span style="color:var(--dim);font-size:11px">('+m.provider+')</span></span>'+
            '<span class="badge '+(ok?'badge-ok':'badge-err')+'">'+(ok?'ready':'no key')+'</span></div>';
    });
    document.getElementById('model-list').innerHTML = mh;

    document.getElementById('task-summary').innerHTML = '<div class="card"><h3>Last Task</h3><div class="value" style="font-size:13px">'+(tr.results&&tr.results[0]?escHtml(tr.results[0].instruction.slice(0,60)):'&mdash;')+'</div><div class="sub">'+(tr.results&&tr.results[0]?(tr.results[0].success?'&#9989;':'&#10060;')+' &middot; '+tr.results[0].duration_ms+'ms':'no tasks yet')+'</div></div>';
  } catch(e) { console.error(e); }
}

let resPage = 0;
async function loadResults() {
  try {
    const [r, sys] = await Promise.all([
      fetch('/api/task-results?limit=30&offset='+(resPage*30)).then(r=>r.json()),
      fetch('/api/system').then(r=>r.json())
    ]);
    const stats = sys.task_stats || {};
    document.getElementById('results-header').innerHTML =
      '<div class="grid" style="margin-bottom:14px">'+
      '<div class="card"><h3>Total Tasks</h3><div class="value">'+stats.total+'</div></div>'+
      '<div class="card"><h3>Success Rate</h3><div class="value" style="color:'+(stats.success_rate>=80?'var(--green)':'var(--red)')+'">'+(stats.success_rate||0).toFixed(0)+'%</div><div class="bar-wrap"><div class="bar-fill bar-green" style="width:'+(stats.success_rate||0)+'%"></div></div></div>'+
      '<div class="card"><h3>Avg Duration</h3><div class="value">'+(stats.avg_duration_ms||0)+'ms</div></div>'+
      '<div class="card"><h3>Success / Fail</h3><div class="value">'+stats.successes+' / '+stats.failures+'</div></div>'+
      '</div>';

    let rows = '';
    (r.results||[]).forEach((m, i) => {
      const rowId = 'res-row-'+i;
      rows += '<tr class="expand-row" onclick="toggleExpand(\x27'+rowId+'\x27)"><td>'+m.id+'</td><td class="msg-content">'+escHtml(m.instruction.slice(0,80))+'</td>'+
              '<td>'+m.model+'</td><td><span class="badge '+(m.success?'badge-ok':'badge-err')+'">'+(m.success?'OK':'ERR')+'</span></td>'+
              '<td>'+m.duration_ms+'ms</td><td style="white-space:nowrap;color:var(--dim);font-size:11px">'+m.created_at+'</td></tr>'+
              '<tr id="'+rowId+'" class="expand-detail"><td colspan="6">'+
              '<strong>Instruction:</strong><pre>'+escHtml(m.instruction)+'</pre>'+
              '<strong>Response:</strong><pre>'+escHtml(m.response)+'</pre>'+
              (m.error_msg ? '<strong style="color:var(--red)">Error:</strong><pre style="color:var(--red)">'+escHtml(m.error_msg)+'</pre>' : '')+
              '</td></tr>';
    });
    document.getElementById('results-body').innerHTML = rows || '<tr><td colspan="6" class="empty-state">No tasks yet</td></tr>';
    const pages = Math.ceil(stats.total/30);
    document.getElementById('results-pager').innerHTML =
      '<button onclick="resPage=0;loadResults()" '+(resPage===0?'disabled':'')+'>First</button>'+
      '<button onclick="resPage--;loadResults()" '+(resPage===0?'disabled':'')+'>Prev</button>'+
      '<span>Page '+(resPage+1)+' of '+(pages||1)+'</span>'+
      '<button onclick="resPage++;loadResults()" '+(resPage>=pages-1?'disabled':'')+'>Next</button>';
  } catch(e) { console.error(e); }
}

let memPage = 0;
async function loadMemory() {
  try {
    const r = await fetch('/api/memory?limit=30&offset='+(memPage*30)).then(r=>r.json());
    let rows = '';
    r.messages.forEach(m => {
      const content = m.content.length > 250 ? m.content.slice(0,250)+'...' : m.content;
      const reasoning = m.reasoning_content ? (m.reasoning_content.length > 150 ? m.reasoning_content.slice(0,150)+'...' : m.reasoning_content) : '&mdash;';
      rows += '<tr><td>'+m.id+'</td><td><span class="role-badge role-'+m.role+'">'+m.role+'</span></td>'+
              '<td class="msg-content">'+escHtml(content)+'</td>'+
              '<td class="msg-content" style="color:var(--dim)">'+escHtml(reasoning)+'</td>'+
              '<td style="white-space:nowrap;color:var(--dim);font-size:11px">'+m.created_at+'</td></tr>';
    });
    document.getElementById('memory-body').innerHTML = rows || '<tr><td colspan="5" class="empty-state">No messages</td></tr>';
    const pages = Math.ceil(r.total/30);
    document.getElementById('memory-pager').innerHTML =
      '<button onclick="memPage=0;loadMemory()" '+(memPage===0?'disabled':'')+'>First</button>'+
      '<button onclick="memPage--;loadMemory()" '+(memPage===0?'disabled':'')+'>Prev</button>'+
      '<span>Page '+(memPage+1)+' of '+(pages||1)+' ('+r.total+' msgs)</span>'+
      '<button onclick="memPage++;loadMemory()" '+(memPage>=pages-1?'disabled':'')+'>Next</button>';
  } catch(e) { console.error(e); }
}

async function loadTools() {
  try {
    const r = await fetch('/api/tools').then(r=>r.json());
    let html = '';
    r.tools.forEach(t => {
      html += '<div class="tool-card"><h3>'+t.name+'</h3><div class="tool-desc">'+t.description+'</div><div class="tool-params">';
      const props = t.parameters.properties || {};
      const reqs = t.parameters.required || [];
      Object.keys(props).forEach(k => {
        const isReq = reqs.includes(k);
        html += '<span style="margin-right:12px"><code>'+k+'</code>: '+props[k].type+(isReq?' <span class="param-req">required</span>':'')+'</span>';
      });
      html += '</div></div>';
    });
    document.getElementById('tools-list').innerHTML = html;
  } catch(e) { console.error(e); }
}

async function loadSystem() {
  try {
    const [sys, status] = await Promise.all([
      fetch('/api/system').then(r=>r.json()),
      fetch('/api/status').then(r=>r.json())
    ]);
    document.getElementById('system-cards').innerHTML =
      '<div class="card"><h3>Embeddings</h3><div class="value">'+sys.embeddings_count+'</div><div class="sub">vector memory entries</div></div>'+
      '<div class="card"><h3>Messages</h3><div class="value">'+sys.messages_count+'</div><div class="sub">conversation history</div></div>'+
      '<div class="card"><h3>Tasks</h3><div class="value">'+(sys.task_stats.total||0)+'</div><div class="sub">'+(sys.task_stats.successes||0)+' ok / '+(sys.task_stats.failures||0)+' fail</div></div>'+
      '<div class="card"><h3>Models</h3><div class="value">'+status.models.length+'</div><div class="sub">'+(status.models.filter(function(m){return m.available==='true'}).length)+' ready</div></div>';
    document.getElementById('system-prompt').textContent = sys.system_prompt || 'Not available';
  } catch(e) { console.error(e); }
}

async function sendTask() {
  const input = document.getElementById('instruction');
  const instruction = input.value.trim();
  if (!instruction) return;
  document.getElementById('send-btn').disabled = true;
  document.getElementById('stop-btn').disabled = false;
  const log = document.getElementById('event-log');
  log.innerHTML = '';
  try {
    const resp = await fetch('/api/task/stream', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({instruction})});
    const reader = resp.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '', eventType = '', dataBuffer = '';
    while (true) {
      const {done, value} = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, {stream:true});
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';
      for (const line of lines) {
        if (line === '') {
          if (dataBuffer) { appendEvent(eventType, dataBuffer, log); eventType = ''; dataBuffer = ''; }
          continue;
        }
        if (line.startsWith('event: ')) eventType = line.slice(7);
        else if (line.startsWith('data: ')) dataBuffer = line.slice(6);
      }
    }
  } catch(e) { log.innerHTML += '<div class="event event-error"><span class="evt-type">[error]</span>'+escHtml(e.message)+'</div>'; }
  document.getElementById('send-btn').disabled = false;
  document.getElementById('stop-btn').disabled = true;
}

function appendEvent(type, data, log) {
  let evt;
  try { evt = JSON.parse(data); } catch(e) { evt = {type:type, message:data}; }
  const cls = 'event-' + (type || 'unknown');
  let html = '<div class="event '+cls+'">';
  switch(type) {
    case 'task_start': const d=evt.data||{}; html+='<span class="evt-type">[start]</span><span class="evt-label">Model:</span> '+escHtml(d.model||'')+' <span class="evt-label">Instruction:</span> '+escHtml(d.instruction||''); break;
    case 'thinking': const td=evt.data||{}; if(td.reasoning)html+='<span class="evt-type">[thinking]</span>'+escHtml(td.reasoning); if(td.content)html+='<div style="margin-top:4px">'+escHtml(td.content)+'</div>'; break;
    case 'tool_call': const tcd=evt.data||{}; html+='<span class="evt-type">[tool]</span><span class="evt-label">'+escHtml(tcd.name||'')+'</span> &rarr; '+escHtml(String(tcd.args||'')); break;
    case 'tool_result': const trd=evt.data||{}; html+='<span class="evt-type">[result]</span><span class="evt-label">'+escHtml(trd.name||'')+'</span> &rarr; <pre style="margin:0;white-space:pre-wrap;font-size:11px">'+escHtml(String(trd.result||''))+'</pre>'; break;
    case 'task_complete': const ccd=evt.data||{}; html+='<span class="evt-type">[complete]</span><div style="white-space:pre-wrap;margin-top:4px">'+escHtml(ccd.response||'')+'</div>'; break;
    case 'task_error': const erd=evt.data||{}; html+='<span class="evt-type">[error]</span>'+escHtml(erd.error||evt.message||''); break;
    default: html+='<span class="evt-type">['+type+']</span>'+escHtml(data);
  }
  html += '</div>';
  log.innerHTML += html;
  log.scrollTop = log.scrollHeight;
}

function stopTask() { location.reload(); }
function fmtDuration(sec) {
  if (sec<60) return sec+'s';
  if (sec<3600) return Math.floor(sec/60)+'m '+(sec%60)+'s';
  return Math.floor(sec/3600)+'h '+Math.floor((sec%3600)/60)+'m';
}
function escHtml(s) { const d=document.createElement('div'); d.textContent=s; return d.innerHTML; }

function toggleExpand(id) { const el=document.getElementById(id); el.style.display=el.style.display==='table-row'?'none':'table-row'; }

let memType = 'messages';
function switchMemoryType(type) {
  memType = type;
  document.querySelectorAll('.mem-subnav button').forEach(b => b.classList.remove('active'));
  event.target.classList.add('active');
  if (type === 'messages') {
    document.querySelector('#tab-memory .memory-table thead').innerHTML = '<tr><th>ID</th><th>Role</th><th>Content</th><th>Reasoning</th><th>Time</th></tr>';
    loadMemory();
  } else {
    loadEmbeddings();
  }
}

async function loadEmbeddings() {
  try {
    const r = await fetch('/api/embeddings?limit=50').then(r=>r.json());
    document.querySelector('#tab-memory .memory-table thead').innerHTML = '<tr><th>Source</th><th>Content</th></tr>';
    let rows = '';
    (r.embeddings||[]).forEach(e => {
      rows += '<tr><td><span class="role-badge role-'+e.source+'">'+e.source+'</span></td><td class="msg-content">'+escHtml(e.content.slice(0,300))+'</td></tr>';
    });
    document.getElementById('memory-body').innerHTML = rows || '<tr><td colspan="2" class="empty-state">No embeddings</td></tr>';
    document.getElementById('memory-pager').innerHTML = '';
  } catch(e) { console.error(e); }
}

loadStatus(); setInterval(loadStatus, 30000);

// --- Swarm Tab ---
let swarmRefresh = null;

async function loadSwarmWorkers() {
  try {
    const workers = await fetch('/api/swarm/workers').then(r=>r.json());
    if (!workers.length) {
      document.getElementById('swarm-worker-cards').innerHTML = '<div class="card" style="grid-column:1/-1"><div class="empty-state" style="padding:20px">No workers running. Use <strong>swarm_spawn</strong> to start one.</div></div>';
      return;
    }
    let html = '';
    workers.forEach(w => {
      const uptime = fmtDuration(w.uptime_sec || 0);
      const logSize = w.log_size > 1024 ? (w.log_size/1024).toFixed(1)+' KB' : (w.log_size||0)+' B';
      html += '<div class="card" style="cursor:pointer" onclick="showWorkerLog(\''+w.name+'\')">'+
        '<h3>'+escHtml(w.name)+'</h3>'+
        '<div class="value" style="font-size:16px">'+w.port+'</div>'+
        '<div class="sub">'+uptime+' &middot; '+logSize+'</div>'+
        '</div>';
    });
    document.getElementById('swarm-worker-cards').innerHTML = html;
  } catch(e) { console.error(e); }
}

async function showWorkerLog(name) {
  document.getElementById('swarm-worker-cards').style.display = 'none';
  document.getElementById('swarm-worker-detail').style.display = 'block';
  document.getElementById('swarm-detail-name').textContent = name;
  document.getElementById('swarm-worker-input').value = 'localhost:' + (await getWorkerPort(name));
  await refreshWorkerLog(name);
  if (swarmRefresh) clearInterval(swarmRefresh);
  swarmRefresh = setInterval(() => refreshWorkerLog(name), 3000);
}

async function getWorkerPort(name) {
  try {
    const workers = await fetch('/api/swarm/workers').then(r=>r.json());
    const w = workers.find(x => x.name === name);
    return w ? w.port : '8081';
  } catch(e) { return '8081'; }
}

async function refreshWorkerLog(name) {
  try {
    const log = await fetch('/api/swarm/logs?worker='+encodeURIComponent(name)+'&lines=100').then(r=>r.text());
    document.getElementById('swarm-log-view').innerHTML = '<pre style="margin:0;white-space:pre-wrap;font-size:11px;line-height:1.5">'+escHtml(log)+'</pre>';
  } catch(e) { document.getElementById('swarm-log-view').innerHTML = '<div class="event event-error">Error loading logs: '+escHtml(e.message)+'</div>'; }
}

document.addEventListener('click', function(e) {
  if (e.target.id === 'swarm-back-btn') {
    document.getElementById('swarm-worker-cards').style.display = '';
    document.getElementById('swarm-worker-detail').style.display = 'none';
    if (swarmRefresh) { clearInterval(swarmRefresh); swarmRefresh = null; }
  }
  if (e.target.id === 'swarm-refresh-log') {
    const name = document.getElementById('swarm-detail-name').textContent;
    if (name) refreshWorkerLog(name);
  }
});

async function dispatchSwarmTask() {
  const worker = document.getElementById('swarm-worker-input').value.trim();
  const instruction = document.getElementById('swarm-instruction-input').value.trim();
  if (!worker || !instruction) return;
  const resultDiv = document.getElementById('swarm-dispatch-result');
  resultDiv.style.display = 'block';
  resultDiv.innerHTML = '<div class="event event-start"><span class="evt-type">[dispatching]</span> Sending to '+escHtml(worker)+'...</div>';
  try {
    const resp = await fetch('/api/swarm/dispatch', {
      method:'POST',
      headers:{'Content-Type':'application/json'},
      body:JSON.stringify({worker, instruction})
    }).then(r=>r.json());
    if (resp.error) {
      resultDiv.innerHTML = '<div class="event event-error"><span class="evt-type">[error]</span>'+escHtml(resp.error)+'</div>';
    } else {
      resultDiv.innerHTML = '<div class="event event-complete"><pre style="margin:0;white-space:pre-wrap;font-size:11px">'+escHtml(resp.response||'')+'</pre></div>';
    }
  } catch(e) {
    resultDiv.innerHTML = '<div class="event event-error"><span class="evt-type">[error]</span>'+escHtml(e.message)+'</div>';
  }
}
</script>
</body>
</html>`
