package main

var dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Sera Log Analyzer - Master Dashboard</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: 'Segoe UI', system-ui, sans-serif; background: #0f1117; color: #e1e4e8; }
  .header { background: linear-gradient(135deg, #1a1c2e 0%, #2d1b4e 100%); padding: 20px 30px; border-bottom: 1px solid #333; }
  .header h1 { font-size: 24px; color: #a78bfa; }
  .header p { color: #8b8fa3; margin-top: 4px; font-size: 13px; }
  .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
  .stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 24px; }
  .stat-card { background: #1a1c2e; border: 1px solid #2a2d3e; border-radius: 12px; padding: 20px; }
  .stat-card .label { color: #8b8fa3; font-size: 12px; text-transform: uppercase; letter-spacing: 1px; }
  .stat-card .value { font-size: 28px; font-weight: 700; margin-top: 8px; color: #a78bfa; }
  .stat-card .value.green { color: #34d399; }
  .stat-card .value.yellow { color: #fbbf24; }
  .stat-card .value.red { color: #f87171; }
  .stat-card .value.blue { color: #60a5fa; }
  .tabs { display: flex; gap: 4px; margin-bottom: 20px; border-bottom: 1px solid #2a2d3e; padding-bottom: 0; }
  .tab { padding: 10px 20px; cursor: pointer; border-radius: 8px 8px 0 0; color: #8b8fa3; font-size: 14px; transition: all 0.2s; border: 1px solid transparent; border-bottom: none; }
  .tab.active { background: #1a1c2e; color: #a78bfa; border-color: #2a2d3e; }
  .tab:hover { color: #a78bfa; }
  .panel { display: none; background: #1a1c2e; border: 1px solid #2a2d3e; border-radius: 12px; padding: 20px; }
  .panel.active { display: block; }
  table { width: 100%; border-collapse: collapse; }
  th { text-align: left; padding: 12px; color: #8b8fa3; font-size: 12px; text-transform: uppercase; border-bottom: 1px solid #2a2d3e; }
  td { padding: 12px; border-bottom: 1px solid #1e2030; font-size: 13px; }
  tr:hover { background: #22243a; }
  .badge { padding: 4px 10px; border-radius: 20px; font-size: 11px; font-weight: 600; }
  .badge-online { background: #064e3b; color: #34d399; }
  .badge-offline { background: #450a0a; color: #f87171; }
  .badge-scanning { background: #451a03; color: #fbbf24; }
  .badge-critical { background: #450a0a; color: #f87171; }
  .badge-warning { background: #451a03; color: #fbbf24; }
  .badge-info { background: #1e3a5f; color: #60a5fa; }
  .badge-pending { background: #1e2030; color: #8b8fa3; }
  .badge-monitoring { background: #1e3a5f; color: #60a5fa; }
  .badge-processing { background: #451a03; color: #fbbf24; }
  .badge-done { background: #064e3b; color: #34d399; }
  .badge-sent { background: #064e3b; color: #34d399; }
  .badge-notsent { background: #1e2030; color: #8b8fa3; }
  .btn { padding: 8px 16px; border-radius: 8px; border: none; cursor: pointer; font-size: 13px; font-weight: 600; transition: all 0.2s; }
  .btn-primary { background: #7c3aed; color: white; }
  .btn-primary:hover { background: #6d28d9; }
  .btn-success { background: #059669; color: white; }
  .btn-success:hover { background: #047857; }
  .btn-danger { background: #dc2626; color: white; }
  .btn-danger:hover { background: #b91c1c; }
  .btn-sm { padding: 5px 12px; font-size: 12px; }
  .form-group { margin-bottom: 16px; }
  .form-group label { display: block; margin-bottom: 6px; color: #8b8fa3; font-size: 13px; }
  .form-group input, .form-group select, .form-group textarea {
    width: 100%; padding: 10px 14px; background: #0f1117; border: 1px solid #2a2d3e;
    border-radius: 8px; color: #e1e4e8; font-size: 13px; outline: none;
  }
  .form-group input:focus, .form-group select:focus { border-color: #7c3aed; }
  .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
  .checkbox-row { display: flex; align-items: center; gap: 8px; padding: 8px 0; }
  .checkbox-row input[type="checkbox"] { width: 16px; height: 16px; accent-color: #7c3aed; }
  .storage-bar { height: 20px; background: #0f1117; border-radius: 10px; overflow: hidden; margin-top: 8px; }
  .storage-fill { height: 100%; background: linear-gradient(90deg, #34d399, #fbbf24, #f87171); transition: width 0.3s; border-radius: 10px; }
  .report-card { background: #0f1117; border: 1px solid #2a2d3e; border-radius: 8px; padding: 16px; margin-bottom: 12px; }
  .report-card .report-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
  .report-card .report-path { color: #60a5fa; font-size: 13px; font-family: monospace; }
  .report-card .report-summary { color: #e1e4e8; font-size: 14px; margin-bottom: 8px; }
  .report-card .report-details { color: #8b8fa3; font-size: 12px; }
  .empty-state { text-align: center; padding: 40px; color: #8b8fa3; }
  .flex { display: flex; gap: 12px; align-items: center; }
  .ml-auto { margin-left: auto; }
  .toast { position: fixed; bottom: 20px; right: 20px; background: #7c3aed; color: white; padding: 12px 20px; border-radius: 8px; display: none; z-index: 1000; font-size: 13px; }
</style>
</head>
<body>
<div class="header">
  <h1>🔍 Sera Log Analyzer</h1>
  <p>Master Dashboard — Manage agents, scan files, monitor AI analysis results</p>
</div>

<div class="container">
  <div class="stats" id="stats">
    <div class="stat-card"><div class="label">Agents Online</div><div class="value green" id="stat-agents">0</div></div>
    <div class="stat-card"><div class="label">Files Monitoring</div><div class="value blue" id="stat-files">0</div></div>
    <div class="stat-card"><div class="label">Reports Generated</div><div class="value yellow" id="stat-reports">0</div></div>
    <div class="stat-card"><div class="label">Storage Used</div><div class="value" id="stat-storage">0 MB</div></div>
  </div>

  <div class="tabs">
    <div class="tab active" onclick="showTab('agents')">Agents</div>
    <div class="tab" onclick="showTab('files')">Files</div>
    <div class="tab" onclick="showTab('reports')">Reports</div>
    <div class="tab" onclick="showTab('settings')">Settings</div>
  </div>

  <!-- Agents Panel -->
  <div class="panel active" id="panel-agents">
    <div class="flex" style="margin-bottom:16px">
      <h3 style="flex:1">Registered Agents</h3>
      <button class="btn btn-primary btn-sm" onclick="refreshAll()">↻ Refresh</button>
    </div>
    <table>
      <thead><tr><th>Name</th><th>ID</th><th>IP</th><th>Status</th><th>Last Heartbeat</th><th>Actions</th></tr></thead>
      <tbody id="agents-table"><tr><td colspan="6" class="empty-state">Loading...</td></tr></tbody>
    </table>
  </div>

  <!-- Files Panel -->
  <div class="panel" id="panel-files">
    <div class="flex" style="margin-bottom:16px">
      <h3 style="flex:1">Discovered Files</h3>
      <select id="file-agent-filter" style="padding:6px 12px;background:#0f1117;border:1px solid #2a2d3e;color:#e1e4e8;border-radius:6px;font-size:13px" onchange="loadFiles()">
        <option value="">All Agents</option>
      </select>
      <button class="btn btn-primary btn-sm" onclick="loadFiles()">↻ Refresh</button>
    </div>
    <div style="margin-bottom:16px" id="file-actions">
      <div class="flex">
        <label style="font-size:13px;color:#8b8fa3">Chunk Size (lines):</label>
        <input type="number" id="chunk-size" value="3" min="1" max="50" style="width:80px;padding:6px;background:#0f1117;border:1px solid #2a2d3e;color:#e1e4e8;border-radius:6px;font-size:13px">
        <button class="btn btn-success btn-sm" onclick="selectAllFiles()">Select All</button>
        <button class="btn btn-primary btn-sm" onclick="sendScanCommand()">🔍 Scan Files</button>
      </div>
    </div>
    <table>
      <thead><tr><th><input type="checkbox" id="select-all-cb" onchange="toggleAllCheckboxes(this)"></th><th>Path</th><th>Agent</th><th>Size</th><th>Status</th></tr></thead>
      <tbody id="files-table"><tr><td colspan="5" class="empty-state">No files found. Click "Scan Files" to start.</td></tr></tbody>
    </table>
  </div>

  <!-- Reports Panel -->
  <div class="panel" id="panel-reports">
    <div class="flex" style="margin-bottom:16px">
      <h3 style="flex:1">AI Analysis Reports</h3>
      <button class="btn btn-primary btn-sm" onclick="loadReports()">↻ Refresh</button>
    </div>
    <div id="reports-container"><div class="empty-state">No reports yet.</div></div>
  </div>

  <!-- Settings Panel -->
  <div class="panel" id="panel-settings">
    <h3 style="margin-bottom:20px">Configuration</h3>

    <div class="form-row">
      <div>
        <h4 style="margin-bottom:12px;color:#a78bfa">AI Configuration</h4>
        <div class="form-group">
          <label>Provider</label>
          <select id="cfg-ai-provider"><option>ollama</option><option>openai</option><option>openai-compatible</option></select>
        </div>
        <div class="form-group">
          <label>Base URL</label>
          <input type="text" id="cfg-ai-url" placeholder="http://ollama:11434">
        </div>
        <div class="form-group">
          <label>Model</label>
          <input type="text" id="cfg-ai-model" placeholder="qwen2.5:0.5b">
        </div>
        <div class="form-group">
          <label>API Key (for OpenAI)</label>
          <input type="password" id="cfg-ai-key" placeholder="sk-...">
        </div>
        <div class="form-group">
          <label>Chunk Size (lines per request)</label>
          <input type="number" id="cfg-chunk" value="3" min="1" max="50">
        </div>
        <button class="btn btn-primary" onclick="saveAIConfig()">Save AI Config</button>
      </div>

      <div>
        <h4 style="margin-bottom:12px;color:#a78bfa">Telegram Configuration</h4>
        <div class="form-group">
          <label>Bot Token</label>
          <input type="password" id="cfg-tg-token" placeholder="123456:ABC-DEF...">
        </div>
        <div class="form-group">
          <label>Chat ID</label>
          <input type="text" id="cfg-tg-chat" placeholder="-100123456789">
        </div>
        <div class="form-group">
          <label><input type="checkbox" id="cfg-tg-enabled"> Enable Telegram Notifications</label>
        </div>
        <button class="btn btn-primary" onclick="saveTGConfig()">Save Telegram Config</button>

        <h4 style="margin-top:30px;margin-bottom:12px;color:#a78bfa">Storage</h4>
        <div class="form-group">
          <label>Max Storage (MB)</label>
          <input type="number" id="cfg-max-storage" value="500" min="10" max="10000">
        </div>
        <div class="storage-bar"><div class="storage-fill" id="storage-fill" style="width:0%"></div></div>
        <p style="font-size:12px;color:#8b8fa3;margin-top:4px" id="storage-text">0 / 500 MB</p>
      </div>
    </div>
  </div>
</div>

<div class="toast" id="toast"></div>

<script>
const API = '/api';
const KEY = new URLSearchParams(window.location.search).get('key') || 'sera-default-key';
const headers = {'Content-Type':'application/json','X-API-Key':KEY};
let agents = [];
let files = [];

function toast(msg) {
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.style.display = 'block';
  setTimeout(() => t.style.display = 'none', 3000);
}

function showTab(name) {
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
  event.target.classList.add('active');
  document.getElementById('panel-'+name).classList.add('active');
}

async function api(path, method='GET', body=null) {
  const opts = {method, headers};
  if (body) opts.body = JSON.stringify(body);
  const resp = await fetch(API+path, opts);
  return resp.json();
}

async function refreshAll() {
  await Promise.all([loadAgents(), loadFiles(), loadReports(), loadStorage()]);
}

async function loadAgents() {
  const res = await api('/agents');
  agents = res.data || [];
  document.getElementById('stat-agents').textContent = agents.filter(a => a.status==='online').length;
  const tbody = document.getElementById('agents-table');
  const filter = document.getElementById('file-agent-filter');
  filter.innerHTML = '<option value="">All Agents</option>';
  if (!agents.length) { tbody.innerHTML = '<tr><td colspan="6" class="empty-state">No agents registered yet</td></tr>'; return; }
  let html = '';
  agents.forEach(a => {
    const badge = a.status==='online' ? 'badge-online' : a.status==='scanning' ? 'badge-scanning' : 'badge-offline';
    const hb = new Date(a.last_heartbeat).toLocaleString();
    html += '<tr><td>'+a.name+'</td><td style="font-family:monospace;font-size:11px">'+a.id.slice(0,8)+'...</td><td>'+a.ip+'</td><td><span class="badge '+badge+'">'+a.status+'</span></td><td>'+hb+'</td><td><button class="btn btn-primary btn-sm" onclick="sendScanToAgent(\''+a.id+'\')">Scan</button></td></tr>';
    filter.innerHTML += '<option value="'+a.id+'">'+a.name+'</option>';
  });
  tbody.innerHTML = html;
}

async function loadFiles() {
  const agentFilter = document.getElementById('file-agent-filter').value;
  const q = agentFilter ? '?agent_id='+agentFilter : '';
  const res = await api('/files'+q);
  files = res.data || [];
  document.getElementById('stat-files').textContent = files.filter(f => f.status==='monitoring'||f.status==='processing').length;
  const tbody = document.getElementById('files-table');
  if (!files.length) { tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No files found. Click "Scan Files" to start.</td></tr>'; return; }
  let html = '';
  files.forEach(f => {
    const badge = 'badge-'+f.status;
    const sizeKB = (f.size/1024).toFixed(1);
    const agent = agents.find(a => a.id===f.agent_id);
    html += '<tr><td><input type="checkbox" class="file-cb" value="'+f.id+'" data-agent="'+f.agent_id+'"></td><td style="font-family:monospace;font-size:12px">'+f.path+'</td><td>'+(agent?agent.name:'?')+'</td><td>'+sizeKB+' KB</td><td><span class="badge '+badge+'">'+f.status+'</span></td></tr>';
  });
  tbody.innerHTML = html;
}

async function loadReports() {
  const res = await api('/reports');
  const reports = res.data || [];
  document.getElementById('stat-reports').textContent = reports.length;
  const container = document.getElementById('reports-container');
  if (!reports.length) { container.innerHTML = '<div class="empty-state">No reports yet.</div>'; return; }
  let html = '';
  reports.forEach(r => {
    const badge = 'badge-'+r.severity;
    const sentBadge = r.sent_to_tg ? 'badge-sent' : 'badge-notsent';
    const ts = new Date(r.created_at).toLocaleString();
    html += '<div class="report-card"><div class="report-header"><span class="badge '+badge+'">'+r.severity.toUpperCase()+'</span><span style="font-size:12px;color:#8b8fa3">'+ts+'</span></div><div class="report-path">'+r.file_path+' (chunk #'+r.chunk_num+')</div><div class="report-summary">'+r.summary+'</div>'+(r.details?'<div class="report-details">'+r.details+'</div>':'')+'<div style="margin-top:8px"><span class="badge '+sentBadge+'" style="font-size:10px">'+(r.sent_to_tg?'TG Sent':'No TG')+'</span></div></div>';
  });
  container.innerHTML = html;
}

async function loadStorage() {
  const res = await api('/storage');
  const s = res.data || {};
  document.getElementById('stat-storage').textContent = (s.used_mb||0).toFixed(1)+' MB';
  document.getElementById('storage-fill').style.width = Math.min(s.percentage||0, 100)+'%';
  document.getElementById('storage-text').textContent = (s.used_mb||0).toFixed(1)+' / '+(s.max_mb||500)+' MB ('+(s.percentage||0).toFixed(1)+'%)'+(s.is_full?' ⚠️ FULL':'');
  document.getElementById('cfg-max-storage').value = s.max_mb || 500;
}

async function sendScanToAgent(agentID) {
  const ext = prompt('Enter file extensions (comma-separated, e.g. .log,.log.2,.err):', '.log');
  if (!ext) return;
  const roots = prompt('Enter root paths (comma-separated, e.g. /var/log,/tmp):', '/var/log');
  if (!roots) return;
  const payload = JSON.stringify({extensions:ext.split(',').map(s=>s.trim()), root_paths:roots.split(',').map(s=>s.trim()), max_depth:5});
  const res = await api('/command','POST',{agent_id:agentID,type:'scan_files',payload});
  if (res.success) toast('Scan command sent!');
  else toast('Error: '+res.error);
  setTimeout(loadFiles, 2000);
}

function sendScanCommand() {
  if (!agents.length) { toast('No agents registered!'); return; }
  agents.forEach(a => sendScanToAgent(a.id));
}

function toggleAllCheckboxes(cb) {
  document.querySelectorAll('.file-cb').forEach(c => c.checked = cb.checked);
}

async function selectAllFiles() {
  const agentFilter = document.getElementById('file-agent-filter').value;
  if (!agentFilter) { toast('Select an agent first'); return; }
  const chunkSize = parseInt(document.getElementById('chunk-size').value) || 3;
  const res = await api('/files/select','POST',{select_all:true, agent_id:agentFilter, chunk_size:chunkSize});
  if (res.success) { toast('All files selected for monitoring!'); loadFiles(); }
  else toast('Error: '+res.error);
}

async function saveAIConfig() {
  const res = await api('/config/ai','PUT',{
    provider: document.getElementById('cfg-ai-provider').value,
    base_url: document.getElementById('cfg-ai-url').value,
    model: document.getElementById('cfg-ai-model').value,
    api_key: document.getElementById('cfg-ai-key').value,
    max_tokens: 512,
    temperature: 0.3,
    chunk_size: parseInt(document.getElementById('cfg-chunk').value)||3,
  });
  if (res.success) toast('AI config saved!');
}

async function saveTGConfig() {
  const res = await api('/config/telegram','PUT',{
    bot_token: document.getElementById('cfg-tg-token').value,
    chat_id: document.getElementById('cfg-tg-chat').value,
    enabled: document.getElementById('cfg-tg-enabled').checked,
  });
  if (res.success) toast('Telegram config saved!');
}

// Auto-load
async function init() {
  const res = await api('/config/ai');
  if (res.data) {
    document.getElementById('cfg-ai-provider').value = res.data.provider||'ollama';
    document.getElementById('cfg-ai-url').value = res.data.base_url||'';
    document.getElementById('cfg-ai-model').value = res.data.model||'';
    document.getElementById('cfg-ai-key').value = res.data.api_key||'';
    document.getElementById('cfg-chunk').value = res.data.chunk_size||3;
  }
  const tg = await api('/config/telegram');
  if (tg.data) {
    document.getElementById('cfg-tg-token').value = tg.data.bot_token||'';
    document.getElementById('cfg-tg-chat').value = tg.data.chat_id||'';
    document.getElementById('cfg-tg-enabled').checked = tg.data.enabled||false;
  }
  refreshAll();
}

init();
setInterval(refreshAll, 10000);
</script>
</body>
</html>`
