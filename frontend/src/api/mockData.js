/**
 * Moodify Realistic Mock Seed Data
 * In accordance with Phase 5 of @frontend skill (Hybrid API Engine).
 * Allows full offline preview, client-side mutations, and realistic UX.
 */

export const initialMockData = {
  songs: [
    {
      id: "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
      filename: "midnight_city.flac",
      original_name: "M83 - Midnight City.flac",
      format: "flac",
      size_bytes: 34102840,
      title: "Midnight City",
      artist: "M83",
      album: "Hurry Up, We're Dreaming",
      inferred_genre: "Synthwave / Electronic",
      matched_moods: ["euphoric", "energetic"],
      tempo_bpm: 105.0,
      key_signature: "B Minor",
      duration_sec: 243.5,
      valence: 0.78,
      energy: 0.88,
      danceability: 0.71,
      status: "analyzed",
      created_at: "2026-09-18T10:14:00Z"
    },
    {
      id: "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      filename: "resonator_drift.mp3",
      original_name: "Tycho - Awake.mp3",
      format: "mp3",
      size_bytes: 11420100,
      title: "Awake",
      artist: "Tycho",
      album: "Awake",
      inferred_genre: "Ambient / Chillwave",
      matched_moods: ["chill", "neutral"],
      tempo_bpm: 92.4,
      key_signature: "D Major",
      duration_sec: 283.1,
      valence: 0.54,
      energy: 0.46,
      danceability: 0.62,
      status: "analyzed",
      created_at: "2026-09-19T14:22:10Z"
    },
    {
      id: "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      filename: "space_song.mp3",
      original_name: "Beach House - Space Song.mp3",
      format: "mp3",
      size_bytes: 12890400,
      title: "Space Song",
      artist: "Beach House",
      album: "Depression Cherry",
      inferred_genre: "Dream Pop / Indie",
      matched_moods: ["melancholic", "chill"],
      tempo_bpm: 74.0,
      key_signature: "Eb Major",
      duration_sec: 320.8,
      valence: 0.32,
      energy: 0.38,
      danceability: 0.44,
      status: "analyzed",
      created_at: "2026-09-20T08:05:30Z"
    },
    {
      id: "8a15fd78-6542-4f32-849c-3a819b1b72e1",
      filename: "aerodynamic.flac",
      original_name: "Daft Punk - Aerodynamic.flac",
      format: "flac",
      size_bytes: 28940120,
      title: "Aerodynamic",
      artist: "Daft Punk",
      album: "Discovery",
      inferred_genre: "French House / Electro",
      matched_moods: ["energetic", "euphoric"],
      tempo_bpm: 123.0,
      key_signature: "D Minor",
      duration_sec: 212.0,
      valence: 0.82,
      energy: 0.94,
      danceability: 0.85,
      status: "analyzed",
      created_at: "2026-09-20T12:30:15Z"
    },
    {
      id: "4b81c2d9-1122-4889-bc33-899120deaa44",
      filename: "nightcall.wav",
      original_name: "Kavinsky - Nightcall.wav",
      format: "wav",
      size_bytes: 42910200,
      title: "Nightcall",
      artist: "Kavinsky",
      album: "OutRun",
      inferred_genre: "Outrun / Dark Synth",
      matched_moods: ["dark", "chill"],
      tempo_bpm: 91.5,
      key_signature: "A Minor",
      duration_sec: 259.4,
      valence: 0.41,
      energy: 0.58,
      danceability: 0.64,
      status: "analyzed",
      created_at: "2026-09-20T13:45:00Z"
    }
  ],

  moodClusters: [
    {
      mood: "chill",
      label: "Chill & Ambient",
      song_count: 8,
      avg_tempo: 88.5,
      avg_energy: 0.42,
      description: "Low-pulse, tranquil soundscapes with harmonic pads and soft percussion."
    },
    {
      mood: "energetic",
      label: "High Voltage & Drive",
      song_count: 14,
      avg_tempo: 124.8,
      avg_energy: 0.91,
      description: "Punchy transients, high kinetic momentum, and driving rhythmic pulse."
    },
    {
      mood: "euphoric",
      label: "Euphoric & Uplifting",
      song_count: 11,
      avg_tempo: 118.2,
      avg_energy: 0.85,
      description: "Bright harmonic resolution, expansive synth leads, and peak emotional release."
    },
    {
      mood: "melancholic",
      label: "Nocturne & Melancholy",
      song_count: 6,
      avg_tempo: 76.0,
      avg_energy: 0.35,
      description: "Introspective minor tonalities, atmospheric reverbs, and deep vocal timbre."
    }
  ],

  lyricsSample: {
    synced: true,
    plain: "Waiting in a car\nWaiting for a ride in the dark\nThe night city grows\nLook and see her eyes, they keep dreaming\nWaiting in a car\nWaiting for a ride in the dark\nDrinking in the lights\nFollowing the neon signs",
    lines: [
      { time: 14.5, text: "Waiting in a car" },
      { time: 19.2, text: "Waiting for a ride in the dark" },
      { time: 24.8, text: "The night city grows" },
      { time: 29.5, text: "Look and see her eyes, they keep dreaming" },
      { time: 35.1, text: "Waiting in a car" },
      { time: 39.8, text: "Waiting for a ride in the dark" },
      { time: 45.4, text: "Drinking in the lights" },
      { time: 51.0, text: "Following the neon signs" }
    ],
    emotions: {
      "Euphoria": 0.84,
      "Nostalgia": 0.76,
      "Excitement": 0.71,
      "Calmness": 0.38,
      "Melancholy": 0.22,
      "Darkness": 0.18
    }
  },

  queueStats: {
    pending: 0,
    in_flight: 0,
    completed: 42,
    failed: 1,
    uptime_seconds: 14820
  }
};
