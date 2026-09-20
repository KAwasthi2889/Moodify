/**
 * Moodify Library Drawer (Burger Side Menu)
 * Slide-out panel that displays tracks stored on the server.
 * Provides the same rich per-track capabilities and multi-select as the Workshop:
 * - Multi-select checkboxes + master Select All checkbox
 * - Bulk download selected as streaming moodify_songs.zip
 * - Single track download with clean [original | english.ext] filename
 * - Instant audition playback
 * - [ ★ Find Similar ] navigation to Similarity Studio
 * - [ ✏️ Edit Tags ] metadata editor modal
 * - [ 🗑️ Remove ] glassmorphic deletion modal
 * - Real-time title/artist/filename search filter
 */

import { MoodifyAPI } from '../api/endpoints.js';
import { playTrack } from './AudioPlayer.js';
import { openMetadataModal } from './MetadataModal.js';
import { promptDeleteModal, showToast } from './Modal.js';

let drawerMount = null;
let isOpen = false;
let drawerSongs = [];
let selectedDrawerSongIds = new Set();
let drawerSearchFilter = '';
let currentNavigate = null;

export function initSessionDrawer(mountElement) {
  drawerMount = mountElement;
}

export async function openSessionDrawer(onNavigate) {
  if (!drawerMount) return;
  isOpen = true;
  currentNavigate = onNavigate;
  selectedDrawerSongIds.clear();
  drawerSearchFilter = '';

  renderSkeleton();
  await loadSessionTracks();
}

export function closeSessionDrawer() {
  if (!drawerMount) return;
  const panel = drawerMount.querySelector('#session-drawer-panel');
  if (panel) {
    panel.classList.remove('open');
    setTimeout(() => {
      drawerMount.innerHTML = '';
      isOpen = false;
    }, 200);
  } else {
    drawerMount.innerHTML = '';
    isOpen = false;
  }
}

function renderSkeleton() {
  drawerMount.innerHTML = `
    <div class="drawer-overlay" id="session-drawer-overlay">
      <div class="drawer-panel wide" id="session-drawer-panel">
        <!-- Drawer Header -->
        <div class="drawer-header">
          <div class="drawer-title-group">
            <h3 class="drawer-title">Uploaded Library</h3>
            <span class="drawer-sub">Stored tracks ready for discovery and playback</span>
          </div>
          <button class="drawer-close-btn" id="btn-close-drawer" title="Close Drawer" aria-label="Close Drawer">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        <!-- Drawer Body -->
        <div class="drawer-body" id="session-drawer-content">
          <div class="drawer-loading">
            <span class="spinner-sm"></span> Loading library tracks from server...
          </div>
        </div>
      </div>
    </div>
  `;

  // Attach close events
  const overlay = drawerMount.querySelector('#session-drawer-overlay');
  const closeBtn = drawerMount.querySelector('#btn-close-drawer');

  overlay?.addEventListener('click', (e) => {
    if (e.target === overlay) closeSessionDrawer();
  });
  closeBtn?.addEventListener('click', closeSessionDrawer);
}

async function loadSessionTracks() {
  try {
    const res = await MoodifyAPI.listSongs();
    drawerSongs = (res && res.songs) ? res.songs : [];
    renderDrawerContent();
  } catch (err) {
    console.error('Failed to load drawer tracks:', err);
    const contentEl = drawerMount?.querySelector('#session-drawer-content');
    if (contentEl) {
      contentEl.innerHTML = `
        <div class="drawer-empty-state">
          <div class="empty-icon" style="color: var(--color-text-dim);">
            <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
            </svg>
          </div>
          <h4>Server Offline</h4>
          <p>Could not connect to Go backend server at <code class="mono-code">/api/v1/songs</code>.</p>
        </div>
      `;
    }
  }
}

