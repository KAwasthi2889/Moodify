/**
 * Moodify Library Drawer
 * Slide-out panel that displays tracks stored on the server.
 * Zero dummy data: strictly reflects authentic Go backend state.
 */

import { MoodifyAPI } from '../api/endpoints.js';
import { playTrack } from './AudioPlayer.js';

let drawerMount = null;
let isOpen = false;

export function initSessionDrawer(mountElement) {
  drawerMount = mountElement;
}

export async function openSessionDrawer() {
  if (!drawerMount) return;
  isOpen = true;

  drawerMount.innerHTML = `
    <div class="drawer-overlay" id="session-drawer-overlay">
      <div class="drawer-panel" id="session-drawer-panel">
        <!-- Drawer Header -->
        <div class="drawer-header">
          <div class="drawer-title-group">
            <h3 class="drawer-title">Uploaded Library</h3>
            <span class="drawer-sub">Stored tracks ready for playback</span>
          </div>
          <button class="drawer-close-btn" id="btn-close-drawer" title="Close Drawer" aria-label="Close Drawer">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        <!-- Drawer Content -->
        <div class="drawer-body" id="session-drawer-content">
          <div class="drawer-loading">
            <span class="spinner"></span> Loading library tracks from server...
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

  // Fetch real tracks from the Go backend
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

async function loadSessionTracks() {
  const contentEl = drawerMount?.querySelector('#session-drawer-content');
  if (!contentEl) return;

  try {
    const res = await MoodifyAPI.listSongs();
    const songs = (res && res.songs) ? res.songs : [];

    if (songs.length === 0) {
      contentEl.innerHTML = `
        <div class="drawer-empty-state">
          <div class="empty-icon">
            <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>
            </svg>
          </div>
          <h4>No Uploaded Tracks</h4>
          <p>No audio files have been uploaded yet. Open the Workshop to ingest your library.</p>
        </div>
      `;
      return;
    }

    contentEl.innerHTML = `
      <div class="drawer-tracks-list">
        <div class="drawer-list-meta">
          <span>Found <strong class="mono-num">${songs.length}</strong> tracks in library</span>
        </div>
        ${songs.map(song => {
          const status = song.status || 'uploaded';
          return `
            <div class="drawer-track-item" data-id="${escapeHtml(song.id)}">
              <button class="btn-drawer-play" data-action="play" data-id="${escapeHtml(song.id)}" title="Play Track">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="6 3 20 12 6 21 6 3"/></svg>
              </button>
              <div class="drawer-track-info">
                <span class="drawer-track-title">${escapeHtml(song.title || song.filename)}</span>
                <span class="drawer-track-artist">${escapeHtml(song.artist || 'Unknown Artist')} • <span class="format-pill">${escapeHtml((song.format || 'mp3').toUpperCase())}</span></span>
              </div>
              <span class="drawer-status-badge ${escapeHtml(status)}">${escapeHtml(status)}</span>
            </div>
          `;
        }).join('')}
      </div>
    `;

    // Hook up play buttons
    const items = contentEl.querySelectorAll('.drawer-track-item');
    items.forEach(item => {
      item.addEventListener('click', () => {
        const songId = item.dataset.id;
        const song = songs.find(s => s.id === songId);
        if (song) {
          playTrack(song);
        }
      });
    });
  } catch (err) {
    contentEl.innerHTML = `
      <div class="drawer-empty-state">
        <div class="empty-icon" style="color: var(--color-text-dim);">
          <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        </div>
        <h4>Server Offline</h4>
        <p>Could not connect to Go backend server at <code class="mono-code">/api/v1/songs</code>. Ensure the backend is running to access tracks.</p>
      </div>
    `;
  }
}

function escapeHtml(str) {
  if (!str) return '';
  return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
