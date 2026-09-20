/**
 * Moodify Track Studio Modal Component
 * Deep-dive inspector for audio features, metadata tagging, AcoustID identification,
 * and synchronized lyrics with 28-D emotion radar.
 */

import { MoodifyAPI } from '../api/endpoints.js';
import { executeOptimisticAction, showToast } from './Toast.js';

let modalContainer = null;
let activeSong = null;

export function initTrackStudioModal() {
  modalContainer = document.createElement('div');
  modalContainer.className = 'modal-backdrop';
  modalContainer.id = 'track-studio-modal';

  modalContainer.addEventListener('click', (e) => {
    if (e.target === modalContainer) closeModal();
  });

  document.body.appendChild(modalContainer);
}

export async function openTrackStudio(song, onUpdateCallback) {
  activeSong = song;
  renderModalContent(onUpdateCallback);
  modalContainer.classList.add('open');

  // Load latest lyrics and features
  try {
    const [featuresRes, lyricsRes] = await Promise.all([
      MoodifyAPI.getSongFeatures(song.id),
      MoodifyAPI.getSongLyrics(song.id)
    ]);
    if (featuresRes?.features) {
      activeSong = { ...activeSong, ...featuresRes.features };
      updateFeatureElements();
    }
    if (lyricsRes?.lyrics) {
      updateLyricsElements(lyricsRes.lyrics);
    }
  } catch (e) {
    console.warn('Could not fetch extra song telemetry', e);
  }
}

export function closeModal() {
  if (modalContainer) {
    modalContainer.classList.remove('open');
  }
}

