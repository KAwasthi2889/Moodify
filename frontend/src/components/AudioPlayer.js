/**
 * Moodify Persistent Docked Audio Player & Dynamic Aura Broadcaster
 * Synthesized from @frontend skill & FRONTEND_INTEGRATION_GUIDE.md.
 * Modulates background aura and kinetic Three.js wave physics.
 */

import { setPlaybackKinetics } from '../scene3d/scene.js';
import { MoodifyAPI } from '../api/endpoints.js';

let audioElement = null;
let currentTrack = null;
let isPlaying = false;
let playerContainer = null;
let timeUpdateListeners = [];

const MOOD_AURA_MAP = {
  'joy & optimism': {
    color: 'rgba(245, 158, 11, 0.22)',
    glow: 'rgba(245, 158, 11, 0.45)',
    border: '#f59e0b'
  },
  'desire & love': {
    color: 'rgba(244, 63, 94, 0.22)',
    glow: 'rgba(244, 63, 94, 0.45)',
    border: '#f43f5e'
  },
  'sadness & grief': {
    color: 'rgba(99, 102, 241, 0.22)',
    glow: 'rgba(56, 189, 248, 0.45)',
    border: '#6366f1'
  },
  'anger & annoyance': {
    color: 'rgba(239, 68, 68, 0.25)',
    glow: 'rgba(239, 68, 68, 0.50)',
    border: '#ef4444'
  },
  'calm & neutral': {
    color: 'rgba(16, 185, 129, 0.22)',
    glow: 'rgba(6, 182, 212, 0.45)',
    border: '#10b981'
  }
};

/**
 * Initializes the audio player in the persistent player container
 */
export function initAudioPlayer() {
  playerContainer = document.getElementById('persistent-player-container');
  if (!playerContainer) return;

  audioElement = new Audio();
  audioElement.preload = 'metadata';

  // Attach audio events
  audioElement.addEventListener('timeupdate', handleTimeUpdate);
  audioElement.addEventListener('ended', () => {
    isPlaying = false;
    updatePlayState(false);
  });
  audioElement.addEventListener('play', () => {
    isPlaying = true;
    updatePlayState(true);
  });
  audioElement.addEventListener('pause', () => {
    isPlaying = false;
    updatePlayState(false);
  });
}

/**
 * Subscribes to audio time updates
 */
export function onAudioTimeUpdate(callback) {
  timeUpdateListeners.push(callback);
}

export function getCurrentPlayback() {
  return {
    track: currentTrack,
    currentTime: audioElement?.currentTime || 0,
    duration: audioElement?.duration || 0,
    isPlaying
  };
}

function handleTimeUpdate() {
  if (!audioElement) return;
  const current = audioElement.currentTime;
  const duration = audioElement.duration || 1;

  // Update progress bar
  const progressFill = playerContainer?.querySelector('#player-progress-fill');
  const timeCurrentEl = playerContainer?.querySelector('#player-time-current');

  if (progressFill) {
    const percent = Math.min(100, (current / duration) * 100);
    progressFill.style.width = `${percent}%`;
  }

  if (timeCurrentEl) {
    timeCurrentEl.textContent = formatTime(current);
  }

  timeUpdateListeners.forEach(cb => {
    try {
      cb(current, duration);
    } catch (e) {
      console.error(e);
    }
  });
}

function updatePlayState(playing) {
  if (!playerContainer) return;
  const playBtn = playerContainer.querySelector('#btn-player-playpause');
  if (playBtn) {
    playBtn.innerHTML = playing
      ? `<svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16" rx="1"/><rect x="14" y="4" width="4" height="16" rx="1"/></svg>`
      : `<svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><polygon points="6 3 20 12 6 21 6 3"/></svg>`;
  }

  // Modulate 3D kinetic canvas & dynamic background aura
  if (currentTrack) {
    setPlaybackKinetics(playing, currentTrack.tempo || 120, currentTrack.dominant_mood || '');
    updateAuraTheme(playing ? currentTrack.dominant_mood : '');
  }

  const eq = playerContainer.querySelector('.eq-bars');
  if (eq) {
    eq.classList.toggle('active', playing);
  }
}

function updateAuraTheme(dominantMood) {
  const root = document.documentElement;
  const aura = MOOD_AURA_MAP[dominantMood] || {
    color: 'rgba(0, 240, 255, 0.16)',
    glow: 'rgba(0, 240, 255, 0.35)',
    border: 'rgba(0, 240, 255, 0.6)'
  };

  root.style.setProperty('--current-aura-color', aura.color);
  root.style.setProperty('--current-aura-glow', aura.glow);

  const playerBar = playerContainer?.querySelector('.docked-player');
  if (playerBar) {
    playerBar.style.boxShadow = `0 16px 40px -6px rgba(0, 0, 0, 0.8), 0 0 28px ${aura.glow}`;
    playerBar.style.borderColor = aura.border;
  }
}

