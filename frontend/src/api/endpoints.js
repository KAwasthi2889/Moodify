/**
 * Moodify API Endpoints Service
 * Strongly-typed endpoint wrappers communicating via hybrid client.
 */

import { hybridRequest, inMemoryStore, apiState } from './client.js';

export const MoodifyAPI = {
  /**
   * Pings server health
   */
  async getHealth() {
    return hybridRequest('/api/v1/health', { method: 'GET' }, null, () => ({
      status: 'ok',
      db: apiState.isLive ? 'connected' : 'mock_active',
      storage: 'ready',
    }));
  },

  /**
   * Retrieves library songs
   */
  async listSongs(sessionID = '', status = '', limit = 50, offset = 0) {
    const params = new URLSearchParams();
    if (sessionID) params.set('session_id', sessionID);
    if (status) params.set('status', status);
    if (limit) params.set('limit', String(limit));
    if (offset) params.set('offset', String(offset));

    const query = params.toString() ? `?${params.toString()}` : '';
    return hybridRequest(`/api/v1/songs${query}`, { method: 'GET' }, 'songs');
  },

  /**
   * Single file upload
   */
  async uploadSong(file, sessionID = '') {
    if (apiState.isLive) {
      const formData = new FormData();
      formData.append('file', file);
      if (sessionID) formData.append('session_id', sessionID);

      try {
        const res = await fetch('/api/v1/songs/upload', {
          method: 'POST',
          body: formData,
        });
        if (res.ok) return await res.json();
      } catch (e) {
        console.warn('Upload live error, falling back to mock');
      }
    }

    // Mock upload response
    const newSong = {
      id: `song-${Date.now()}`,
      filename: file.name,
      original_name: file.name,
      format: file.name.split('.').pop() || 'mp3',
      size_bytes: file.size,
      title: file.name.replace(/\.[^/.]+$/, ''),
      artist: 'Unknown Artist',
      album: 'Unassigned',
      inferred_genre: 'Electronic',
      matched_moods: ['chill'],
      tempo_bpm: 110.0,
      key_signature: 'C Major',
      duration_sec: 180.0,
      valence: 0.6,
      energy: 0.65,
      danceability: 0.6,
      status: 'uploaded',
      created_at: new Date().toISOString()
    };
    inMemoryStore.songs.unshift(newSong);
    return { status: 'created', song: newSong };
  },

  /**
   * Batch Upload
   */
  async batchUpload(files, sessionID = '') {
    if (apiState.isLive) {
      const formData = new FormData();
      for (const f of files) {
        formData.append('files', f);
      }
      try {
        const res = await fetch(`/api/v1/songs/batch/upload${sessionID ? `?session_id=${sessionID}` : ''}`, {
          method: 'POST',
          body: formData,
        });
        if (res.ok) return await res.json();
      } catch (e) {
        console.warn('Batch upload live error, using mock');
      }
    }

    const uploaded = Array.from(files).map((f, i) => ({
      id: `mock-batch-${Date.now()}-${i}`,
      filename: f.name,
      original_name: f.name,
      format: f.name.split('.').pop() || 'mp3',
      size_bytes: f.size,
      title: f.name.replace(/\.[^/.]+$/, ''),
      artist: 'Batch Ingest',
      album: 'Library',
      inferred_genre: 'Various',
      matched_moods: ['chill', 'energetic'],
      tempo_bpm: 100 + Math.floor(Math.random() * 40),
      key_signature: 'A Minor',
      duration_sec: 210,
      valence: 0.5,
      energy: 0.7,
      danceability: 0.6,
      status: 'uploaded',
      created_at: new Date().toISOString()
    }));

    inMemoryStore.songs.unshift(...uploaded);
    return {
      status: 'ok',
      session_id: sessionID || `sess-${Date.now()}`,
      uploaded_count: uploaded.length,
      failed_count: 0,
      songs: uploaded
    };
  },

  /**
   * Batch Analyze
   */
  async batchAnalyze(sessionID, songIDs = []) {
    return hybridRequest('/api/v1/songs/batch/analyze', {
      method: 'POST',
      body: JSON.stringify({ session_id: sessionID, song_ids: songIDs, limit: 50 })
    }, null, () => {
      inMemoryStore.songs.forEach(s => s.status = 'analyzed');
      return {
        status: 'ok',
        queued_count: songIDs.length || inMemoryStore.songs.length,
        message: 'Songs queued for parallel feature extraction'
      };
    });
  },

  /**
   * Batch Queue Status
   */
  async getBatchStatus(sessionID = '') {
    return hybridRequest(`/api/v1/songs/batch/status?session_id=${sessionID}`, {
      method: 'GET'
    }, null, () => ({
      status: 'ok',
      session_id: sessionID,
      queue_stats: inMemoryStore.queueStats
    }));
  },

  /**
   * Fingerprint & identify track
   */
  async identifySong(id) {
    return hybridRequest(`/api/v1/songs/${id}/identify`, { method: 'POST' }, null, () => ({
      status: 'ok',
      identified: true,
      metadata: {
        title: 'Identified Audio Track',
        artist: 'AcoustID Match',
        album: 'Studio Master',
        year: 2024,
        genre: 'Electronic / Ambient'
      }
    }));
  },

  /**
   * Save song metadata
   */
  async saveMetadata(id, meta) {
    return hybridRequest(`/api/v1/songs/${id}/metadata`, {
      method: 'PUT',
      body: JSON.stringify(meta)
    }, null, () => {
      const s = inMemoryStore.songs.find(item => item.id === id);
      if (s) Object.assign(s, meta);
      return { status: 'ok', updated: true };
    });
  },

  /**
   * Embed metadata tags into audio file
   */
  async embedTags(id) {
    return hybridRequest(`/api/v1/songs/${id}/embed`, { method: 'POST' }, null, () => ({
      status: 'ok',
      message: 'ID3 tags embedded successfully'
    }));
  },

  /**
   * Rename file based on tags
   */
  async renameSong(id) {
    return hybridRequest(`/api/v1/songs/${id}/rename`, { method: 'POST' }, null, () => ({
      status: 'ok',
      message: 'File renamed on disk to matching artist and title'
    }));
  },

  /**
   * Run Python audio feature extraction
   */
  async analyzeSong(id) {
    return hybridRequest(`/api/v1/songs/${id}/analyze`, { method: 'POST' }, null, () => {
      const s = inMemoryStore.songs.find(item => item.id === id);
      if (s) s.status = 'analyzed';
      return {
        status: 'ok',
        features: {
          tempo_bpm: 120.0,
          key_signature: 'C Major',
          danceability: 0.75,
          energy: 0.82,
          valence: 0.68,
          matched_moods: ['energetic', 'euphoric']
        }
      };
    });
  },

  /**
   * Get audio features
   */
  async getSongFeatures(id) {
    return hybridRequest(`/api/v1/songs/${id}/features`, { method: 'GET' }, null, () => {
      const s = inMemoryStore.songs.find(item => item.id === id);
      return {
        status: 'ok',
        features: s || inMemoryStore.songs[0]
      };
    });
  },

  /**
   * Fetch synced/plain lyrics and RoBERTa emotion embedding
   */
  async syncLyrics(id, payload = {}) {
    return hybridRequest(`/api/v1/songs/${id}/lyrics/sync`, {
      method: 'POST',
      body: JSON.stringify(payload)
    }, null, () => ({
      status: 'ok',
      lyrics: inMemoryStore.lyricsSample
    }));
  },

  /**
   * Get lyrics
   */
  async getSongLyrics(id) {
    return hybridRequest(`/api/v1/songs/${id}/lyrics`, { method: 'GET' }, null, () => ({
      status: 'ok',
      lyrics: inMemoryStore.lyricsSample
    }));
  },

  /**
   * Mood clusters overview
   */
  async getMoodClusters() {
    return hybridRequest('/api/v1/songs/clusters', { method: 'GET' }, null, () => ({
      status: 'ok',
      clusters: inMemoryStore.moodClusters
    }));
  },

  /**
   * Find nearest neighbor tracks
   */
  async getSimilarSongs(id, mode = 'multimodal', threshold = 0.65, limit = 10) {
    return hybridRequest(`/api/v1/songs/${id}/similar?mode=${mode}&threshold=${threshold}&limit=${limit}`, {
      method: 'GET'
    }, null, () => {
      const neighbors = inMemoryStore.songs
        .filter(s => s.id !== id)
        .map(s => ({
          ...s,
          similarity: +(0.72 + Math.random() * 0.25).toFixed(2)
        }));
      return { status: 'ok', similar_songs: neighbors };
    });
  },

  /**
   * Generate curated playlist
   */
  async generatePlaylist(payload = {}) {
    return hybridRequest('/api/v1/playlists/generate', {
      method: 'POST',
      body: JSON.stringify(payload)
    }, null, () => {
      const tracks = inMemoryStore.songs.slice(0, payload.limit || 10).map(s => ({
        id: s.id,
        title: s.title,
        artist: s.artist,
        album: s.album,
        inferred_genre: s.inferred_genre,
        matched_moods: s.matched_moods,
        duration_sec: s.duration_sec,
        tempo_bpm: s.tempo_bpm,
        download_url: `/api/v1/songs/${s.id}/download`
      }));
      return {
        status: 'ok',
        title: payload.title || (payload.mood ? `Moodify — ${payload.mood} Vibe` : 'Moodify Curated Mix'),
        track_count: tracks.length,
        tracks
      };
    });
  },

  /**
   * Delete track
   */
  async deleteSong(id) {
    return hybridRequest(`/api/v1/songs/${id}`, { method: 'DELETE' }, null, () => {
      inMemoryStore.songs = inMemoryStore.songs.filter(s => s.id !== id);
      return { status: 'ok', message: 'song deleted' };
    });
  },

  /**
   * Audio stream / download URL
   */
  getDownloadUrl(id) {
    return `/api/v1/songs/${id}/download`;
  }
};
