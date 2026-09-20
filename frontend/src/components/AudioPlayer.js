/**
 * Moodify Floating Audio Player Component
 * Robust HTML5 audio streaming with Web Audio synthesizer fallback,
 * animated spectrum visualizer, playlist queue, and 3D kinetic linking.
 */

import { setPlaybackKinetics } from '../scene3d/scene.js';
import { MoodifyAPI } from '../api/endpoints.js';
import { showToast } from './Toast.js';

let audioElement = null;
let currentSong = null;
let playlistQueue = [];
let currentIndex = -1;
let isPlaying = false;
let isDraggingSeeker = false;

// Web Audio API Synthesizer fallback for mock/missing tracks
let audioCtx = null;
let synthInterval = null;
let synthGain = null;
let synthTime = 0;
let isUsingSynth = false;

export function initAudioPlayer() {
  audioElement = new Audio();
  audioElement.preload = 'metadata';

  // Native HTML5 Audio Event Listeners
  audioElement.addEventListener('loadedmetadata', () => {
    isUsingSynth = false;
    stopSynth();
    updateTimeline();
  });

  audioElement.addEventListener('timeupdate', () => {
    if (!isUsingSynth && !isDraggingSeeker) {
      updateTimeline();
    }
  });

  audioElement.addEventListener('play', () => {
    setPlayingState(true);
  });

  audioElement.addEventListener('pause', () => {
    setPlayingState(false);
  });

  audioElement.addEventListener('ended', () => {
    playNextTrack();
  });

  audioElement.addEventListener('error', () => {
    console.warn('[Moodify Audio] Physical audio stream unavailable or failed. Engaging Web Audio synthesizer preview.');
    startSynthFallback();
  });

  renderPlayerDOM();
}

function renderPlayerDOM() {
  const playerBar = document.createElement('div');
  playerBar.className = 'audio-player-bar';
  playerBar.id = 'moodify-player';

  playerBar.innerHTML = `
    <!-- Left: Track Metadata -->
    <div class="player-track-info">
      <div class="player-art-placeholder" id="player-art">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>
        </svg>
      </div>
      <div class="track-info">
        <span class="track-title" id="player-title">Select a track to play</span>
        <span class="track-artist" id="player-artist">Moodify Audio Engine</span>
      </div>
    </div>

    <!-- Center: Playback Controls & Timeline -->
    <div class="player-center-controls">
      <div class="player-buttons">
        <button class="player-ctrl-btn" id="player-prev-btn" title="Previous Track">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><polygon points="19 20 9 12 19 4 19 20"/><line x1="5" y1="19" x2="5" y2="5" stroke="currentColor" stroke-width="2"/></svg>
        </button>
        <button class="player-ctrl-btn play-master" id="player-toggle-btn" title="Play / Pause">
          <svg id="play-icon" width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
          <svg id="pause-icon" style="display:none;" width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
        </button>
        <button class="player-ctrl-btn" id="player-next-btn" title="Next Track">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 4 15 12 5 20 5 4"/><line x1="19" y1="5" x2="19" y2="19" stroke="currentColor" stroke-width="2"/></svg>
        </button>
      </div>

      <div class="player-timeline-wrapper">
        <span class="mono-num" id="player-time-current" style="font-size:11px; color:var(--color-text-muted);">0:00</span>
        <input type="range" class="timeline-slider" id="player-seek" min="0" max="100" value="0" />
        <span class="mono-num" id="player-time-total" style="font-size:11px; color:var(--color-text-muted);">0:00</span>
      </div>
    </div>

    <!-- Right: Kinetic Waveform Spectrum & Volume -->
    <div class="player-wave-container" style="display:flex; align-items:center; justify-content:flex-end; gap:16px;">
      <div class="waveform-bars" id="visualizer-bars">
        <div class="wave-bar"></div>
        <div class="wave-bar"></div>
        <div class="wave-bar"></div>
        <div class="wave-bar"></div>
        <div class="wave-bar"></div>
        <div class="wave-bar"></div>
        <div class="wave-bar"></div>
        <div class="wave-bar"></div>
      </div>
      <div style="display:flex; align-items:center; gap:8px;">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color:var(--color-text-muted);">
          <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><path d="M19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07"/>
        </svg>
        <input type="range" class="timeline-slider" id="player-volume" min="0" max="1" step="0.05" value="0.85" style="width:70px;" />
      </div>
    </div>
  `;

  document.body.appendChild(playerBar);

  // Bind Master Controls
  document.getElementById('player-toggle-btn').addEventListener('click', togglePlay);
  document.getElementById('player-prev-btn').addEventListener('click', playPreviousTrack);
  document.getElementById('player-next-btn').addEventListener('click', playNextTrack);

  // Bind Seeker Events (smooth drag without jitter)
  const seekSlider = document.getElementById('player-seek');
  seekSlider.addEventListener('mousedown', () => { isDraggingSeeker = true; });
  seekSlider.addEventListener('touchstart', () => { isDraggingSeeker = true; }, { passive: true });

  seekSlider.addEventListener('input', (e) => {
    const dur = getActiveDuration();
    const targetSec = (e.target.value / 100) * dur;
    document.getElementById('player-time-current').textContent = formatTime(targetSec);
  });

  seekSlider.addEventListener('change', (e) => {
    const dur = getActiveDuration();
    const targetSec = (e.target.value / 100) * dur;
    if (isUsingSynth) {
      synthTime = targetSec;
    } else if (audioElement && audioElement.duration) {
      audioElement.currentTime = targetSec;
    }
    isDraggingSeeker = false;
  });

  // Bind Volume Slider
  const volSlider = document.getElementById('player-volume');
  volSlider.addEventListener('input', (e) => {
    const val = parseFloat(e.target.value);
    if (audioElement) audioElement.volume = val;
    if (synthGain) synthGain.gain.setValueAtTime(val * 0.15, audioCtx?.currentTime || 0);
  });
}

