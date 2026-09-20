/**
 * Moodify Mood Galaxy & Similarity Radar Component
 * Explores 4D mood clusters and multimodal nearest neighbors.
 */

import { MoodifyAPI } from '../api/endpoints.js';
import { playTrack } from './AudioPlayer.js';
import { showToast } from './Toast.js';

export function renderMoodGalaxy(container, songs, onSelectMoodFilter) {
  if (!container) return;

  container.innerHTML = `
    <div class="glass-panel" style="margin-bottom:24px;">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" style="color:var(--color-action);">
              <circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/>
            </svg>
            Mood Clusters & Audio Galaxy
          </h2>
          <p class="panel-subtitle">Multimodal classification mapping acoustic features & RoBERTa lyric sentiment</p>
        </div>
      </div>

      <!-- Cluster Bento Cards -->
      <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(240px, 1fr)); gap:16px; margin-bottom:24px;" id="clusters-grid">
        <div class="stat-card" style="border-left:3px solid var(--mood-chill); cursor:pointer;" data-mood="chill">
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <span class="mood-badge chill">Chill & Ambient</span>
            <span class="mono-num" style="font-size:12px; color:var(--color-text-muted);">~88 BPM</span>
          </div>
          <span style="font-size:13px; color:var(--color-text-secondary); margin-top:6px;">Tranquil soundscapes, harmonic pads, and soft acoustic percussion.</span>
          <div style="margin-top:10px; font-size:11px; color:var(--color-action); font-weight:600;">Filter Library →</div>
        </div>

        <div class="stat-card" style="border-left:3px solid var(--mood-energetic); cursor:pointer;" data-mood="energetic">
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <span class="mood-badge energetic">High Voltage & Drive</span>
            <span class="mono-num" style="font-size:12px; color:var(--color-text-muted);">~125 BPM</span>
          </div>
          <span style="font-size:13px; color:var(--color-text-secondary); margin-top:6px;">High kinetic momentum, punchy transients, and driving rhythm.</span>
          <div style="margin-top:10px; font-size:11px; color:var(--color-action); font-weight:600;">Filter Library →</div>
        </div>

        <div class="stat-card" style="border-left:3px solid var(--mood-euphoric); cursor:pointer;" data-mood="euphoric">
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <span class="mood-badge euphoric">Euphoric & Uplifting</span>
            <span class="mono-num" style="font-size:12px; color:var(--color-text-muted);">~118 BPM</span>
          </div>
          <span style="font-size:13px; color:var(--color-text-secondary); margin-top:6px;">Bright harmonic resolution, expansive leads, and peak emotional release.</span>
          <div style="margin-top:10px; font-size:11px; color:var(--color-action); font-weight:600;">Filter Library →</div>
        </div>

        <div class="stat-card" style="border-left:3px solid var(--mood-melancholic); cursor:pointer;" data-mood="melancholic">
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <span class="mood-badge melancholic">Nocturne & Melancholy</span>
            <span class="mono-num" style="font-size:12px; color:var(--color-text-muted);">~76 BPM</span>
          </div>
          <span style="font-size:13px; color:var(--color-text-secondary); margin-top:6px;">Introspective minor tonalities, atmospheric reverbs, and deep vocal timbre.</span>
          <div style="margin-top:10px; font-size:11px; color:var(--color-action); font-weight:600;">Filter Library →</div>
        </div>
      </div>

      <!-- Nearest Neighbor Similarity Radar Tool -->
      <div style="background:rgba(0,0,0,0.25); border:1px solid var(--color-surface-border); border-radius:var(--radius-md); padding:18px;">
        <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:14px; flex-wrap:wrap; gap:10px;">
          <div>
            <h3 style="font-size:14px; font-weight:600; color:var(--color-text-primary);">Multimodal Nearest Neighbors Radar</h3>
            <span style="font-size:11px; color:var(--color-text-muted);">Query 64-D cosine similarity to find acoustically and lyrically cohesive tracks</span>
          </div>
          <div style="display:flex; align-items:center; gap:8px;">
            <select id="similarity-mode-select" class="search-input" style="width:140px; padding:6px 10px; font-size:12px;">
              <option value="multimodal">Multimodal (64-D)</option>
              <option value="acoustic">Acoustic Only (36-D)</option>
              <option value="lyrics">Lyrics Only (28-D)</option>
            </select>
            <select id="similarity-seed-select" class="search-input" style="width:200px; padding:6px 10px; font-size:12px;">
              ${songs.map(s => `<option value="${s.id}">${s.title || s.original_name}</option>`).join('')}
            </select>
            <button class="btn btn-primary btn-sm" id="btn-run-similarity">Find Neighbors</button>
          </div>
        </div>

        <div id="similarity-results" style="display:grid; grid-template-columns:repeat(auto-fill, minmax(280px, 1fr)); gap:12px;">
          <p style="font-size:12px; color:var(--color-text-muted); grid-column: 1/-1;">Click "Find Neighbors" to calculate real-time vector matches.</p>
        </div>
      </div>
    </div>
  `;

  // Bind Cluster click filters
  const clusterCards = container.querySelectorAll('#clusters-grid .stat-card');
  clusterCards.forEach(card => {
    card.addEventListener('click', () => {
      const mood = card.dataset.mood;
      if (onSelectMoodFilter) onSelectMoodFilter(mood);
      showToast(`Filtered library by ${mood} vibe`, 'success');
    });
  });

  // Bind Similarity Finder
  const runSimBtn = container.querySelector('#btn-run-similarity');
  if (runSimBtn) {
    runSimBtn.addEventListener('click', async () => {
      const seedId = container.querySelector('#similarity-seed-select').value;
      const mode = container.querySelector('#similarity-mode-select').value;
      if (!seedId) return;

      runSimBtn.disabled = true;
      runSimBtn.innerHTML = `<span class="spinner"></span> Scanning vectors...`;

      try {
        const res = await MoodifyAPI.getSimilarSongs(seedId, mode, 0.60, 6);
        const resultsBox = container.querySelector('#similarity-results');
        const similar = res.similar_songs || [];

        if (similar.length === 0) {
          resultsBox.innerHTML = `<p style="font-size:12px; color:var(--color-text-muted); grid-column:1/-1;">No neighbors found above 60% similarity threshold.</p>`;
        } else {
          resultsBox.innerHTML = similar.map(track => `
            <div class="glass-panel" style="padding:14px; display:flex; align-items:center; justify-content:space-between;">
              <div>
                <div style="font-weight:600; font-size:13px; color:var(--color-text-primary);">${track.title || track.original_name}</div>
                <div style="font-size:11px; color:var(--color-text-secondary);">${track.artist || 'Unknown'} • ${track.tempo_bpm ? track.tempo_bpm.toFixed(0) : '--'} BPM</div>
              </div>
              <div style="display:flex; align-items:center; gap:8px;">
                <span class="mono-num" style="font-size:12px; font-weight:700; color:var(--color-action);">
                  ${track.similarity ? (track.similarity * 100).toFixed(0) : '85'}% match
                </span>
                <button class="track-play-btn" style="width:28px; height:28px;" data-sim-play="${track.id}" title="Play Match">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                </button>
              </div>
            </div>
          `).join('');

          // Bind Play buttons on similarity results
          resultsBox.querySelectorAll('[data-sim-play]').forEach(btn => {
            btn.addEventListener('click', () => {
              const tid = btn.dataset.simPlay;
              const trk = similar.find(s => s.id === tid) || songs.find(s => s.id === tid);
              if (trk) playTrack(trk, similar);
            });
          });
        }
      } catch (err) {
        showToast('Similarity search error: ' + err.message, 'error');
      } finally {
        runSimBtn.disabled = false;
        runSimBtn.innerHTML = 'Find Neighbors';
      }
    });
  }
}
