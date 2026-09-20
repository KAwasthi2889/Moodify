/**
 * Moodify API Client & Session Manager
 * Transparently attaches X-Session-ID header to backend requests.
 * Zero dummy data: strictly performs authentic HTTP requests to the Go backend.
 */

const API_BASE_URL = import.meta.env?.VITE_API_URL || '';

export const apiState = {
  isLive: false,
  checked: false,
  statusListeners: []
};

/**
 * Gets or initializes the client session ID from localStorage
 */
export function getSessionId() {
  let sessionId = localStorage.getItem('moodify_session_id');
  if (!sessionId) {
    sessionId = typeof crypto.randomUUID === 'function' 
      ? crypto.randomUUID() 
      : 'session-' + Math.random().toString(36).substring(2, 11);
    localStorage.setItem('moodify_session_id', sessionId);
  }
  return sessionId;
}

/**
 * Subscribe to API status changes
 */
export function onApiStatusChange(callback) {
  apiState.statusListeners.push(callback);
}

function notifyStatus() {
  apiState.statusListeners.forEach(cb => {
    try {
      cb(apiState.isLive);
    } catch (e) {
      console.error('Error in status listener', e);
    }
  });
}

/**
 * Pings backend health
 */
export async function checkBackendHealth() {
  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 2000);

    const res = await fetch(`${API_BASE_URL}/api/v1/health`, {
      method: 'GET',
      signal: controller.signal
    });
    clearTimeout(timeoutId);

    apiState.isLive = res.ok;
  } catch (err) {
    apiState.isLive = false;
  } finally {
    apiState.checked = true;
    notifyStatus();
  }
}

/**
 * Clean HTTP fetch wrapper passing session headers
 */
export async function apiFetch(endpoint, options = {}) {
  const sessionId = getSessionId();
  const headers = {
    'X-Session-ID': sessionId,
    ...(options.headers || {})
  };

  if (!(options.body instanceof FormData) && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json';
  }

  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers
  });

  if (!response.ok) {
    const errorText = await response.text().catch(() => response.statusText);
    let errorMessage = `HTTP ${response.status}: ${errorText}`;
    if (errorText.includes('ECONNREFUSED') || errorText.includes('Proxy error')) {
      errorMessage = 'Backend server is offline (connection refused at http://localhost:8080). Please ensure the Go server is running.';
    } else {
      try {
        const parsed = JSON.parse(errorText);
        if (parsed.error) errorMessage = parsed.error;
      } catch (_) {}
    }
    throw new Error(errorMessage);
  }

  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    return await response.json();
  }
  return response;
}