/**
 * Plays a specified song track
 * @param {Object} track - Song entity
 */
export function playTrack(track) {
  if (!track) return;
  currentTrack = track;

  const streamUrl = MoodifyAPI.getAudioStreamUrl(track.id);
  audioElement.src = streamUrl;

  renderActivePlayer(track);

  audioElement.play().catch(e => {
    console.warn('Playback error or offline stream:', e);
    isPlaying = false;
    updatePlayState(false);
  });
}

export function togglePlayPause() {
  if (!audioElement || !currentTrack) return;
  if (audioElement.paused) {
    audioElement.play().catch(e => console.warn(e));
  } else {
    audioElement.pause();
  }
}

function renderActivePlayer(track) {
  const mood = track.dominant_mood || 'calm & neutral';
  const duration = track.duration_sec || 180;

  playerContainer.innerHTML = `
    <div class="docked-player active">
      <!-- Scrubbing Timeline -->
      <div class="player-progress-bar" id="player-progress-bar">
        <div class="player-progress-track">
          <div class="player-progress-fill" id="player-progress-fill" style="width:0%;"></div>
        </div>
      </div>

      <div class="player-content">
        <!-- Left: Track Metadata -->
        <div class="player-left">
          <div class="player-artwork">
            <div class="eq-bars active">
              <span></span><span></span><span></span><span></span>
            </div>
          </div>
          <div class="player-meta">
            <div class="player-title" title="${escapeHtml(track.title || track.filename)}">${escapeHtml(track.title || track.filename)}</div>
            <div class="player-artist-row">
              <span class="player-artist">${escapeHtml(track.artist || 'Session Track')}</span>
              <span class="badge-format">${(track.format || 'mp3').toUpperCase()}</span>
            </div>
          </div>
        </div>

        <!-- Center: Playback Controls -->
        <div class="player-center">
          <div class="controls-row">
            <button class="btn-ctrl" id="btn-player-rw" title="Rewind 10s">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="11 19 2 12 11 5 11 19"/><polygon points="22 19 13 12 22 5 22 19"/></svg>
            </button>
            <button class="btn-ctrl-play" id="btn-player-playpause" title="Play/Pause">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16" rx="1"/><rect x="14" y="4" width="4" height="16" rx="1"/></svg>
            </button>
            <button class="btn-ctrl" id="btn-player-ff" title="Forward 10s">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 19 22 12 13 5 13 19"/><polygon points="2 19 11 12 2 5 2 19"/></svg>
            </button>
          </div>
          <div class="time-row">
            <span class="time-label mono-num" id="player-time-current">00:00</span>
            <span class="time-sep">/</span>
            <span class="time-label mono-num" id="player-time-total">${formatTime(duration)}</span>
          </div>
        </div>

        <!-- Right: Volume & Download -->
        <div class="player-right">
          <div class="volume-slider-wrapper" title="Volume">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
            </svg>
            <input type="range" class="volume-slider" id="player-volume" min="0" max="1" step="0.05" value="0.85" />
          </div>

          <a href="${MoodifyAPI.getAudioStreamUrl(track.id, true)}" class="btn-player-action icon-only" title="Download Audio File" download>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
          </a>
        </div>
      </div>
    </div>
  `;

  // Attach listeners
  playerContainer.querySelector('#btn-player-playpause')?.addEventListener('click', togglePlayPause);
  playerContainer.querySelector('#btn-player-rw')?.addEventListener('click', () => {
    if (audioElement) audioElement.currentTime = Math.max(0, audioElement.currentTime - 10);
  });
  playerContainer.querySelector('#btn-player-ff')?.addEventListener('click', () => {
    if (audioElement) audioElement.currentTime = Math.min(audioElement.duration || 0, audioElement.currentTime + 10);
  });

  const progressTrack = playerContainer.querySelector('#player-progress-bar');
  progressTrack?.addEventListener('click', (e) => {
    if (!audioElement || !audioElement.duration) return;
    const rect = progressTrack.getBoundingClientRect();
    const ratio = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
    audioElement.currentTime = ratio * audioElement.duration;
  });

  playerContainer.querySelector('#player-volume')?.addEventListener('input', (e) => {
    if (audioElement) audioElement.volume = parseFloat(e.target.value);
  });

  updateAuraTheme(mood);
}

function formatTime(seconds) {
  if (isNaN(seconds) || seconds < 0) return '00:00';
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins < 10 ? '0' : ''}${mins}:${secs < 10 ? '0' : ''}${secs}`;
}

function escapeHtml(str) {
  if (!str) return '';
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
