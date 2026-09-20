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
let playlist = [];
let isSourceChanging = false;

const MOOD_AURA_MAP = {
  'energetic': { color: 'rgba(236, 72, 153, 0.18)', glow: 'rgba(236, 72, 153, 0.35)', border: 'rgba(236, 72, 153, 0.6)' },
  'calm': { color: 'rgba(0, 240, 255, 0.16)', glow: 'rgba(0, 240, 255, 0.35)', border: 'rgba(0, 240, 255, 0.6)' },
  'happy': { color: 'rgba(16, 185, 129, 0.18)', glow: 'rgba(16, 185, 129, 0.35)', border: 'rgba(16, 185, 129, 0.6)' },
  'sad': { color: 'rgba(99, 102, 241, 0.18)', glow: 'rgba(99, 102, 241, 0.35)', border: 'rgba(99, 102, 241, 0.6)' },
  'dark': { color: 'rgba(239, 68, 68, 0.18)', glow: 'rgba(239, 68, 68, 0.35)', border: 'rgba(239, 68, 68, 0.6)' },
  'dreamy': { color: 'rgba(168, 85, 247, 0.18)', glow: 'rgba(168, 85, 247, 0.35)', border: 'rgba(168, 85, 247, 0.6)' },
};

/**
 * Sets or updates the active playlist
 */
export function setPlaylist(tracks) {
  if (Array.isArray(tracks)) {
    playlist = tracks;
  }
}

/**
 * Initializes the audio player in the persistent player container
 */
