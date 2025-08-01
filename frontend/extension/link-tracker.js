// HyprLnk Link Tracker v1.1
const DEBUG_MODE = false; // Set to true for development logging - shows click tracking and URL data

// Debug logging helper
function debugLog(message, ...args) {
  if (DEBUG_MODE) {
    debugLog(message, ...args);
  }
}

debugLog('[HyprLnk] Link tracker v1.1 loaded on:', window.location.href);

// Immediate test - this should appear in console
setTimeout(() => {
    debugLog('[HyprLnk] Delayed initialization test');
}, 100);

// Check if we're in the right context
if (window === window.top) {
    debugLog('[HyprLnk] Running in main frame');
} else {
    debugLog('[HyprLnk] Running in iframe/frame');
}

// Chrome extension API check
if (typeof chrome !== 'undefined' && chrome.runtime && chrome.runtime.id) {
    debugLog('[HyprLnk] Chrome extension API available, ID:', chrome.runtime.id);
} else {
    debugLog('[HyprLnk] Chrome extension API NOT available');
}

// Add click listener
function handleClick(event) {
    debugLog('[HyprLnk] RAW CLICK on:', event.target.tagName, event.target);
    
    // Find the anchor element
    let link = event.target;
    let attempts = 0;
    while (link && link.tagName !== 'A' && attempts < 10) {
        link = link.parentElement;
        attempts++;
    }
    
    if (link && link.tagName === 'A' && link.href) {
        debugLog('[HyprLnk] LINK FOUND:', {
            href: link.href,
            text: link.textContent?.trim(),
            target: link.target
        });
        
        // Don't track certain types of links
        if (link.href.startsWith('javascript:') || 
            link.href.startsWith('mailto:') || 
            link.href.startsWith('tel:') ||
            link.href === '#' ||
            link.href.endsWith('#')) {
            debugLog('[HyprLnk] Skipping special link:', link.href);
            return;
        }
        
        const clickData = {
            destinationUrl: link.href,
            destinationTitle: link.title || link.textContent?.trim() || 'Link',
            sourceUrl: window.location.href,
            sourceTitle: document.title,
            linkText: (link.textContent?.trim() || '').substring(0, 200),
            clickType: link.href.startsWith(window.location.origin) ? 'internal_link' : 'external_link',
            timestamp: Date.now(),
            domain: window.location.hostname,
            isNewTab: event.ctrlKey || event.metaKey || link.target === '_blank'
        };
        
        debugLog('[HyprLnk] TRACKING CLICK:', clickData);
        
        // Send to background script
        if (chrome && chrome.runtime && chrome.runtime.sendMessage) {
            chrome.runtime.sendMessage({
                action: 'trackLinkClicks',
                clicks: [clickData]
            }, function(response) {
                if (chrome.runtime.lastError) {
                    console.error('[HyprLnk] Send error:', chrome.runtime.lastError.message);
                } else {
                    debugLog('[HyprLnk] Send success:', response);
                }
            });
        } else {
            console.error('[HyprLnk] Cannot send message - chrome.runtime not available');
        }
    } else {
        debugLog('[HyprLnk] No link found, clicked on:', event.target.tagName);
    }
}

// Attach listener
document.addEventListener('click', handleClick, true);
debugLog('[HyprLnk] Click listener attached to document');