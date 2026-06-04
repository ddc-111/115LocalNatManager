const DEFAULT_API_BASE = 'http://localhost:11580';
let apiBase = DEFAULT_API_BASE;
let currentCID = '0';
let selectedFolder = null;

document.addEventListener('DOMContentLoaded', async () => {
  await loadSettings();
  initNavigation();
  initEventListeners();
  checkServerStatus();
  loadDashboard();
});

async function loadSettings() {
  const settings = await chrome.storage.local.get(['serverUrl', 'refreshToken', 'accessToken']);
  if (settings.serverUrl) {
    apiBase = settings.serverUrl;
    document.getElementById('server-url').value = settings.serverUrl;
  }
  if (settings.refreshToken) {
    document.getElementById('refresh-token').value = settings.refreshToken;
  }
  if (settings.accessToken) {
    document.getElementById('access-token').value = settings.accessToken;
  }
}

function initNavigation() {
  document.querySelectorAll('.nav-links li').forEach(item => {
    item.addEventListener('click', () => {
      const page = item.dataset.page;
      showPage(page);
    });
  });
}

function showPage(page) {
  document.querySelectorAll('.nav-links li').forEach(item => {
    item.classList.toggle('active', item.dataset.page === page);
  });
  
  document.querySelectorAll('.page').forEach(p => {
    p.classList.toggle('active', p.id === `page-${page}`);
  });
  
  const titles = {
    dashboard: '控制台',
    magnets: '磁力链接',
    downloads: '下载任务',
    files: '文件管理',
    settings: '设置'
  };
  document.getElementById('page-title').textContent = titles[page] || page;
  
  if (page === 'downloads') loadTasks();
  if (page === 'files') loadFiles(currentCID);
}

function initEventListeners() {
  document.getElementById('refresh-btn').addEventListener('click', () => {
    const activePage = document.querySelector('.nav-links li.active')?.dataset.page;
    if (activePage === 'dashboard') loadDashboard();
    if (activePage === 'downloads') loadTasks();
    if (activePage === 'files') loadFiles(currentCID);
  });
  
  document.getElementById('open-settings').addEventListener('click', () => showPage('settings'));
  
  document.getElementById('add-magnet-btn').addEventListener('click', addMagnets);
  document.getElementById('scan-page-btn').addEventListener('click', scanCurrentPage);
  
  document.getElementById('refresh-tasks-btn').addEventListener('click', loadTasks);
  document.getElementById('clear-tasks-btn').addEventListener('click', clearAllTasks);
  
  document.getElementById('new-folder-btn').addEventListener('click', createNewFolder);
  document.getElementById('refresh-files-btn').addEventListener('click', () => loadFiles(currentCID));
  document.getElementById('file-search').addEventListener('keyup', debounce(searchFiles, 300));
  
  document.getElementById('test-connection-btn').addEventListener('click', testConnection);
  document.getElementById('save-token-btn').addEventListener('click', saveToken);
  document.getElementById('save-config-btn').addEventListener('click', saveConfig);
  
  document.getElementById('toggle-token-visibility').addEventListener('click', () => {
    const input = document.getElementById('refresh-token');
    const icon = document.querySelector('#toggle-token-visibility i');
    input.type = input.type === 'password' ? 'text' : 'password';
    icon.classList.toggle('fa-eye');
    icon.classList.toggle('fa-eye-slash');
  });
  
  document.getElementById('toggle-access-token-visibility').addEventListener('click', () => {
    const input = document.getElementById('access-token');
    const icon = document.querySelector('#toggle-access-token-visibility i');
    input.type = input.type === 'password' ? 'text' : 'password';
    icon.classList.toggle('fa-eye');
    icon.classList.toggle('fa-eye-slash');
  });
  
  document.getElementById('toggle-monitor-btn').addEventListener('click', toggleMonitor);
  
  document.querySelectorAll('.filter-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      filterTasks(btn.dataset.filter);
    });
  });
  
  document.querySelectorAll('.modal-close').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.modal').forEach(m => m.classList.remove('active'));
    });
  });
  
  document.getElementById('select-folder-btn').addEventListener('click', openFolderModal);
  document.getElementById('confirm-folder-btn').addEventListener('click', confirmFolderSelection);
}

