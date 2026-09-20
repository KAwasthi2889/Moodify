/**
 * Moodify Batch Station Component
 * Handles streaming multipart uploads, client-side constraint pre-validation,
 * and real-time worker queue telemetry monitoring.
 */

import { MoodifyAPI } from '../api/endpoints.js';
import { showToast, executeOptimisticAction } from './Toast.js';

const ALLOWED_EXTENSIONS = ['mp3', 'flac', 'wav', 'ogg', 'm4a', 'aac', 'opus'];
const MAX_FILE_SIZE_BYTES = 35 * 1024 * 1024; // 35 MB

let queuePollInterval = null;

export function renderBatchStation(container, onUploadSuccess) {
  if (!container) return;

  container.innerHTML = `
    <div class="glass-panel" style="margin-bottom:24px;">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" style="color:var(--color-action);">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
            </svg>
            Batch Ingest & Worker Queue Station
          </h2>
          <p class="panel-subtitle">Streaming multipart ingestion with background parallel feature extraction</p>
        </div>

        <!-- Progressive Disclosure Constraint Affordance -->
        <div class="constraint-affordance" title="Per-file ceiling: 35MB. Aggregate session upload: Unlimited. Formats: FLAC, MP3, WAV, OGG, M4A, AAC.">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
          <span>Max 35MB/file • Audio Formats Only</span>
        </div>
      </div>

      <!-- Drag & Drop Zone -->
      <div class="dropzone" id="batch-dropzone">
        <div class="dropzone-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/><path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/></svg>
        </div>
        <div>
          <div class="dropzone-text">Drag & drop audio library tracks or click to browse</div>
          <div class="dropzone-subtext">Streaming upload directly to storage with automatic session tagging</div>
        </div>
        <input type="file" id="batch-file-input" multiple accept=".mp3,.flac,.wav,.ogg,.m4a,.aac,.opus" style="display:none;" />
      </div>

      <!-- Queue Telemetry & Trigger Bar -->
      <div class="queue-strip">
        <div style="display:flex; align-items:center; gap:20px;">
          <div>
            <span style="font-size:11px; color:var(--color-text-muted); text-transform:uppercase; font-weight:600;">Worker Queue</span>
            <div style="display:flex; align-items:center; gap:8px; margin-top:3px;">
              <span class="mono-num" id="queue-active-count" style="font-size:16px; font-weight:700; color:var(--color-action);">0 in flight</span>
              <span style="color:var(--color-text-muted);">•</span>
              <span class="mono-num" id="queue-pending-count" style="font-size:13px; color:var(--color-text-secondary);">0 pending</span>
            </div>
          </div>
        </div>

        <button class="btn btn-primary btn-sm" id="btn-trigger-batch-analyze">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="5 3 19 12 5 21 5 3"/></svg>
          Analyze All Pending
        </button>
      </div>

      <div class="progress-bar-container">
        <div class="progress-bar-fill" id="queue-progress-bar" style="width: 100%;"></div>
      </div>
    </div>
  `;

  const dropzone = container.querySelector('#batch-dropzone');
  const fileInput = container.querySelector('#batch-file-input');

  dropzone.addEventListener('click', () => fileInput.click());

  dropzone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropzone.classList.add('drag-over');
  });

  dropzone.addEventListener('dragleave', () => {
    dropzone.classList.remove('drag-over');
  });

  dropzone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropzone.classList.remove('drag-over');
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFiles(e.dataTransfer.files);
    }
  });

  fileInput.addEventListener('change', () => {
    if (fileInput.files && fileInput.files.length > 0) {
      handleFiles(fileInput.files);
    }
  });

  // Client-side pre-validation & upload execution
  async function handleFiles(fileList) {
    const validFiles = [];
    for (const f of fileList) {
      const ext = f.name.split('.').pop()?.toLowerCase();
      if (!ALLOWED_EXTENSIONS.includes(ext)) {
        showToast(`Skipping ${f.name}: unsupported audio format (.${ext})`, 'error');
        continue;
      }
      if (f.size > MAX_FILE_SIZE_BYTES) {
        showToast(`Skipping ${f.name}: exceeds 35MB file ceiling (${(f.size / (1024*1024)).toFixed(1)}MB)`, 'error');
        continue;
      }
      validFiles.push(f);
    }

    if (validFiles.length === 0) return;

    showToast(`Uploading ${validFiles.length} audio files...`, 'success');
    dropzone.classList.add('loading');

    try {
      const res = await MoodifyAPI.batchUpload(validFiles);
      showToast(`Successfully ingested ${res.uploaded_count || validFiles.length} tracks!`, 'success');
      if (onUploadSuccess) onUploadSuccess(res.songs);
    } catch (err) {
      showToast('Batch upload error: ' + err.message, 'error');
    } finally {
      dropzone.classList.remove('loading');
      fileInput.value = '';
    }
  }

  // Trigger batch parallel analysis
  const triggerBtn = container.querySelector('#btn-trigger-batch-analyze');
  triggerBtn.addEventListener('click', async (e) => {
    await executeOptimisticAction({
      button: e.currentTarget,
      apiCall: () => MoodifyAPI.batchAnalyze(''),
      successMessage: 'Queued unanalyzed tracks for parallel extraction'
    });
    pollQueue();
  });

  // Start polling queue status
  pollQueue();
  clearInterval(queuePollInterval);
  queuePollInterval = setInterval(pollQueue, 5000);

  async function pollQueue() {
    try {
      const res = await MoodifyAPI.getBatchStatus();
      const stats = res?.queue_stats;
      if (stats) {
        const activeElem = container.querySelector('#queue-active-count');
        const pendingElem = container.querySelector('#queue-pending-count');
        if (activeElem) activeElem.textContent = `${stats.in_flight || 0} in flight`;
        if (pendingElem) pendingElem.textContent = `${stats.pending || 0} pending`;
      }
    } catch (e) {
      // Ignore polling hiccups
    }
  }
}