function renderModalContent(onUpdateCallback) {
  const song = activeSong;
  modalContainer.innerHTML = `
    <div class="modal-dialog">
      <!-- Header -->
      <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:20px;">
        <div style="display:flex; align-items:center; gap:12px;">
          <div style="width:40px; height:40px; border-radius:var(--radius-sm); background:linear-gradient(135deg, #00f0ff, #8b5cf6); display:flex; align-items:center; justify-content:center; color:#040814; font-weight:800;">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
          </div>
          <div>
            <h2 class="panel-title" style="font-size:20px;">${song.title || song.original_name}</h2>
            <p class="panel-subtitle">${song.artist || 'Unknown Artist'} • <span class="format-tag">${song.format}</span> • ${song.filename}</p>
          </div>
        </div>
        <button class="modal-close-btn" id="modal-close-btn">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>

      <!-- Audio Telemetry & Feature Dials -->
      <div style="margin-bottom:24px;">
        <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:8px;">
          <span style="font-size:12px; font-weight:600; text-transform:uppercase; letter-spacing:0.06em; color:var(--color-text-muted);">Acoustic Intelligence Telemetry</span>
          <button class="btn btn-secondary btn-sm" id="btn-reanalyze">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/></svg>
            Re-Analyze Features
          </button>
        </div>
        <div class="feature-metric-grid" id="feature-grid">
          <div class="feature-pill">
            <span class="feature-pill-name">Tempo</span>
            <span class="feature-pill-value">${song.tempo_bpm ? song.tempo_bpm.toFixed(1) : '--'} <span style="font-size:11px; color:var(--color-text-muted);">BPM</span></span>
            <div class="feature-pill-bar"><div class="feature-pill-bar-fill" style="width:${Math.min(100, ((song.tempo_bpm || 100) / 180) * 100)}%;"></div></div>
          </div>
          <div class="feature-pill">
            <span class="feature-pill-name">Key</span>
            <span class="feature-pill-value">${song.key_signature || 'Unknown'}</span>
            <div class="feature-pill-bar"><div class="feature-pill-bar-fill" style="width:75%; background:#8b5cf6;"></div></div>
          </div>
          <div class="feature-pill">
            <span class="feature-pill-name">Energy</span>
            <span class="feature-pill-value">${song.energy ? (song.energy * 100).toFixed(0) : '75'}%</span>
            <div class="feature-pill-bar"><div class="feature-pill-bar-fill" style="width:${(song.energy || 0.75) * 100}%; background:var(--mood-energetic);"></div></div>
          </div>
          <div class="feature-pill">
            <span class="feature-pill-name">Valence</span>
            <span class="feature-pill-value">${song.valence ? (song.valence * 100).toFixed(0) : '60'}%</span>
            <div class="feature-pill-bar"><div class="feature-pill-bar-fill" style="width:${(song.valence || 0.6) * 100}%; background:var(--mood-chill);"></div></div>
          </div>
          <div class="feature-pill">
            <span class="feature-pill-name">Danceability</span>
            <span class="feature-pill-value">${song.danceability ? (song.danceability * 100).toFixed(0) : '68'}%</span>
            <div class="feature-pill-bar"><div class="feature-pill-bar-fill" style="width:${(song.danceability || 0.68) * 100}%; background:var(--mood-euphoric);"></div></div>
          </div>
        </div>
      </div>

      <!-- Metadata Editing & Studio Operations -->
      <div style="display:grid; grid-template-columns:1fr 1fr; gap:20px; margin-bottom:24px;">
        <!-- Left: Metadata Form -->
        <div style="background:rgba(255,255,255,0.02); border:1px solid var(--color-surface-border); border-radius:var(--radius-md); padding:18px;">
          <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:14px;">
            <span style="font-weight:600; font-size:13px; color:var(--color-text-primary);">Metadata Tags</span>
            <button class="btn btn-secondary btn-sm" id="btn-identify" title="Auto-detect tags using AcoustID & MusicBrainz">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
              AcoustID Match
            </button>
          </div>
          <div style="display:flex; flex-direction:column; gap:10px;">
            <div>
              <label style="font-size:11px; color:var(--color-text-muted); display:block; margin-bottom:4px;">Title</label>
              <input type="text" id="meta-title" value="${song.title || ''}" class="search-input" style="padding-left:12px;" />
            </div>
            <div style="display:grid; grid-template-columns:1fr 1fr; gap:10px;">
              <div>
                <label style="font-size:11px; color:var(--color-text-muted); display:block; margin-bottom:4px;">Artist</label>
                <input type="text" id="meta-artist" value="${song.artist || ''}" class="search-input" style="padding-left:12px;" />
              </div>
              <div>
                <label style="font-size:11px; color:var(--color-text-muted); display:block; margin-bottom:4px;">Album</label>
                <input type="text" id="meta-album" value="${song.album || ''}" class="search-input" style="padding-left:12px;" />
              </div>
            </div>
            <div>
              <label style="font-size:11px; color:var(--color-text-muted); display:block; margin-bottom:4px;">Inferred Genre</label>
              <input type="text" id="meta-genre" value="${song.inferred_genre || ''}" class="search-input" style="padding-left:12px;" />
            </div>
          </div>
          <div style="display:flex; align-items:center; gap:8px; margin-top:16px;">
            <button class="btn btn-primary btn-sm" id="btn-save-meta">Save Tags</button>
            <button class="btn btn-secondary btn-sm" id="btn-embed-tags" title="Embed tags into physical ID3 headers">Embed to Audio</button>
            <button class="btn btn-secondary btn-sm" id="btn-rename-file" title="Rename file to Artist - Title format">Rename File</button>
          </div>
        </div>

        <!-- Right: Lyrics & RoBERTa Emotion Embedding -->
        <div style="background:rgba(255,255,255,0.02); border:1px solid var(--color-surface-border); border-radius:var(--radius-md); padding:18px; display:flex; flex-direction:column;">
          <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:14px;">
            <span style="font-weight:600; font-size:13px; color:var(--color-text-primary);">Synchronized Lyrics</span>
            <button class="btn btn-secondary btn-sm" id="btn-sync-lyrics">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z"/><line x1="4" y1="22" x2="4" y2="15"/></svg>
              LRCLIB Sync
            </button>
          </div>
          <div class="lyrics-box" id="lyrics-container">
            <p style="color:var(--color-text-muted); font-size:13px; margin-top:40px;">Fetching synchronized lyrics & emotion vector...</p>
          </div>
          <div id="emotion-radar" style="margin-top:12px; display:flex; gap:6px; flex-wrap:wrap;"></div>
        </div>
      </div>
    </div>
  `;

  // Bind close button
  document.getElementById('modal-close-btn').addEventListener('click', closeModal);

  // Bind Metadata Save
  document.getElementById('btn-save-meta').addEventListener('click', async (e) => {
    const updatedData = {
      title: document.getElementById('meta-title').value,
      artist: document.getElementById('meta-artist').value,
      album: document.getElementById('meta-album').value,
      inferred_genre: document.getElementById('meta-genre').value,
    };
    await executeOptimisticAction({
      button: e.currentTarget,
      apiCall: () => MoodifyAPI.saveMetadata(song.id, updatedData),
      onOptimisticApply: () => {
        Object.assign(activeSong, updatedData);
        if (onUpdateCallback) onUpdateCallback(activeSong);
      },
      successMessage: 'Metadata updated successfully'
    });
  });

  // Bind AcoustID Match
  document.getElementById('btn-identify').addEventListener('click', async (e) => {
    await executeOptimisticAction({
      button: e.currentTarget,
      apiCall: async () => {
        const res = await MoodifyAPI.identifySong(song.id);
        if (res?.metadata) {
          document.getElementById('meta-title').value = res.metadata.title;
          document.getElementById('meta-artist').value = res.metadata.artist;
          document.getElementById('meta-album').value = res.metadata.album;
          document.getElementById('meta-genre').value = res.metadata.genre;
          Object.assign(activeSong, res.metadata);
          if (onUpdateCallback) onUpdateCallback(activeSong);
        }
      },
      successMessage: 'AcoustID matched track fingerprint!'
    });
  });

  // Bind Embed ID3
  document.getElementById('btn-embed-tags').addEventListener('click', async (e) => {
    await executeOptimisticAction({
      button: e.currentTarget,
      apiCall: () => MoodifyAPI.embedTags(song.id),
      successMessage: 'Tags embedded into audio stream'
    });
  });

  // Bind Rename File
  document.getElementById('btn-rename-file').addEventListener('click', async (e) => {
    await executeOptimisticAction({
      button: e.currentTarget,
      apiCall: () => MoodifyAPI.renameSong(song.id),
      successMessage: 'File renamed on disk'
    });
  });

  // Bind Re-Analyze
  document.getElementById('btn-reanalyze').addEventListener('click', async (e) => {
    await executeOptimisticAction({
      button: e.currentTarget,
      apiCall: async () => {
        const res = await MoodifyAPI.analyzeSong(song.id);
        if (res?.features) {
          Object.assign(activeSong, res.features);
          updateFeatureElements();
          if (onUpdateCallback) onUpdateCallback(activeSong);
        }
      },
      successMessage: 'Audio analysis completed'
    });
  });

  // Bind Lyrics Sync
  document.getElementById('btn-sync-lyrics').addEventListener('click', async (e) => {
    await executeOptimisticAction({
      button: e.currentTarget,
      apiCall: async () => {
        const res = await MoodifyAPI.syncLyrics(song.id, { track: song.title, artist: song.artist });
        if (res?.lyrics) updateLyricsElements(res.lyrics);
      },
      successMessage: 'Lyrics synchronized via LRCLIB'
    });
  });
}