async function checkServerStatus() {
  try {
    const response = await fetch(`${apiBase}/health`, { signal: AbortSignal.timeout(3000) });
    const result = await response.json();
    
    const statusEl = document.getElementById('server-status');
    if (result.status === 'ok') {
      statusEl.innerHTML = '<span class="status-dot online"></span><span>已连接</span>';
      return true;
    }
  } catch (error) {}
  
  const statusEl = document.getElementById('server-status');
  statusEl.innerHTML = '<span class="status-dot offline"></span><span>未连接</span>';
  return false;
}

async function loadDashboard() {
  const isConnected = await checkServerStatus();
  if (!isConnected) return;
  
  try {
    const [tasksRes, quotaRes, userRes] = await Promise.all([
      apiGet('/api/download/tasks'),
      apiGet('/api/download/quota'),
      apiGet('/api/user')
    ]);
    
    if (tasksRes.state && tasksRes.data?.tasks) {
      const tasks = tasksRes.data.tasks;
      const active = tasks.filter(t => t.status === 0 || t.status === 1).length;
      const completed = tasks.filter(t => t.status === 2).length;
      
      document.getElementById('active-tasks').textContent = active;
      document.getElementById('completed-tasks').textContent = completed;
      
      renderRecentTasks(tasks.slice(0, 5));
    }
    
    if (quotaRes.state && quotaRes.data) {
      const quota = quotaRes.data;
      document.getElementById('download-quota').textContent = 
        `${quota.surplus || 0}/${quota.count || 0}`;
    }
    
    if (userRes.state && userRes.data) {
      const user = userRes.data;
      document.getElementById('storage-used').textContent = 
        user.rt_space_info?.all_use?.size_format || '--';
    }
  } catch (error) {
    console.error('失败 to load dashboard:', error);
  }
}

function renderRecentTasks(tasks) {
  const container = document.getElementById('recent-downloads');
  if (!tasks.length) {
    container.innerHTML = '<div class="empty-state"><i class="fas fa-inbox"></i><p>暂无下载记录</p></div>';
    return;
  }
  
  container.innerHTML = tasks.map(task => {
    const statusClass = task.status === 2 ? 'completed' : task.status === -1 ? 'failed' : 'downloading';
    const statusIcon = task.status === 2 ? 'check-circle' : task.status === -1 ? 'times-circle' : 'spinner fa-spin';
    
    return `
      <div class="task-item">
        <div class="task-icon ${statusClass}">
          <i class="fas fa-${statusIcon}"></i>
        </div>
        <div class="task-info">
          <div class="task-name">${escapeHtml(task.name || '未知')}</div>
          <div class="task-meta">${formatSize(task.size)} · ${formatTime(task.add_time)}</div>
        </div>
        <div class="task-progress">
          <div class="progress-bar">
            <div class="progress-fill ${statusClass}" style="width:${task.percentDone}%"></div>
          </div>
          <span class="progress-text">${task.percentDone}%</span>
        </div>
      </div>
    `;
  }).join('');
}

async function addMagnets() {
  const input = document.getElementById('magnet-input').value.trim();
  if (!input) {
    showToast('请输入磁力链接', 'error');
    return;
  }
  
  const urls = input.split('\n').filter(u => u.trim().startsWith('magnet:')).join('\n');
  if (!urls) {
    showToast('未找到有效的磁力链接', 'error');
    return;
  }
  
  try {
    const result = await apiPost('/api/download', { urls, path_id: selectedFolder?.id || '' });
    if (result.state) {
      showToast('任务添加成功！', 'success');
      document.getElementById('magnet-input').value = '';
    } else {
      showToast(result.message || '添加任务失败', 'error');
    }
  } catch (error) {
    showToast('连接错误', 'error');
  }
}

async function scanCurrentPage() {
  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    const response = await chrome.tabs.sendMessage(tab.id, { action: 'getMagnets' });
    
    const container = document.getElementById('detected-magnets');
    if (response?.magnets?.length > 0) {
      container.innerHTML = response.magnets.map((magnet, i) => `
        <div class="magnet-item">
          <div class="magnet-info">
            <div class="magnet-hash">${escapeHtml(magnet)}</div>
          </div>
          <div class="magnet-actions">
            <button class="btn btn-sm btn-primary add-detected" data-url="${escapeHtml(magnet)}">
              <i class="fas fa-plus"></i> 添加
            </button>
          </div>
        </div>
      `).join('');
      
      container.querySelectorAll('.add-detected').forEach(btn => {
        btn.addEventListener('click', async () => {
          const url = btn.dataset.url;
          try {
            const result = await apiPost('/api/download', { urls: url });
            if (result.state) {
              showToast('任务已添加！', 'success');
              btn.disabled = true;
              btn.textContent = '添加ed';
            }
          } catch (error) {
            showToast('添加任务失败', 'error');
          }
        });
      });
    } else {
      container.innerHTML = '<div class="empty-state"><i class="fas fa-magnet"></i><p>当前页面未检测到磁力链接</p></div>';
    }
  } catch (error) {
    showToast('无法扫描页面', 'error');
  }
}

