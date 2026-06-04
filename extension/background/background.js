const API_BASE = 'http://localhost:11580/api';

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: 'send-to-115',
    title: 'Send to 115 Cloud',
    contexts: ['link']
  });
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
  if (info.menuItemId === 'send-to-115') {
    const url = info.linkUrl;
    if (url && (url.startsWith('magnet:') || url.startsWith('http'))) {
      sendToCloud(url);
    }
  }
});

async function sendToCloud(url) {
  try {
    const response = await fetch(`${API_BASE}/download`, {
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

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.action === 'api') {
    handleAPI(request)
      .then(sendResponse)
      .catch(error => sendResponse({ error: error.message }));
    return true;
  }
});

async function handleAPI(request) {
  const { method, path, body } = request;
  const options = {
    method: method || 'GET',
    headers: { 'Content-Type': 'application/json' }
  };

  if (body) {
    options.body = JSON.stringify(body);
  }

  const response = await fetch(`${API_BASE}${path}`, options);
  return response.json();
}
