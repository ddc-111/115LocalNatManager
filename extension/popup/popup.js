const API_BASE = 'http://localhost:11580/api';

document.addEventListener('DOMContentLoaded', () => {
  initTabs();
  checkStatus();
  loadMagnets();
  loadTasks();
  loadFiles();
  initMonitorToggle();
});

function initTabs() {
  document.querySelectorAll('.tab').forEach(tab => {
    tab.addEventListener('click', () => {
      document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
      document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
      tab.classList.add('active');
      document.getElementById(`${tab.dataset.tab}-tab`).classList.add('active');
    });
  });
}

async function checkStatus() {
  try {
    const response = await fetch(`${API_BASE}/token`);
    const result = await response.json();
    
    const statusEl = document.getElementById('status');
    if (result.state && result.data?.has_refresh_token) {
      statusEl.className = 'status-badge online';
      statusEl.innerHTML = '<span class="status-dot"></span><span>Online</span>';
    } else {
      statusEl.className = 'status-badge offline';
      statusEl.innerHTML = '<span class="status-dot"></span><span>No Token</span>';
    }
  } catch (error) {
    const statusEl = document.getElementById('status');
    statusEl.className = 'status-badge offline';
    statusEl.innerHTML = '<span class="status-dot"></span><span>Offline</span>';
  }
}

async function loadMagnets() {
  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    const response = await chrome.tabs.sendMessage(tab.id, { action: 'getMagnets' });
    
    const listEl = document.getElementById('magnet-list');
    const sendAllBtn = document.getElementById('send-all-btn');
    
    if (response?.magnets?.length > 0) {
      listEl.innerHTML = response.magnets.map((magnet, i) => `
        <div class="magnet-item">
          <div class="magnet-info">
            <div class="magnet-name">Magnet Link ${i + 1}</div>
            <div class="magnet-url">${magnet.substring(0, 50)}...</div>
          </div>
          <button class="btn btn-sm btn-success send-magnet" data-url="${magnet}">Send</button>
        </div>
      `).join('');
      
      sendAllBtn.style.display = 'block';
      sendAllBtn.onclick = () => sendAllMagnets(response.magnets);
      
      document.querySelectorAll('.send-magnet').forEach(btn => {
        btn.addEventListener('click', () => sendMagnet(btn.dataset.url));
      });
    } else {
      listEl.innerHTML = '<div class="empty-state"><p>No magnet links detected on this page</p></div>';
      sendAllBtn.style.display = 'none';
    }
  } catch (error) {
    document.getElementById('magnet-list').innerHTML = 
      '<div class="empty-state"><p>Could not scan page for magnets</p></div>';
  }
}

async function sendMagnet(url) {
  try {
    const response = await fetch(`${API_BASE}/download`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ urls: url })
    });
    const result = await response.json();
    
    if (result.state) {
      showToast('Task added successfully', 'success');
    } else {
      showToast(result.message || 'Failed to add task', 'error');
    }
  } catch (error) {
    showToast('Connection error', 'error');
  }
}

async function sendAllMagnets(magnets) {
  const urls = magnets.join('\n');
  await sendMagnet(urls);
}

async function loadTasks() {
  try {
    const response = await fetch(`${API_BASE}/download/tasks`);
    const result = await response.json();
    
    const listEl = document.getElementById('task-list');
    
    if (result.state && result.data?.tasks?.length > 0) {
      listEl.innerHTML = result.data.tasks.map(task => {
        const statusText = ['', '', 'Done', 'Failed'][task.status + 1] || 'Downloading';
        return `
          <div class="task-item">
            <div class="task-info">
              <div class="task-name">${task.name || 'Unknown'}</div>
              <div class="task-progress">
                <div class="progress-bar">
                  <div class="progress-fill" style="width:${task.percentDone}%"></div>
                </div>
                <span class="progress-text">${task.percentDone}%</span>
              </div>
            </div>
            <button class="btn btn-sm btn-danger delete-task" data-hash="${task.info_hash}">X</button>
          </div>
        `;
      }).join('');
      
      document.querySelectorAll('.delete-task').forEach(btn => {
        btn.addEventListener('click', () => deleteTask(btn.dataset.hash));
      });
    } else {
      listEl.innerHTML = '<div class="empty-state"><p>No download tasks</p></div>';
    }
  } catch (error) {
    document.getElementById('task-list').innerHTML = 
      '<div class="empty-state"><p>Failed to load tasks</p></div>';
  }
}

