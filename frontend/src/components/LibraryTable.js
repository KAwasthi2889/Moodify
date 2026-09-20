/**
 * Moodify Track Library Table Component
 * Displays ingested songs with real-time filters, mood badges, telemetry,
 * and direct triggers for playback, inspection, and deletion.
 */

import { playTrack } from './AudioPlayer.js';
import { openTrackStudio } from './TrackStudioModal.js';
import { MoodifyAPI } from '../api/endpoints.js';
import { showToast } from './Toast.js';

export function renderLibraryTable(container, songs, onRefreshNeeded) {
  if (!container) return;

  let currentSearch = '';
  let currentMoodFilter = 'all';

  function filterSongs() {
    return songs.filter(s => {
      const matchSearch = !currentSearch ||
        (s.title && s.title.toLowerCase().includes(currentSearch)) ||
        (s.artist && s.artist.toLowerCase().includes(currentSearch)) ||
        (s.original_name && s.original_name.toLowerCase().includes(currentSearch));

      const matchMood = currentMoodFilter === 'all' ||
        (s.matched_moods && s.matched_moods.includes(currentMoodFilter));

      return matchSearch && matchMood;
    });
  }

  function render() {
    const filtered = filterSongs();

    // Calculate metrics
    const totalTracks = songs.length;
    const avgBpm = totalTracks > 0
      ? (songs.reduce((acc, s) => acc + (s.tempo_bpm || 110), 0) / totalTracks).toFixed(0)
      : '--';

    container.innerHTML = `
      <!-- Telemetry Strip -->
      <div class="stats-strip" style="margin-bottom:24px;">
        <div class="stat-card">
          <span class="stat-label">Total Audio Ingested</span>
          <span class="stat-value">${totalTracks} <span style="font-size:12px; color:var(--color-text-muted);">tracks</span></span>
        </div>
        <div class="stat-card">
          <span class="stat-label">Library Avg Tempo</span>
          <span class="stat-value">${avgBpm} <span style="font-size:12px; color:var(--color-text-muted);">BPM</span></span>
        </div>
        <div class="stat-card">
          <span class="stat-label">Multimodal Embeddings</span>
          <span class="stat-value">64-D <span style="font-size:12px; color:var(--color-action);">cosine</span></span>
        </div>
        <div class="stat-card">
          <span class="stat-label">Audio Formats</span>
          <span class="stat-value" style="font-size:16px; margin-top:4px;">FLAC • MP3 • WAV • OGG</span>
        </div>
      </div>

      <!-- Library Glass Panel -->
      <div class="glass-panel">
        <div class="table-toolbar">
          <div style="display:flex; align-items:center; gap:12px; flex:1;">
            <div class="search-input-wrapper">
              <svg class="search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
              </svg>
              <input type="text" class="search-input" id="library-search" placeholder="Search track, artist, filename..." value="${currentSearch}" />
            </div>

            <select id="mood-filter-dropdown" class="search-input" style="width:140px; padding:7px 10px; font-size:12px;">
              <option value="all" ${currentMoodFilter === 'all' ? 'selected' : ''}>All Moods</option>
              <option value="chill" ${currentMoodFilter === 'chill' ? 'selected' : ''}>Chill</option>
              <option value="energetic" ${currentMoodFilter === 'energetic' ? 'selected' : ''}>Energetic</option>
              <option value="euphoric" ${currentMoodFilter === 'euphoric' ? 'selected' : ''}>Euphoric</option>
              <option value="melancholic" ${currentMoodFilter === 'melancholic' ? 'selected' : ''}>Melancholic</option>
              <option value="dark" ${currentMoodFilter === 'dark' ? 'selected' : ''}>Dark</option>
            </select>
          </div>

          <div style="font-size:12px; color:var(--color-text-muted);">
            Showing <strong style="color:var(--color-text-primary);">${filtered.length}</strong> of ${totalTracks}
          </div>
        </div>

        <!-- Table -->
        <div class="table-container">
          <table class="tracks-table">
            <thead>
              <tr>
                <th style="width:40%;">Track & Artist</th>
                <th>Format</th>
                <th>Tempo</th>
                <th>Key</th>
                <th>Mood Clusters</th>
                <th>Duration</th>
                <th style="text-align:right;">Actions</th>
              </tr>
            </thead>
            <tbody>
              ${filtered.length === 0 ? `
                <tr>
                  <td colspan="7" style="text-align:center; padding:36px; color:var(--color-text-muted);">
                    No tracks found matching filter criteria.
                  </td>
                </tr>
              ` : filtered.map(song => `
                <tr data-song-id="${song.id}">
                  <td>
                    <div class="track-main-cell">
                      <button class="track-play-btn" data-action="play" data-id="${song.id}" title="Play Track">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                      </button>
                      <div class="track-info">
                        <span class="track-title">${song.title || song.original_name}</span>
                        <span class="track-artist">${song.artist || 'Unknown Artist'}</span>
                      </div>
                    </div>
                  </td>
                  <td>
                    <span class="format-tag">${song.format || 'mp3'}</span>
                  </td>
                  <td>
                    <span class="mono-num" style="font-weight:600; color:var(--color-text-primary);">
                      ${song.tempo_bpm ? song.tempo_bpm.toFixed(0) : '--'}
                    </span>
                    <span style="font-size:10px; color:var(--color-text-muted);"> BPM</span>
                  </td>
                  <td>
                    <span style="font-size:12px; color:var(--color-text-secondary);">${song.key_signature || 'Unknown'}</span>
                  </td>
                  <td>
                    <div style="display:flex; gap:4px; flex-wrap:wrap;">
                      ${(song.matched_moods || ['neutral']).map(m => `
                        <span class="mood-badge ${m.toLowerCase()}">${m}</span>
                      `).join('')}
                    </div>
                  </td>
                  <td>
                    <span class="mono-num" style="font-size:12px; color:var(--color-text-secondary);">
                      ${formatDuration(song.duration_sec)}
                    </span>
                  </td>
                  <td style="text-align:right;">
                    <div style="display:inline-flex; align-items:center; gap:6px;">
                      <button class="btn btn-secondary btn-sm" data-action="inspect" data-id="${song.id}" title="Open Audio Telemetry & ID3 Studio">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
                        Studio
                      </button>
                      <button class="btn btn-secondary btn-sm" data-action="delete" data-id="${song.id}" title="Delete track" style="color:var(--color-error);">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                      </button>
                    </div>
                  </td>
                </tr>
              `).join('')}
            </tbody>
          </table>
        </div>
      </div>
    `;

    // Bind Search Input
    const searchEl = container.querySelector('#library-search');
    searchEl.addEventListener('input', (e) => {
      currentSearch = e.target.value.toLowerCase();
      render();
    });

    // Bind Mood Dropdown
    const moodEl = container.querySelector('#mood-filter-dropdown');
    moodEl.addEventListener('change', (e) => {
      currentMoodFilter = e.target.value;
      render();
    });

    // Bind Table Action Delegations
    container.querySelectorAll('[data-action="play"]').forEach(btn => {
      btn.addEventListener('click', () => {
        const id = btn.dataset.id;
        const song = songs.find(s => s.id === id);
        if (song) playTrack(song, filtered);
      });
    });

    container.querySelectorAll('[data-action="inspect"]').forEach(btn => {
      btn.addEventListener('click', () => {
        const id = btn.dataset.id;
        const song = songs.find(s => s.id === id);
        if (song) openTrackStudio(song, onRefreshNeeded);
      });
    });

    container.querySelectorAll('[data-action="delete"]').forEach(btn => {
      btn.addEventListener('click', async () => {
        const id = btn.dataset.id;
        const song = songs.find(s => s.id === id);
        if (!confirm(`Delete "${song?.title || song?.original_name}"?`)) return;

        try {
          await MoodifyAPI.deleteSong(id);
          showToast('Track deleted successfully', 'success');
          if (onRefreshNeeded) onRefreshNeeded();
        } catch (err) {
          showToast('Failed to delete track: ' + err.message, 'error');
        }
      });
    });
  }

  render();
}

function formatDuration(sec) {
  if (!sec) return '3:45';
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
}
