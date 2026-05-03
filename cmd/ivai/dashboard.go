package main


const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Ivai OS — Admin Dashboard</title>
<style>
  :root {
    --bg: #0d1117;
    --surface: #161b22;
    --border: #30363d;
    --text: #c9d1d9;
    --dim: #8b949e;
    --accent: #58a6ff;
    --green: #3fb950;
    --red: #f85149;
    --yellow: #d29922;
    --purple: #a371f7;
    --orange: #d2693a;
    --radius: 6px;
  }
  * { margin:0; padding:0; box-sizing:border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif; background: var(--bg); color: var(--text); height:100vh; display:flex; flex-direction:column; }
  header { background: var(--surface); border-bottom:1px solid var(--border); padding:12px 20px; display:flex; align-items:center; gap:16px; flex-shrink:0; }
  header h1 { font-size:18px; font-weight:600; }
  header .dot { width:8px; height:8px; border-radius:50%; background:var(--green); display:inline-block; margin-right:6px; }
  nav { display:flex; gap:4px; margin-left:24px; }
  nav button { background:none; border:1px solid transparent; color:var(--dim); padding:6px 14px; border-radius:var(--radius); cursor:pointer; font-size:13px; transition:all .15s; }
  nav button:hover { color:var(--text); background:var(--border); }
  nav button.active { color:var(--accent); border-color:var(--accent); background:rgba(88,166,255,0.1); }
  main { flex:1; overflow:auto; padding:20px; }
  .tab { display:none; }
  .tab.active { display:block; }

  /* Dashboard cards */
  .grid { display:grid; grid-template-columns:repeat(auto-fill, minmax(260px, 1fr)); gap:16px; margin-bottom:24px; }
  .card { background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:16px; }
  .card h3 { font-size:12px; text-transform:uppercase; letter-spacing:.5px; color:var(--dim); margin-bottom:8px; }
  .card .value { font-size:28px; font-weight:700; }
  .card .sub { font-size:12px; color:var(--dim); margin-top:4px; }

  .model-row { display:flex; justify-content:space-between; align-items:center; padding:8px 0; border-bottom:1px solid var(--border); }
  .model-row:last-child { border-bottom:none; }
  .badge { font-size:10px; padding:2px 8px; border-radius:10px; font-weight:600; text-transform:uppercase; }
  .badge-ok { background:rgba(63,185,80,0.15); color:var(--green); }
  .badge-err { background:rgba(248,81,73,0.15); color:var(--red); }

  /* Console */
  .console-input { display:flex; gap:8px; margin-bottom:16px; }
  .console-input input { flex:1; background:var(--surface); border:1px solid var(--border); color:var(--text); padding:10px 14px; border-radius:var(--radius); font-size:14px; outline:none; }
  .console-input input:focus { border-color:var(--accent); }
  .console-input button { background:var(--accent); color:#fff; border:none; padding:10px 20px; border-radius:var(--radius); cursor:pointer; font-weight:600; font-size:13px; }
  .console-input button:hover { opacity:.9; }
  .console-input button:disabled { opacity:.5; cursor:default; }

  .event-log { background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); max-height:60vh; overflow:auto; padding:12px; font-family:'SF Mono', 'Cascadia Code', 'Fira Code', monospace; font-size:12px; line-height:1.6; }
  .event { padding:4px 8px; border-left:3px solid transparent; margin-bottom:2px; }
  .event-start { border-color:var(--accent); }
  .event-thinking { border-color:var(--purple); color:var(--dim); }
  .event-tool_call { border-color:var(--yellow); }
  .event-tool_result { border-color:var(--orange); }
  .event-complete { border-color:var(--green); }
  .event-error { border-color:var(--red); }
  .evt-type { color:var(--dim); margin-right:8px; }
  .evt-label { font-weight:600; }

  /* Memory table */
  .memory-table { width:100%; border-collapse:collapse; background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); overflow:hidden; }
  .memory-table th { text-align:left; padding:10px 14px; font-size:11px; text-transform:uppercase; letter-spacing:.5px; color:var(--dim); background:var(--bg); border-bottom:1px solid var(--border); }
  .memory-table td { padding:10px 14px; border-bottom:1px solid var(--border); font-size:13px; vertical-align:top; }
  .memory-table tr:last-child td { border-bottom:none; }
  .memory-table tr:hover td { background:rgba(88,166,255,0.04); }
  .role-badge { display:inline-block; padding:2px 8px; border-radius:4px; font-size:11px; font-weight:600; text-transform:uppercase; }
  .role-user { background:rgba(88,166,255,0.15); color:var(--accent); }
  .role-assistant { background:rgba(63,185,80,0.15); color:var(--green); }
  .role-tool { background:rgba(210,153,34,0.15); color:var(--yellow); }
  .role-system { background:rgba(139,148,158,0.15); color:var(--dim); }
  .msg-content { max-width:500px; white-space:pre-wrap; word-break:break-word; }
  .pagination { display:flex; gap:8px; align-items:center; margin-top:12px; justify-content:center; }
  .pagination button { background:var(--surface); border:1px solid var(--border); color:var(--text); padding:6px 14px; border-radius:var(--radius); cursor:pointer; font-size:13px; }
  .pagination button:disabled { opacity:.3; cursor:default; }
  .pagination span { color:var(--dim); font-size:13px; }

  /* Tools */
  .tool-card { background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:16px; margin-bottom:12px; }
  .tool-card h3 { font-size:15px; font-weight:600; margin-bottom:6px; }
  .tool-card .tool-desc { color:var(--dim); font-size:13px; margin-bottom:10px; }
  .tool-card .tool-params { font-size:12px; color:var(--dim); }
  .tool-card .tool-params code { background:var(--bg); padding:1px 6px; border-radius:3px; font-size:11px; }
  .param-req { color:var(--red); font-size:11px; }

  .empty-state { text-align:center; padding:40px; color:var(--dim); }
  .toast { position:fixed; bottom:20px; right:20px; background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:12px 20px; box-shadow:0 4px 12px rgba(0,0,0,.4); z-index:100; animation:slideIn .3s ease; }
  @keyframes slideIn { from { transform:translateY(20px); opacity:0; } to { transform:translateY(0); opacity:1; } }