async function deleteTask(hash) {
  try {
    await fetch(`${API_BASE}/download/tasks/${hash}`, { method: 'DELETE' });
    loadTasks();
    showToast('Task deleted', 'success');
  } catch (error) {
    showToast('Failed to delete task', 'error');
  }
}

let currentCID = '0';

async function loadFiles(cid = '0') {
  currentCID = cid;
  try {
    const response = await fetch(`${API_BASE}/files?cid=${cid}&limit=50`);
    const result = await response.json();
    
    const listEl = document.getElementById('file-list');
    
    if (result.state && result.data?.length > 0) {
      listEl.innerHTML = result.data.map(file => {
        const isFolder = file.fc === '0';
        const icon = isFolder ? '📁' : '📄';
        const size = isFolder ? '' : formatSize(file.fs);
        
        return `
          <div class="file-item" data-id="${file.fid}" data-is-folder="${isFolder}">
            <div class="file-icon">${icon}</div>
            <div class="file-details">
              <div class="file-name">${file.fn}</div>
              <div class="file-meta">${size}</div>
            </div>
            <button class="btn btn-sm btn-danger delete-file" data-id="${file.fid}">X</button>
          </div>
        `;
      }).join('');
      
      document.querySelectorAll('.file-item').forEach(item => {
        item.addEventListener('click', (e) => {
          if (e.target.classList.contains('delete-file')) return;
          if (item.dataset.isFolder === 'true') {
            loadFiles(item.dataset.id);
            updateBreadcrumb(item.dataset.id, item.querySelector('.file-name').textContent);
          }
        });
      });
      
      document.querySelectorAll('.delete-file').forEach(btn => {
        btn.addEventListener('click', () => deleteFile(btn.dataset.id));
      });
    } else {
      listEl.innerHTML = '<div class="empty-state"><p>Empty folder</p></div>';
    }
  } catch (error) {
    document.getElementById('file-list').innerHTML = 
      '<div class="empty-state"><p>Failed to load files</p></div>';
  }
}

function updateBreadcrumb(cid, name) {
  const breadcrumb = document.getElementById('breadcrumb');
  breadcrumb.innerHTML = `
    <span class="breadcrumb-item" data-cid="0">Root</span>
    <span class="breadcrumb-separator">/</span>
    <span class="breadcrumb-item" data-cid="${cid}">${name}</span>
  `;
  
  breadcrumb.querySelectorAll('.breadcrumb-item').forEach(item => {
    item.addEventListener('click', () => {
      loadFiles(item.dataset.cid);
    });
  });
}

async function deleteFile(fileId) {
  if (!confirm('Delete this file?')) return;
  
  try {
    await fetch(`${API_BASE}/files/delete`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ file_ids: fileId })
    });
    loadFiles(currentCID);
    showToast('File deleted', 'success');
  } catch (error) {
    showToast('Failed to delete file', 'error');
  }
}

async function initMonitorToggle() {
  const toggle = document.getElementById('monitor-toggle');
  
  try {
    const response = await fetch(`${API_BASE}/download/monitor`);
    const result = await response.json();
    toggle.checked = result.data?.running || false;
  } catch (error) {}
  
  toggle.addEventListener('change', async () => {
    try {
      await fetch(`${API_BASE}/download/monitor`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: toggle.checked ? 'start' : 'stop' })
      });
      showToast(`Monitor ${toggle.checked ? 'started' : 'stopped'}`, 'success');
    } catch (error) {
      toggle.checked = !toggle.checked;
      showToast('Failed to toggle monitor', 'error');
    }
  });
}

document.getElementById('refresh-tasks-btn')?.addEventListener('click', loadTasks);
document.getElementById('refresh-files-btn')?.addEventListener('click', () => loadFiles(currentCID));

document.getElementById('new-folder-btn')?.addEventListener('click', async () => {
  const name = prompt('Enter folder name:');
  if (!name) return;
  
  try {
    await fetch(`${API_BASE}/folders`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ parent_id: currentCID, name })
    });
    loadFiles(currentCID);
    showToast('Folder created', 'success');
  } catch (error) {
    showToast('Failed to create folder', 'error');
  }
});

function formatSize(bytes) {
  if (!bytes) return '';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0;
  while (bytes >= 1024 && i < units.length - 1) {
    bytes /= 1024;
    i++;
  }
  return `${bytes.toFixed(1)} ${units[i]}`;
}

function showToast(message, type = 'success') {
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.textContent = message;
  document.body.appendChild(toast);
  setTimeout(() => toast.remove(), 3000);
}
