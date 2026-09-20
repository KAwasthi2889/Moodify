/**
 * Moodify Glassmorphism Modal & Notification System
 * Replaces native browser alert/confirm with modern frosted-glass dialogs.
 */

function escapeHtml(str) {
  if (!str) return '';
  return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

/**
 * Displays a non-blocking toast alert.
 */
export function showToast(message, type = 'info') {
  const existing = document.querySelector('.moodify-toast');
  if (existing) existing.remove();

  const toast = document.createElement('div');
  toast.className = `moodify-toast toast-${type}`;
  toast.innerHTML = `
    <div class="toast-content">
      <span class="toast-text">${escapeHtml(message)}</span>
    </div>
  `;
  document.body.appendChild(toast);

  requestAnimationFrame(() => toast.classList.add('visible'));
  setTimeout(() => {
    toast.classList.remove('visible');
    setTimeout(() => toast.remove(), 350);
  }, 4000);
}

/**
 * Displays a glassmorphic confirmation modal for deleting a track.
 */
export function promptDeleteModal(trackName, onConfirm) {
  const overlay = document.createElement('div');
  overlay.className = 'glass-modal-overlay active';
  overlay.innerHTML = `
    <div class="glass-modal-card">
      <div class="modal-icon-badge danger">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/>
        </svg>
      </div>
      <h3 class="modal-title">Remove Track?</h3>
      <p class="modal-body-text">
        Are you sure you want to remove <strong>${escapeHtml(trackName)}</strong> from your library?
      </p>
      <div class="modal-actions-row">
        <button class="btn-modal-secondary" id="btn-modal-cancel">Cancel</button>
        <button class="btn-modal-danger" id="btn-modal-confirm">Remove Track</button>
      </div>
    </div>
  `;

  document.body.appendChild(overlay);

  const close = () => {
    overlay.classList.remove('active');
    setTimeout(() => overlay.remove(), 250);
  };

  overlay.querySelector('#btn-modal-cancel')?.addEventListener('click', close);
  overlay.addEventListener('click', (e) => {
    if (e.target === overlay) close();
  });

  overlay.querySelector('#btn-modal-confirm')?.addEventListener('click', async () => {
    close();
    try {
      await onConfirm();
    } catch (err) {
      showToast(`Failed to remove track: ${err.message}`, 'error');
    }
  });
}

/**
 * Displays Modal B for batch duplicate track resolution when tagging completes.
 */
export function showDuplicateSummaryModal(unresolvedDuplicates, { onReplace, onDiscard, onKeepAll }) {
  const overlay = document.createElement('div');
  overlay.className = 'glass-modal-overlay active';
  overlay.innerHTML = `
    <div class="glass-modal-card duplicate-summary-modal">
      <div class="modal-icon-badge warning">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
          <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
      </div>
      <h3 class="modal-title">Duplicate Tracks Detected</h3>
      <p class="modal-body-text">
        Tagging identified <strong>${unresolvedDuplicates.length}</strong> track(s) with matching acoustic audio fingerprints to existing songs in your library.
      </p>

      <div class="duplicate-modal-list">
        ${unresolvedDuplicates.map(song => {
          const title = song.title || song.original_name || song.filename || 'Untitled';
          const artist = song.artist || 'Unknown Artist';
          return `
            <div class="duplicate-modal-item" data-id="${escapeHtml(song.id)}">
              <div class="dup-item-info">
                <div class="dup-item-title">${escapeHtml(title)}</div>
                <div class="dup-item-artist">${escapeHtml(artist)}</div>
              </div>
              <div class="dup-item-actions">
                <button class="btn-dup-modal-replace" data-action="replace" data-id="${escapeHtml(song.id)}" data-target="${escapeHtml(song.duplicate_of || '')}">
                  Replace Older
                </button>
                <button class="btn-dup-modal-discard" data-action="discard" data-id="${escapeHtml(song.id)}">
                  Discard
                </button>
              </div>
            </div>
          `;
        }).join('')}
      </div>

      <div class="modal-actions-row dual">
        <button class="btn-modal-secondary" id="btn-dup-keep-all">Keep All Duplicates</button>
        <button class="btn-modal-danger" id="btn-dup-discard-all">Discard All Duplicates</button>
      </div>
    </div>
  `;

  document.body.appendChild(overlay);

  const close = () => {
    overlay.classList.remove('active');
    setTimeout(() => overlay.remove(), 250);
  };

  overlay.querySelector('#btn-dup-keep-all')?.addEventListener('click', () => {
    close();
    onKeepAll(unresolvedDuplicates);
  });

  overlay.querySelector('#btn-dup-discard-all')?.addEventListener('click', async () => {
    close();
    for (const song of unresolvedDuplicates) {
      await onDiscard(song.id);
    }
  });

  // Individual item actions
  overlay.querySelectorAll('button[data-action="replace"]').forEach(btn => {
    btn.addEventListener('click', async () => {
      const songId = btn.dataset.id;
      const targetId = btn.dataset.target;
      const row = overlay.querySelector(`.duplicate-modal-item[data-id="${songId}"]`);
      if (row) row.style.opacity = '0.4';
      await onReplace(songId, targetId);
      if (row) row.remove();
      if (overlay.querySelectorAll('.duplicate-modal-item').length === 0) close();
    });
  });

  overlay.querySelectorAll('button[data-action="discard"]').forEach(btn => {
    btn.addEventListener('click', async () => {
      const songId = btn.dataset.id;
      const row = overlay.querySelector(`.duplicate-modal-item[data-id="${songId}"]`);
      if (row) row.style.opacity = '0.4';
      await onDiscard(songId);
      if (row) row.remove();
      if (overlay.querySelectorAll('.duplicate-modal-item').length === 0) close();
    });
  });
}