</style>
</head>
<body>
<header>
  <div><span class="dot"></span><strong>Ivai OS</strong> __VERSION__</div>
  <nav>
    <button class="active" data-tab="dashboard">Dashboard</button>
    <button data-tab="console">Task Console</button>
    <button data-tab="memory">Memory</button>
    <button data-tab="tools">Tools</button>
  </nav>
  <div style="margin-left:auto;font-size:12px;color:var(--dim)" id="uptime"></div>
</header>
<main>
  <div id="tab-dashboard" class="tab active">
    <div class="grid" id="status-cards"></div>
    <div class="card">
      <h3>LLM Providers</h3>
      <div id="model-list">Loading...</div>
    </div>
  </div>
  <div id="tab-console" class="tab">
    <div class="console-input">
      <input id="instruction" placeholder="Enter instruction (e.g., list all Go files and count their lines)" />
      <button id="send-btn" onclick="sendTask()">▶ Send</button>
      <button id="stop-btn" style="background:var(--red)" onclick="stopTask()" disabled>⏹ Stop</button>
    </div>
    <div class="event-log" id="event-log">
      <div class="empty-state">Send a task to see live events here...</div>
    </div>
  </div>
  <div id="tab-memory" class="tab">
    <table class="memory-table">
      <thead><tr><th>ID</th><th>Role</th><th>Content</th><th>Reasoning</th><th>Time</th></tr></thead>
      <tbody id="memory-body"><tr><td colspan="5" class="empty-state">Loading...</td></tr></tbody>
    </table>
    <div class="pagination" id="memory-pager"></div>
  </div>
  <div id="tab-tools" class="tab">
    <div id="tools-list"></div>
  </div>
</main>

<script>
// --- Tab switching ---
document.querySelectorAll('nav button').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('nav button').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById('tab-' + btn.dataset.tab).classList.add('active');
    if (btn.dataset.tab === 'dashboard') loadStatus();
    if (btn.dataset.tab === 'memory') loadMemory();
    if (btn.dataset.tab === 'tools') loadTools();
  });
});

