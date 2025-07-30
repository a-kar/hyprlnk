const API_BASE = 'http://localhost:4381/api';

document.addEventListener('DOMContentLoaded', async () => {
  await loadCurrentTab();
  await loadLatestSession();
  
  document.getElementById('saveBookmark').addEventListener('click', saveBookmark);
  document.getElementById('saveNewSession').addEventListener('click', saveNewSession);
  document.getElementById('updateLatestSession').addEventListener('click', updateLatestSession);
});

async function loadCurrentTab() {
  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (tab) {
      document.getElementById('title').value = tab.title || '';
      document.getElementById('url').value = tab.url || '';
    }
  } catch (error) {
    console.error('Error loading current tab:', error);
  }
}

async function saveBookmark() {
  const title = document.getElementById('title').value;
  const url = document.getElementById('url').value;
  const tagsInput = document.getElementById('tags').value;
  
  if (!title || !url) {
    showStatus('Please fill in title and URL', 'error');
    return;
  }
  
  const tags = tagsInput ? tagsInput.split(',').map(tag => tag.trim()) : [];
  
  const bookmark = {
    title,
    url,
    description: '',
    tags
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
      showStatus('Bookmark saved successfully!', 'success');
      document.getElementById('tags').value = '';
    } else {
      throw new Error('Failed to save bookmark');
    }
  } catch (error) {
    showStatus('Error saving bookmark: ' + error.message, 'error');
  }
}

let latestSession = null;

async function loadLatestSession() {
  try {
    const response = await fetch(`${API_BASE}/sessions`);
    if (response.ok) {
      const sessions = await response.json();
      if (sessions && sessions.length > 0) {
        // Sort by created_at to get the latest session
        sessions.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
        latestSession = sessions[0];
        
        // Show latest session info
        document.getElementById('latestSessionName').textContent = latestSession.name;
        document.getElementById('latestSessionDetails').textContent = 
          `${latestSession.tabs.length} tabs • ${formatDate(latestSession.created_at)}`;
        document.getElementById('latestSessionInfo').style.display = 'block';
        document.getElementById('updateLatestSession').style.display = 'block';
      }
    }
  } catch (error) {
    console.error('Error loading latest session:', error);
  }
}

async function saveNewSession() {
  const sessionName = document.getElementById('sessionName').value || `Session ${new Date().toLocaleString()}`;
  
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
      showStatus(`New session "${sessionName}" saved with ${sessionTabs.length} tabs!`, 'success');
      document.getElementById('sessionName').value = '';
      await loadLatestSession(); // Refresh latest session info
    } else {
      throw new Error('Failed to save session');
    }
  } catch (error) {
    showStatus('Error saving session: ' + error.message, 'error');
  }
}

async function updateLatestSession() {
  if (!latestSession) {
    showStatus('No session to update', 'error');
    return;
  }
  
  try {
    const tabs = await chrome.tabs.query({});
    const sessionTabs = tabs.map((tab, index) => ({
      url: tab.url,
      title: tab.title,
      active: tab.active,
      index: index
    }));
    
    const updatedSession = {
      ...latestSession,
      description: `${sessionTabs.length} tabs updated`,
      tabs: sessionTabs,
      updated_at: new Date().toISOString()
    };
    
    const response = await fetch(`${API_BASE}/sessions/${latestSession.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(updatedSession)
    });
    
    if (response.ok) {
      showStatus(`Updated "${latestSession.name}" with ${sessionTabs.length} tabs!`, 'success');
      await loadLatestSession(); // Refresh latest session info
    } else {
      throw new Error('Failed to update session');
    }
  } catch (error) {
    showStatus('Error updating session: ' + error.message, 'error');
  }
}

function formatDate(dateString) {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now - date;
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);
  
  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  
  return date.toLocaleDateString();
}

function showStatus(message, type) {
  const statusEl = document.getElementById('status');
  statusEl.textContent = message;
  statusEl.className = `status ${type}`;
  statusEl.style.display = 'block';
  
  setTimeout(() => {
    statusEl.style.display = 'none';
  }, 3000);
}