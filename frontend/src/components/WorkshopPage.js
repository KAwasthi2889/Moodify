/**
 * Moodify Workshop & Batch Ingestion Studio
 * High-aesthetic upload workspace supporting drag-and-drop batch ingestion,
 * tagged file selection & bulk downloading, song removal, real-time filename/title search,
 * and seamless navigation to the Similarity Studio for ready tracks.
 */

import { MoodifyAPI } from '../api/endpoints.js';
import { openMetadataModal } from './MetadataModal.js';
import { playTrack } from './AudioPlayer.js';
import { promptDeleteModal, showToast, showDuplicateSummaryModal } from './Modal.js';

let uploadedSongs = [];
let selectedSongIds = new Set();
let searchFilter = '';
let isUploading = false;
let isAutoTagging = false;
let isMoodifying = false;
let statusPollInterval = null;
let resolvedDuplicates = new Set();
let hasShownDuplicateModal = false;

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

    // Filter songs by search input (space-insensitive and punctuation-tolerant)
    const filteredSongs = uploadedSongs.filter(song => MoodifyAPI.matchesSongSearch(song, searchFilter));

    const selectedCount = selectedSongIds.size;
    const isAllSelected = filteredSongs.length > 0 && filteredSongs.every(s => selectedSongIds.has(s.id));

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
                  All ${totalCount} Tracks Ready in 64-D Hyperspace
                </span>
              ` : `
                <span class="summary-status-pill pending">${unanalyzedCount} Ready to Moodify</span>
              `}
            </div>

            <div class="actions-buttons">
              ${selectedCount > 0 ? `
                <button class="btn-bulk-download" id="btn-download-selected">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                  <span>${selectedCount === 1 ? 'Download Selected' : `Download Selected (${selectedCount})`}</span>
                </button>
                <button class="btn-bulk-remove" id="btn-remove-selected" title="Remove Selected Tracks">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
                  <span>Remove Selected (${selectedCount})</span>
                </button>
              ` : ''}

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
                <span class="panel-subtitle">Select tagged tracks to download, remove tracks, or find similar songs in 64-D vector space</span>
              </div>

              <!-- Real-time search by filename, title, or artist -->
              <div class="table-search-box">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
                <input type="text" id="workshop-search-input" placeholder="Search by filename, title, artist..." value="${escapeHtml(searchFilter)}" />
                ${searchFilter ? `
                  <button class="btn-clear-search" id="btn-clear-search" title="Clear search">×</button>
                ` : ''}
              </div>
            </div>

            <div class="tracks-table-container">
              <table class="workshop-table">
                <thead>
                  <tr>
                    <th style="width: 36px; text-align: center;">
                      <input type="checkbox" id="select-all-checkbox" class="track-checkbox" ${isAllSelected ? 'checked' : ''} title="Select all filtered songs" />
                    </th>
                    <th style="width: 36px;">#</th>
                    <th style="width: 44px;">Play</th>
                    <th>Track Title & Artist</th>
                    <th>Format</th>
                    <th>Size</th>
                    <th>Processing Status</th>
                    <th style="text-align: right;">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  ${filteredSongs.length > 0 ? filteredSongs.map((song, index) => {
                    const status = song.status || 'uploaded';
                    const displayTitle = MoodifyAPI.getCleanTitle(song);
                    const displayArtist = song.artist || 'Unknown Artist';
                    const format = (song.format || 'mp3').toUpperCase();
                    const isFormatCorrected = MoodifyAPI.isFormatCorrected(song);
                    const sizeMb = song.size_bytes ? (song.size_bytes / (1024 * 1024)).toFixed(1) + ' MB' : '—';
                    const isSelected = selectedSongIds.has(song.id);
                    const isReady = status === 'ready';

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
                      <tr class="track-row ${status} ${isSelected ? 'row-selected' : ''}" data-id="${escapeHtml(song.id)}">
                        <td class="cell-checkbox" style="text-align: center;">
                          <input type="checkbox" class="track-checkbox track-select-cb" data-id="${escapeHtml(song.id)}" ${isSelected ? 'checked' : ''} />
                        </td>
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
                          <span class="format-pill ${isFormatCorrected ? 'format-corrected' : ''}" title="${escapeHtml(isFormatCorrected ? `Detected & auto-corrected format: ${format}` : `Format: ${format}`)}">
                            ${escapeHtml(format)}
                          </span>
                        </td>
                        <td class="cell-size mono-num">${escapeHtml(sizeMb)}</td>
                        <td class="cell-status">
                          <span class="status-badge ${badgeClass}">
                            ${badgeContent}
                          </span>
                        </td>
                        <td class="cell-actions" style="text-align: right;">
                          ${isReady ? `
                            <button class="btn-find-similar" data-action="find-similar" data-id="${escapeHtml(song.id)}" title="Find Similar Songs in 64-D Space">
                              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
                              <span>Find Similar</span>
                            </button>
                          ` : ''}

                          <a class="btn-row-download" href="/api/v1/songs/${escapeHtml(song.id)}/download?download=true" download="${escapeHtml(MoodifyAPI.getCleanFilename(song))}" title="Download: ${escapeHtml(MoodifyAPI.getCleanFilename(song))}">
                            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                            <span>Download</span>
                          </a>

                          <button class="btn-correct-meta" data-action="edit-meta" data-id="${escapeHtml(song.id)}" title="Manually Adjust Metadata">
                            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
                            <span>Edit Tags</span>
                          </button>

                          <button class="btn-row-remove" data-action="remove-song" data-id="${escapeHtml(song.id)}" title="Remove Track from Library">
                            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
                          </button>
                        </td>
                      </tr>
                      ${(song.is_duplicate && !resolvedDuplicates.has(song.id)) ? `
                        <tr class="duplicate-banner-row" data-duplicate-id="${escapeHtml(song.id)}">
                          <td colspan="8">
                            <div class="inline-duplicate-banner">
                              <div class="dup-text-group">
                                <span class="dup-pill">⚠️ Duplicate Audio</span>
                                <span class="dup-msg">Matches an existing track in your library (identical acoustic fingerprint).</span>
                              </div>
                              <div class="dup-action-group">
                                <button class="btn-dup-replace" data-action="dup-replace" data-id="${escapeHtml(song.id)}" data-target="${escapeHtml(song.duplicate_of || '')}">
                                  🔄 Replace Existing
                                </button>
                                <button class="btn-dup-discard" data-action="dup-discard" data-id="${escapeHtml(song.id)}">
                                  ⏭️ Discard
                                </button>
                                <button class="btn-dup-keep" data-action="dup-keep" data-id="${escapeHtml(song.id)}">
                                  ➕ Keep Both
                                </button>
                              </div>
                            </div>
                          </td>
                        </tr>
                      ` : ''}
                    `;
                  }).join('') : `
                    <tr>
                      <td colspan="8" style="text-align: center; padding: 40px; color: var(--color-text-muted);">
                        No songs match your search query: "${escapeHtml(searchFilter)}"
                      </td>
                    </tr>
                  `}
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

    // Search input event
    const searchInput = container.querySelector('#workshop-search-input');
    searchInput?.addEventListener('input', (e) => {
      searchFilter = e.target.value;
      render();
      // Keep focus on search input after render
      const newInput = container.querySelector('#workshop-search-input');
      if (newInput) {
        newInput.focus();
        newInput.setSelectionRange(newInput.value.length, newInput.value.length);
      }
    });

    const clearSearchBtn = container.querySelector('#btn-clear-search');
    clearSearchBtn?.addEventListener('click', () => {
      searchFilter = '';
      render();
    });

    // Checkbox selection: Select All
    const selectAllCb = container.querySelector('#select-all-checkbox');
    selectAllCb?.addEventListener('change', (e) => {
      const isChecked = e.target.checked;
      const filtered = uploadedSongs.filter(song => MoodifyAPI.matchesSongSearch(song, searchFilter));

      filtered.forEach(song => {
        if (isChecked) {
          selectedSongIds.add(song.id);
        } else {
          selectedSongIds.delete(song.id);
        }
      });
      render();
    });

    // Checkbox selection: Individual rows
    const rowCheckboxes = container.querySelectorAll('.track-select-cb');
    rowCheckboxes.forEach(cb => {
      cb.addEventListener('change', (e) => {
        const id = cb.dataset.id;
        if (e.target.checked) {
          selectedSongIds.add(id);
        } else {
          selectedSongIds.delete(id);
        }
        render();
      });
    });

    // Bulk or single download selected songs
    const bulkDownloadBtn = container.querySelector('#btn-download-selected');
    bulkDownloadBtn?.addEventListener('click', async () => {
      if (selectedSongIds.size === 0) return;
      const idsToDownload = Array.from(selectedSongIds);
      
      // If only 1 song is selected, download directly as individual file with clean filename
      if (idsToDownload.length === 1) {
        const singleId = idsToDownload[0];
        const song = uploadedSongs.find(s => s.id === singleId);
        const cleanName = song ? MoodifyAPI.getCleanFilename(song) : 'track.mp3';
        const a = document.createElement('a');
        a.href = `/api/v1/songs/${encodeURIComponent(singleId)}/download?download=true`;
        a.download = cleanName;
        document.body.appendChild(a);
        a.click();
        a.remove();
        showToast(`Downloading "${cleanName}"`, 'success');
        return;
      }

      // If multiple songs selected, stream ZIP archive
      try {
        bulkDownloadBtn.disabled = true;
        const origText = bulkDownloadBtn.innerHTML;
        bulkDownloadBtn.innerHTML = `<span class="spinner-sm"></span> Packaging ZIP...`;
        await MoodifyAPI.batchDownloadZip(idsToDownload);
        showToast(`Downloaded moodify_songs.zip (${idsToDownload.length} tracks)`, 'success');
        bulkDownloadBtn.innerHTML = origText;
      } catch (err) {
        showToast(`Failed to download zip: ${err.message}`, 'error');
      } finally {
        bulkDownloadBtn.disabled = false;
      }
    });

    // Bulk remove selected songs
    const bulkRemoveBtn = container.querySelector('#btn-remove-selected');
    bulkRemoveBtn?.addEventListener('click', () => {
      if (selectedSongIds.size === 0) return;
      const count = selectedSongIds.size;
      const targetLabel = count === 1 ? '1 selected track' : `${count} selected tracks`;
      promptDeleteModal(targetLabel, async () => {
        const ids = Array.from(selectedSongIds);
        for (const id of ids) {
          try {
            await MoodifyAPI.deleteSong(id);
          } catch (err) {
            console.warn(`Failed to delete song ${id}:`, err);
          }
        }
        uploadedSongs = uploadedSongs.filter(s => !selectedSongIds.has(s.id));
        selectedSongIds.clear();
        showToast(`Removed ${count} track${count > 1 ? 's' : ''} from library`, 'info');
        render();
      });
    });

    // Remove song handlers with custom glassmorphic modal
    const removeBtns = container.querySelectorAll('button[data-action="remove-song"]');
    removeBtns.forEach(btn => {
      btn.addEventListener('click', () => {
        const id = btn.dataset.id;
        const song = uploadedSongs.find(s => s.id === id);
        const name = song ? (song.title || song.original_name || song.filename || 'track') : 'track';
        
        promptDeleteModal(name, async () => {
          await MoodifyAPI.deleteSong(id);
          uploadedSongs = uploadedSongs.filter(s => s.id !== id);
          selectedSongIds.delete(id);
          showToast(`Removed "${name}" from library`, 'info');
          render();
        });
      });
    });

    // Inline Duplicate Action Handlers (Approach A)
    container.querySelectorAll('button[data-action="dup-replace"]').forEach(btn => {
      btn.addEventListener('click', async () => {
        const songId = btn.dataset.id;
        const targetId = btn.dataset.target;
        resolvedDuplicates.add(songId);
        if (targetId) {
          try {
            await MoodifyAPI.deleteSong(targetId);
            uploadedSongs = uploadedSongs.filter(s => s.id !== targetId);
            selectedSongIds.delete(targetId);
            showToast('Replaced older version with this track', 'success');
          } catch (e) {
            console.warn(e);
          }
        }
        render();
      });
    });

    container.querySelectorAll('button[data-action="dup-discard"]').forEach(btn => {
      btn.addEventListener('click', async () => {
        const songId = btn.dataset.id;
        resolvedDuplicates.add(songId);
        try {
          await MoodifyAPI.deleteSong(songId);
          uploadedSongs = uploadedSongs.filter(s => s.id !== songId);
          selectedSongIds.delete(songId);
          showToast('Discarded duplicate track', 'info');
        } catch (e) {
          console.warn(e);
        }
        render();
      });
    });

    container.querySelectorAll('button[data-action="dup-keep"]').forEach(btn => {
      btn.addEventListener('click', () => {
        const songId = btn.dataset.id;
        resolvedDuplicates.add(songId);
        showToast('Kept both versions in library', 'info');
        render();
      });
    });

    // Find Similar handler (navigates to Similarity Page)
    const similarBtns = container.querySelectorAll('button[data-action="find-similar"]');
    similarBtns.forEach(btn => {
      btn.addEventListener('click', () => {
        const id = btn.dataset.id;
        const song = uploadedSongs.find(s => s.id === id);
        if (song && onNavigate) {
          if (statusPollInterval) {
            clearInterval(statusPollInterval);
            statusPollInterval = null;
          }
          onNavigate('similarity', { seedSong: song });
        }
      });
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

    // Play track handlers (play button & track row body)
    const playBtns = container.querySelectorAll('button[data-action="play"]');
    playBtns.forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const id = btn.dataset.id;
        const song = uploadedSongs.find(s => s.id === id);
        if (song) playTrack(song, uploadedSongs);
      });
    });

    // Clicking anywhere on the track row body plays the song (unless clicking on interactive inputs/buttons/links)
    const trackRows = container.querySelectorAll('.track-row');
    trackRows.forEach(row => {
      row.addEventListener('click', (e) => {
        if (e.target.closest('input, button, a, .cell-checkbox, .inline-duplicate-banner')) {
          return;
        }
        const id = row.dataset.id;
        const song = uploadedSongs.find(s => s.id === id);
        if (song) playTrack(song, uploadedSongs);
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
      const incomingList = (res && res.successes && res.successes.length > 0) ? res.successes : ((res && res.songs) ? res.songs : []);

      if (incomingList.length > 0) {
        newlyUploaded = incomingList;
        uploadedSongs = [...incomingList, ...uploadedSongs];
      } else {
        const listRes = await MoodifyAPI.listSongs();
        uploadedSongs = (listRes && listRes.songs) ? listRes.songs : uploadedSongs;
      }

      isUploading = false;
      render();

      // Automatically tag only newly uploaded songs
      if (newlyUploaded.length > 0) {
        autoCorrectMetadata(newlyUploaded);
      }
    } catch (err) {
      alert(`Batch upload failed: ${err.message}`);
      isUploading = false;
      render();
    }
  }

  async function autoCorrectMetadata(songsToTag) {
    if (!songsToTag || songsToTag.length === 0) return;
    isAutoTagging = true;

    const targetSongs = songsToTag.filter(s => s.status !== 'ready' && s.status !== 'tagged');
    if (targetSongs.length === 0) {
      isAutoTagging = false;
      return;
    }

    targetSongs.forEach(s => {
      const idx = uploadedSongs.findIndex(item => item.id === s.id);
      if (idx !== -1) {
        uploadedSongs[idx].status = 'identifying';
      }
    });
    render();

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
              uploadedSongs[idx].year = topMatch.release_year || topMatch.year || uploadedSongs[idx].year || '';
              uploadedSongs[idx].genre = topMatch.genre || topMatch.inferred_genre || uploadedSongs[idx].genre || '';
              uploadedSongs[idx].english_title = topMatch.english_title || uploadedSongs[idx].english_title || '';
              uploadedSongs[idx].status = 'tagged';
            } else {
              uploadedSongs[idx].status = 'uploaded';
            }
          }
        } catch (err) {
          console.warn(`Auto-tagging failed for ${song.id}:`, err);
          const idx = uploadedSongs.findIndex(item => item.id === song.id);
          if (idx !== -1) {
            uploadedSongs[idx].status = 'uploaded';
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

    // Refresh library from backend to get updated metadata & duplicate flags
    try {
      const refreshed = await MoodifyAPI.listSongs();
      if (refreshed && refreshed.songs) {
        uploadedSongs = uploadedSongs.map(localSong => {
          const remote = refreshed.songs.find(r => r.id === localSong.id);
          if (!remote) return localSong;
          return {
            ...remote,
            status: localSong.status === 'tagged' ? 'tagged' : remote.status,
            extension_corrected: localSong.extension_corrected || remote.extension_corrected,
            format_warning: localSong.format_warning || remote.format_warning,
            title: localSong.title || remote.title,
            artist: localSong.artist || remote.artist,
            album: localSong.album || remote.album,
            year: localSong.year || remote.year,
            genre: localSong.genre || remote.genre,
            english_title: remote.english_title || localSong.english_title || ''
          };
        });
      }
    } catch (_) {}

    isAutoTagging = false;
    render();

    // Hybrid Approach B: Prompt user with Modal B for any remaining duplicates
    checkUnresolvedDuplicatesModal();
  }

  function checkUnresolvedDuplicatesModal() {
    const unresolved = uploadedSongs.filter(s => s.is_duplicate && !resolvedDuplicates.has(s.id));
    if (unresolved.length > 0 && !hasShownDuplicateModal) {
      hasShownDuplicateModal = true;
      showDuplicateSummaryModal(unresolved, {
        onReplace: async (songId, targetId) => {
          resolvedDuplicates.add(songId);
          if (targetId) {
            try {
              await MoodifyAPI.deleteSong(targetId);
              uploadedSongs = uploadedSongs.filter(s => s.id !== targetId);
              selectedSongIds.delete(targetId);
              showToast('Replaced older version with this track', 'success');
            } catch (_) {}
          }
          render();
        },
        onDiscard: async (songId) => {
          resolvedDuplicates.add(songId);
          try {
            await MoodifyAPI.deleteSong(songId);
            uploadedSongs = uploadedSongs.filter(s => s.id !== songId);
            selectedSongIds.delete(songId);
            showToast('Discarded duplicate track', 'info');
          } catch (_) {}
          render();
        },
        onKeepAll: (allUnresolved) => {
          allUnresolved.forEach(s => resolvedDuplicates.add(s.id));
          showToast('Kept all duplicate tracks in library', 'info');
          render();
        }
      });
    }
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
