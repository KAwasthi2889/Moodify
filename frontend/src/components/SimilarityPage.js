/**
 * Moodify Similarity & Dynamic Playlist Studio Page
 * Displays seed song anchor, interactive similarity / suitability slider (0.50 - 0.95),
 * real-time 64-D vector matching, instant audio auditioning, and M3U8 playlist export.
 */

import { MoodifyAPI } from '../api/endpoints.js';
import { playTrack } from './AudioPlayer.js';

export async function renderSimilarityPage(container, seedSong, onNavigate) {
  if (!container || !seedSong) return;

  let currentThreshold = 0.65;
  let currentMode = 'multimodal'; // 'multimodal' | 'acoustic' | 'lyrics'
  let matchingSongs = [];
  let isLoading = false;
  let errorMsg = null;

  async function fetchSimilarSongs() {
    isLoading = true;
    errorMsg = null;
    render();

    try {
      const res = await MoodifyAPI.getSimilarSongs(seedSong.id, currentMode, currentThreshold, 100);
      const raw = (res && res.similar_songs) ? res.similar_songs : [];
      // Exclude seed song itself and identical duplicate title+artist matches
      const seedTitleNorm = (seedSong.title || seedSong.original_name || '').toLowerCase().trim();
      const seedArtistNorm = (seedSong.artist || '').toLowerCase().trim();

      matchingSongs = raw.filter(song => {
        if (song.id === seedSong.id) return false;
        const titleNorm = (song.title || song.original_name || '').toLowerCase().trim();
        const artistNorm = (song.artist || '').toLowerCase().trim();
        if (seedTitleNorm && titleNorm && seedTitleNorm === titleNorm && seedArtistNorm === artistNorm) {
          return false;
        }
        return true;
      });
    } catch (err) {
      console.error('Failed to query similar songs:', err);
      let msg = err.message || 'Failed to calculate nearest neighbor similarity';
      if (msg.includes('features or lyrics have not been analyzed')) {
        msg = 'This track has not been analyzed for 64-D vectors yet. Click "Moodify" in the workshop to analyze and extract mood vectors.';
      }
      errorMsg = msg;
      matchingSongs = [];
    } finally {
      isLoading = false;
      try {
        render();
      } catch (renderErr) {
        console.error('Error rendering similarity results:', renderErr);
      }
    }
  }

  function downloadM3U8() {
    if (matchingSongs.length === 0) return;

    let m3u8Content = '#EXTM3U\n';
    m3u8Content += `#PLAYLIST:Similar to ${seedSong.title || seedSong.original_name || 'Seed'}\n\n`;

    // Add seed song first
    const seedDuration = Math.round(seedSong.duration_sec || 180);
    const seedArtist = seedSong.artist || 'Unknown';
    const seedTitle = seedSong.title || seedSong.original_name || 'Seed Song';
    m3u8Content += `#EXTINF:${seedDuration},${seedArtist} - ${seedTitle}\n`;
    m3u8Content += `${window.location.origin}/api/v1/songs/${seedSong.id}/download\n\n`;

    // Add matched songs
    matchingSongs.forEach(song => {
      const duration = Math.round(song.duration_sec || 180);
      const artist = song.artist || 'Unknown';
      const title = song.title || song.filename || 'Untitled';
      m3u8Content += `#EXTINF:${duration},${artist} - ${title}\n`;
      m3u8Content += `${window.location.origin}/api/v1/songs/${song.id}/download\n`;
    });

    const blob = new Blob([m3u8Content], { type: 'application/x-mpegurl' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    const safeTitle = (seedSong.title || 'Moodify').replace(/[^a-zA-Z0-9_-]/g, '_');
    a.download = `${safeTitle}_similarity_playlist.m3u8`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  function render() {
    const seedTitle = seedSong.title || seedSong.original_name || seedSong.filename || 'Seed Track';
    const seedArtist = seedSong.artist || 'Unknown Artist';
    const seedAlbum = seedSong.album ? `• ${seedSong.album}` : '';
    const seedMood = seedSong.dominant_mood || (seedSong.matched_moods && seedSong.matched_moods[0]) || 'Neutral';
    const seedGenre = seedSong.inferred_genre || 'Universal';
    const seedBpm = seedSong.tempo_bpm || seedSong.tempo ? Math.round(seedSong.tempo_bpm || seedSong.tempo) : 120;
    const thresholdPercent = Math.round(currentThreshold * 100);

    container.innerHTML = `
      <div class="similarity-page">
        <!-- ── Navigation & Header ────────────────────────────── -->
        <div class="similarity-header">
          <button class="btn-back-nav" id="btn-back-workshop" title="Return to Uploaded Library">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="15 18 9 12 15 6"/></svg>
            <span>Back to Workshop</span>
          </button>
          <div class="similarity-title-group">
            <h1 class="similarity-title">
              Similarity <span class="gradient-text">Studio</span>
            </h1>
            <span class="similarity-subtitle">Real-time nearest-neighbor mood discovery in 64-D hyperspace</span>
          </div>
        </div>

        <!-- ── Selected Seed Song Anchor Banner ───────────────── -->
        <div class="glass-panel seed-anchor-card">
          <div class="seed-play-col">
            <button class="btn-seed-play" id="btn-seed-play-trigger" title="Play Anchor Track">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><polygon points="6 3 20 12 6 21 6 3"/></svg>
            </button>
          </div>
          <div class="seed-info-col">
            <div class="seed-eyebrow">Anchor Seed Song</div>
            <h2 class="seed-title">${escapeHtml(seedTitle)}</h2>
            <div class="seed-artist">${escapeHtml(seedArtist)} ${escapeHtml(seedAlbum)}</div>
          </div>
          <div class="seed-tags-col">
            <span class="pill-mood glow">${escapeHtml(seedMood)}</span>
            <span class="pill-genre">${escapeHtml(seedGenre)}</span>
            <span class="pill-bpm mono-num">${seedBpm} BPM</span>
          </div>
        </div>

        <!-- ── Similarity Controls & Threshold Slider ─────────── -->
        <div class="glass-panel similarity-controls-panel">
          <div class="controls-top-row">
            <div class="slider-group">
              <div class="slider-header-row">
                <label for="similarity-slider" class="slider-label">
                  Suitability Threshold
                </label>
                <div class="slider-stats-badges">
                  <span class="slider-match-counter mono-num" id="slider-match-counter">
                    Showing ${matchingSongs.length} matches ≥ ${thresholdPercent}%
                  </span>
                  <span class="slider-value-pill mono-num" id="slider-value-display">
                    ≥ ${thresholdPercent}% Match
                  </span>
                </div>
              </div>
              <input 
                type="range" 
                id="similarity-slider" 
                class="mood-slider" 
                min="0.50" 
                max="0.95" 
                step="0.05" 
                value="${currentThreshold}" 
              />
              <div class="slider-hints-row">
                <span>0.50 (Broad discovery)</span>
                <span>0.70 (Harmonic flow)</span>
                <span>0.95 (Near identical)</span>
              </div>
            </div>

            <div class="mode-selector-group">
              <span class="mode-selector-label">Vector Space Mode</span>
              <div class="mode-button-pill-group">
                <button class="btn-mode-pill ${currentMode === 'multimodal' ? 'active' : ''}" data-mode="multimodal">
                  64-D Multimodal
                </button>
                <button class="btn-mode-pill ${currentMode === 'acoustic' ? 'active' : ''}" data-mode="acoustic">
                  36-D DSP Acoustic
                </button>
                <button class="btn-mode-pill ${currentMode === 'lyrics' ? 'active' : ''}" data-mode="lyrics">
                  28-D Lyrics
                </button>
              </div>
            </div>
          </div>

          <div class="controls-bottom-row">
            <div class="match-count-indicator">
              ${isLoading ? `
                <span class="spinner-sm"></span> <span>Searching 64-D vector space...</span>
              ` : `
                <span>Found <strong class="color-action mono-num">${matchingSongs.length}</strong> matching songs with ≥ ${thresholdPercent}% suitability</span>
              `}
            </div>
            
            <button class="btn-hero-primary btn-export-m3u8" id="btn-export-playlist" ${matchingSongs.length === 0 || isLoading ? 'disabled' : ''}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
              <span>Export .M3U8 Playlist</span>
            </button>
          </div>
        </div>

        <!-- ── Matching Tracks Results Table ──────────────────── -->
        <div class="glass-panel similarity-results-panel">
          ${errorMsg ? `
            <div class="error-banner">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
              <span>${escapeHtml(errorMsg)}</span>
            </div>
          ` : isLoading ? `
            <div class="similarity-loading-state">
              <div class="spinner-large"></div>
              <p>Traversing pgvector HNSW cosine graph...</p>
            </div>
          ` : matchingSongs.length > 0 ? `
            <div class="tracks-table-container">
              <table class="workshop-table">
                <thead>
                  <tr>
                    <th style="width: 40px;">#</th>
                    <th style="width: 44px;">Play</th>
                    <th>Track Title & Artist</th>
                    <th>Genre & Mood</th>
                    <th>BPM</th>
                    <th>Suitability Score</th>
                    <th style="text-align: right;">Download</th>
                  </tr>
                </thead>
                <tbody>
                  ${matchingSongs.map((song, index) => {
                    const title = song.title || song.original_name || song.filename || 'Untitled';
                    const artist = song.artist || 'Unknown Artist';
                    const album = song.album ? `• ${song.album}` : '';
                    const genre = song.inferred_genre || 'Universal';
                    const mood = (song.matched_moods && song.matched_moods[0]) || song.dominant_mood || 'Neutral';
                    const bpm = song.tempo_bpm || song.tempo ? Math.round(song.tempo_bpm || song.tempo) : 120;
                    const rawScore = typeof song.similarity_score === 'number' ? song.similarity_score : (typeof song.similarity === 'number' ? song.similarity : 0);
                    const simScore = Math.round(rawScore * 100);
                    const cleanDlName = MoodifyAPI.getCleanFilename(song);

                    let scoreColorClass = 'score-high';
                    if (simScore < 70) scoreColorClass = 'score-med';
                    if (simScore < 60) scoreColorClass = 'score-low';

                    return `
                      <tr class="track-row" data-id="${escapeHtml(song.id)}">
                        <td class="cell-index mono-num">${index + 1}</td>
                        <td class="cell-play">
                          <button class="btn-row-play" data-action="play-match" data-id="${escapeHtml(song.id)}" title="Audition Track">
                            <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="6 3 20 12 6 21 6 3"/></svg>
                          </button>
                        </td>
                        <td class="cell-main">
                          <div class="track-primary-title">${escapeHtml(title)}</div>
                          <div class="track-secondary-artist">${escapeHtml(artist)} ${escapeHtml(album)}</div>
                        </td>
                        <td class="cell-genre-mood">
                          <span class="genre-tag">${escapeHtml(genre)}</span>
                          <span class="mood-tag">${escapeHtml(mood)}</span>
                        </td>
                        <td class="cell-bpm mono-num">${bpm} BPM</td>
                        <td class="cell-suitability">
                          <div class="suitability-pill ${scoreColorClass} mono-num">
                            <span class="suitability-bar" style="width: ${simScore}%;"></span>
                            <span class="suitability-text">${simScore}% Match</span>
                          </div>
                        </td>
                        <td class="cell-actions" style="text-align: right;">
                          <a class="btn-row-download" href="/api/v1/songs/${escapeHtml(song.id)}/download?download=true" download="${escapeHtml(cleanDlName)}" title="Download: ${escapeHtml(cleanDlName)}">
                            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                            <span>Download</span>
                          </a>
                        </td>
                      </tr>
                    `;
                  }).join('')}
                </tbody>
              </table>
            </div>
          ` : `
            <div class="similarity-empty-state">
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M8 12h8"/><path d="M12 8v8"/></svg>
              <h3>No Matching Songs Found</h3>
              <p>Try lowering the suitability threshold slider (e.g. to 0.50 or 0.60) or switching vector modes above.</p>
            </div>
          `}
        </div>
      </div>
    `;

    bindEvents();
  }

  function bindEvents() {
    // Back navigation to Workshop
    const backBtn = container.querySelector('#btn-back-workshop');
    backBtn?.addEventListener('click', () => {
      if (onNavigate) onNavigate('workshop');
    });

    // Seed play trigger
    const seedPlayBtn = container.querySelector('#btn-seed-play-trigger');
    seedPlayBtn?.addEventListener('click', () => {
      playTrack(seedSong);
    });

    // Slider interaction with real-time fetch
    const slider = container.querySelector('#similarity-slider');
    slider?.addEventListener('input', (e) => {
      currentThreshold = parseFloat(e.target.value);
      const display = container.querySelector('#slider-value-display');
      if (display) {
        display.textContent = `≥ ${Math.round(currentThreshold * 100)}% Match`;
      }
    });

    slider?.addEventListener('change', (e) => {
      currentThreshold = parseFloat(e.target.value);
      fetchSimilarSongs();
    });

    // Mode button toggles
    const modeBtns = container.querySelectorAll('.btn-mode-pill');
    modeBtns.forEach(btn => {
      btn.addEventListener('click', () => {
        const selectedMode = btn.dataset.mode;
        if (selectedMode && selectedMode !== currentMode) {
          currentMode = selectedMode;
          fetchSimilarSongs();
        }
      });
    });

    // Export playlist button
    const exportBtn = container.querySelector('#btn-export-playlist');
    exportBtn?.addEventListener('click', downloadM3U8);

    // Matching songs play buttons
    const playBtns = container.querySelectorAll('button[data-action="play-match"]');
    playBtns.forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const id = btn.dataset.id;
        const song = matchingSongs.find(s => s.id === id);
        if (song) playTrack(song, [seedSong, ...matchingSongs]);
      });
    });

    // Similar track row body click-to-play
    container.querySelectorAll('.similarity-results-panel .track-row').forEach(row => {
      row.style.cursor = 'pointer';
      row.addEventListener('click', (e) => {
        if (e.target.closest('button, a, input')) return;
        const id = row.dataset.id;
        const song = matchingSongs.find(s => s.id === id);
        if (song) playTrack(song, [seedSong, ...matchingSongs]);
      });
    });
  }

  // Initial fetch
  fetchSimilarSongs();
}

function escapeHtml(str) {
  if (!str) return '';
  return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
