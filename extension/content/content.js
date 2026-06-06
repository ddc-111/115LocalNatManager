const MAGNET_REGEX = /magnet:\?xt=urn:[a-zA-Z0-9]+:[a-zA-Z0-9]{32,}/gi;
let API_BASE = 'http://localhost:11580';
let lastClipboardContent = '';
let checkInterval = null;

async function loadApiBase() {
  try {
    const settings = await chrome.storage.local.get(['serverUrl']);
    if (settings.serverUrl) {
      API_BASE = settings.serverUrl;
    }
  } catch (e) {}
}

loadApiBase();

chrome.storage.onChanged.addListener((changes) => {
  if (changes.serverUrl) {
    API_BASE = changes.serverUrl.newValue || 'http://localhost:11580';
  }
});

function scanForMagnets() {
  const pageText = document.body.innerText;
  const links = document.querySelectorAll('a[href^="magnet:"]');
  const magnets = new Set();

  links.forEach(link => {
    const href = link.getAttribute('href');
    if (href && href.startsWith('magnet:')) {
      magnets.add(href.split('&')[0]);
    }
  });

  const textMatches = pageText.match(MAGNET_REGEX);
  if (textMatches) {
    textMatches.forEach(m => magnets.add(m));
  }

  return Array.from(magnets);
}

async function addDownloadTask(url) {
  try {
    const settings = await chrome.storage.local.get(['serverUrl']);
    const apiBase = settings.serverUrl || API_BASE;
    
    const response = await fetch(`${apiBase}/api/download`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ urls: url })
    });
    return await response.json();
  } catch (error) {
    return { state: false, message: error.message };
  }
}

function showToast(message, type = 'success') {
  const existing = document.getElementById('cloud-115-clipboard-toast');
  if (existing) existing.remove();

  const toast = document.createElement('div');
  toast.id = 'cloud-115-clipboard-toast';
  toast.textContent = message;
  toast.style.cssText = `
    position: fixed;
    bottom: 20px;
    right: 20px;
    padding: 12px 20px;
    background: ${type === 'success' ? '#34a853' : type === 'error' ? '#ea4335' : '#1a73e8'};
    color: white;
    border-radius: 8px;
    font-size: 14px;
    z-index: 10000;
    box-shadow: 0 4px 12px rgba(0,0,0,0.3);
    animation: cloud115SlideIn 0.3s ease;
    font-family: -apple-system, BlinkMacSystemFont, sans-serif;
  `;
  
  document.body.appendChild(toast);
  
  setTimeout(() => {
    toast.style.animation = 'cloud115SlideOut 0.3s ease';
    setTimeout(() => toast.remove(), 300);
  }, 3000);
}

const style = document.createElement('style');
style.textContent = `
  @keyframes cloud115SlideIn {
    from { transform: translateY(20px); opacity: 0; }
    to { transform: translateY(0); opacity: 1; }
  }
  @keyframes cloud115SlideOut {
    from { transform: translateY(0); opacity: 1; }
    to { transform: translateY(20px); opacity: 0; }
  }
`;
document.head.appendChild(style);

async function checkClipboard() {
  try {
    const text = await navigator.clipboard.readText();
    if (!text || text === lastClipboardContent) return;
    
    const magnetMatch = text.match(MAGNET_REGEX);
    if (magnetMatch && magnetMatch.length > 0) {
      const magnetUrl = magnetMatch[0];
      if (magnetUrl !== lastClipboardContent) {
        lastClipboardContent = magnetUrl;
        
        const result = await addDownloadTask(magnetUrl);
        if (result.state) {
          showToast('磁力链已自动添加到115云下载', 'success');
        } else {
          showToast(result.message || '添加任务失败', 'error');
        }
      }
    }
  } catch (e) {
    // Clipboard API requires user gesture or permission
  }
}

function startClipboardMonitor() {
  if (checkInterval) return;
  
  document.addEventListener('copy', () => {
    setTimeout(checkClipboard, 100);
  });
  
  document.addEventListener('cut', () => {
    setTimeout(checkClipboard, 100);
  });
  
  checkInterval = true;
}

startClipboardMonitor();

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.action === 'getMagnets') {
    const magnets = scanForMagnets();
    sendResponse({ magnets });
  }
  return true;
});
