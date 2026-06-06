const DEFAULT_API_BASE = 'http://localhost:11580';

chrome.action.onClicked.addListener(async () => {
  const url = chrome.runtime.getURL('dashboard/dashboard.html');
  const tabs = await chrome.tabs.query({ url: url });
  
  if (tabs.length > 0) {
    chrome.tabs.update(tabs[0].id, { active: true });
  } else {
    chrome.tabs.create({ url: url });
  }
});

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: 'send-to-115',
    title: 'Send to 115 Cloud',
    contexts: ['link']
  });
});

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (info.menuItemId === 'send-to-115') {
    const url = info.linkUrl;
    if (url && (url.startsWith('magnet:') || url.startsWith('http'))) {
      const settings = await chrome.storage.local.get(['serverUrl']);
      const apiBase = (settings.serverUrl || DEFAULT_API_BASE).replace(/\/+$/, '');
      
      try {
        const response = await fetch(`${apiBase}/api/download`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ urls: url })
        });
        const result = await response.json();
        
        chrome.notifications.create({
          type: 'basic',
          iconUrl: 'icons/icon128.png',
          title: '115 Cloud Manager',
          message: result.state ? 'Task added successfully' : `Failed: ${result.message}`
        });
      } catch (error) {
        chrome.notifications.create({
          type: 'basic',
          iconUrl: 'icons/icon128.png',
          title: '115 Cloud Manager',
          message: `Error: ${error.message}`
        });
      }
    }
  }
});

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.action === 'api') {
    handleAPI(request)
      .then(sendResponse)
      .catch(error => sendResponse({ error: error.message }));
    return true;
  }
  
  if (request.action === 'addDownload') {
    handleAddDownload(request)
      .then(sendResponse)
      .catch(error => sendResponse({ state: false, message: error.message }));
    return true;
  }
  
  if (request.action === 'getServerUrl') {
    chrome.storage.local.get(['serverUrl']).then(settings => {
      sendResponse({ serverUrl: settings.serverUrl || DEFAULT_API_BASE });
    });
    return true;
  }
});

async function handleAddDownload(request) {
  const { data } = request;
  
  const settings = await chrome.storage.local.get(['serverUrl']);
  const apiBase = (settings.serverUrl || DEFAULT_API_BASE).replace(/\/+$/, '');
  
  let pathId = data.path_id || '';
  
  if (!pathId) {
    try {
      const configResp = await fetch(`${apiBase}/api/config`);
      const configResult = await configResp.json();
      if (configResult.state && configResult.data?.default_save_path) {
        pathId = configResult.data.default_save_path;
      }
    } catch (e) {}
  }
  
  const postData = { ...data };
  if (pathId) {
    postData.path_id = pathId;
  }
  
  const response = await fetch(`${apiBase}/api/download`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(postData)
  });
  
  if (!response.ok) {
    const text = await response.text();
    return { state: false, message: `Server error: ${response.status} ${text}` };
  }
  
  try {
    const text = await response.text();
    return JSON.parse(text);
  } catch (e) {
    return { state: false, message: 'Invalid response from server' };
  }
}

async function handleAPI(request) {
  const settings = await chrome.storage.local.get(['serverUrl']);
  const apiBase = (settings.serverUrl || DEFAULT_API_BASE).replace(/\/+$/, '');
  const { method, path, body } = request;
  
  const options = {
    method: method || 'GET',
    headers: { 'Content-Type': 'application/json' }
  };

  if (body) {
    options.body = JSON.stringify(body);
  }

  const response = await fetch(`${apiBase}${path}`, options);
  return response.json();
}