// --- Status API ---
async function loadStatus() {
  try {
    const r = await fetch('/api/status');
    const d = await r.json();
    document.getElementById('status-cards').innerHTML =
      '<div class="card"><h3>Uptime</h3><div class="value">' + fmtDuration(d.uptime_sec) + '</div><div class="sub">since start</div></div>' +
      '<div class="card"><h3>Go Runtime</h3><div class="value">' + d.go_version + '</div><div class="sub">' + d.goroutines + ' goroutines · ' + d.heap_alloc_mb.toFixed(1) + ' MB heap</div></div>' +
      '<div class="card"><h3>System</h3><div class="value">' + d.os + '/' + d.arch + '</div><div class="sub">' + d.num_cpu + ' CPUs</div></div>' +
      '<div class="card"><h3>Active Model</h3><div class="value" style="font-size:20px">' + d.active_model + '</div></div>';
    document.getElementById('uptime').textContent = 'Up ' + fmtDuration(d.uptime_sec);

    let mh = '';
    d.models.forEach(m => {
      const ok = m.available === 'true';
      mh += '<div class="model-row"><span>' + m.id + ' <span style="color:var(--dim);font-size:12px">(' + m.provider + ')</span></span>' +
            '<span class="badge ' + (ok ? 'badge-ok' : 'badge-err') + '">' + (ok ? 'ready' : 'no key') + '</span></div>';
    });
    document.getElementById('model-list').innerHTML = mh;
  } catch(e) { console.error(e); }
}

// --- Memory API ---
let memPage = 0;
const memLimit = 30;

async function loadMemory() {
  try {
    const r = await fetch('/api/memory?limit=' + memLimit + '&offset=' + (memPage * memLimit));
    const d = await r.json();
    let rows = '';
    d.messages.forEach(m => {
      const content = m.content.length > 300 ? m.content.slice(0, 300) + '...' : m.content;
      const reasoning = m.reasoning_content ? (m.reasoning_content.length > 200 ? m.reasoning_content.slice(0, 200) + '...' : m.reasoning_content) : '—';
      const roleClass = 'role-' + m.role;
      rows += '<tr><td>' + m.id + '</td><td><span class="role-badge ' + roleClass + '">' + m.role + '</span></td>' +
              '<td class="msg-content">' + escHtml(content) + '</td>' +
              '<td class="msg-content" style="color:var(--dim)">' + escHtml(reasoning) + '</td>' +
              '<td style="white-space:nowrap;color:var(--dim);font-size:11px">' + m.created_at + '</td></tr>';
    });
    document.getElementById('memory-body').innerHTML = rows || '<tr><td colspan="5" class="empty-state">No messages yet</td></tr>';
    const totalPages = Math.ceil(d.total / memLimit);
    document.getElementById('memory-pager').innerHTML =
      '<button onclick="memPage=0;loadMemory()" ' + (memPage === 0 ? 'disabled' : '') + '>First</button>' +
      '<button onclick="memPage--;loadMemory()" ' + (memPage === 0 ? 'disabled' : '') + '>← Prev</button>' +
      '<span>Page ' + (memPage + 1) + ' of ' + (totalPages || 1) + ' (' + d.total + ' messages)</span>' +
      '<button onclick="memPage++;loadMemory()" ' + (memPage >= totalPages - 1 ? 'disabled' : '') + '>Next →</button>' +
      '<button onclick="memPage=' + (totalPages - 1) + ';loadMemory()" ' + (memPage >= totalPages - 1 ? 'disabled' : '') + '>Last</button>';
  } catch(e) { console.error(e); }
}

// --- Tools API ---
async function loadTools() {
  try {
    const r = await fetch('/api/tools');
    const d = await r.json();
    let html = '';
    d.tools.forEach(t => {
      html += '<div class="tool-card"><h3>' + t.name + '</h3><div class="tool-desc">' + t.description + '</div><div class="tool-params">';
      const props = t.parameters.properties || {};
      const reqs = t.parameters.required || [];
      Object.keys(props).forEach(k => {
        const isReq = reqs.includes(k);
        html += '<span style="margin-right:12px"><code>' + k + '</code>: ' + props[k].type + (isReq ? ' <span class="param-req">required</span>' : '') + '</span>';
      });
      html += '</div></div>';
    });
    document.getElementById('tools-list').innerHTML = html;
  } catch(e) { console.error(e); }
}

// --- SSE Task Console ---
let currentEventSource = null;

