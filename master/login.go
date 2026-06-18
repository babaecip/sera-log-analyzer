package main

var loginPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Sera Log Analyzer - Login</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: 'Segoe UI', system-ui, sans-serif; background: #0f1117; color: #e1e4e8; min-height: 100vh; display: flex; align-items: center; justify-content: center; }
  .login-container { width: 100%; max-width: 420px; padding: 20px; }
  .login-card { background: #1a1c2e; border: 1px solid #2a2d3e; border-radius: 16px; padding: 40px 32px; }
  .logo { text-align: center; margin-bottom: 32px; }
  .logo h1 { font-size: 28px; color: #a78bfa; margin-bottom: 4px; }
  .logo p { color: #8b8fa3; font-size: 13px; }
  .form-group { margin-bottom: 20px; }
  .form-group label { display: block; margin-bottom: 6px; color: #8b8fa3; font-size: 13px; font-weight: 500; }
  .form-group input { width: 100%; padding: 12px 14px; background: #0f1117; border: 1px solid #2a2d3e; border-radius: 8px; color: #e1e4e8; font-size: 14px; outline: none; transition: border-color 0.2s; }
  .form-group input:focus { border-color: #7c3aed; }
  .captcha-box { background: #0f1117; border: 1px solid #2a2d3e; border-radius: 8px; padding: 16px; margin-bottom: 20px; display: flex; align-items: center; gap: 10px; flex-wrap: nowrap; overflow: hidden; }
  .captcha-question { font-size: 18px; font-weight: 700; color: #a78bfa; font-family: monospace; white-space: nowrap; flex-shrink: 0; text-align: center; background: #1a1c2e; padding: 8px 14px; border-radius: 8px; }
  .captcha-input { flex: 1 1 0; min-width: 0; padding: 10px 10px; background: #1a1c2e; border: 1px solid #2a2d3e; border-radius: 8px; color: #e1e4e8; font-size: 18px; text-align: center; outline: none; }
  .captcha-input:focus { border-color: #7c3aed; }
  .btn-login { width: 100%; padding: 14px; background: #7c3aed; color: white; border: none; border-radius: 8px; font-size: 15px; font-weight: 600; cursor: pointer; transition: all 0.2s; margin-top: 8px; }
  .btn-login:hover { background: #6d28d9; }
  .btn-login:disabled { background: #4a4a5a; cursor: not-allowed; }
  .error-msg { background: #450a0a; color: #f87171; padding: 10px 14px; border-radius: 8px; font-size: 13px; margin-bottom: 16px; display: none; }
  .error-msg.show { display: block; }
  .refresh-captcha { background: none; border: none; color: #8b8fa3; cursor: pointer; font-size: 18px; padding: 4px; transition: color 0.2s; }
  .refresh-captcha:hover { color: #a78bfa; }
  .loading { display: inline-block; width: 16px; height: 16px; border: 2px solid rgba(255,255,255,0.3); border-radius: 50%; border-top-color: white; animation: spin 0.6s linear infinite; vertical-align: middle; margin-right: 8px; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
</head>
<body>
<div class="login-container">
  <div class="login-card">
    <div class="logo">
      <h1>🔍 Sera Log Analyzer</h1>
      <p>Admin Dashboard Login</p>
    </div>

    <div class="error-msg" id="error-msg"></div>

    <form id="login-form" onsubmit="return handleLogin(event)">
      <div class="form-group">
        <label>Username</label>
        <input type="text" id="username" autocomplete="username" required>
      </div>

      <div class="form-group">
        <label>Password</label>
        <input type="password" id="password" autocomplete="current-password" required>
      </div>

      <div class="captcha-box">
        <div class="captcha-question" id="captcha-question">Loading...</div>
        <input type="number" class="captcha-input" id="captcha-answer" placeholder="?" required>
        <button type="button" class="refresh-captcha" onclick="loadCaptcha()" title="New captcha">🔄</button>
      </div>

      <button type="submit" class="btn-login" id="btn-login">Login</button>
    </form>
  </div>
</div>

<script>
let captchaId = '';

async function loadCaptcha() {
  try {
    const resp = await fetch('/api/captcha');
    const data = await resp.json();
    if (data.success) {
      captchaId = data.data.captcha_id;
      document.getElementById('captcha-question').textContent = data.data.question;
      document.getElementById('captcha-answer').value = '';
      document.getElementById('captcha-answer').focus();
    }
  } catch(e) {
    document.getElementById('captcha-question').textContent = 'Error';
  }
}

async function handleLogin(e) {
  e.preventDefault();
  const btn = document.getElementById('btn-login');
  const errDiv = document.getElementById('error-msg');

  btn.disabled = true;
  btn.innerHTML = '<span class="loading"></span>Logging in...';
  errDiv.classList.remove('show');

  try {
    const resp = await fetch('/api/login', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        username: document.getElementById('username').value,
        password: document.getElementById('password').value,
        captcha_id: captchaId,
        captcha_answer: parseInt(document.getElementById('captcha-answer').value) || 0
      })
    });

    const data = await resp.json();

    if (data.success) {
      window.location.href = '/';
    } else {
      errDiv.textContent = data.error || 'Login failed';
      errDiv.classList.add('show');
      loadCaptcha();
    }
  } catch(e) {
    errDiv.textContent = 'Connection error. Please try again.';
    errDiv.classList.add('show');
    loadCaptcha();
  }

  btn.disabled = false;
  btn.textContent = 'Login';
  return false;
}

// Load captcha on page load
loadCaptcha();

// Enter key on captcha submits form
document.getElementById('captcha-answer').addEventListener('keydown', function(e) {
  if (e.key === 'Enter') {
    document.getElementById('login-form').requestSubmit();
  }
});
</script>
</body>
</html>`