async function loadTasks() {
  const container = document.getElementById('download-tasks');
  container.innerHTML = '<div class="loading"><div class="spinner"></div><p>加载中...</p></div>';
  
  try {
    const result = await apiGet('/api/download/tasks');
    if (result.state && result.data?.tasks?.length > 0) {
      renderTasks(result.data.tasks);
    } else {
      container.innerHTML = '<div class="empty-state"><i class="fas fa-cloud-download-alt"></i><p>暂无下载任务</p></div>';
    }
  } catch (error) {
    container.innerHTML = '<div class="empty-state"><i class="fas fa-exclamation-circle"></i><p>加载任务失败</p></div>';
  }
}

function renderTasks(tasks) {
  const container = document.getElementById('download-tasks');
  container.innerHTML = tasks.map(task => {
    const statusClass = task.status === 2 ? 'completed' : task.status === -1 ? 'failed' : task.status === 0 ? 'pending' : 'downloading';
    const statusIcon = task.status === 2 ? 'check-circle' : task.status === -1 ? 'times-circle' : task.status === 0 ? 'clock' : 'spinner fa-spin';
    const statusText = task.status === 2 ? '已完成' : task.status === -1 ? '失败' : task.status === 0 ? '等待中' : '下载中';
    
    return `
      <div class="task-item" data-status="${statusClass}">
        <div class="task-icon ${statusClass}">
          <i class="fas fa-${statusIcon}"></i>
        </div>
        <div class="task-info">
          <div class="task-name">${escapeHtml(task.name || '未知')}</div>
          <div class="task-meta">${formatSize(task.size)} · ${statusText}</div>
        </div>
        <div class="task-progress">
          <div class="progress-bar">
            <div class="progress-fill ${statusClass}" style="width:${task.percentDone}%"></div>
          </div>
          <span class="progress-text">${task.percentDone}%</span>
        </div>
        <div class="task-actions">
          <button class="btn btn-icon btn-danger delete-task" data-hash="${task.info_hash}" title="Delete">
            <i class="fas fa-trash"></i>
          </button>
        </div>
      </div>
    `;
  }).join('');
  
  container.querySelectorAll('.delete-task').forEach(btn => {
    btn.addEventListener('click', async () => {
      if (confirm('确定删除此任务？')) {
        try {
          await apiDelete(`/api/download/tasks/${btn.dataset.hash}`);
          showToast('任务已删除', 'success');
          loadTasks();
        } catch (error) {
          showToast('失败 to delete task', 'error');
        }
      }
    });
  });
}

function filterTasks(filter) {
  document.querySelectorAll('.task-item').forEach(item => {
    if (filter === 'all' || item.dataset.status === filter) {
      item.style.display = '';
    } else {
      item.style.display = 'none';
    }
  });
}

async function clearAllTasks() {
  if (!confirm('确定清空所有下载任务？')) return;
  
  try {
    await apiPost('/api/download/clear', { flag: 1 });
    showToast('所有任务已清空', 'success');
    loadTasks();
  } catch (error) {
    showToast('失败 to clear tasks', 'error');
  }
}

async function loadFiles(cid = '0') {
  currentCID = cid;
  const container = document.getElementById('file-list');
  container.innerHTML = '<div class="loading"><div class="spinner"></div><p>加载中...</p></div>';
  
  try {
    const result = await apiGet(`/api/files?cid=${cid}&limit=100`);
    if (result.state && result.data?.length > 0) {
      renderFiles(result.data);
    } else {
      container.innerHTML = '<div class="empty-state"><i class="fas fa-folder-open"></i><p>此文件夹为空</p></div>';
    }
  } catch (error) {
    container.innerHTML = '<div class="empty-state"><i class="fas fa-exclamation-circle"></i><p>失败 to load files</p></div>';
  }
}

