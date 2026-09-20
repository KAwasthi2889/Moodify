/**
 * Moodify Main Application Bootstrap & Client Router
 * Switches smoothly between the Commercial Landing Showcase and the Moodify Workshop.
 */

import './styles/tokens.css';
import './styles/base.css';
import './styles/components.css';

import { initScene } from './scene3d/scene.js';
import { checkBackendHealth } from './api/client.js';
import { initAudioPlayer } from './components/AudioPlayer.js';
import { initSessionDrawer, openSessionDrawer } from './components/SessionDrawer.js';
import { renderLandingPage } from './components/LandingPage.js';
import { renderWorkshopPage } from './components/WorkshopPage.js';

let currentView = 'landing';

export function navigateTo(viewName) {
  currentView = viewName;
  const viewContainer = document.getElementById('tab-view-container');
  if (!viewContainer) return;

  window.scrollTo({ top: 0, behavior: 'smooth' });

  if (viewName === 'workshop') {
    renderWorkshopPage(viewContainer, navigateTo);
  } else {
    renderLandingPage(viewContainer, navigateTo);
  }
}

async function initApp() {
  // 1. Initialize Background 3D Procedural Soundwave Canvas
  const canvas = document.getElementById('webgl-canvas');
  initScene(canvas);

  // 2. Initialize Docked Audio Player
  initAudioPlayer();

  // 3. Initialize Session Drawer
  const drawerMount = document.getElementById('session-drawer-mount');
  initSessionDrawer(drawerMount);

  // 4. Check Backend Connectivity
  await checkBackendHealth();

  // 5. Setup Hamburger Menu Drawer Event
  const hamburgerBtn = document.getElementById('btn-open-session-drawer');
  hamburgerBtn?.addEventListener('click', () => {
    openSessionDrawer();
  });

  // 6. Header navigation interactions
  const brandEl = document.getElementById('header-brand');
  brandEl?.addEventListener('click', () => {
    navigateTo('landing');
  });

  const workshopNavBtn = document.getElementById('nav-btn-workshop');
  workshopNavBtn?.addEventListener('click', () => {
    navigateTo('workshop');
  });

  const pipelineLink = document.getElementById('nav-link-pipeline');
  pipelineLink?.addEventListener('click', (e) => {
    if (currentView !== 'landing') {
      e.preventDefault();
      navigateTo('landing');
      setTimeout(() => {
        document.getElementById('architecture')?.scrollIntoView({ behavior: 'smooth' });
      }, 100);
    }
  });

  const hyperspaceLink = document.getElementById('nav-link-hyperspace');
  hyperspaceLink?.addEventListener('click', (e) => {
    if (currentView !== 'landing') {
      e.preventDefault();
      navigateTo('landing');
      setTimeout(() => {
        document.getElementById('hyperspace')?.scrollIntoView({ behavior: 'smooth' });
      }, 100);
    }
  });

  // 7. Initial View Render (Landing)
  navigateTo('landing');
}

// Bootstrap once DOM is loaded
document.addEventListener('DOMContentLoaded', initApp);
