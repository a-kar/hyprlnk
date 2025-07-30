const API_BASE = 'http://localhost:4381/api';

// Context menu setup
chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: 'saveBookmark',
    title: 'Save to Hyprlink',
    contexts: ['page']
  });

  chrome.contextMenus.create({
    id: 'saveSession',
    title: 'Save Session to Hyprlink',
    contexts: ['page']
  });

  // Start periodic history sync
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
      console.log('Bookmark saved successfully:', bookmark.title);
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
      console.log(`Session saved successfully: ${sessionTabs.length} tabs`);
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
      console.log(`Session "${sessionName}" saved successfully: ${sessionTabs.length} tabs`);
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
  // Sync immediately
  syncTodaysHistory();
  
  // Set up periodic sync every 10 minutes
  setInterval(syncTodaysHistory, 10 * 60 * 1000);
  
  // Also sync when user becomes active
  chrome.idle.onStateChanged.addListener((newState) => {
    if (newState === 'active') {
      syncTodaysHistory();
    }
  });
}

async function syncTodaysHistory() {
  try {
    const now = new Date();
    const startOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    
    const historyItems = await chrome.history.search({
      text: '',
      startTime: startOfDay,
      maxResults: 1000
    });

    // Filter and format history for today
    const todaysHistory = historyItems
      .filter(item => item.lastVisitTime >= startOfDay)
      .map(item => ({
        url: item.url,
        title: item.title || 'Untitled',
        visit_count: item.visitCount || 1,
        last_visit_time: new Date(item.lastVisitTime).toISOString()
      }));

    if (todaysHistory.length > 0) {
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
        console.log(`History synced: ${result.synced_count} entries`);
      }
    }
  } catch (error) {
    console.error('Error syncing history:', error);
  }
}

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
});

async function restoreSession(session) {
  try {
    // Get current window and all its tabs
    const currentWindow = await chrome.windows.getCurrent({ populate: true });
    const currentTabs = currentWindow.tabs;
    
    // Find the Hyprlink tab (localhost:4381)
    const hyprLinkTab = currentTabs.find(tab => 
      tab.url && (tab.url.includes('localhost:4381') || tab.url.includes('127.0.0.1:4381'))
    );
    
    // Get tabs to close (all except Hyprlink tab)
    const tabsToClose = currentTabs.filter(tab => tab.id !== hyprLinkTab?.id);
    
    // Close non-Hyprlink tabs
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
    
    // Focus the Hyprlink tab after restoration
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