function renderFiles(files) {
  const container = document.getElementById('file-list');
  container.innerHTML = files.map(file => {
    const isFolder = file.fc === '0';
    const icon = isFolder ? 'folder' : getFileIcon(file.ico);
    const size = isFolder ? '' : formatSize(file.fs);
    
    return `
      <div class="file-item" data-id="${file.fid}" data-is-folder="${isFolder}" data-name="${escapeHtml(file.fn)}">
        <div class="file-icon ${isFolder ? 'folder' : 'file'}">
          <i class="fas fa-${icon}"></i>
        </div>
        <div class="file-name">${escapeHtml(file.fn)}</div>
        <div class="file-size">${size}</div>
      </div>
    `;
  }).join('');
  
  container.querySelectorAll('.file-item').forEach(item => {
    item.addEventListener('click', (e) => {
      if (e.ctrlKey || e.metaKey) {
        item.classList.toggle('selected');
      } else if (item.dataset.isFolder === 'true') {
        loadFiles(item.dataset.id);
        updateBreadcrumb(item.dataset.id, item.dataset.name);
      }
    });
    
    item.addEventListener('contextmenu', (e) => {
      e.preventDefault();
      showFileContextMenu(e, item.dataset.id, item.dataset.isFolder === 'true');
    });
  });
}

function getFileIcon(ext) {
  const icons = {
    mp4: 'video', mkv: 'video', avi: 'video', mov: 'video',
    mp3: 'music', flac: 'music', wav: 'music', aac: 'music',
    jpg: 'image', jpeg: 'image', png: 'image', gif: 'image', webp: 'image',
    pdf: 'file-pdf', doc: 'file-word', docx: 'file-word',
    xls: 'file-excel', xlsx: 'file-excel',
    zip: 'file-archive', rar: 'file-archive', '7z': 'file-archive',
    txt: 'file-alt', md: 'file-alt',
    exe: 'file', app: 'file',
    torrent: 'magnet'
  };
  return icons[ext?.toLowerCase()] || 'file';
}

function updateBreadcrumb(cid, name) {
  const breadcrumb = document.getElementById('file-breadcrumb');
  if (cid === '0') {
    breadcrumb.innerHTML = '<span class="crumb active" data-cid="0"><i class="fas fa-home"></i> Root</span>';
  } else {
    breadcrumb.innerHTML = `
      <span class="crumb" data-cid="0"><i class="fas fa-home"></i> Root</span>
      <span class="crumb-separator">/</span>
      <span class="crumb active" data-cid="${cid}">${escapeHtml(name)}</span>
    `;
  }
  
  breadcrumb.querySelectorAll('.crumb').forEach(crumb => {
    crumb.addEventListener('click', () => {
      loadFiles(crumb.dataset.cid);
      if (crumb.dataset.cid === '0') {
        breadcrumb.innerHTML = '<span class="crumb active" data-cid="0"><i class="fas fa-home"></i> Root</span>';
      }
    });
  });
}

async function searchFiles() {
  const query = document.getElementById('file-search').value.trim();
  if (!query) {
    loadFiles(currentCID);
    return;
  }
  
  const container = document.getElementById('file-list');
  container.innerHTML = '<div class="loading"><div class="spinner"></div><p>搜索中...</p></div>';
  
  try {
    const result = await apiGet(`/api/files/search?q=${encodeURIComponent(query)}&limit=50`);
    if (result.state && result.data?.length > 0) {
      renderFiles(result.data);
    } else {
      container.innerHTML = '<div class="empty-state"><i class="fas fa-search"></i><p>未找到文件</p></div>';
    }
  } catch (error) {
    container.innerHTML = '<div class="empty-state"><i class="fas fa-exclamation-circle"></i><p>搜索失败</p></div>';
  }
}

async function createNewFolder() {
  const name = prompt('输入文件夹名称：');
  if (!name) return;
  
  try {
    const result = await apiPost('/api/folders', { parent_id: currentCID, name });
    if (result.state) {
      showToast('文件夹已创建', 'success');
      loadFiles(currentCID);
    } else {
      showToast(result.message || '失败 to create folder', 'error');
    }
  } catch (error) {
    showToast('失败 to create folder', 'error');
  }
}

