// Content script that runs on Hyprlink web pages
// This enables communication between the web page and the extension

// Listen for extension detection requests
window.addEventListener('hyprlink-check-extension', (event) => {
  // Respond that the extension is installed
  window.dispatchEvent(new CustomEvent('hyprlink-extension-response', {
    detail: { extensionInstalled: true, version: '1.0.0' }
  }));
});

// Enable message passing from web page to extension
window.addEventListener('hyprlink-message', (event) => {
  if (event.detail && event.detail.action) {
    chrome.runtime.sendMessage(event.detail, (response) => {
      window.dispatchEvent(new CustomEvent('hyprlink-response', {
        detail: { 
          id: event.detail.id,
          response: response 
        }
      }));
    });
  }
});

// Inject a marker that the extension is present
const marker = document.createElement('div');
marker.id = 'hyprlink-extension-marker';
marker.style.display = 'none';
marker.dataset.version = '1.0.0';
document.body.appendChild(marker);