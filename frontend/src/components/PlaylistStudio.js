/**
 * Moodify Curated Playlist Generator Studio
 * Generates BPM-sequenced playlists from multimodal seed songs, moods, or genres,
 * with instant preview and M3U8 export.
 */

import { MoodifyAPI } from '../api/endpoints.js';
import { playTrack } from './AudioPlayer.js';
import { showToast } from './Toast.js';

let generatedTracks = [];

export function renderPlaylistStudio(container, songs) {
  if (!container) return;

  container.innerHTML = `
    <div class="glass-panel" style="margin-bottom:24px;">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" style="color:var(--color-action);">
              <polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/>
            </svg>
            Curated Playlist Generator
          </h2>
          <p class="panel-subtitle">Harmonic BPM transitions & multimodal semantic flow sequencing</p>
        </div>
      </div>

      <!-- Controls Row -->
      <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(200px, 1fr)); gap:14px; margin-bottom:20px; align-items:end;">
        <div>
          <label style="font-size:11px; color:var(--color-text-muted); display:block; margin-bottom:5px;">Curation Strategy</label>
          <select id="playlist-curation-mode" class="search-input" style="width:100%;">
            <option value="seed">Seed Song Multimodal (64-D Flow)</option>
            <option value="mood">Mood Category</option>
            <option value="genre">Genre Mix</option>
          </select>
        </div>

        <div id="seed-select-wrap">
          <label style="font-size:11px; color:var(--color-text-muted); display:block; margin-bottom:5px;">Seed Anchor Song</label>
          <select id="playlist-seed-select" class="search-input" style="width:100%;">
            ${songs.map(s => `<option value="${s.id}">${s.title || s.original_name}</option>`).join('')}
          </select>
        </div>

        <div id="mood-select-wrap" style="display:none;">
          <label style="font-size:11px; color:var(--color-text-muted); display:block; margin-bottom:5px;">Target Mood</label>
          <select id="playlist-mood-select" class="search-input" style="width:100%;">
            <option value="chill">Chill & Ambient</option>
            <option value="energetic">High Voltage & Drive</option>
            <option value="euphoric">Euphoric & Uplifting</option>
            <option value="melancholic">Nocturne & Melancholy</option>
          </select>
        </div>

        <div>
          <label style="font-size:11px; color:var(--color-text-muted); display:block; margin-bottom:5px;">Track Count</label>
          <input type="number" id="playlist-limit-input" value="10" min="3" max="50" class="search-input" style="width:100%;" />
        </div>

        <div>
          <button class="btn btn-primary" id="btn-generate-playlist" style="width:100%;">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
            Generate Set
          </button>
        </div>
      </div>

      <!-- Generated Playlist Result Table -->
      <div id="playlist-results-box" style="background:rgba(0,0,0,0.25); border:1px solid var(--color-surface-border); border-radius:var(--radius-md); padding:16px;">
        <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:12px;">
          <span style="font-size:13px; font-weight:600; color:var(--color-text-primary);" id="playlist-output-title">Playlist Preview</span>
          <div style="display:flex; gap:8px;">
            <button class="btn btn-secondary btn-sm" id="btn-export-m3u8" disabled>
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
              Export M3U8
            </button>
            <button class="btn btn-secondary btn-sm" id="btn-export-json" disabled>
              Export JSON
            </button>
          </div>
        </div>

        <div id="playlist-track-list">
          <p style="font-size:12px; color:var(--color-text-muted); text-align:center; padding:24px 0;">Configure parameters above and click "Generate Set".</p>
        </div>
      </div>
    </div>
  `;

  // Toggle dropdowns based on mode
  const modeSelect = container.querySelector('#playlist-curation-mode');
  const seedWrap = container.querySelector('#seed-select-wrap');
  const moodWrap = container.querySelector('#mood-select-wrap');

  modeSelect.addEventListener('change', () => {
    if (modeSelect.value === 'seed') {
      seedWrap.style.display = 'block';
      moodWrap.style.display = 'none';
    } else if (modeSelect.value === 'mood') {
      seedWrap.style.display = 'none';
      moodWrap.style.display = 'block';
    } else {
      seedWrap.style.display = 'none';
      moodWrap.style.display = 'none';
    }
  });

  // Bind Generate Button
  const genBtn = container.querySelector('#btn-generate-playlist');
  genBtn.addEventListener('click', async () => {
    genBtn.disabled = true;
    genBtn.innerHTML = `<span class="spinner"></span> Curating flow...`;

    const limit = parseInt(container.querySelector('#playlist-limit-input').value, 10) || 10;
    const mode = modeSelect.value;
    const payload = { limit };

    if (mode === 'seed') {
      payload.seed_song_id = container.querySelector('#playlist-seed-select').value;
    } else if (mode === 'mood') {
      payload.mood = container.querySelector('#playlist-mood-select').value;
    }

    try {
      const res = await MoodifyAPI.generatePlaylist(payload);
      generatedTracks = res.tracks || [];
      const titleSpan = container.querySelector('#playlist-output-title');
      titleSpan.textContent = `${res.title || 'Moodify Curated Mix'} (${generatedTracks.length} tracks)`;

      const listContainer = container.querySelector('#playlist-track-list');
      if (generatedTracks.length === 0) {
        listContainer.innerHTML = `<p style="font-size:12px; color:var(--color-text-muted); text-align:center; padding:20px 0;">No tracks matched criteria.</p>`;
      } else {
        listContainer.innerHTML = `
          <div style="display:flex; flex-direction:column; gap:8px;">
            ${generatedTracks.map((t, i) => `
              <div style="display:flex; align-items:center; justify-content:space-between; padding:10px 14px; background:rgba(255,255,255,0.02); border-radius:var(--radius-sm); border:1px solid rgba(255,255,255,0.04);">
                <div style="display:flex; align-items:center; gap:12px;">
                  <span class="mono-num" style="font-size:11px; color:var(--color-text-muted); width:20px;">#${i + 1}</span>
                  <button class="track-play-btn" style="width:28px; height:28px;" data-pl-play="${t.id}" title="Play Track">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                  </button>
                  <div>
                    <div style="font-size:13px; font-weight:600; color:var(--color-text-primary);">${t.title}</div>
                    <div style="font-size:11px; color:var(--color-text-secondary);">${t.artist || 'Unknown'} • ${t.inferred_genre || 'Vibe'}</div>
                  </div>
                </div>
                <div style="display:flex; align-items:center; gap:16px;">
                  <span class="mono-num" style="font-size:12px; color:var(--color-action);">${t.tempo_bpm ? t.tempo_bpm.toFixed(0) : '--'} BPM</span>
                  <span class="mono-num" style="font-size:11px; color:var(--color-text-muted);">${formatSecs(t.duration_sec)}</span>
                </div>
              </div>
            `).join('')}
          </div>
        `;

        // Bind Play
        listContainer.querySelectorAll('[data-pl-play]').forEach(btn => {
          btn.addEventListener('click', () => {
            const tid = btn.dataset.plPlay;
            const trk = generatedTracks.find(t => t.id === tid);
            if (trk) playTrack(trk, generatedTracks);
          });
        });

        container.querySelector('#btn-export-m3u8').disabled = false;
        container.querySelector('#btn-export-json').disabled = false;
        showToast(`Playlist generated with ${generatedTracks.length} tracks`, 'success');
      }
    } catch (err) {
      showToast('Failed to generate playlist: ' + err.message, 'error');
    } finally {
      genBtn.disabled = false;
      genBtn.innerHTML = `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg> Generate Set`;
    }
  });

  // Bind Exports
  container.querySelector('#btn-export-m3u8').addEventListener('click', () => {
    if (generatedTracks.length === 0) return;
    let m3u = '#EXTM3U\n#PLAYLIST:Moodify Curated Flow\n';
    generatedTracks.forEach(t => {
      m3u += `#EXTINF:${Math.round(t.duration_sec || 180)},${t.artist} - ${t.title}\n`;
      m3u += `${window.location.origin}/api/v1/songs/${t.id}/download\n`;
    });
    downloadBlob(m3u, 'moodify_playlist.m3u8', 'audio/x-mpegurl');
    showToast('Exported M3U8 Playlist file', 'success');
  });

  container.querySelector('#btn-export-json').addEventListener('click', () => {
    if (generatedTracks.length === 0) return;
    const jsonStr = JSON.stringify({ tracks: generatedTracks }, null, 2);
    downloadBlob(jsonStr, 'moodify_playlist.json', 'application/json');
    showToast('Exported JSON Playlist', 'success');
  });
}

function downloadBlob(content, filename, type) {
  const blob = new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function formatSecs(sec) {
  if (!sec) return '3:00';
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
}