function showFileContextMenu(e, fileId, isFolder) {
  const existing = document.querySelector('.context-menu');
  if (existing) existing.remove();
  
  const menu = document.createElement('div');
  menu.className = 'context-menu';
  menu.style.cssText = `
    position: fixed;
    left: ${e.clientX}px;
    top: ${e.clientY}px;
    background: white;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    padding: 8px;
    z-index: 1000;
    min-width: 160px;
  `;
  
  const items = [
    { icon: 'trash', label: 'Delete', action: () => deleteFile(fileId) },
    { icon: 'edit', label: 'Rename', action: () => renameFile(fileId) },
    { icon: 'download', label: 'Get Link', action: () => getFileLink(fileId), show: !isFolder }
  ].filter(i => i.show !== false);
  
  menu.innerHTML = items.map(item => `
    <div class="context-item" style="display:flex;align-items:center;gap:8px;padding:8px 12px;cursor:pointer;border-radius:4px;">
      <i class="fas fa-${item.icon}"></i>
      <span>${item.label}</span>
    </div>
  `).join('');
  
  menu.querySelectorAll('.context-item').forEach((item, i) => {
    item.addEventListener('click', () => {
      items[i].action();
      menu.remove();
    });
    item.addEventListener('mouseenter', () => item.style.background = '#f3f4f6');
    item.addEventListener('mouseleave', () => item.style.background = '');
  });
  
  document.body.appendChild(menu);
  
  document.addEventListener('click', () => menu.remove(), { once: true });
}

async function deleteFile(fileId) {
  if (!confirm('确定删除此文件？')) return;
  
  try {
    const result = await apiPost('/api/files/delete', { file_ids: fileId });
    if (result.state) {
      showToast('文件已删除', 'success');
      loadFiles(currentCID);
    }
  } catch (error) {
    showToast('失败 to delete file', 'error');
  }
}

async function renameFile(fileId) {
  const name = prompt('输入新名称：');
  if (!name) return;
  
  try {
    const result = await apiPut(`/api/files/${fileId}`, { name });
    if (result.state) {
      showToast('文件已重命名', 'success');
      loadFiles(currentCID);
    }
  } catch (error) {
    showToast('失败 to rename file', 'error');
  }
}

async function getFileLink(fileId) {
  try {
    const result = await apiGet(`/api/files/${fileId}`);
    if (result.state && result.data?.pick_code) {
      const dlResult = await apiPost('/api/files/download', { pick_code: result.data.pick_code });
      if (dlResult.state) {
        const url = Object.values(dlResult.data)[0]?.url?.url;
        if (url) {
          await navigator.clipboard.writeText(url);
          showToast('下载链接已复制！', 'success');
        }
      }
    }
  } catch (error) {
    showToast('失败 to get link', 'error');
  }
}

async function openFolderModal() {
  const modal = document.getElementById('folder-modal');
  modal.classList.add('active');
  
  const tree = document.getElementById('folder-tree');
  tree.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
  
  try {
    const result = await apiGet('/api/files?cid=0&limit=100');
    if (result.state) {
      const folders = result.data.filter(f => f.fc === '0');
      tree.innerHTML = folders.map(folder => `
        <div class="folder-item" data-id="${folder.fid}" data-name="${escapeHtml(folder.fn)}">
          <i class="fas fa-folder"></i>
          <span>${escapeHtml(folder.fn)}</span>
        </div>
      `).join('') || '<div class="empty-state"><p>暂无文件夹</p></div>';
      
      tree.querySelectorAll('.folder-item').forEach(item => {
        item.addEventListener('click', () => {
          tree.querySelectorAll('.folder-item').forEach(i => i.classList.remove('selected'));
          item.classList.add('selected');
          selectedFolder = { id: item.dataset.id, name: item.dataset.name };
        });
      });
    }
  } catch (error) {
    tree.innerHTML = '<div class="empty-state"><p>失败 to load folders</p></div>';
  }
}

function confirmFolderSelection() {
  if (selectedFolder) {
    document.getElementById('magnet-folder').value = selectedFolder.name;
  }
  document.getElementById('folder-modal').classList.remove('active');
}

async function testConnection() {
  const url = document.getElementById('server-url').value.trim();
  const resultEl = document.getElementById('connection-result');
  
  try {
    const response = await fetch(`${url}/health`, { signal: AbortSignal.timeout(5000) });
    const result = await response.json();
    
    if (result.status === 'ok') {
      resultEl.className = 'alert alert-success';
      resultEl.textContent = '连接成功！';
      apiBase = url;
      await chrome.storage.local.set({ serverUrl: url });
    } else {
      throw new Error('无效响应');
    }
  } catch (error) {
    resultEl.className = 'alert alert-error';
    resultEl.textContent = '连接失败，请检查服务器是否运行';
  }
  resultEl.style.display = 'block';
}

