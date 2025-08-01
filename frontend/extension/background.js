const API_BASE = 'http://localhost:4381/api';
const DEBUG_MODE = false; // Set to true for development logging - shows URL tracking and sync events

// Debug logging helper
function debugLog(message, ...args) {
  if (DEBUG_MODE) {
    debugLog(message, ...args);
  }
}

debugLog('[HyprLnk] Background script loaded, API_BASE:', API_BASE);

// Context menu setup
chrome.runtime.onInstalled.addListener(() => {
  debugLog('[HyprLnk] Extension installed/reloaded - setting up...');
  
  chrome.contextMenus.create({
    id: 'saveBookmark',
    title: 'Save to HyprLnk',
    contexts: ['page']
  });

  chrome.contextMenus.create({
    id: 'saveSession',
    title: 'Save Session to HyprLnk',
    contexts: ['page']
  });

  // Start periodic history sync
  debugLog('[HyprLnk] Starting history sync system...');
  startHistorySync();
});

// Handle context menu clicks
chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (info.menuItemId === 'saveBookmark') {
    await saveBookmark(tab);
  } else if (info.menuItemId === 'saveSession') {
    await saveCurrentSession();
  }
});

async function saveBookmark(tab) {
  const bookmark = {
    url: tab.url,
    title: tab.title,
    description: '',
    tags: []
  };

  try {
    const response = await fetch(`${API_BASE}/bookmarks`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(bookmark)
    });

    if (response.ok) {
      debugLog('Bookmark saved successfully:', bookmark.title);
    }
  } catch (error) {
    console.error('Error saving bookmark:', error);
  }
}

async function saveCurrentSession() {
  try {
    const tabs = await chrome.tabs.query({});
    const sessionTabs = tabs.map((tab, index) => ({
      url: tab.url,
      title: tab.title,
      active: tab.active,
      index: index
    }));

    const session = {
      name: `Session ${new Date().toLocaleString()}`,
      description: `${sessionTabs.length} tabs saved`,
      tabs: sessionTabs,
      is_active: true
    };

    const response = await fetch(`${API_BASE}/sessions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(session)
    });

    if (response.ok) {
      debugLog(`Session saved successfully: ${sessionTabs.length} tabs`);
    }
  } catch (error) {
    console.error('Error saving session:', error);
  }
}

async function saveCurrentSessionWithName(sessionName) {
  try {
    const tabs = await chrome.tabs.query({});
    const sessionTabs = tabs.map((tab, index) => ({
      url: tab.url,
      title: tab.title,
      active: tab.active,
      index: index
    }));

    const session = {
      name: sessionName,
      description: `${sessionTabs.length} tabs saved`,
      tabs: sessionTabs,
      is_active: true
    };

    const response = await fetch(`${API_BASE}/sessions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(session)
    });

    if (response.ok) {
      const result = await response.json();
      debugLog(`Session "${sessionName}" saved successfully: ${sessionTabs.length} tabs`);
      return { message: `Session "${sessionName}" saved with ${sessionTabs.length} tabs`, session: result };
    } else {
      throw new Error(`Failed to save session: ${response.status} ${response.statusText}`);
    }
  } catch (error) {
    console.error('Error saving session:', error);
    throw error;
  }
}