function sendTask() {
  const input = document.getElementById('instruction');
  const instruction = input.value.trim();
  if (!instruction) return;

  document.getElementById('send-btn').disabled = true;
  document.getElementById('stop-btn').disabled = false;
  document.getElementById('event-log').innerHTML = '';

  const es = new EventSource('/api/task/stream?' + new URLSearchParams({_:Date.now()}));

  // EventSource is GET-only but our endpoint is POST. We need fetch+ReadableStream.
  // Fallback to fetch-based SSE parsing.
  sendTaskViaFetch(instruction);
}

async function sendTaskViaFetch(instruction) {
  const log = document.getElementById('event-log');
  log.innerHTML = '';

  try {
    const resp = await fetch('/api/task/stream', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({instruction})
    });

    const reader = resp.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let eventType = '';
    let dataBuffer = '';

    while (true) {
      const {done, value} = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, {stream: true});
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line === '') {
          if (dataBuffer) {
            appendEvent(eventType, dataBuffer, log);
            eventType = '';
            dataBuffer = '';
          }
          continue;
        }
        if (line.startsWith('event: ')) eventType = line.slice(7);
        else if (line.startsWith('data: ')) dataBuffer = line.slice(6);
      }
    }

    document.getElementById('send-btn').disabled = false;
    document.getElementById('stop-btn').disabled = true;
  } catch(e) {
    log.innerHTML += '<div class="event event-error"><span class="evt-type">[error]</span>' + escHtml(e.message) + '</div>';
    document.getElementById('send-btn').disabled = false;
    document.getElementById('stop-btn').disabled = true;
  }
}

function appendEvent(type, data, log) {
  let evt;
  try { evt = JSON.parse(data); } catch(e) { evt = {type:type, message:data}; }

  const cls = 'event-' + (type || 'unknown');
  let html = '<div class="event ' + cls + '">';

  switch(type) {
    case 'task_start':
      const d = evt.data || {};
      html += '<span class="evt-type">[start]</span><span class="evt-label">Model:</span> ' + escHtml(d.model || '') +
              ' <span class="evt-label">Instruction:</span> ' + escHtml(d.instruction || '');
      break;
    case 'thinking':
      const td = evt.data || {};
      if (td.reasoning) html += '<span class="evt-type">[thinking]</span>' + escHtml(td.reasoning);
      if (td.content) html += '<div style="margin-top:4px">' + escHtml(td.content) + '</div>';
      break;
    case 'tool_call':
      const tcd = evt.data || {};
      html += '<span class="evt-type">[tool]</span><span class="evt-label">' + escHtml(tcd.name || '') + '</span> → ' + escHtml(String(tcd.args || ''));
      break;
    case 'tool_result':
      const trd = evt.data || {};
      html += '<span class="evt-type">[result]</span><span class="evt-label">' + escHtml(trd.name || '') + '</span> → <pre style="margin:0;white-space:pre-wrap;font-size:11px">' + escHtml(String(trd.result || '')) + '</pre>';
      break;
    case 'task_complete':
      const ccd = evt.data || {};
      html += '<span class="evt-type">[complete]</span><div style="white-space:pre-wrap;margin-top:4px">' + escHtml(ccd.response || '') + '</div>';
      break;
    case 'task_error':
      const erd = evt.data || {};
      html += '<span class="evt-type">[error]</span>' + escHtml(erd.error || evt.message || '');
      break;
    default:
      html += '<span class="evt-type">[' + type + ']</span>' + escHtml(data);
  }

  html += '</div>';
  log.innerHTML += html;
  log.scrollTop = log.scrollHeight;
}

function stopTask() {
  // SSE will be stopped when fetch aborts — but we can't easily abort with current design.
  // Reload page as a simple stop mechanism.
  location.reload();
}

// --- Helpers ---
function fmtDuration(sec) {
  if (sec < 60) return sec + 's';
  if (sec < 3600) return Math.floor(sec/60) + 'm ' + (sec%60) + 's';
  const h = Math.floor(sec/3600), m = Math.floor((sec%3600)/60);
  return h + 'h ' + m + 'm';
}

function escHtml(s) {
  const d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

// --- Init ---
loadStatus();
setInterval(loadStatus, 30000);
</script>
</body>
</html>`