/**
 * Initiates playback for a given track and updates playlist queue context.
 * @param {Object} song - Track object
 * @param {Array} queue - Optional list of songs for next/prev sequencing
 */
export function playTrack(song, queue = null) {
  if (!song) return;
  currentSong = song;

  if (queue && Array.isArray(queue) && queue.length > 0) {
    playlistQueue = queue;
    currentIndex = playlistQueue.findIndex(s => s.id === song.id);
  } else if (!playlistQueue.some(s => s.id === song.id)) {
    playlistQueue.push(song);
    currentIndex = playlistQueue.length - 1;
  } else {
    currentIndex = playlistQueue.findIndex(s => s.id === song.id);
  }

  // Update DOM labels
  document.getElementById('player-title').textContent = song.title || song.original_name || 'Untitled Track';
  document.getElementById('player-artist').textContent = song.artist || song.inferred_genre || 'Audio File';

  // Stop any active synth
  stopSynth();
  isUsingSynth = false;

  // Reset Seeker
  document.getElementById('player-seek').value = 0;
  document.getElementById('player-time-current').textContent = '0:00';
  document.getElementById('player-time-total').textContent = formatTime(song.duration_sec || 180);

  // Check if song is an in-memory mock or real server song
  const isMockId = String(song.id).startsWith('mock-') || String(song.id).startsWith('song-');

  if (isMockId) {
    startSynthFallback();
    showToast(`Playing preview: ${song.title || song.original_name}`, 'success');
    return;
  }

  // Load Real Stream from Backend
  const downloadUrl = MoodifyAPI.getDownloadUrl(song.id);
  audioElement.src = downloadUrl;

  const playPromise = audioElement.play();
  if (playPromise !== undefined) {
    playPromise
      .then(() => {
        setPlayingState(true);
        showToast(`Streaming: ${song.title || song.original_name}`, 'success');
      })
      .catch((err) => {
        console.warn('[Moodify Audio] Autoplay policy or media source error:', err);
        // If real audio format cannot decode or stream errors, fallback to harmonic synthesizer
        startSynthFallback();
      });
  }
}

export function togglePlay() {
  if (!currentSong && playlistQueue.length > 0) {
    playTrack(playlistQueue[0]);
    return;
  }
  if (!currentSong) return;

  if (isPlaying) {
    if (isUsingSynth) {
      pauseSynth();
    } else {
      audioElement.pause();
    }
    setPlayingState(false);
  } else {
    if (isUsingSynth) {
      resumeSynth();
    } else {
      audioElement.play().catch(() => startSynthFallback());
    }
    setPlayingState(true);
  }
}

export function playNextTrack() {
  if (playlistQueue.length === 0) return;
  currentIndex = (currentIndex + 1) % playlistQueue.length;
  playTrack(playlistQueue[currentIndex], playlistQueue);
}

export function playPreviousTrack() {
  if (playlistQueue.length === 0) return;
  currentIndex = (currentIndex - 1 + playlistQueue.length) % playlistQueue.length;
  playTrack(playlistQueue[currentIndex], playlistQueue);
}