// History sync functionality
function startHistorySync() {
  debugLog('[HyprLnk] Setting up automatic sync listeners...');
  
  // Sync immediately
  syncTodaysHistory();
  
  // Set up periodic sync every 10 minutes
  debugLog('[HyprLnk] Setting up 10-minute periodic sync...');
  setInterval(() => {
    debugLog('[HyprLnk] Periodic sync triggered (10 minutes)');
    syncTodaysHistory();
  }, 10 * 60 * 1000);
  
  // Also sync when user becomes active
  debugLog('[HyprLnk] Setting up idle state listener...');
  chrome.idle.onStateChanged.addListener((newState) => {
    debugLog('[HyprLnk] Idle state changed to:', newState);
    if (newState === 'active') {
      debugLog('[HyprLnk] User became active, syncing history');
      syncTodaysHistory();
    }
  });
  
  // Sync when tabs are updated (new page visits)
  debugLog('[HyprLnk] Setting up tab update listener...');
  chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    debugLog('[HyprLnk] Tab updated:', tabId, changeInfo, tab?.url);
    if (changeInfo.status === 'complete' && tab.url && !tab.url.startsWith('chrome://') && !tab.url.startsWith('chrome-extension://')) {
      debugLog('[HyprLnk] Page loaded, scheduling sync:', tab.url);
      // Use debounced sync to prevent multiple rapid syncs
      debouncedSync();
    }
  });
  
  // Also sync on navigation completed
  debugLog('[HyprLnk] Setting up navigation listener...');
  chrome.webNavigation.onCompleted.addListener((details) => {
    debugLog('[HyprLnk] Navigation event:', details.frameId, details.url);
    if (details.frameId === 0) { // Main frame only
      debugLog('[HyprLnk] Navigation completed, scheduling sync:', details.url);
      debouncedSync();
    }
  });
  
  debugLog('[HyprLnk] All sync listeners set up successfully');
}

// Debounced sync to prevent multiple rapid syncs
function debouncedSync() {
  // Clear any pending sync
  if (pendingSyncTimeout) {
    clearTimeout(pendingSyncTimeout);
  }
  
  // Schedule new sync with 3 second delay
  pendingSyncTimeout = setTimeout(() => {
    debugLog('[HyprLnk] Debounced sync executing...');
    syncTodaysHistory();
    pendingSyncTimeout = null;
  }, 3000);
}

async function syncTodaysHistory() {
  // Prevent concurrent syncs
  if (isSyncing) {
    debugLog('[HyprLnk] Sync already in progress, skipping...');
    return;
  }
  
  try {
    isSyncing = true;
    debugLog('[HyprLnk] Starting history sync...');
    const now = new Date();
    const startOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    
    debugLog(`[HyprLnk] Searching history from: ${new Date(startOfDay).toISOString()}`);
    
    const historyItems = await chrome.history.search({
      text: '',
      startTime: startOfDay,
      maxResults: 1000
    });

    debugLog(`[HyprLnk] Found ${historyItems.length} history items from Chrome`);

    // Filter and format history for today
    const todaysHistory = historyItems
      .filter(item => item.lastVisitTime >= startOfDay)
      .map(item => ({
        url: item.url,
        title: item.title || 'Untitled',
        visit_count: item.visitCount || 1,
        last_visit_time: new Date(item.lastVisitTime).toISOString()
      }));

    debugLog(`[HyprLnk] Filtered to ${todaysHistory.length} entries for today`);

    if (todaysHistory.length > 0) {
      debugLog('[HyprLnk] Syncing history to backend...');
      const response = await fetch(`${API_BASE}/history/sync`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          history: todaysHistory
        })
      });

      if (response.ok) {
        const result = await response.json();
        debugLog(`[HyprLnk] History synced successfully: ${result.synced_count} entries`);
      } else {
        console.error('[HyprLnk] History sync failed:', response.status, response.statusText);
      }
    } else {
      debugLog('[HyprLnk] No history to sync for today');
    }
  } catch (error) {
    console.error('[HyprLnk] Error syncing history:', error);
  } finally {
    isSyncing = false;
  }
}

// Link click tracking storage
let linkClickBuffer = [];

// Sync state management
let isSyncing = false;
let pendingSyncTimeout = null;

// Session restoration functionality
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.action === 'ping') {
    sendResponse({ pong: true });
    return true;
  }
  
  if (request.action === 'saveSession') {
    saveCurrentSessionWithName(request.sessionName)
      .then(result => sendResponse({ success: true, result }))
      .catch(error => sendResponse({ success: false, error: error.message }));
    return true; // Keep message channel open for async response
  }
  
  if (request.action === 'restoreSession') {
    restoreSession(request.session)
      .then(result => sendResponse({ success: true, result }))
      .catch(error => sendResponse({ success: false, error: error.message }));
    return true; // Keep message channel open for async response
  }

  if (request.action === 'trackLinkClicks') {
    handleLinkClicks(request.clicks);
    sendResponse({ success: true });
    return true;
  }

  if (request.action === 'triggerSync') {
    debugLog('[HyprLnk] Manual sync triggered from content script');
    syncTodaysHistory();
    sendResponse({ success: true });
    return true;
  }
});

