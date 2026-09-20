/**
 * Moodify API Endpoints Service
 * Zero dummy data: strictly interacts with the Go backend API.
 */

import { apiFetch, getSessionId } from './client.js';

export const MoodifyAPI = {
  /**
   * Health status check
   */
  async getHealth() {
    return apiFetch('/api/v1/health', { method: 'GET' });
  },

  /**
   * Lists songs for the current session
   */
  async listSongs(search = '', limit = 100, offset = 0) {
    const sessionId = getSessionId();
    const params = new URLSearchParams();
    if (sessionId) params.set('session_id', sessionId);
    if (search) params.set('search', search);
    if (limit) params.set('limit', String(limit));
    if (offset) params.set('offset', String(offset));

    const query = params.toString() ? `?${params.toString()}` : '';
    return apiFetch(`/api/v1/songs${query}`, { method: 'GET' });
  },

  /**
   * Batch Upload audio files to Go backend
   * Supports streaming multipart upload of multiple files
   */
  async batchUpload(files) {
    const formData = new FormData();
    const sessionId = getSessionId();
    formData.append('session_id', sessionId);

    for (const file of files) {
      formData.append('files', file);
    }

    return apiFetch('/api/v1/songs/batch/upload', {
      method: 'POST',
      body: formData
    });
  },

  /**
   * Fingerprints audio with Chromaprint and automatically identifies/corrects metadata via AcoustID + MusicBrainz
   * POST /api/v1/songs/{id}/identify?auto_save=true
   */
  async identifySong(songId, autoSave = true) {
    return apiFetch(`/api/v1/songs/${songId}/identify?auto_save=${autoSave}`, {
      method: 'POST'
    });
  },

  /**
   * Updates/corrects metadata manually for an individual song
   * PUT /api/v1/songs/{id}/metadata
   */
  async updateSongMetadata(songId, metadata) {
    return apiFetch(`/api/v1/songs/${songId}/metadata`, {
      method: 'PUT',
      body: JSON.stringify(metadata)
    });
  },

  /**
   * Enqueues batch analysis (Tier 2 64-D neural moodifying)
   * POST /api/v1/songs/batch/analyze
   */
  async batchAnalyze(songIds = []) {
    const sessionId = getSessionId();
    return apiFetch('/api/v1/songs/batch/analyze', {
      method: 'POST',
      body: JSON.stringify({
        session_id: sessionId,
        song_ids: songIds,
        limit: songIds.length || 100
      })
    });
  },

  /**
   * Polls asynchronous queue status for batch analysis
   * GET /api/v1/songs/batch/status
   */
  async getBatchStatus() {
    const sessionId = getSessionId();
    return apiFetch(`/api/v1/songs/batch/status?session_id=${encodeURIComponent(sessionId)}`, {
      method: 'GET'
    });
  },

  /**
   * Gets audio stream URL for a song
   */
  getAudioStreamUrl(songId) {
    const sessionId = getSessionId();
    return `/api/v1/songs/${songId}/audio?session_id=${encodeURIComponent(sessionId)}`;
  }
};
