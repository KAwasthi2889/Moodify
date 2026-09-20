/**
 * Moodify Status Badge Component
 * Shows Live Go API vs Preview Mode state with tooltips.
 */

import { onApiStatusChange, apiState } from '../api/client.js';

export function renderStatusBadge(container) {
  if (!container) return;
  const badge = document.createElement('div');
  badge.className = 'status-badge';
  badge.id = 'api-status-badge';

  function update(isLive) {
    badge.innerHTML = isLive
      ? '<span class="dot green"></span> Live API'
      : '<span class="dot amber"></span> Preview Mode (Mocks)';
    badge.title = isLive
      ? 'Connected to local Moodify Go backend (port 8080)'
      : 'Go backend offline or initializing. Running interactive mock engine.';
  }

  update(apiState.isLive);
  onApiStatusChange(update);
  container.appendChild(badge);
}