async function restoreSession(session) {
  try {
    // Get current window and all its tabs
    const currentWindow = await chrome.windows.getCurrent({ populate: true });
    const currentTabs = currentWindow.tabs;
    
    // Find the HyprLnk tab (localhost:4381)
    const hyprLinkTab = currentTabs.find(tab => 
      tab.url && (tab.url.includes('localhost:4381') || tab.url.includes('127.0.0.1:4381'))
    );
    
    // Get tabs to close (all except HyprLnk tab)
    const tabsToClose = currentTabs.filter(tab => tab.id !== hyprLinkTab?.id);
    
    // Close non-HyprLnk tabs
    if (tabsToClose.length > 0) {
      const tabIds = tabsToClose.map(tab => tab.id);
      await chrome.tabs.remove(tabIds);
    }
    
    // Create new tabs from session
    const validTabs = session.tabs.filter(tab => 
      tab.url && 
      !tab.url.startsWith('chrome://') && 
      !tab.url.startsWith('chrome-extension://') &&
      !tab.url.startsWith('about:') &&
      !tab.url.startsWith('moz-extension://')
    );
    
    const createdTabs = [];
    for (let i = 0; i < validTabs.length; i++) {
      const sessionTab = validTabs[i];
      try {
        const newTab = await chrome.tabs.create({
          url: sessionTab.url,
          active: i === 0, // Make first tab active
          windowId: currentWindow.id
        });
        createdTabs.push(newTab);
        
        // Small delay between tab creation to avoid overwhelming the browser
        if (i < validTabs.length - 1) {
          await new Promise(resolve => setTimeout(resolve, 100));
        }
      } catch (error) {
        console.error(`Failed to create tab for ${sessionTab.url}:`, error);
      }
    }
    
    // Focus the HyprLnk tab after restoration
    if (hyprLinkTab) {
      await chrome.tabs.update(hyprLinkTab.id, { active: true });
    }
    
    return {
      message: `Restored session "${session.name}" with ${createdTabs.length} tabs`,
      tabsCreated: createdTabs.length,
      tabsClosed: tabsToClose.length
    };
    
  } catch (error) {
    console.error('Error restoring session:', error);
    throw error;
  }
}

// Link click tracking functionality
function handleLinkClicks(clicks) {
  // Add clicks to buffer
  linkClickBuffer.push(...clicks);
  debugLog(`[HyprLnk] Received ${clicks.length} link clicks, buffer size: ${linkClickBuffer.length}`);
  
  // Sync if buffer is getting large or periodically
  if (linkClickBuffer.length >= 10) {
    syncLinkClicks();
  }
}

async function syncLinkClicks() {
  if (linkClickBuffer.length === 0) return;

  const clicksToSync = [...linkClickBuffer];
  linkClickBuffer = [];

  try {
    const response = await fetch(`${API_BASE}/link-clicks/sync`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        clicks: clicksToSync
      })
    });

    if (response.ok) {
      const result = await response.json();
      debugLog(`[HyprLnk] Link clicks synced: ${result.synced_count || clicksToSync.length} entries`);
    } else {
      // Put clicks back in buffer if sync failed
      linkClickBuffer.unshift(...clicksToSync);
      console.error(`[HyprLnk] Failed to sync link clicks: ${response.status}`);
    }
  } catch (error) {
    // Put clicks back in buffer if sync failed
    linkClickBuffer.unshift(...clicksToSync);
    console.error('[HyprLnk] Error syncing link clicks:', error);
  }
}

// Periodic sync of link clicks
setInterval(() => {
  if (linkClickBuffer.length > 0) {
    debugLog('[HyprLnk] Periodic sync of link clicks');
    syncLinkClicks();
  }
}, 5 * 60 * 1000); // Every 5 minutes

// Sync on extension startup
chrome.runtime.onStartup.addListener(() => {
  debugLog('[HyprLnk] Extension startup - syncing any buffered link clicks');
  syncLinkClicks();
});