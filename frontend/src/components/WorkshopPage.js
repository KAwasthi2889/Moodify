/**
 * Moodify Workshop & Batch Ingestion Studio
 * High-aesthetic upload workspace supporting drag-and-drop batch ingestion,
 * dynamic 'Upload More Files' state transition, automatic metadata correction,
 * and live real-time visual progress tracking for 64-D neural Moodify queue analysis.
 */

import { MoodifyAPI } from '../api/endpoints.js';
import { openMetadataModal } from './MetadataModal.js';
import { playTrack } from './AudioPlayer.js';

let uploadedSongs = [];
let isUploading = false;
let isAutoTagging = false;
let isMoodifying = false;
let statusPollInterval = null;

export async function renderWorkshopPage(container, onNavigate) {
  if (!container) return;

  // Fetch already uploaded session songs from the Go backend
  try {
    const res = await MoodifyAPI.listSongs();
    uploadedSongs = (res && res.songs) ? res.songs : [];
  } catch (err) {
    console.warn('Could not fetch existing songs:', err);
    uploadedSongs = [];
  }

  function render() {
    const hasSongs = uploadedSongs.length > 0;
    const uploadBtnText = hasSongs ? 'Upload More Files' : 'Upload Files';
    const identifyingCount = uploadedSongs.filter(s => s.status === 'identifying').length;
    const queuedCount = uploadedSongs.filter(s => s.status === 'queued').length;
    const analyzingCount = uploadedSongs.filter(s => s.status === 'analyzing').length;
    const readyCount = uploadedSongs.filter(s => s.status === 'ready').length;
    const totalCount = uploadedSongs.length;
    const unanalyzedCount = uploadedSongs.filter(s => s.status !== 'ready').length;
    const percentReady = totalCount > 0 ? Math.round((readyCount / totalCount) * 100) : 0;
    const isProcessing = queuedCount > 0 || analyzingCount > 0 || isMoodifying;

    container.innerHTML = `
      <div class="workshop-page">
        <!-- ── Workshop Header & Navigation ────────────────────── -->
        <div class="workshop-header">
          <div class="workshop-header-left">
            <button class="btn-back-nav" id="btn-workshop-back" title="Back to Overview">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="15 18 9 12 15 6"/></svg>
              <span>Overview</span>
            </button>
            <div class="workshop-title-group">
              <h1 class="workshop-title">
                Moodify <span class="gradient-text">Workshop</span>
              </h1>
              <span class="workshop-subtitle">Whole-Library Batch Ingestion & 64-D Hyperspace Analysis</span>
            </div>
          </div>
        </div>

        <!-- ── Batch Upload Dropzone ────────────────────────────── -->
        <section class="glass-panel dropzone-card ${isUploading ? 'uploading' : ''}" id="dropzone-area">
          <input type="file" id="file-input-element" multiple accept=".mp3,.wav,.flac,.m4a,.opus,audio/*" style="display: none;" />
          
          <div class="dropzone-content">
            <div class="dropzone-icon-wrapper">
              ${isUploading ? `
                <div class="spinner-large"></div>
              ` : `
                <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="17 8 12 3 7 8"/>
                  <line x1="12" y1="3" x2="12" y2="15"/>
                </svg>
              `}
            </div>

            <div class="dropzone-text-group">
              <h3 class="dropzone-heading">
                ${isUploading ? 'Streaming Audio Files to Server...' : 'Drag & Drop Audio Files Here'}
              </h3>
              <p class="dropzone-sub">
                ${isUploading ? 'Writing multipart streams directly to Go storage' : 'Supports MP3, FLAC, WAV, M4A, OPUS (up to 700 files per batch, max 35MB each)'}
              </p>
            </div>

            <button class="btn-hero-primary btn-upload-trigger" id="btn-trigger-upload" ${isUploading ? 'disabled' : ''}>
              ${isUploading ? `
                <span class="spinner"></span> Uploading...
              ` : `
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/><path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/></svg>
                <span>${uploadBtnText}</span>
              `}
            </button>
          </div>

          <div class="upload-progress-strip" id="upload-progress-strip" style="display: ${isUploading ? 'block' : 'none'};">
            <div class="progress-bar-container">
              <div class="progress-bar-fill animated-glow" style="width: 100%;"></div>
            </div>
          </div>
        </section>

        <!-- ── Uploaded Tracks Action Bar ──────────────────────── -->
        ${hasSongs ? `
          <section class="workshop-actions-bar">
            <div class="actions-summary">
              <span class="summary-count mono-num">${totalCount}</span>
              <span class="summary-label">Uploaded Tracks</span>
              <span class="summary-sep">•</span>
              
              ${isProcessing ? `
                <span class="summary-status-pill analyzing">
                  <span class="spinner-sm"></span> 
                  Analyzing: <strong class="mono-num">${readyCount}/${totalCount} Ready (${percentReady}%)</strong>
                </span>
              ` : identifyingCount > 0 ? `
                <span class="summary-status-pill identifying">
                  <span class="spinner-sm"></span> Auto-Tagging Metadata (${identifyingCount} remaining)
                </span>
              ` : unanalyzedCount === 0 ? `
                <span class="summary-status-pill ready">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg>
                  All ${totalCount} Tracks 64-D Indexed
                </span>
              ` : `
                <span class="summary-status-pill pending">${unanalyzedCount} Ready to Moodify</span>
              `}
            </div>

            <div class="actions-buttons">
              <button class="btn-hero-primary btn-moodify-batch ${isProcessing ? 'loading' : ''}" id="btn-moodify-batch" ${unanalyzedCount === 0 || isProcessing || identifyingCount > 0 ? 'disabled' : ''}>
                ${isProcessing ? `
                  <span class="spinner"></span> 
                  <span>Analyzing (${percentReady}%)...</span>
                ` : `
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
                  <span>Moodify Uploaded List</span>
                `}
              </button>
            </div>

            ${isProcessing ? `
              <div class="batch-live-telemetry">
                <div class="telemetry-info-row">
                  <span class="telemetry-detail">
                    Neural DSP Worker Pool: 
                    <strong class="color-action">${analyzingCount} Active</strong> • 
                    <strong class="color-muted">${queuedCount} In Queue</strong> • 
                    <strong class="color-success">${readyCount} Completed</strong>
                  </span>
                  <span class="telemetry-percent mono-num">${percentReady}%</span>
                </div>
                <div class="progress-bar-container">
                  <div class="progress-bar-fill animated-glow" style="width: ${percentReady}%;"></div>
                </div>
              </div>
            ` : ''}
          </section>

          <!-- ── Live Uploaded Tracks Table ──────────────────────── -->
          <section class="glass-panel uploaded-list-panel">
            <div class="panel-header">
              <div class="panel-title-group">
                <h3 class="panel-title">Uploaded Tracks</h3>
                <span class="panel-subtitle">Tracks automatically auto-tag via Chromaprint & calculate 64-D vectors in pgvector</span>
              </div>
            </div>

            <div class="tracks-table-container">
              <table class="workshop-table">
                <thead>
                  <tr>
                    <th style="width: 40px;">#</th>
                    <th style="width: 44px;">Play</th>
                    <th>Track Title & Artist</th>
                    <th>Format</th>
                    <th>Size</th>
                    <th>Processing Status</th>
                    <th style="text-align: right;">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  ${uploadedSongs.map((song, index) => {
                    const status = song.status || 'uploaded';
                    const displayTitle = song.title || song.original_name || song.filename || 'Untitled';
                    const displayArtist = song.artist || 'Unknown Artist';
                    const format = (song.format || 'mp3').toUpperCase();
                    const sizeMb = song.size_bytes ? (song.size_bytes / (1024 * 1024)).toFixed(1) + ' MB' : '—';

                    let badgeContent = escapeHtml(status);
                    let badgeClass = status;

                    if (status === 'ready') {
                      badgeContent = `
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" style="margin-right:4px;"><polyline points="20 6 9 17 4 12"/></svg>
                        Ready (64-D)
                      `;
                    } else if (status === 'analyzing') {
                      badgeContent = `<span class="spinner-sm"></span> DSP Analyzing`;
                    } else if (status === 'queued') {
                      badgeContent = `<span class="spinner-sm"></span> Queued`;
                    } else if (status === 'identifying') {
                      badgeContent = `<span class="spinner-sm"></span> Auto-Tagging`;
                    } else if (status === 'tagged') {
                      badgeContent = `Tagged`;
                    }

                    return `
                      <tr class="track-row ${status}" data-id="${escapeHtml(song.id)}">
                        <td class="cell-index mono-num">${index + 1}</td>
                        <td class="cell-play">
                          <button class="btn-row-play" data-action="play" data-id="${escapeHtml(song.id)}" title="Play Audio">
                            <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="6 3 20 12 6 21 6 3"/></svg>
                          </button>
                        </td>
                        <td class="cell-main">
                          <div class="track-primary-title" id="title-${escapeHtml(song.id)}">${escapeHtml(displayTitle)}</div>
                          <div class="track-secondary-artist" id="artist-${escapeHtml(song.id)}">${escapeHtml(displayArtist)} ${song.album ? `• ${escapeHtml(song.album)}` : ''}</div>
                        </td>
                        <td class="cell-format">
                          <span class="format-pill">${escapeHtml(format)}</span>
                        </td>
                        <td class="cell-size mono-num">${escapeHtml(sizeMb)}</td>
                        <td class="cell-status">
                          <span class="status-badge ${badgeClass}">
                            ${badgeContent}
                          </span>
                        </td>
                        <td class="cell-actions" style="text-align: right;">
                          <a class="btn-row-download" href="/api/v1/songs/${escapeHtml(song.id)}/download?download=true" download="${escapeHtml(song.filename || 'track')}" title="Download Track">
                            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                            <span>Download</span>
                          </a>
                          <button class="btn-correct-meta" data-action="edit-meta" data-id="${escapeHtml(song.id)}" title="Manually Adjust Metadata">
                            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
                            <span>Edit Tags</span>
                          </button>
                        </td>
                      </tr>
                    `;
                  }).join('')}
                </tbody>
              </table>
            </div>
          </section>
        ` : `
          <div class="workshop-empty-hint">
            <p>Ready for ingestion. Select multiple audio files or drop an entire album above.</p>
          </div>
        `}
      </div>
    `;

    bindEvents();
  }

  function bindEvents() {
    // Back navigation
    const backBtn = container.querySelector('#btn-workshop-back');
    backBtn?.addEventListener('click', () => {
      if (statusPollInterval) {
        clearInterval(statusPollInterval);
        statusPollInterval = null;
      }
      if (onNavigate) onNavigate('landing');
    });

    // File input trigger
    const fileInput = container.querySelector('#file-input-element');
    const triggerBtn = container.querySelector('#btn-trigger-upload');
    const dropzone = container.querySelector('#dropzone-area');

    triggerBtn?.addEventListener('click', () => {
      fileInput?.click();
    });

    fileInput?.addEventListener('change', (e) => {
      const files = Array.from(e.target.files || []);
      if (files.length > 0) {
        handleUploadFiles(files);
      }
    });

    // Drag and drop handlers
    dropzone?.addEventListener('dragover', (e) => {
      e.preventDefault();
      dropzone.classList.add('drag-active');
    });

    dropzone?.addEventListener('dragleave', () => {
      dropzone.classList.remove('drag-active');
    });

    dropzone?.addEventListener('drop', (e) => {
      e.preventDefault();
      dropzone.classList.remove('drag-active');
      const files = Array.from(e.dataTransfer?.files || []);
      if (files.length > 0) {
        handleUploadFiles(files);
      }
    });

    // Play track handlers
    const playBtns = container.querySelectorAll('button[data-action="play"]');
    playBtns.forEach(btn => {
      btn.addEventListener('click', () => {
        const id = btn.dataset.id;
        const song = uploadedSongs.find(s => s.id === id);
        if (song) playTrack(song);
      });
    });

    // Edit metadata handlers (manual override)
    const editBtns = container.querySelectorAll('button[data-action="edit-meta"]');
    editBtns.forEach(btn => {
      btn.addEventListener('click', () => {
        const id = btn.dataset.id;
        const song = uploadedSongs.find(s => s.id === id);
        if (song) {
          openMetadataModal(song, (updatedSong) => {
            const idx = uploadedSongs.findIndex(s => s.id === id);
            if (idx !== -1) {
              uploadedSongs[idx] = { ...uploadedSongs[idx], ...updatedSong };
              render();
            }
          });
        }
      });
    });

    // Moodify batch trigger
    const moodifyBtn = container.querySelector('#btn-moodify-batch');
    moodifyBtn?.addEventListener('click', handleMoodifyBatch);
  }

  async function handleUploadFiles(files) {
    if (files.length === 0) return;
    isUploading = true;
    render();

    try {
      const res = await MoodifyAPI.batchUpload(files);
      let newlyUploaded = [];
      if (res && res.songs && res.songs.length > 0) {
        newlyUploaded = res.songs;
        uploadedSongs = [...res.songs, ...uploadedSongs];
      } else {
        const listRes = await MoodifyAPI.listSongs();
        uploadedSongs = (listRes && listRes.songs) ? listRes.songs : uploadedSongs;
        newlyUploaded = uploadedSongs;
      }

      isUploading = false;
      render();

      // AUTOMATIC METADATA CORRECTION:
      // Start immediately without waiting for user input!
      autoCorrectMetadata(newlyUploaded);
    } catch (err) {
      alert(`Batch upload failed: ${err.message}`);
      isUploading = false;
      render();
    }
  }

  /**
   * Automatically fingerprints and corrects metadata for uploaded songs
   * Runs in the background without requiring user clicks or waiting for user input
   */
  async function autoCorrectMetadata(songsToTag) {
    if (!songsToTag || songsToTag.length === 0) return;
    isAutoTagging = true;

    // Filter to songs that are untagged
    const targetSongs = songsToTag.filter(s => s.status !== 'ready' && s.status !== 'tagged');
    if (targetSongs.length === 0) {
      isAutoTagging = false;
      return;
    }

    // Mark as identifying
    targetSongs.forEach(s => {
      const idx = uploadedSongs.findIndex(item => item.id === s.id);
      if (idx !== -1) {
        uploadedSongs[idx].status = 'identifying';
      }
    });
    render();

    // Process with controlled concurrency (2 at a time to be polite to MusicBrainz/AcoustID)
    const concurrency = 2;
    let index = 0;

    async function processNext() {
      while (index < targetSongs.length) {
        const song = targetSongs[index++];
        try {
          const result = await MoodifyAPI.identifySong(song.id, true);
          const topMatch = (result && result.matches && result.matches.length > 0) ? result.matches[0] : null;

          const idx = uploadedSongs.findIndex(item => item.id === song.id);
          if (idx !== -1) {
            if (topMatch) {
              uploadedSongs[idx].title = topMatch.title || uploadedSongs[idx].title || song.original_name;
              uploadedSongs[idx].artist = topMatch.artist || uploadedSongs[idx].artist || 'Unknown Artist';
              uploadedSongs[idx].album = topMatch.album || uploadedSongs[idx].album || '';
            }
            uploadedSongs[idx].status = 'tagged';
          }
        } catch (err) {
          console.warn(`Auto-tagging failed for ${song.id}:`, err);
          const idx = uploadedSongs.findIndex(item => item.id === song.id);
          if (idx !== -1) {
            uploadedSongs[idx].status = 'tagged';
          }
        }
        render();
      }
    }

    const workers = [];
    for (let i = 0; i < Math.min(concurrency, targetSongs.length); i++) {
      workers.push(processNext());
    }
    await Promise.all(workers);

    isAutoTagging = false;
    render();
  }

  function startStatusPolling() {
    if (statusPollInterval) return;
    isMoodifying = true;

    statusPollInterval = setInterval(async () => {
      try {
        const listRes = await MoodifyAPI.listSongs();
        if (listRes && listRes.songs) {
          uploadedSongs = listRes.songs;
          render();

          const stillWorking = uploadedSongs.some(s => s.status === 'queued' || s.status === 'analyzing');
          if (!stillWorking) {
            clearInterval(statusPollInterval);
            statusPollInterval = null;
            isMoodifying = false;
            render();
          }
        }
      } catch (err) {
        console.warn('Status polling error:', err);
      }
    }, 2000);
  }

  async function handleMoodifyBatch() {
    const unanalyzedIds = uploadedSongs.filter(s => s.status !== 'ready').map(s => s.id);
    if (unanalyzedIds.length === 0) return;

    isMoodifying = true;

    // Mark songs as analyzing in local UI immediately
    uploadedSongs = uploadedSongs.map(s => {
      if (unanalyzedIds.includes(s.id)) {
        return { ...s, status: 'queued' };
      }
      return s;
    });
    render();

    try {
      await MoodifyAPI.batchAnalyze(unanalyzedIds);
      startStatusPolling();
    } catch (err) {
      alert(`Batch analysis enqueue failed: ${err.message}`);
      isMoodifying = false;
      render();
    }
  }

  // Initial render
  render();

  // If there are existing untagged songs, auto-correct them automatically
  const untaggedExisting = uploadedSongs.filter(s => s.status === 'uploaded');
  if (untaggedExisting.length > 0) {
    autoCorrectMetadata(untaggedExisting);
  }

  // If any songs are queued or analyzing in the database, automatically poll in real-time
  const hasPendingAnalysis = uploadedSongs.some(s => s.status === 'queued' || s.status === 'analyzing');
  if (hasPendingAnalysis) {
    startStatusPolling();
  }
}

function escapeHtml(str) {
  if (!str) return '';
  return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
