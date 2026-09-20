/**
 * Moodify Hybrid API Engine
 * Synthesized from @frontend skill HYBRID-API-ENGINE.
 * Seamlessly transitions between Live Go API and Mock Preview mode with local mutations.
 */

import { initialMockData } from './mockData.js';

// Base API configuration (proxied in vite.config.js)
const API_BASE_URL = import.meta.env?.VITE_API_URL || '';

// In-memory state store for mock mutations during interactive preview
export const inMemoryStore = JSON.parse(JSON.stringify(initialMockData));

export const apiState = {
  isLive: false,
  checked: false,
  statusListeners: []
};

/**
 * Subscribes to API status changes (Live vs Mock Preview)
 */
export function onApiStatusChange(callback) {
  apiState.statusListeners.push(callback);
}

function notifyStatus() {
  apiState.statusListeners.forEach(cb => {
    try {
      cb(apiState.isLive);
    } catch (e) {
      console.error(e);
    }
  });
}

/**
 * Pings backend health on application bootstrap
 */
export async function checkBackendHealth() {
  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 2000); // 2s fast timeout

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
    console.log(`[Moodify Hybrid API] Engine running in ${apiState.isLive ? '🟢 LIVE API' : '🟡 PREVIEW MODE (Interactive Mocks)'}`);
  }
}

/**
 * Universal hybrid request executor
 * @param {string} endpoint - API path e.g. '/api/v1/songs'
 * @param {Object} options - Fetch options
 * @param {string} mockCollectionKey - Key in inMemoryStore
 * @param {Function} mockCustomHandler - Optional custom handler for specific endpoints
 */
export async function hybridRequest(endpoint, options = {}, mockCollectionKey, mockCustomHandler) {
  if (apiState.isLive) {
    try {
      const res = await fetch(`${API_BASE_URL}${endpoint}`, {
        headers: {
          'Content-Type': 'application/json',
          ...(options.headers || {})
        },
        ...options
      });
      if (res.ok) {
        return await res.json();
      }
      console.warn(`[Hybrid API] Live endpoint ${endpoint} returned ${res.status}. Falling back to mock engine.`);
    } catch (err) {
      console.warn(`[Hybrid API] Live endpoint ${endpoint} failed: ${err.message}. Falling back to mock engine.`);
    }
  }

  // Mock Engine Fallback
  if (mockCustomHandler) {
    return mockCustomHandler(options.method || 'GET', options.body);
  }

  return handleMockCollectionMutation(options.method || 'GET', mockCollectionKey, options.body);
}

function handleMockCollectionMutation(method, key, body) {
  if (!key || !inMemoryStore[key]) {
    return { status: 'ok', success: true };
  }

  let parsed = null;
  if (body) {
    try {
      parsed = typeof body === 'string' ? JSON.parse(body) : body;
    } catch (e) {
      parsed = body;
    }
  }

  switch (method.toUpperCase()) {
    case 'GET':
      return { status: 'ok', songs: inMemoryStore[key], total: inMemoryStore[key].length };

    case 'POST': {
      const newItem = {
        id: `mock-${Date.now()}`,
        status: 'analyzed',
        created_at: new Date().toISOString(),
        ...parsed
      };
      inMemoryStore[key] = [newItem, ...inMemoryStore[key]];
      return { status: 'ok', song: newItem };
    }

    case 'PUT': {
      if (parsed?.id) {
        inMemoryStore[key] = inMemoryStore[key].map(item => item.id === parsed.id ? { ...item, ...parsed } : item);
      }
      return { status: 'ok', updated: true };
    }

    case 'DELETE': {
      if (parsed?.id) {
        inMemoryStore[key] = inMemoryStore[key].filter(item => item.id !== parsed.id);
      }
      return { status: 'ok', deleted: true };
    }

    default:
      return { status: 'ok', data: inMemoryStore[key] };
  }
}
