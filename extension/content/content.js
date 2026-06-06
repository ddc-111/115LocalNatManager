const MAGNET_REGEX = /magnet:\?xt=urn:[a-zA-Z0-9]+:[a-zA-Z0-9]{32,}/gi;
let API_BASE = 'http://localhost:11580';

let detectedMagnets = [];

async function loadApiBase() {
  try {
    const settings = await chrome.storage.local.get(['serverUrl']);
    if (settings.serverUrl) {
      API_BASE = settings.serverUrl;
    }
  } catch (e) {}
}

loadApiBase();

// 监听storage变化
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

function createMagnetButton(magnetUrl) {
  const btn = document.createElement('button');
  btn.className = 'cloud-115-magnet-btn';
  btn.innerHTML = `
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
      <polyline points="7 10 12 15 17 10"/>
      <line x1="12" y1="15" x2="12" y2="3"/>
    </svg>
    <span>Send to 115</span>
  `;
  btn.title = 'Send to 115 Cloud Download';

  btn.addEventListener('click', async (e) => {
    e.preventDefault();
    e.stopPropagation();
    
    btn.disabled = true;
    btn.querySelector('span').textContent = 'Sending...';

    try {
      const result = await new Promise((resolve, reject) => {
        chrome.runtime.sendMessage({
          action: 'addDownload',
          data: { urls: magnetUrl }
        }, (response) => {
          if (chrome.runtime.lastError) {
            reject(new Error(chrome.runtime.lastError.message));
          } else if (!response) {
            reject(new Error('No response from background'));
          } else {
            resolve(response);
          }
        });
      });
      
      if (result.state) {
        btn.querySelector('span').textContent = 'Sent!';
        btn.classList.add('success');
        showToast('Task added to 115 cloud download', 'success');
      } else {
        throw new Error(result.message || 'Failed to add task');
      }
    } catch (error) {
      btn.querySelector('span').textContent = 'Failed';
      btn.classList.add('error');
      showToast(error.message, 'error');
    }

    setTimeout(() => {
      btn.disabled = false;
      btn.querySelector('span').textContent = 'Send to 115';
      btn.classList.remove('success', 'error');
    }, 2000);
  });

  return btn;
}

function injectButtons() {
  const magnetLinks = document.querySelectorAll('a[href^="magnet:"]');
  
  magnetLinks.forEach(link => {
    if (link.dataset.cloud115Injected) return;
    link.dataset.cloud115Injected = 'true';

    const btn = createMagnetButton(link.href);
    btn.style.cssText = `
      display: inline-flex;
      align-items: center;
      gap: 4px;
      margin-left: 8px;
      padding: 4px 10px;
      background: #1a73e8;
      color: white;
      border: none;
      border-radius: 4px;
      font-size: 12px;
      cursor: pointer;
      vertical-align: middle;
    `;

    link.parentNode.insertBefore(btn, link.nextSibling);
  });
}

function showToast(message, type = 'success') {
  const toast = document.createElement('div');
  toast.className = `cloud-115-toast ${type}`;
  toast.textContent = message;
  toast.style.cssText = `
    position: fixed;
    bottom: 20px;
    right: 20px;
    padding: 12px 20px;
    background: ${type === 'success' ? '#34a853' : '#ea4335'};
    color: white;
    border-radius: 8px;
    font-size: 14px;
    z-index: 10000;
    animation: cloud115SlideIn 0.3s ease;
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
  .cloud-115-magnet-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px 10px;
    background: #1a73e8;
    color: white;
    border: none;
    border-radius: 4px;
    font-size: 12px;
    cursor: pointer;
    transition: background 0.2s;
  }
  .cloud-115-magnet-btn:hover {
    background: #1557b0;
  }
  .cloud-115-magnet-btn.success {
    background: #34a853;
  }
  .cloud-115-magnet-btn.error {
    background: #ea4335;
  }
  .cloud-115-magnet-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }
`;
document.head.appendChild(style);

injectButtons();

const observer = new MutationObserver(() => {
  injectButtons();
});

observer.observe(document.body, {
  childList: true,
  subtree: true
});

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.action === 'getMagnets') {
    const magnets = scanForMagnets();
    sendResponse({ magnets });
  }
  return true;
});
