document.addEventListener('DOMContentLoaded', () => {
  loadSettings();
  
  document.getElementById('save-token-btn').addEventListener('click', saveToken);
  document.getElementById('test-connection-btn').addEventListener('click', testConnection);
  document.getElementById('save-config-btn').addEventListener('click', saveConfig);
});

async function loadSettings() {
  const settings = await chrome.storage.local.get(['serverUrl', 'refreshToken']);
  
  if (settings.serverUrl) {
    document.getElementById('server-url').value = settings.serverUrl;
  }
  if (settings.refreshToken) {
    document.getElementById('refresh-token').value = settings.refreshToken;
  }
  
  await loadConfig();
}

async function loadConfig() {
  const serverUrl = document.getElementById('server-url').value;
  
  try {
    const response = await fetch(`${serverUrl}/api/config`);
    const result = await response.json();
    
    if (result.state && result.data) {
      document.getElementById('download-dir').value = result.data.download_dir || '';
      document.getElementById('monitor-interval').value = result.data.monitor_interval || 30;
    }
  } catch (error) {}
}

async function saveToken() {
  const token = document.getElementById('refresh-token').value.trim();
  const serverUrl = document.getElementById('server-url').value;
  const statusEl = document.getElementById('token-status');
  
  if (!token) {
    showStatus(statusEl, 'Please enter a refresh token', 'error');
    return;
  }
  
  try {
    const response = await fetch(`${serverUrl}/api/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: token })
    });
    
    const result = await response.json();
    
    if (result.state) {
      await chrome.storage.local.set({ refreshToken: token });
      showStatus(statusEl, 'Token saved successfully!', 'success');
    } else {
      showStatus(statusEl, result.message || 'Failed to save token', 'error');
    }
  } catch (error) {
    showStatus(statusEl, 'Cannot connect to server', 'error');
  }
}

async function testConnection() {
  const serverUrl = document.getElementById('server-url').value;
  const statusEl = document.getElementById('connection-status');
  
  try {
    const response = await fetch(`${serverUrl}/health`);
    const result = await response.json();
    
    if (result.status === 'ok') {
      await chrome.storage.local.set({ serverUrl });
      showStatus(statusEl, 'Connection successful!', 'success');
    } else {
      showStatus(statusEl, 'Server responded with error', 'error');
    }
  } catch (error) {
    showStatus(statusEl, 'Cannot connect to server. Is it running?', 'error');
  }
}

async function saveConfig() {
  const serverUrl = document.getElementById('server-url').value;
  const downloadDir = document.getElementById('download-dir').value.trim();
  const monitorInterval = parseInt(document.getElementById('monitor-interval').value);
  const statusEl = document.getElementById('config-status');
  
  if (monitorInterval < 10 || monitorInterval > 300) {
    showStatus(statusEl, 'Monitor interval must be between 10 and 300 seconds', 'error');
    return;
  }
  
  try {
    const config = {};
    if (downloadDir) config.download_dir = downloadDir;
    config.monitor_interval = monitorInterval;
    
    const response = await fetch(`${serverUrl}/api/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config)
    });
    
    const result = await response.json();
    
    if (result.state) {
      showStatus(statusEl, 'Configuration saved!', 'success');
    } else {
      showStatus(statusEl, result.message || 'Failed to save config', 'error');
    }
  } catch (error) {
    showStatus(statusEl, 'Cannot connect to server', 'error');
  }
}

function showStatus(element, message, type) {
  element.textContent = message;
  element.className = `status ${type}`;
  element.style.display = 'block';
  
  setTimeout(() => {
    element.style.display = 'none';
  }, 5000);
}
