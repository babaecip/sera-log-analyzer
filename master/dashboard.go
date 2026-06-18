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
  .panel { display: none; background: #1a1c2e; border: 1px solid #2a2d3e; border-radius: 12px; padding: 20px; overflow-x: auto; }
  .panel.active { display: block; }
  table { width: 100%; border-collapse: collapse; table-layout: fixed; }
  table th, table td { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .col-path { width: 35%; max-width: 400px; }
  .col-id { width: 12%; }
  .col-agent { width: 12%; }
  .col-size { width: 8%; }
  .col-progress { width: 18%; }
  .col-status { width: 15%; }
  .col-checkbox { width: 40px; }
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
  .step-indicator { display:flex;align-items:center;gap:6px;padding:8px 16px;border-radius:20px;font-size:12px;font-weight:600;color:#8b8fa3;background:#1e2030;cursor:default;white-space:nowrap; }
  .step-indicator.active { background:#7c3aed;color:white; }
  .step-indicator.done { background:#064e3b;color:#34d399; }
  .step-num { display:inline-flex;align-items:center;justify-content:center;width:20px;height:20px;border-radius:50%;background:rgba(255,255,255,0.15);font-size:11px; }
  .step-line { flex:1;min-width:20px;height:2px;background:#2a2d3e; }
  .progress-bar { width:100%;height:6px;background:#1e2030;border-radius:3px;overflow:hidden;min-width:80px; }
  .progress-fill { height:100%;background:linear-gradient(90deg,#7c3aed,#a78bfa);border-radius:3px;transition:width 0.3s; }
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
  .report-card { background: #0f1117; border: 1px solid #2a2d3e; border-radius: 8px; padding: 16px; margin-bottom: 12px; overflow: hidden; }
  .report-card .report-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
  .report-card .report-path { color: #60a5fa; font-size: 13px; font-family: monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 100%; display: block; }
  .report-card .report-summary { color: #e1e4e8; font-size: 14px; margin-bottom: 8px; }
  .report-card .report-details { color: #8b8fa3; font-size: 12px; }
  .empty-state { text-align: center; padding: 40px; color: #8b8fa3; }
  .flex { display: flex; gap: 12px; align-items: center; }
  .ml-auto { margin-left: auto; }
  .toast { position: fixed; bottom: 20px; right: 20px; background: #7c3aed; color: white; padding: 12px 20px; border-radius: 8px; display: none; z-index: 1000; font-size: 13px; }
  .ai-log-card { background: #0f1117; border: 1px solid #2a2d3e; border-radius: 8px; padding: 14px; margin-bottom: 10px; }
  .ai-log-card.success { border-left: 3px solid #34d399; }
  .ai-log-card.error { border-left: 3px solid #f87171; }
  .ai-log-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; flex-wrap: wrap; gap: 6px; }
  .ai-log-meta { color: #8b8fa3; font-size: 12px; }
  .ai-log-url { color: #60a5fa; font-size: 12px; font-family: monospace; word-break: break-all; }
  .ai-log-body { margin-top: 8px; }
  .ai-log-toggle { cursor: pointer; color: #a78bfa; font-size: 12px; font-weight: 600; user-select: none; }
  .ai-log-toggle:hover { color: #c4b5fd; }
  .ai-log-pre { background: #1a1c2e; border: 1px solid #2a2d3e; border-radius: 6px; padding: 10px; font-size: 11px; font-family: monospace; color: #e1e4e8; white-space: pre-wrap; word-break: break-all; max-height: 300px; overflow-y: auto; margin-top: 6px; display: none; }
  .btn-delete { background: #450a0a; color: #f87171; border: 1px solid #7f1d1d; }
  .btn-delete:hover { background: #7f1d1d; color: #fca5a5; }
</style>
</head>
<body>
<div class="header" style="display:flex;align-items:center;justify-content:space-between">
  <div><h1>🔍 Sera Log Analyzer</h1><p>Master Dashboard — Manage agents, scan files, monitor AI analysis results</p></div>
  <div style="display:flex;align-items:center;gap:12px"><span id="user-info" style="color:#8b8fa3;font-size:13px"></span><button class="btn btn-sm" style="background:#dc2626;color:white" onclick="logout()">Logout</button></div>
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
    <div class="tab" onclick="showTab('ai-monitor')">🤖 AI Monitor</div>
    <div class="tab" onclick="showTab('settings')">Settings</div>
  </div>

  <!-- Agents Panel -->
  <div class="panel active" id="panel-agents">
    <div class="flex" style="margin-bottom:16px">
      <h3 style="flex:1">Registered Agents</h3>
      <button class="btn btn-primary btn-sm" onclick="refreshAll()">↻ Refresh</button>
    </div>
    <table>
      <thead><tr><th style="width:15%">Name</th><th class="col-id">ID</th><th style="width:10%">IP</th><th class="col-status">Status</th><th style="width:18%">Last Heartbeat</th><th style="width:18%">Actions</th></tr></thead>
      <tbody id="agents-table"><tr><td colspan="6" class="empty-state">Loading...</td></tr></tbody>
    </table>
  </div>

  <!-- Files Panel -->
  <div class="panel" id="panel-files">
    <!-- Step Progress -->
    <div style="display:flex;align-items:center;margin-bottom:24px;gap:0">
      <div class="step-indicator active" id="step-1"><span class="step-num">1</span> Scan</div>
      <div class="step-line"></div>
      <div class="step-indicator" id="step-2"><span class="step-num">2</span> Select</div>
      <div class="step-line"></div>
      <div class="step-indicator" id="step-3"><span class="step-num">3</span> Monitor</div>
      <div class="step-line"></div>
      <div class="step-indicator" id="step-4"><span class="step-num">4</span> Done</div>
    </div>

    <!-- Actions Bar -->
    <div class="flex" style="margin-bottom:16px;flex-wrap:wrap">
      <select id="file-agent-filter" style="padding:6px 12px;background:#0f1117;border:1px solid #2a2d3e;color:#e1e4e8;border-radius:6px;font-size:13px" onchange="loadFiles()">
        <option value="">All Agents</option>
      </select>
      <button class="btn btn-primary btn-sm" onclick="sendScanCommand()">🔍 1. Scan Files</button>
      <div id="select-controls" style="display:none;align-items:center;gap:8px">
        <label style="font-size:13px;color:#8b8fa3">Chunk:</label>
        <input type="number" id="chunk-size" value="3" min="1" max="50" style="width:60px;padding:4px 8px;background:#0f1117;border:1px solid #2a2d3e;color:#e1e4e8;border-radius:6px;font-size:13px">
        <button class="btn btn-success btn-sm" onclick="selectAllFiles()">✅ 2. Select All</button>
        <button class="btn btn-primary btn-sm" onclick="startMonitoring()">▶ 3. Start Monitoring</button>
      </div>
      <button class="btn btn-sm" style="background:#374151;color:#e1e4e8;margin-left:auto" onclick="loadFiles()">↻ Refresh</button>
    </div>

    <!-- File Table -->
    <table>
      <thead><tr>
        <th class="col-checkbox"><input type="checkbox" id="select-all-cb" onchange="toggleAllCheckboxes(this)"></th>
        <th class="col-path">Path</th><th class="col-agent">Agent</th><th class="col-size">Size</th><th class="col-progress">Progress</th><th class="col-status">Status</th><th style="width:50px"></th>
      </tr></thead>
      <tbody id="files-table"><tr><td colspan="6" class="empty-state">Click <b>🔍 Scan Files</b> to discover log files</td></tr></tbody>
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

  <!-- AI Monitor Panel -->
  <div class="panel" id="panel-ai-monitor">
    <div class="flex" style="margin-bottom:16px">
      <h3 style="flex:1">🤖 AI Request/Response Monitor</h3>
      <button class="btn btn-sm" style="background:#374151;color:#e1e4e8" onclick="clearAILogs()">🗑 Clear Logs</button>
      <button class="btn btn-primary btn-sm" onclick="loadAILogs()">↻ Refresh</button>
    </div>
    <div id="ai-logs-container"><div class="empty-state">No AI requests yet.</div></div>
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
const headers = {'Content-Type':'application/json'};
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
  const opts = {method, headers, credentials:'same-origin'};
  if (body) opts.body = JSON.stringify(body);
  const resp = await fetch(API+path, opts);
  if (resp.status === 401 || resp.redirected) { window.location.href = '/login'; return {}; }
  return resp.json();
}

async function logout() {
  await fetch('/api/logout', {method:'POST', credentials:'same-origin'});
  window.location.href = '/login';
}

async function refreshAll() {
  await Promise.all([loadAgents(), loadFiles(), loadReports(), loadStorage(), loadAILogs()]);
}

async function loadAgents() {
  const res = await api('/agents');
  agents = res.data || [];
  document.getElementById('stat-agents').textContent = agents.filter(a => a.status==='online').length;
  const tbody = document.getElementById('agents-table');
  const filter = document.getElementById('file-agent-filter');
  const prevVal = filter.value;
  filter.innerHTML = '<option value="">All Agents</option>';
  if (!agents.length) { tbody.innerHTML = '<tr><td colspan="7" class="empty-state">No agents registered yet</td></tr>'; return; }
  let html = '';
  agents.forEach(a => {
    const badge = a.status==='online' ? 'badge-online' : a.status==='scanning' ? 'badge-scanning' : 'badge-offline';
    const hb = new Date(a.last_heartbeat).toLocaleString();
    let actions = '<button class="btn btn-primary btn-sm" onclick="sendScanToAgent(\''+a.id+'\')">🔍 Scan</button>';
    if (a.status === 'offline') {
      actions += ' <button class="btn btn-delete btn-sm" onclick="deleteAgent(\''+a.id+'\',\''+a.name+'\')">🗑 Delete</button>';
    }
    html += '<tr><td>'+a.name+'</td><td style="font-family:monospace;font-size:11px" title="'+a.id+'">'+a.id.slice(0,8)+'...</td><td>'+a.ip+'</td><td><span class="badge '+badge+'">'+a.status+'</span></td><td>'+hb+'</td><td>'+actions+'</td></tr>';
    filter.innerHTML += '<option value="'+a.id+'">'+a.name+'</option>';
  });
  filter.value = prevVal;
  tbody.innerHTML = html;
}

async function loadFiles() {
  const agentFilter = document.getElementById('file-agent-filter').value;
  const q = agentFilter ? '?agent_id='+agentFilter : '';
  const res = await api('/files'+q);
  files = res.data || [];
  document.getElementById('stat-files').textContent = files.filter(f => f.status==='monitoring'||f.status==='processing').length;
  updateStepIndicator();
  renderFileTable();
}

function updateStepIndicator() {
  const pending = files.filter(f => f.status==='pending').length;
  const monitoring = files.filter(f => f.status==='monitoring'||f.status==='processing').length;
  const done = files.filter(f => f.status==='done').length;
  const hasAny = pending > 0 || monitoring > 0 || done > 0;
  const s1 = document.getElementById('step-1');
  const s2 = document.getElementById('step-2');
  const s3 = document.getElementById('step-3');
  const s4 = document.getElementById('step-4');
  const selectCtrl = document.getElementById('select-controls');

  s1.className = 'step-indicator' + (hasAny ? ' done' : ' active');
  s2.className = 'step-indicator' + (pending > 0 && monitoring === 0 ? ' active' : (pending === 0 && (monitoring > 0 || done > 0) ? ' done' : ''));
  s3.className = 'step-indicator' + (monitoring > 0 ? ' active' : (monitoring === 0 && done > 0 ? ' done' : ''));
  s4.className = 'step-indicator' + (done > 0 && monitoring === 0 ? ' done' : (done > 0 ? ' active' : ''));
  selectCtrl.style.display = (pending > 0 || files.length > 0) ? 'flex' : 'none';
}

function renderFileTable() {
  const tbody = document.getElementById('files-table');
  if (!files.length) {
    tbody.innerHTML = '<tr><td colspan="7" class="empty-state">Click <b>🔍 Scan Files</b> to discover log files</td></tr>';
    return;
  }
  let html = '';
  files.forEach(f => {
    const sizeKB = (f.size/1024).toFixed(1);
    const agent = agents.find(a => a.id===f.agent_id);
    const isPending = f.status === 'pending';
    const isMonitoring = f.status === 'monitoring' || f.status === 'processing';
    const isDone = f.status === 'done';
    let statusBadge = '';
    switch(f.status) {
      case 'pending': statusBadge = '<span class="badge badge-pending">⏳ Pending</span>'; break;
      case 'monitoring': statusBadge = '<span class="badge badge-monitoring">👁 Monitoring</span>'; break;
      case 'processing': statusBadge = '<span class="badge badge-processing">⚙ Processing</span>'; break;
      case 'done': statusBadge = '<span class="badge badge-done">✅ Done</span>'; break;
      default: statusBadge = '<span class="badge badge-pending">'+f.status+'</span>';
    }
    let progressHtml = '<div style="color:#8b8fa3;font-size:12px">—</div>';
    if (isMonitoring || isDone) {
      const reports = f.total_reports || 0;
      const actions = f.action_reports || 0;
      const pct = isDone ? 100 : Math.min(90, reports * 5);
      progressHtml = '<div class="progress-bar"><div class="progress-fill" style="width:'+pct+'%"></div></div>'
        + '<div style="font-size:11px;color:#8b8fa3;margin-top:2px">'+reports+' chunks analyzed'
        + (actions > 0 ? ' · <span style="color:#fbbf24">'+actions+' action needed</span>' : '')
        + '</div>';
    }
    const cbHtml = isPending
      ? '<td><input type="checkbox" class="file-cb" value="'+f.id+'" data-agent="'+f.agent_id+'"></td>'
      : '<td><input type="checkbox" disabled checked></td>';
    const deleteBtn = '<td><button class="btn btn-delete btn-sm" onclick="deleteFile(\''+f.id+'\')" title="Delete">🗑</button></td>';
    html += '<tr'+(isDone?' style="opacity:0.6"':'')+'>'+cbHtml
      +'<td class="col-path" style="font-family:monospace;font-size:12px" title="'+f.path+'">'+f.path+'</td>'
      +'<td>'+(agent?agent.name:'?')+'</td>'
      +'<td>'+sizeKB+' KB</td>'
      +'<td>'+progressHtml+'</td>'
      +'<td>'+statusBadge+'</td>'
      +deleteBtn+'</tr>';
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
    html += '<div class="report-card"><div class="report-header"><span class="badge '+badge+'">'+r.severity.toUpperCase()+'</span><span style="font-size:12px;color:#8b8fa3">'+ts+'</span></div><div class="report-path" title="'+r.file_path+'">'+r.file_path+' (chunk #'+r.chunk_num+')</div><div class="report-summary">'+r.summary+'</div>'+(r.details?'<div class="report-details">'+r.details+'</div>':'')+'<div style="margin-top:8px"><span class="badge '+sentBadge+'" style="font-size:10px">'+(r.sent_to_tg?'TG Sent':'No TG')+'</span></div></div>';
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

function sendScanToAgent(agentID) {
  const ext = prompt('File extensions (comma-separated):', '.log');
  if (!ext) return;
  const roots = prompt('Root paths (comma-separated):', '/var/log');
  if (!roots) return;
  const payload = JSON.stringify({extensions:ext.split(',').map(s=>s.trim()), root_paths:roots.split(',').map(s=>s.trim()), max_depth:5});
  api('/command','POST',{agent_id:agentID,type:'scan_files',payload}).then(res => {
    if (res.success) toast('Scan command sent! Waiting for results...');
    else toast('Error: '+res.error);
    setTimeout(loadFiles, 3000);
  });
}

async function deleteAgent(agentID, agentName) {
  if (!confirm('Delete agent "'+agentName+'" and ALL its data (files, reports, commands)?')) return;
  const res = await api('/agents/'+agentID, 'DELETE');
  if (res.success) { toast('Agent deleted!'); loadAgents(); loadFiles(); }
  else toast('Error: '+res.error);
}

async function deleteFile(fileID) {
  if (!confirm('Delete this file from the list?')) return;
  const res = await api('/files/'+fileID, 'DELETE');
  if (res.success) { toast('File deleted!'); loadFiles(); }
  else toast('Error: '+res.error);
}

async function loadAILogs() {
  const res = await api('/ai-logs');
  const logs = res.data || [];
  const container = document.getElementById('ai-logs-container');
  if (!logs.length) { container.innerHTML = '<div class="empty-state">No AI requests yet.</div>'; return; }
  let html = '';
  logs.forEach((l, i) => {
    const cls = l.success ? 'success' : 'error';
    const statusBadge = l.success
      ? '<span class="badge badge-done">✓ OK</span>'
      : '<span class="badge badge-critical">✗ Error</span>';
    const ts = new Date(l.created_at).toLocaleString();
    let reqPreview = '';
    try {
      const r = JSON.parse(l.request);
      if (r.messages && r.messages.length) {
        const userMsg = r.messages.find(m => m.role==='user');
        reqPreview = userMsg ? userMsg.content.substring(0,150)+'...' : l.request.substring(0,150)+'...';
      } else { reqPreview = l.request.substring(0,150)+'...'; }
    } catch(e) { reqPreview = l.request.substring(0,150)+'...'; }

    let respPreview = '';
    try {
      const r = JSON.parse(l.response);
      if (r.message && r.message.content) respPreview = r.message.content.substring(0,150)+'...';
      else if (r.choices && r.choices[0]) respPreview = r.choices[0].message.content.substring(0,150)+'...';
      else respPreview = l.response.substring(0,150)+'...';
    } catch(e) { respPreview = l.response.substring(0,150)+'...'; }

    html += '<div class="ai-log-card '+cls+'">'
      +'<div class="ai-log-header">'
        +'<div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap">'
          +statusBadge
          +'<span class="badge badge-info">'+l.provider+'</span>'
          +'<span style="color:#e1e4e8;font-size:13px;font-weight:600">'+l.model+'</span>'
          +'<span class="ai-log-meta">'+l.duration_ms+'ms</span>'
        +'</div>'
        +'<span class="ai-log-meta">'+ts+'</span>'
      +'</div>'
      +'<div class="ai-log-url">'+l.url+'</div>'
      +(l.error ? '<div style="color:#f87171;font-size:12px;margin-top:4px">Error: '+l.error+'</div>' : '')
      +'<div class="ai-log-body">'
        +'<div class="ai-log-toggle" onclick="toggleAILog('+i+')">▸ Request</div>'
        +'<pre class="ai-log-pre" id="ai-req-'+i+'">'+formatJSON(l.request)+'</pre>'
        +'<div class="ai-log-toggle" onclick="toggleAILogResp('+i+')">▸ Response</div>'
        +'<pre class="ai-log-pre" id="ai-resp-'+i+'">'+formatJSON(l.response)+'</pre>'
      +'</div>'
    +'</div>';
  });
  container.innerHTML = html;
}

function toggleAILog(i) {
  const el = document.getElementById('ai-req-'+i);
  el.style.display = el.style.display === 'none' ? 'block' : 'none';
}
function toggleAILogResp(i) {
  const el = document.getElementById('ai-resp-'+i);
  el.style.display = el.style.display === 'none' ? 'block' : 'none';
}

function formatJSON(str) {
  try { return JSON.stringify(JSON.parse(str), null, 2); } catch(e) { return str; }
}

async function clearAILogs() {
  if (!confirm('Clear all AI request logs?')) return;
  const res = await api('/ai-logs', 'DELETE');
  if (res.success) { toast('AI logs cleared!'); loadAILogs(); }
  else toast('Error: '+res.error);
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
  if (res.success) { toast('Files selected! Click ▶ Start Monitoring'); loadFiles(); }
  else toast('Error: '+res.error);
}

async function startMonitoring() {
  const agentFilter = document.getElementById('file-agent-filter').value;
  if (!agentFilter) { toast('Select an agent first'); return; }
  const chunkSize = parseInt(document.getElementById('chunk-size').value) || 3;
  await api('/files/select','POST',{select_all:true, agent_id:agentFilter, chunk_size:chunkSize});
  toast('Monitoring started!');
  loadFiles();
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

async function init() {
  const sess = await api('/session');
  if (!sess.success || !sess.data || !sess.data.authenticated) {
    window.location.href = '/login';
    return;
  }
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
  document.getElementById('user-info').textContent = sess.data.username;
  refreshAll();
}

init();
setInterval(refreshAll, 10000);
</script>
</body>
</html>`