async function saveToken() {
  const refreshToken = document.getElementById('refresh-token').value.trim();
  const accessToken = document.getElementById('access-token').value.trim();
  const resultEl = document.getElementById('token-status');
  
  if (!refreshToken && !accessToken) {
    resultEl.className = 'alert alert-error';
    resultEl.textContent = '请输入刷新令牌或访问令牌';
    resultEl.style.display = 'block';
    return;
  }
  
  try {
    const body = {};
    if (refreshToken) body.refresh_token = refreshToken;
    if (accessToken) body.access_token = accessToken;
    
    const result = await apiPost('/api/token', body);
    if (result.state) {
      if (refreshToken) await chrome.storage.local.set({ refreshToken });
      if (accessToken) await chrome.storage.local.set({ accessToken });
      resultEl.className = 'alert alert-success';
      resultEl.textContent = '令牌保存成功！';
    } else {
      resultEl.className = 'alert alert-error';
      resultEl.textContent = result.message || '保存令牌失败';
    }
  } catch (error) {
    resultEl.className = 'alert alert-error';
    resultEl.textContent = '连接错误';
  }
  resultEl.style.display = 'block';
}
  
  try {
    const result = await apiPost('/api/token', { refresh_token: token });
    if (result.state) {
      await chrome.storage.local.set({ refreshToken: token });
      resultEl.className = 'alert alert-success';
      resultEl.textContent = '令牌保存成功！';
    } else {
      resultEl.className = 'alert alert-error';
      resultEl.textContent = result.message || '失败 to save token';
    }
  } catch (error) {
    resultEl.className = 'alert alert-error';
    resultEl.textContent = '连接错误';
  }
  resultEl.style.display = 'block';
}

async function saveConfig() {
  const resultEl = document.getElementById('config-status');
  const config = {
    download_dir: document.getElementById('download-dir').value,
    monitor_interval: parseInt(document.getElementById('monitor-interval').value)
  };
  
  try {
    const result = await apiPut('/api/config', config);
    if (result.state) {
      resultEl.className = 'alert alert-success';
      resultEl.textContent = '配置已保存！';
    } else {
      resultEl.className = 'alert alert-error';
      resultEl.textContent = result.message || '失败 to save config';
    }
  } catch (error) {
    resultEl.className = 'alert alert-error';
    resultEl.textContent = '连接错误';
  }
  resultEl.style.display = 'block';
}

async function toggleMonitor() {
  const btn = document.getElementById('toggle-monitor-btn');
  const isRunning = btn.querySelector('i').classList.contains('fa-pause');
  
  try {
    const result = await apiPost('/api/download/monitor', { 
      action: isRunning ? 'stop' : 'start' 
    });
    
    if (result.state) {
      const icon = btn.querySelector('i');
      const text = btn.querySelector('span');
      if (isRunning) {
        icon.classList.replace('fa-pause', 'fa-play');
        text.textContent = '启动监控';
        showToast('监控已停止', 'info');
      } else {
        icon.classList.replace('fa-play', 'fa-pause');
        text.textContent = '停止监控';
        showToast('监控已启动', 'success');
      }
    }
  } catch (error) {
    showToast('失败 to toggle monitor', 'error');
  }
}

async function apiGet(path) {
  const response = await fetch(`${apiBase}${path}`);
  return response.json();
}

async function apiPost(path, body) {
  const response = await fetch(`${apiBase}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  return response.json();
}

async function apiPut(path, body) {
  const response = await fetch(`${apiBase}${path}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  return response.json();
}

async function apiDelete(path) {
  const response = await fetch(`${apiBase}${path}`, { method: 'DELETE' });
  return response.json();
}

function showToast(message, type = 'info') {
  const container = document.getElementById('toast-container');
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.innerHTML = `<i class="fas fa-${type === 'success' ? 'check' : type === 'error' ? 'times' : 'info-circle'}"></i> ${message}`;
  container.appendChild(toast);
  
  setTimeout(() => {
    toast.style.animation = 'slideIn 0.3s ease reverse';
    setTimeout(() => toast.remove(), 300);
  }, 3000);
}

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

function formatTime(timestamp) {
  if (!timestamp) return '';
  return new Date(timestamp * 1000).toLocaleString();
}

function escapeHtml(str) {
  if (!str) return '';
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

function debounce(func, wait) {
  let timeout;
  return function executedFunction(...args) {
    clearTimeout(timeout);
    timeout = setTimeout(() => func.apply(this, args), wait);
  };
}