export function initAudioPlayer() {
  playerContainer = document.getElementById('persistent-player-container');
  if (!playerContainer) return;

  audioElement = new Audio();
  audioElement.preload = 'auto';

  // Load persistent volume from localStorage
  const savedVol = parseFloat(localStorage.getItem('moodify_volume') || '0.85');
  audioElement.volume = isNaN(savedVol) ? 0.85 : Math.max(0, Math.min(1, savedVol));

  // Attach audio events
  audioElement.addEventListener('timeupdate', handleTimeUpdate);
  audioElement.addEventListener('ended', () => {
    isPlaying = false;
    updatePlayState(false);
    playNextTrack();
  });
  audioElement.addEventListener('play', () => {
    isPlaying = true;
    updatePlayState(true);
  });
  audioElement.addEventListener('playing', () => {
    isSourceChanging = false;
    isPlaying = true;
    updatePlayState(true);
  });
  audioElement.addEventListener('pause', () => {
    if (isSourceChanging || !currentTrack) {
      // Ignore synthetic pause events emitted while resetting or changing media sources, or when closing player
      return;
    }
    isPlaying = false;
    updatePlayState(false);
  });

  // Global Keyboard Shortcuts (Space: Play/Pause, ArrowRight: Next, ArrowLeft: Prev, Esc: Close)
  window.addEventListener('keydown', (e) => {
    const activeEl = document.activeElement;
    const tag = activeEl ? activeEl.tagName.toLowerCase() : '';
    if (tag === 'input' || tag === 'textarea' || tag === 'select' || activeEl?.isContentEditable) {
      return;
    }

    if (e.code === 'Space') {
      e.preventDefault();
      togglePlayPause();
    } else if (e.code === 'ArrowRight') {
      e.preventDefault();
      playNextTrack();
    } else if (e.code === 'ArrowLeft') {
      e.preventDefault();
      playPreviousTrack();
    } else if (e.code === 'Escape') {
      if (currentTrack || (playerContainer && playerContainer.children.length > 0)) {
        closeAudioPlayer(e);
      }
    }
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

  // Sync position with OS MediaSession (Wayland, Windows SMTC, Android, iOS)
  if ('mediaSession' in navigator && 'setPositionState' in navigator.mediaSession && audioElement && audioElement.duration) {
    try {
      navigator.mediaSession.setPositionState({
        duration: audioElement.duration,
        playbackRate: audioElement.playbackRate || 1,
        position: audioElement.currentTime,
      });
    } catch (_) {}
  }
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

  // Sync playback state with OS MediaSession
  if ('mediaSession' in navigator) {
    navigator.mediaSession.playbackState = playing ? 'playing' : 'paused';
  }
}

function updateAuraTheme(dominantMood) {
  const root = document.documentElement;
  const moodKey = (dominantMood || '').toLowerCase().trim();
  const aura = MOOD_AURA_MAP[moodKey] || {
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
 * @param {Array} [trackList] - Optional playlist array
 */
export function playTrack(track, trackList = null) {
  if (!track) return;
  if (!audioElement) {
    initAudioPlayer();
  }

  if (Array.isArray(trackList) && trackList.length > 0) {
    playlist = trackList;
  } else if (!playlist.some(t => t.id === track.id)) {
    playlist.push(track);
  }

  const streamUrl = MoodifyAPI.getAudioStreamUrl(track.id);

  // If clicked track is already loaded in audioElement
  if (currentTrack && currentTrack.id === track.id && audioElement.src && audioElement.src.includes(track.id)) {
    if (audioElement.paused) {
      const p = audioElement.play();
      if (p !== undefined) {
        p.then(() => {
          isPlaying = true;
          updatePlayState(true);
        }).catch(err => {
          console.warn('Playback resume error:', err);
        });
      }
    }
    return;
  }

  currentTrack = track;
  isSourceChanging = true;

  audioElement.src = streamUrl;

  renderActivePlayer(track);
  setupMediaSession(track);

  const playPromise = audioElement.play();
  if (playPromise !== undefined) {
    playPromise.then(() => {
      isSourceChanging = false;
      isPlaying = true;
      updatePlayState(true);
    }).catch(e => {
      isSourceChanging = false;
      if (e.name === 'AbortError') {
        return;
      }
      console.warn('Playback error or offline stream:', e);
      isPlaying = false;
      updatePlayState(false);
    });
  }
}

/**
 * Registers OS MediaSession metadata & action handlers
 */
function setupMediaSession(track) {
  if (!('mediaSession' in navigator) || !track) return;
  const displayTitle = MoodifyAPI.getCleanTitle(track);
  navigator.mediaSession.metadata = new MediaMetadata({
    title: displayTitle,
    artist: track.artist || 'Unknown Artist',
    album: track.album || 'Moodify Audio Library',
    artwork: [
      { src: '/icon-96.png', sizes: '96x96', type: 'image/png' },
      { src: '/icon-128.png', sizes: '128x128', type: 'image/png' },
      { src: '/icon-256.png', sizes: '256x256', type: 'image/png' },
      { src: '/icon-512.png', sizes: '512x512', type: 'image/png' },
    ]
  });

  try {
    navigator.mediaSession.setActionHandler('play', () => {
      if (audioElement && audioElement.paused) togglePlayPause();
    });
    navigator.mediaSession.setActionHandler('pause', () => {
      if (audioElement && !audioElement.paused) togglePlayPause();
    });
    navigator.mediaSession.setActionHandler('nexttrack', () => {
      playNextTrack();
    });
    navigator.mediaSession.setActionHandler('previoustrack', () => {
      playPreviousTrack();
    });
    navigator.mediaSession.setActionHandler('seekbackward', (details) => {
      if (audioElement) audioElement.currentTime = Math.max(0, audioElement.currentTime - (details.seekOffset || 10));
    });
    navigator.mediaSession.setActionHandler('seekforward', (details) => {
      if (audioElement) audioElement.currentTime = Math.min(audioElement.duration || 0, audioElement.currentTime + (details.seekOffset || 10));
    });
    navigator.mediaSession.setActionHandler('seekto', (details) => {
      if (audioElement && details.seekTime !== undefined) audioElement.currentTime = details.seekTime;
    });
  } catch (e) {
    console.warn('MediaSession handler registration:', e);
  }
}

/**
 * Plays the next song in the active playlist
 */
export function playNextTrack() {
  if (!playlist || playlist.length === 0) return;
  const curId = currentTrack?.id;
  const idx = playlist.findIndex(t => t.id === curId);
  const nextIdx = (idx + 1) % playlist.length;
  const nextTrack = playlist[nextIdx];
  if (nextTrack) {
    playTrack(nextTrack, playlist);
  }
}

/**
 * Plays the previous song in the active playlist
 */
export function playPreviousTrack() {
  if (!playlist || playlist.length === 0) return;
  const curId = currentTrack?.id;
  const idx = playlist.findIndex(t => t.id === curId);
  const prevIdx = (idx - 1 + playlist.length) % playlist.length;
  const prevTrack = playlist[prevIdx];
  if (prevTrack) {
    playTrack(prevTrack, playlist);
  }
}

/**
 * Closes the music player, stops playback and cleans up 3D kinetics
 */
export function closeAudioPlayer(e) {
  if (e) {
    e.preventDefault?.();
    e.stopPropagation?.();
  }

  isPlaying = false;
  currentTrack = null;

  try {
    if (audioElement) {
      audioElement.pause();
      audioElement.currentTime = 0;
      audioElement.removeAttribute('src');
    }
  } catch (err) {
    console.warn('Error pausing audio:', err);
  }

  try {
    setPlaybackKinetics(false, 120, '');
  } catch (_) {}

  try {
    updateAuraTheme('');
  } catch (_) {}

  if (playerContainer) {
    playerContainer.innerHTML = '';
  }

  if ('mediaSession' in navigator) {
    try {
      navigator.mediaSession.playbackState = 'none';
    } catch (_) {}
  }
}

export function togglePlayPause() {
  if (!audioElement || !currentTrack) return;
  if (audioElement.paused) {
    const p = audioElement.play();
    if (p !== undefined) {
      p.then(() => {
        isPlaying = true;
        updatePlayState(true);
      }).catch(e => console.warn('Play error:', e));
    }
  } else {
    audioElement.pause();
  }
}

function renderActivePlayer(track) {
  const mood = track.dominant_mood || 'calm & neutral';
  const duration = track.duration_sec || 180;
  const currentVol = audioElement ? audioElement.volume : 0.85;
  const cleanFilename = MoodifyAPI.getCleanFilename(track);

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
            <div class="player-title" title="${escapeHtml(MoodifyAPI.getCleanTitle(track))}">${escapeHtml(MoodifyAPI.getCleanTitle(track))}</div>
            <div class="player-artist-row">
              <span class="player-artist">${escapeHtml(track.artist || 'Unknown Artist')}</span>
              <span class="badge-format ${MoodifyAPI.isFormatCorrected(track) ? 'format-corrected' : ''}" title="${escapeHtml(MoodifyAPI.isFormatCorrected(track) ? `Detected & auto-corrected format: ${track.format}` : `Format: ${track.format}`)}">${(track.format || 'mp3').toUpperCase()}</span>
            </div>
          </div>
        </div>

        <!-- Center: Playback Controls -->
        <div class="player-center">
          <div class="controls-row">
            <button class="btn-ctrl" id="btn-player-prev" title="Previous Track (Left Arrow)">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor"><polygon points="19 20 9 12 19 4 19 20"/><line x1="5" y1="19" x2="5" y2="5" stroke="currentColor" stroke-width="2.5"/></svg>
            </button>
            <button class="btn-ctrl-play" id="btn-player-playpause" title="Play/Pause (Space)">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16" rx="1"/><rect x="14" y="4" width="4" height="16" rx="1"/></svg>
            </button>
            <button class="btn-ctrl" id="btn-player-next" title="Next Track (Right Arrow)">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 4 15 12 5 20 5 4"/><line x1="19" y1="5" x2="19" y2="19" stroke="currentColor" stroke-width="2.5"/></svg>
            </button>
          </div>
          <div class="time-row">
            <span class="time-label mono-num" id="player-time-current">00:00</span>
            <span class="time-sep">/</span>
            <span class="time-label mono-num" id="player-time-total">${formatTime(duration)}</span>
          </div>
        </div>

        <!-- Right: Volume & Download & Close -->
        <div class="player-right">
          <div class="volume-slider-wrapper" title="Volume">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
            </svg>
            <input type="range" class="volume-slider" id="player-volume" min="0" max="1" step="0.05" value="${currentVol}" />
          </div>

          <a href="${MoodifyAPI.getAudioStreamUrl(track.id, true)}" download="${escapeHtml(cleanFilename)}" class="btn-player-action icon-only" title="Download: ${escapeHtml(cleanFilename)}">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
          </a>

          <button type="button" class="btn-player-action icon-only btn-player-close" id="btn-player-close" title="Close Music Player (Esc)">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" pointer-events="none">
              <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
      </div>
    </div>
  `;

  // Attach listeners
  playerContainer.querySelector('#btn-player-playpause')?.addEventListener('click', togglePlayPause);
  playerContainer.querySelector('#btn-player-prev')?.addEventListener('click', playPreviousTrack);
  playerContainer.querySelector('#btn-player-next')?.addEventListener('click', playNextTrack);
  playerContainer.querySelector('#btn-player-close')?.addEventListener('click', (e) => closeAudioPlayer(e));

  const progressTrack = playerContainer.querySelector('#player-progress-bar');
  progressTrack?.addEventListener('click', (e) => {
    if (!audioElement || !audioElement.duration) return;
    const rect = progressTrack.getBoundingClientRect();
    const ratio = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
    audioElement.currentTime = ratio * audioElement.duration;
  });

  playerContainer.querySelector('#player-volume')?.addEventListener('input', (e) => {
    const val = parseFloat(e.target.value);
    if (audioElement) audioElement.volume = val;
    localStorage.setItem('moodify_volume', val.toString());
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
