/**
 * Moodify Metadata Correction Modal
 * Glassmorphic editor allowing users to review and manually correct ID3 / MusicBrainz tags.
 * Persists changes directly via PUT /api/v1/songs/{id}/metadata.
 */

import { MoodifyAPI } from '../api/endpoints.js';

let activeModal = null;

export function openMetadataModal(song, onSaved) {
  if (activeModal) closeMetadataModal();

  const modalContainer = document.createElement('div');
  modalContainer.className = 'modal-overlay';
  modalContainer.id = 'metadata-modal-overlay';

  const title = song.title || song.original_name || song.filename || '';
  const artist = song.artist || '';
  const album = song.album || '';
  const year = song.year || '';
  const genre = song.genre || '';

  modalContainer.innerHTML = `
    <div class="modal-card" id="metadata-modal-card">
      <div class="modal-header">
        <div class="modal-title-group">
          <h3 class="modal-title">Correct Audio Metadata</h3>
          <span class="modal-subtitle">${escapeHtml(song.original_name || song.filename || 'Track')}</span>
        </div>
        <button class="modal-close-btn" id="btn-close-metadata-modal" title="Close">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>

      <form class="metadata-form" id="metadata-form">
        <div class="form-group">
          <label class="form-label" for="meta-title">Track Title</label>
          <input class="form-input" id="meta-title" type="text" value="${escapeHtml(title)}" placeholder="e.g. Midnight City" required />
        </div>

        <div class="form-row">
          <div class="form-group flex-1">
            <label class="form-label" for="meta-artist">Artist / Performer</label>
            <input class="form-input" id="meta-artist" type="text" value="${escapeHtml(artist)}" placeholder="e.g. M83" />
          </div>
          <div class="form-group flex-1">
            <label class="form-label" for="meta-album">Album</label>
            <input class="form-input" id="meta-album" type="text" value="${escapeHtml(album)}" placeholder="e.g. Hurry Up, We're Dreaming" />
          </div>
        </div>

        <div class="form-row">
          <div class="form-group flex-1">
            <label class="form-label" for="meta-year">Release Year</label>
            <input class="form-input" id="meta-year" type="number" value="${escapeHtml(String(year))}" placeholder="e.g. 2011" min="1900" max="2099" />
          </div>
          <div class="form-group flex-1">
            <label class="form-label" for="meta-genre">Genre</label>
            <input class="form-input" id="meta-genre" type="text" value="${escapeHtml(genre)}" placeholder="e.g. Synth-pop" />
          </div>
        </div>

        <div class="metadata-status-msg" id="metadata-status-msg" style="display: none;"></div>

        <div class="modal-footer">
          <button type="button" class="btn-secondary" id="btn-cancel-metadata">Cancel</button>
          <button type="submit" class="btn-primary" id="btn-save-metadata">
            <span>Save Metadata</span>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
          </button>
        </div>
      </form>
    </div>
  `;

  document.body.appendChild(modalContainer);
  activeModal = modalContainer;

  // Focus title input
  setTimeout(() => {
    modalContainer.querySelector('#meta-title')?.focus();
  }, 100);

  // Close handlers
  modalContainer.addEventListener('click', (e) => {
    if (e.target === modalContainer) closeMetadataModal();
  });
  modalContainer.querySelector('#btn-close-metadata-modal')?.addEventListener('click', closeMetadataModal);
  modalContainer.querySelector('#btn-cancel-metadata')?.addEventListener('click', closeMetadataModal);

  // Keydown Esc to close
  const onKeyDown = (e) => {
    if (e.key === 'Escape') {
      closeMetadataModal();
      document.removeEventListener('keydown', onKeyDown);
    }
  };
  document.addEventListener('keydown', onKeyDown);

  // Submit handler
  const form = modalContainer.querySelector('#metadata-form');
  const saveBtn = modalContainer.querySelector('#btn-save-metadata');
  const statusMsg = modalContainer.querySelector('#metadata-status-msg');

  form?.addEventListener('submit', async (e) => {
    e.preventDefault();

    const updatedData = {
      title: modalContainer.querySelector('#meta-title').value.trim(),
      artist: modalContainer.querySelector('#meta-artist').value.trim(),
      album: modalContainer.querySelector('#meta-album').value.trim(),
      year: parseInt(modalContainer.querySelector('#meta-year').value.trim(), 10) || 0,
      genre: modalContainer.querySelector('#meta-genre').value.trim()
    };

    saveBtn.disabled = true;
    saveBtn.innerHTML = `<span class="spinner"></span> Saving...`;

    try {
      await MoodifyAPI.updateSongMetadata(song.id, updatedData);

      const mergedSong = {
        ...song,
        ...updatedData
      };

      if (onSaved) onSaved(mergedSong);
      closeMetadataModal();
    } catch (err) {
      saveBtn.disabled = false;
      saveBtn.innerHTML = `<span>Save Metadata</span>`;
      if (statusMsg) {
        statusMsg.style.display = 'block';
        statusMsg.className = 'metadata-status-msg error';
        statusMsg.textContent = `Failed to save: ${err.message}`;
      }
    }
  });
}

export function closeMetadataModal() {
  if (activeModal && activeModal.parentNode) {
    activeModal.parentNode.removeChild(activeModal);
    activeModal = null;
  }
}

function escapeHtml(str) {
  if (!str) return '';
  return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