function renderDrawerContent() {
  const contentEl = drawerMount?.querySelector('#session-drawer-content');
  if (!contentEl) return;

  if (drawerSongs.length === 0) {
    contentEl.innerHTML = `
      <div class="drawer-empty-state">
        <div class="empty-icon">
          <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>
          </svg>
        </div>
        <h4>No Uploaded Tracks</h4>
        <p>No audio files have been uploaded yet.</p>
        <button class="btn-drawer-action primary" id="btn-drawer-jump-workshop">
          Open Workshop to Ingest
        </button>
      </div>
    `;
    contentEl.querySelector('#btn-drawer-jump-workshop')?.addEventListener('click', () => {
      closeSessionDrawer();
      if (currentNavigate) currentNavigate('workshop');
    });
    return;
  }

  // Filter songs based on search query (space-insensitive and punctuation-tolerant)
  const filteredSongs = drawerSearchFilter.trim()
    ? drawerSongs.filter(s => MoodifyAPI.matchesSongSearch(s, drawerSearchFilter))
    : drawerSongs;

  const selectedCount = selectedDrawerSongIds.size;
  const allFilteredSelected = filteredSongs.length > 0 && filteredSongs.every(s => selectedDrawerSongIds.has(s.id));

  contentEl.innerHTML = `
    <div class="drawer-inner-container">
      <!-- ── Search & Bulk Toolbar ────────────────────────────── -->
      <div class="drawer-toolbar">
        <div class="drawer-search-box">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
          <input 
            type="text" 
            class="drawer-search-input" 
            id="drawer-search-input" 
            placeholder="Search tracks, artists, files..." 
            value="${escapeHtml(drawerSearchFilter)}"
          />
          ${drawerSearchFilter ? `<button class="drawer-clear-search" id="btn-drawer-clear-search">&times;</button>` : ''}
        </div>

        <div class="drawer-selection-bar">
          <label class="drawer-select-all-label">
            <input type="checkbox" class="track-checkbox" id="drawer-select-all-cb" ${allFilteredSelected ? 'checked' : ''} />
            <span>Select All (${filteredSongs.length})</span>
          </label>

          ${selectedCount > 0 ? `
            <div class="drawer-selection-actions">
              <button class="btn-drawer-bulk-dl" id="btn-drawer-bulk-download" title="${selectedCount === 1 ? 'Download Selected Track' : 'Download Selected as ZIP'}">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                <span>${selectedCount === 1 ? 'Download' : `Download (${selectedCount})`}</span>
              </button>
              <button class="btn-drawer-bulk-remove" id="btn-drawer-bulk-remove" title="Remove Selected Tracks">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
                <span>Remove (${selectedCount})</span>
              </button>
            </div>
          ` : ''}
        </div>
      </div>

      <!-- ── Tracks List ──────────────────────────────────────── -->
      <div class="drawer-tracks-scroll">
        ${filteredSongs.length > 0 ? filteredSongs.map((song, index) => {
          const isSelected = selectedDrawerSongIds.has(song.id);
          const isReady = song.status === 'ready';
          const format = (song.format || 'mp3').toUpperCase();
          const isFormatCorrected = Boolean(song.extension_corrected || song.format_warning);
          const title = MoodifyAPI.getCleanTitle(song);
          const artist = song.artist || 'Unknown Artist';
          const album = song.album ? `• ${song.album}` : '';
          const cleanName = MoodifyAPI.getCleanFilename(song);

          return `
            <div class="drawer-track-card ${isSelected ? 'selected' : ''}" data-id="${escapeHtml(song.id)}">
              <div class="drawer-card-top">
                <input type="checkbox" class="track-checkbox drawer-track-cb" data-id="${escapeHtml(song.id)}" ${isSelected ? 'checked' : ''} />
                <button class="btn-drawer-play" data-action="play" data-id="${escapeHtml(song.id)}" title="Play Audio">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="6 3 20 12 6 21 6 3"/></svg>
                </button>
                <div class="drawer-track-text-group">
                  <div class="drawer-track-primary-title" title="${escapeHtml(title)}">${escapeHtml(title)}</div>
                  <div class="drawer-track-sub-text">${escapeHtml(artist)} ${escapeHtml(album)}</div>
                </div>
                <span class="drawer-format-badge ${isFormatCorrected ? 'format-corrected' : ''}" title="${escapeHtml(isFormatCorrected ? `Detected & auto-corrected format: ${format}` : `Format: ${format}`)}">${escapeHtml(format)}</span>
              </div>

              <!-- Action Toolbar per track -->
              <div class="drawer-card-actions">
                ${isReady ? `
                  <button class="btn-drawer-item-action find-sim" data-action="find-similar" data-id="${escapeHtml(song.id)}" title="Find Similar Songs">
                    <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
                    <span>Find Similar</span>
                  </button>
                ` : ''}

                <a class="btn-drawer-item-action" href="/api/v1/songs/${escapeHtml(song.id)}/download?download=true" download="${escapeHtml(cleanName)}" title="Download Track">
                  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                  <span>Download</span>
                </a>

                <button class="btn-drawer-item-action" data-action="edit-meta" data-id="${escapeHtml(song.id)}" title="Edit Metadata">
                  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
                  <span>Edit</span>
                </button>

                <button class="btn-drawer-item-action remove" data-action="remove" data-id="${escapeHtml(song.id)}" title="Remove Track">
                  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                  <span>Remove</span>
                </button>
              </div>
            </div>
          `;
        }).join('') : `
          <div class="drawer-empty-search">
            <p>No tracks match "${escapeHtml(drawerSearchFilter)}"</p>
          </div>
        `}
      </div>
    </div>
  `;

  attachDrawerListeners(filteredSongs);
}