function setPlayingState(playing) {
  isPlaying = playing;
  const playIcon = document.getElementById('play-icon');
  const pauseIcon = document.getElementById('pause-icon');
  const bars = document.querySelectorAll('#visualizer-bars .wave-bar');

  if (playing) {
    if (playIcon) playIcon.style.display = 'none';
    if (pauseIcon) pauseIcon.style.display = 'block';
    bars.forEach((b, i) => {
      b.classList.add('animated');
      b.style.animationDelay = `${(i * 0.12).toFixed(2)}s`;
    });
    setPlaybackKinetics(true, currentSong?.tempo_bpm || 120);
  } else {
    if (playIcon) playIcon.style.display = 'block';
    if (pauseIcon) pauseIcon.style.display = 'none';
    bars.forEach(b => b.classList.remove('animated'));
    setPlaybackKinetics(false);
  }
}

function getActiveDuration() {
  if (isUsingSynth) {
    return currentSong?.duration_sec || 180;
  }
  if (audioElement && audioElement.duration && !isNaN(audioElement.duration) && isFinite(audioElement.duration)) {
    return audioElement.duration;
  }
  return currentSong?.duration_sec || 180;
}

function updateTimeline() {
  const dur = getActiveDuration();
  const cur = isUsingSynth ? synthTime : (audioElement?.currentTime || 0);

  const currentSpan = document.getElementById('player-time-current');
  const totalSpan = document.getElementById('player-time-total');
  const seek = document.getElementById('player-seek');

  if (currentSpan) currentSpan.textContent = formatTime(cur);
  if (totalSpan) totalSpan.textContent = formatTime(dur);
  if (seek && dur > 0 && !isDraggingSeeker) {
    seek.value = Math.min(100, (cur / dur) * 100);
  }
}

function formatTime(secs) {
  if (!secs || isNaN(secs)) return '0:00';
  const m = Math.floor(secs / 60);
  const s = Math.floor(secs % 60);
  return `${m}:${s < 10 ? '0' : ''}${s}`;
}

// ── Web Audio Synthesizer Fallback Engine ──────────────────────────

function ensureAudioContext() {
  if (!audioCtx) {
    const AudioContextClass = window.AudioContext || window.webkitAudioContext;
    if (AudioContextClass) {
      audioCtx = new AudioContextClass();
    }
  }
  if (audioCtx && audioCtx.state === 'suspended') {
    audioCtx.resume();
  }
}

function startSynthFallback() {
  isUsingSynth = true;
  synthTime = 0;
  ensureAudioContext();
  setPlayingState(true);
  resumeSynth();
}

function resumeSynth() {
  ensureAudioContext();
  clearInterval(synthInterval);

  // Chord notes for harmonic generative ambient pad
  const chordNotes = [220.0, 277.18, 329.63, 440.0, 554.37, 659.25]; // A major / C#m ambience
  let step = 0;

  synthInterval = setInterval(() => {
    if (!isPlaying) return;

    synthTime += 0.25;
    updateTimeline();

    const dur = getActiveDuration();
    if (synthTime >= dur) {
      playNextTrack();
      return;
    }

    // Play subtle harmonic notes on beat
    if (audioCtx && audioCtx.state === 'running' && Math.floor(synthTime * 2) > step) {
      step = Math.floor(synthTime * 2);
      playSynthNote(chordNotes[step % chordNotes.length]);
    }
  }, 250);
}

function playSynthNote(freq) {
  if (!audioCtx) return;
  try {
    const osc = audioCtx.createOscillator();
    const gain = audioCtx.createGain();

    osc.type = 'sine';
    osc.frequency.setValueAtTime(freq, audioCtx.currentTime);

    const masterVol = parseFloat(document.getElementById('player-volume')?.value || 0.8);
    gain.gain.setValueAtTime(0.001, audioCtx.currentTime);
    gain.gain.exponentialRampToValueAtTime(masterVol * 0.08, audioCtx.currentTime + 0.05);
    gain.gain.exponentialRampToValueAtTime(0.0001, audioCtx.currentTime + 0.6);

    osc.connect(gain);
    gain.connect(audioCtx.destination);

    osc.start();
    osc.stop(audioCtx.currentTime + 0.65);
  } catch (e) {
    // Ignore synth audio hiccups
  }
}

function pauseSynth() {
  clearInterval(synthInterval);
}

function stopSynth() {
  clearInterval(synthInterval);
  synthTime = 0;
}
