/**
 * Moodify Glassmorphic Toast System & Optimistic Action Runner
 * In accordance with @frontend skill HYBRID-API-ENGINE.
 */

export function showToast(message, type = 'success') {
  let container = document.getElementById('toast-container');
  if (!container) {
    container = document.createElement('div');
    container.id = 'toast-container';
    document.body.appendChild(container);
  }

  const toast = document.createElement('div');
  toast.className = `toast ${type}`;

  const icon = type === 'success'
    ? `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>`
    : `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>`;

  toast.innerHTML = `${icon}<span>${message}</span>`;
  container.appendChild(toast);

  // Trigger smooth slide-in
  requestAnimationFrame(() => {
    toast.classList.add('show');
  });

  setTimeout(() => {
    toast.classList.remove('show');
    setTimeout(() => toast.remove(), 260);
  }, 3200);
}

/**
 * Executes an optimistic action with button feedback and toast notification.
 */
export async function executeOptimisticAction({
  button,
  apiCall,
  onOptimisticApply,
  onRollback,
  successMessage
}) {
  const originalHtml = button ? button.innerHTML : '';
  if (button) {
    button.disabled = true;
    button.innerHTML = `<span class="spinner"></span> Processing...`;
  }

  if (onOptimisticApply) onOptimisticApply();

  try {
    const result = await apiCall();
    showToast(successMessage || 'Action completed successfully', 'success');
    return result;
  } catch (err) {
    if (onRollback) onRollback();
    showToast(err.message || 'Operation failed', 'error');
    throw err;
  } finally {
    if (button) {
      button.disabled = false;
      button.innerHTML = originalHtml;
    }
  }
}