function attachDrawerListeners(filteredSongs) {
  const contentEl = drawerMount?.querySelector('#session-drawer-content');
  if (!contentEl) return;

  // Search input
  const searchInput = contentEl.querySelector('#drawer-search-input');
  searchInput?.addEventListener('input', (e) => {
    drawerSearchFilter = e.target.value;
    renderDrawerContent();
    const newIn = contentEl.querySelector('#drawer-search-input');
    if (newIn) {
      newIn.focus();
      newIn.setSelectionRange(newIn.value.length, newIn.value.length);
    }
  });

  const clearSearchBtn = contentEl.querySelector('#btn-drawer-clear-search');
  clearSearchBtn?.addEventListener('click', () => {
    drawerSearchFilter = '';
    renderDrawerContent();
  });

  // Select all checkbox
  const selectAllCb = contentEl.querySelector('#drawer-select-all-cb');
  selectAllCb?.addEventListener('change', (e) => {
    if (e.target.checked) {
      filteredSongs.forEach(s => selectedDrawerSongIds.add(s.id));
    } else {
      filteredSongs.forEach(s => selectedDrawerSongIds.delete(s.id));
    }
    renderDrawerContent();
  });

  // Individual row checkboxes
  contentEl.querySelectorAll('.drawer-track-cb').forEach(cb => {
    cb.addEventListener('change', (e) => {
      const id = cb.dataset.id;
      if (e.target.checked) {
        selectedDrawerSongIds.add(id);
      } else {
        selectedDrawerSongIds.delete(id);
      }
      renderDrawerContent();
    });
  });

  // Bulk or single download selected tracks
  const bulkDlBtn = contentEl.querySelector('#btn-drawer-bulk-download');
  bulkDlBtn?.addEventListener('click', async () => {
    if (selectedDrawerSongIds.size === 0) return;
    const idsToDownload = Array.from(selectedDrawerSongIds);

    // If only 1 song is selected, download directly as individual file with clean filename
    if (idsToDownload.length === 1) {
      const singleId = idsToDownload[0];
      const song = drawerSongs.find(s => s.id === singleId);
      const cleanName = song ? MoodifyAPI.getCleanFilename(song) : 'track.mp3';
      const a = document.createElement('a');
      a.href = `/api/v1/songs/${encodeURIComponent(singleId)}/download?download=true`;
      a.download = cleanName;
      document.body.appendChild(a);
      a.click();
      a.remove();
      showToast(`Downloading "${cleanName}"`, 'success');
      return;
    }

    // If multiple songs selected, stream ZIP
    try {
      bulkDlBtn.disabled = true;
      bulkDlBtn.innerHTML = `<span class="spinner-sm"></span> Packaging ZIP...`;
      await MoodifyAPI.batchDownloadZip(idsToDownload);
      showToast(`Downloaded moodify_songs.zip (${idsToDownload.length} tracks)`, 'success');
    } catch (err) {
      showToast(`Failed to download zip: ${err.message}`, 'error');
    } finally {
      bulkDlBtn.disabled = false;
      renderDrawerContent();
    }
  });

  // Bulk remove selected tracks from drawer
  const bulkRemoveBtn = contentEl.querySelector('#btn-drawer-bulk-remove');
  bulkRemoveBtn?.addEventListener('click', () => {
    if (selectedDrawerSongIds.size === 0) return;
    const count = selectedDrawerSongIds.size;
    const targetLabel = count === 1 ? '1 selected track' : `${count} selected tracks`;
    promptDeleteModal(targetLabel, async () => {
      const ids = Array.from(selectedDrawerSongIds);
      for (const id of ids) {
        try {
          await MoodifyAPI.deleteSong(id);
        } catch (err) {
          console.warn(`Failed to delete song ${id}:`, err);
        }
      }
      drawerSongs = drawerSongs.filter(s => !selectedDrawerSongIds.has(s.id));
      selectedDrawerSongIds.clear();
      showToast(`Removed ${count} track${count > 1 ? 's' : ''} from library`, 'info');
      renderDrawerContent();
    });
  });

  // Play button click
  contentEl.querySelectorAll('button[data-action="play"]').forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.stopPropagation();
      const id = btn.dataset.id;
      const song = drawerSongs.find(s => s.id === id);
      if (song) playTrack(song, drawerSongs);
    });
  });

  // Clicking anywhere on the track card body plays the song (unless clicking on interactive controls)
  contentEl.querySelectorAll('.drawer-track-card').forEach(card => {
    card.addEventListener('click', (e) => {
      if (e.target.closest('input, button, a')) {
        return;
      }
      const id = card.dataset.id;
      const song = drawerSongs.find(s => s.id === id);
      if (song) playTrack(song, drawerSongs);
    });
  });

  // Find Similar
  contentEl.querySelectorAll('button[data-action="find-similar"]').forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.stopPropagation();
      const id = btn.dataset.id;
      const song = drawerSongs.find(s => s.id === id);
      if (song && currentNavigate) {
        closeSessionDrawer();
        currentNavigate('similarity', { seedSong: song });
      }
    });
  });

  // Edit metadata
  contentEl.querySelectorAll('button[data-action="edit-meta"]').forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.stopPropagation();
      const id = btn.dataset.id;
      const song = drawerSongs.find(s => s.id === id);
      if (song) {
        openMetadataModal(song, (updatedMeta) => {
          song.title = updatedMeta.title || song.title;
          song.artist = updatedMeta.artist || song.artist;
          song.album = updatedMeta.album || song.album;
          showToast(`Updated tags for "${song.title}"`, 'success');
          renderDrawerContent();
        });
      }
    });
  });

  // Remove track
  contentEl.querySelectorAll('button[data-action="remove"]').forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.stopPropagation();
      const id = btn.dataset.id;
      const song = drawerSongs.find(s => s.id === id);
      const name = song ? (song.title || song.original_name || song.filename || 'track') : 'track';

      promptDeleteModal(name, async () => {
        await MoodifyAPI.deleteSong(id);
        drawerSongs = drawerSongs.filter(s => s.id !== id);
        selectedDrawerSongIds.delete(id);
        showToast(`Removed "${name}" from library`, 'info');
        renderDrawerContent();
      });
    });
  });
}

function escapeHtml(str) {
  if (!str) return '';
  return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
