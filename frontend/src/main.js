/**
 * Moodify Main Application Bootstrap
 * Orchestrates Three.js kinetic scene, Hybrid API Engine, navigation tabs,
 * and modular bento components.
 */

import './styles/tokens.css';
import './styles/base.css';
import './styles/components.css';

import { initScene } from './scene3d/scene.js';
import { checkBackendHealth, onApiStatusChange } from './api/client.js';
import { MoodifyAPI } from './api/endpoints.js';
import { renderStatusBadge } from './components/StatusBadge.js';
import { initAudioPlayer } from './components/AudioPlayer.js';
import { initTrackStudioModal } from './components/TrackStudioModal.js';
import { renderLibraryTable } from './components/LibraryTable.js';
import { renderMoodGalaxy } from './components/MoodGalaxy.js';
import { renderPlaylistStudio } from './components/PlaylistStudio.js';
import { renderBatchStation } from './components/BatchStation.js';

let appState = {
  currentTab: 'library',
  songs: [],
  selectedMoodFilter: 'all'
};

async function initApp() {
  // 1. Initialize Background 3D Procedural Canvas
  const canvas = document.getElementById('webgl-canvas');
  initScene(canvas);

  // 2. Initialize Floating Audio Player & Track Studio Modal
  initAudioPlayer();
  initTrackStudioModal();

  // 3. Render Status Badge in Header
  const badgeContainer = document.getElementById('header-badge-wrap');
  renderStatusBadge(badgeContainer);

  // 4. Bind Navigation Tabs
  setupNavigation();

  // 5. Check Backend Connectivity
  await checkBackendHealth();

  // 6. Load Initial Data
  await loadSongs();

  // Re-fetch data if backend comes online/offline
  onApiStatusChange(() => {
    loadSongs();
  });
}

async function loadSongs() {
  try {
    const res = await MoodifyAPI.listSongs();
    appState.songs = res.songs || [];
    renderActiveTab();
  } catch (err) {
    console.error('Failed to load songs', err);
  }
}

function setupNavigation() {
  const tabs = document.querySelectorAll('.nav-tabs .tab-btn');
  tabs.forEach(btn => {
    btn.addEventListener('click', () => {
      tabs.forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      appState.currentTab = btn.dataset.tab;
      renderActiveTab();
    });
  });
}

function renderActiveTab() {
  const viewContainer = document.getElementById('tab-view-container');
  if (!viewContainer) return;

  switch (appState.currentTab) {
    case 'library':
      renderLibraryTable(viewContainer, appState.songs, () => loadSongs());
      break;

    case 'galaxy':
      renderMoodGalaxy(viewContainer, appState.songs, (mood) => {
        // Switch to library tab and filter
        const libBtn = document.querySelector('[data-tab="library"]');
        if (libBtn) libBtn.click();
        const dropdown = document.querySelector('#mood-filter-dropdown');
        if (dropdown) {
          dropdown.value = mood;
          dropdown.dispatchEvent(new Event('change'));
        }
      });
      break;

    case 'playlists':
      renderPlaylistStudio(viewContainer, appState.songs);
      break;

    case 'batch':
      renderBatchStation(viewContainer, () => loadSongs());
      break;

    default:
      renderLibraryTable(viewContainer, appState.songs, () => loadSongs());
  }
}

// Bootstrap once DOM is ready
document.addEventListener('DOMContentLoaded', initApp);