function updateFeatureElements() {
  const s = activeSong;
  const grid = document.getElementById('feature-grid');
  if (!grid) return;
  grid.innerHTML = `
    <div class="feature-pill">
      <span class="feature-pill-name">Tempo</span>
      <span class="feature-pill-value">${s.tempo_bpm ? s.tempo_bpm.toFixed(1) : '--'} <span style="font-size:11px; color:var(--color-text-muted);">BPM</span></span>
      <div class="feature-pill-bar"><div class="feature-pill-bar-fill" style="width:${Math.min(100, ((s.tempo_bpm || 100) / 180) * 100)}%;"></div></div>
    </div>
    <div class="feature-pill">
      <span class="feature-pill-name">Key</span>
      <span class="feature-pill-value">${s.key_signature || 'Unknown'}</span>
      <div class="feature-pill-bar"><div class="feature-pill-bar-fill" style="width:75%; background:#8b5cf6;"></div></div>
    </div>
    <div class="feature-pill">
      <span class="feature-pill-name">Energy</span>
      <span class="feature-pill-value">${s.energy ? (s.energy * 100).toFixed(0) : '75'}%</span>
      <div class="feature-pill-bar"><div class="feature-pill-bar-fill" style="width:${(s.energy || 0.75) * 100}%; background:var(--mood-energetic);"></div></div>
    </div>
    <div class="feature-pill">
      <span class="feature-pill-name">Valence</span>
      <span class="feature-pill-value">${s.valence ? (s.valence * 100).toFixed(0) : '60'}%</span>
      <div class="feature-pill-bar"><div class="feature-pill-bar-fill" style="width:${(s.valence || 0.6) * 100}%; background:var(--mood-chill);"></div></div>
    </div>
    <div class="feature-pill">
      <span class="feature-pill-name">Danceability</span>
      <span class="feature-pill-value">${s.danceability ? (s.danceability * 100).toFixed(0) : '68'}%</span>
      <div class="feature-pill-bar"><div class="feature-pill-bar-fill" style="width:${(s.danceability || 0.68) * 100}%; background:var(--mood-euphoric);"></div></div>
    </div>
  `;
}

function updateLyricsElements(lyricsData) {
  const container = document.getElementById('lyrics-container');
  if (!container) return;

  if (lyricsData.lines && lyricsData.lines.length > 0) {
    container.innerHTML = lyricsData.lines
      .map(line => `<div class="lyric-line">${line.text}</div>`)
      .join('');
  } else if (lyricsData.plain) {
    container.innerHTML = lyricsData.plain
      .split('\n')
      .map(line => `<div class="lyric-line">${line}</div>`)
      .join('');
  }

  const radar = document.getElementById('emotion-radar');
  if (radar && lyricsData.emotions) {
    radar.innerHTML = Object.entries(lyricsData.emotions)
      .map(([emotion, val]) => `
        <span class="mood-badge neutral" style="font-size:10px;">
          ${emotion} ${(val * 100).toFixed(0)}%
        </span>
      `).join('');
  }
}
