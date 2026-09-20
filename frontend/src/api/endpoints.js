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
   * Retrieves stored metadata for an individual song
   * GET /api/v1/songs/{id}/metadata
   */
  async getSongMetadata(songId) {
    return apiFetch(`/api/v1/songs/${songId}/metadata`, {
      method: 'GET'
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
   * Deletes an individual song from storage and database
   * DELETE /api/v1/songs/{id}
   */
  async deleteSong(songId) {
    return apiFetch(`/api/v1/songs/${songId}`, {
      method: 'DELETE'
    });
  },

  /**
   * Finds nearest neighbor similar songs in vector space
   * GET /api/v1/songs/{id}/similar?mode=...&threshold=...&limit=...
   */
  async getSimilarSongs(songId, mode = 'multimodal', threshold = 0.5, limit = 100) {
    const params = new URLSearchParams({
      mode,
      threshold: String(threshold),
      limit: String(limit)
    });
    return apiFetch(`/api/v1/songs/${songId}/similar?${params.toString()}`, {
      method: 'GET'
    });
  },

  /**
   * Generates a curated or dynamic playlist
   * POST /api/v1/playlists/generate
   */
  async generatePlaylist(payload) {
    const sessionId = getSessionId();
    return apiFetch('/api/v1/playlists/generate', {
      method: 'POST',
      body: JSON.stringify({
        session_id: sessionId,
        ...payload
      })
    });
  },

  /**
   * Gets audio stream URL for a song with Range request support.
   * If download is true, sets attachment disposition query.
   */
  getAudioStreamUrl(songId, download = false) {
    return `/api/v1/songs/${songId}/download${download ? '?download=true' : ''}`;
  },

  /**
   * Resolves a clean display track title (without server timestamp suffixes or file extensions).
   */
  getCleanTitle(song) {
    if (!song) return 'Moodify Track';
    let title = song.title || '';
    let englishTitle = song.english_title || '';

    // If englishTitle not directly available, check if title or original_name contains '|'
    if (!englishTitle && title.includes('|')) {
      const parts = title.split('|').map(p => p.trim().replace(/^\((.*)\)$/, '$1').trim());
      if (parts.length >= 2 && parts[0] && parts[1]) {
        title = parts[0];
        englishTitle = parts[1];
      }
    } else if (!englishTitle && song.original_name && song.original_name.includes('|')) {
      const baseOrig = song.original_name.replace(/\.[^/.]+$/, '');
      const parts = baseOrig.split('|').map(p => p.trim().replace(/^\((.*)\)$/, '$1').trim());
      if (parts.length >= 2 && parts[0] && parts[1]) {
        title = title || parts[0];
        englishTitle = parts[1];
      }
    } else if (!englishTitle && /[\u0400-\u04FF\u4E00-\u9FFF\u3040-\u30FF\uAC00-\uD7AF\u0600-\u06FF]/.test(title) && song.original_name) {
      // If title is non-Latin script (Cyrillic, CJK, etc.) and original uploaded filename is Latin/English
      const cleanOrig = song.original_name.replace(/\.[^/.]+$/, '').replace(/_\d{10,}.*$/, '').replace(/_/g, ' ').trim();
      if (/^[a-zA-Z0-9\s\-'.&]+$/.test(cleanOrig) && !/^(track|audio|unknown|untitled)\b/i.test(cleanOrig) && cleanOrig.toLowerCase() !== title.toLowerCase()) {
        englishTitle = cleanOrig;
      }
    }

    const cleanTitle = (title || '').trim().replace(/^\((.*)\)$/, '$1').trim();
    const cleanEng = (englishTitle || '').trim().replace(/^\((.*)\)$/, '$1').trim();

    if (cleanEng && cleanTitle && cleanEng.toLowerCase() !== cleanTitle.toLowerCase()) {
      return `${cleanTitle} | ${cleanEng}`;
    }
    if (cleanTitle) {
      return cleanTitle;
    }
    if (song.original_name) {
      return song.original_name.replace(/\.[^/.]+$/, '');
    }
    if (song.filename) {
      // Strip internal timestamp e.g. "Song_1789927194459733812_1789936449618109140.m4a" -> "Song"
      return song.filename.replace(/_\d{10,}.*$/, '').replace(/\.[^/.]+$/, '');
    }
    return 'Moodify Track';
  },

  /**
   * Resolves a clean filename matching [title | english.ext] or [Title.ext].
   */
  getCleanFilename(song) {
    if (!song) return 'track.mp3';
    const format = (song.format || 'mp3').toLowerCase();
    let title = song.title || '';
    let englishTitle = song.english_title || '';

    if (!englishTitle && title.includes('|')) {
      const parts = title.split('|').map(p => p.trim().replace(/^\((.*)\)$/, '$1').trim());
      if (parts.length >= 2 && parts[0] && parts[1]) {
        title = parts[0];
        englishTitle = parts[1];
      }
    } else if (!englishTitle && song.original_name && song.original_name.includes('|')) {
      const baseOrig = song.original_name.replace(/\.[^/.]+$/, '');
      const parts = baseOrig.split('|').map(p => p.trim().replace(/^\((.*)\)$/, '$1').trim());
      if (parts.length >= 2 && parts[0] && parts[1]) {
        title = title || parts[0];
        englishTitle = parts[1];
      }
    } else if (!englishTitle && /[\u0400-\u04FF\u4E00-\u9FFF\u3040-\u30FF\uAC00-\uD7AF\u0600-\u06FF]/.test(title) && song.original_name) {
      // If title is non-Latin script (Cyrillic, CJK, etc.) and original uploaded filename is Latin/English
      const cleanOrig = song.original_name.replace(/\.[^/.]+$/, '').replace(/_\d{10,}.*$/, '').replace(/_/g, ' ').trim();
      if (/^[a-zA-Z0-9\s\-'.&]+$/.test(cleanOrig) && !/^(track|audio|unknown|untitled)\b/i.test(cleanOrig) && cleanOrig.toLowerCase() !== title.toLowerCase()) {
        englishTitle = cleanOrig;
      }
    }

    const cleanTitle = (title || '').trim().replace(/^\((.*)\)$/, '$1').trim();
    const cleanEng = (englishTitle || '').trim().replace(/^\((.*)\)$/, '$1').trim();

    if (cleanEng && cleanTitle && cleanEng.toLowerCase() !== cleanTitle.toLowerCase()) {
      return `${cleanTitle} | ${cleanEng}.${format}`;
    }
    if (cleanTitle) {
      return `${cleanTitle}.${format}`;
    }
    if (song.original_name) {
      const base = song.original_name.replace(/\.[^/.]+$/, '');
      return `${base}.${format}`;
    }
    if (song.filename) {
      // Strip internal timestamp if present
      const cleaned = song.filename.replace(/_\d{10,}.*$/, '').replace(/\.[^/.]+$/, '');
      return `${cleaned}.${format}`;
    }
    return `track.${format}`;
  },

  /**
   * Downloads multiple songs packaged inside a single streaming zip named moodify_songs.zip.
   */
  async batchDownloadZip(songIds) {
    if (!songIds || songIds.length === 0) return;
    const res = await fetch('/api/v1/songs/batch/download', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ song_ids: songIds }),
    });
    if (!res.ok) {
      throw new Error(`Batch download failed: ${res.statusText}`);
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'moodify_songs.zip';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  },

  /**
   * Space-insensitive and punctuation-tolerant search matching.
   * Matches tracks even if there are spaces in the song name and user types without spaces,
   * or vice versa, plus handles underscores, hyphens, and multi-token queries.
   */
  matchesSongSearch(song, query) {
    if (!song) return false;
    if (!query || !query.trim()) return true;

    const rawQuery = query.trim().toLowerCase();
    const strippedQuery = rawQuery.replace(/[\s\-_.]+/g, '');

    const title = song.title || '';
    const artist = song.artist || '';
    const album = song.album || '';
    const orig = song.original_name || '';
    const fn = song.filename || '';
    const cleanTitle = this.getCleanTitle(song);

    const fields = [title, artist, album, orig, fn, cleanTitle];

    // 1. Direct substring match on any field
    for (const f of fields) {
      if (!f) continue;
      if (f.toLowerCase().includes(rawQuery)) return true;
    }

    // 2. Space-insensitive match (e.g. "Alag Aasmaan" matches "alagaasmaan")
    if (strippedQuery) {
      for (const f of fields) {
        if (!f) continue;
        const strippedField = f.toLowerCase().replace(/[\s\-_.]+/g, '');
        if (strippedField.includes(strippedQuery)) return true;
      }

      // Check combined "Title Artist"
      const combinedStripped = `${title}${artist}${album}${cleanTitle}`.toLowerCase().replace(/[\s\-_.]+/g, '');
      if (combinedStripped.includes(strippedQuery)) return true;
    }

    // 3. Multi-token match (e.g. "anuv alag" matches "Alag Aasmaan" by "Anuv Jain")
    const tokens = rawQuery.split(/\s+/).filter(Boolean);
    if (tokens.length > 1) {
      const combinedAll = `${title} ${artist} ${album} ${orig} ${fn} ${cleanTitle}`.toLowerCase();
      const combinedStrippedAll = combinedAll.replace(/[\s\-_.]+/g, '');
      const allMatch = tokens.every(tok => {
        const strippedTok = tok.replace(/[\s\-_.]+/g, '');
        return combinedAll.includes(tok) || (strippedTok && combinedStrippedAll.includes(strippedTok));
      });
      if (allMatch) return true;
    }

    return false;
  },

  /**
   * Checks if a song had its format auto-corrected (e.g. extension mismatch vs detected magic bytes)
   */
  isFormatCorrected(song) {
    if (!song) return false;
    if (song.extension_corrected || song.format_warning) return true;
    if (!song.original_name || !song.format) return false;
    const match = song.original_name.match(/\.([a-zA-Z0-9]+)$/);
    if (!match) return false;
    const origExt = match[1].toLowerCase();
    const format = song.format.toLowerCase();
    const extToFmt = {
      'mp3': 'mp3',
      'm4a': 'm4a', 'aac': 'm4a', 'mp4': 'm4a',
      'wav': 'wav',
      'flac': 'flac',
      'opus': 'opus', 'ogg': 'opus'
    };
    const expectedFmt = extToFmt[origExt];
    return Boolean(expectedFmt && expectedFmt !== format);
  }
};

