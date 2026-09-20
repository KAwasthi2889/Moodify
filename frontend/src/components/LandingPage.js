/**
 * Moodify Commercial Product Landing Page
 * World-class editorial showcase inspired by Apple, Linear, and Teenage Engineering.
 * Acts as a magnetic visual hook explaining the 64-D multimodal architecture.
 * Features direct launch into the Moodify Workshop.
 */

export function renderLandingPage(container, onNavigate) {
  if (!container) return;

  container.innerHTML = `
    <div class="landing-page">
      <!-- ── Hero Section ───────────────────────────────────── -->
      <section class="hero-section" id="hero">
        <div class="hero-glow" aria-hidden="true"></div>

        <div class="hero-content">
          <div class="hero-pill-badge">
            <span class="badge-sparkle">✨</span>
            <span>Continuous Multimodal Hyperspace</span>
          </div>

          <h1 class="hero-title">
            Audio Intelligence in <span class="gradient-text">64 Dimensions.</span>
          </h1>

          <p class="hero-subtitle">
            Transform raw audio collections into continuous vector hyperspace. Tier-1 zero-CPU
            MusicBrainz tagging meets Tier-2 Librosa acoustic DSP and RoBERTa continuous GoEmotions —
            chained via pgvector cosine similarity.
          </p>

          <div class="hero-cta-group">
            <button class="btn-hero-primary" id="btn-hero-launch">
              <span>Moodify Workshop</span>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="9 18 15 12 9 6"/></svg>
            </button>
            <a href="#architecture" class="btn-hero-secondary">
              <span>Explore Architecture ↓</span>
            </a>
          </div>

          <!-- Kinetic Specs Strip -->
          <div class="hero-specs-strip">
            <div class="spec-item">
              <span class="spec-value mono-num">36-D</span>
              <span class="spec-label">Librosa DSP Acoustics</span>
            </div>
            <div class="spec-sep">•</div>
            <div class="spec-item">
              <span class="spec-value mono-num">28-D</span>
              <span class="spec-label">RoBERTa GoEmotions</span>
            </div>
            <div class="spec-sep">•</div>
            <div class="spec-item">
              <span class="spec-value mono-num">ΔBPM ≤ 12</span>
              <span class="spec-label">Smooth Tempo Pacing</span>
            </div>
            <div class="spec-sep">•</div>
            <div class="spec-item">
              <span class="spec-value mono-num">1 req/s</span>
              <span class="spec-label">MusicBrainz Compliant</span>
            </div>
          </div>
        </div>
      </section>

      <!-- ── Visual Architecture Breakdown ──────────────────── -->
      <section class="section-container" id="architecture">
        <div class="section-header">
          <span class="section-tag">System Architecture</span>
          <h2 class="section-title">The Two-Tier Processing Pipeline</h2>
          <p class="section-desc">Engineered for lightning-fast library imports, followed by deep on-demand neural multimodal analysis.</p>
        </div>

        <div class="architecture-grid">
          <!-- Tier 1 Card -->
          <div class="arch-card tier-1">
            <div class="arch-header">
              <span class="arch-badge tier-1-badge">Tier 1 • Instant Ingest</span>
              <h3 class="arch-title">Zero-CPU Ingestion & Fingerprinting</h3>
            </div>
            <p class="arch-desc">
              Ingests whole music libraries of up to <strong>700 files</strong> (max 35 MB per track).
              Automatically fingerprints audio with Chromaprint (<code class="mono-code">fpcalc</code>) and fetches canonical metadata from MusicBrainz with polite 1 req/sec pacing.
            </p>
            <div class="arch-features-list">
              <div class="arch-feature">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#00f0ff" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                <span>AcoustID & Chromaprint non-destructive audio fingerprinting</span>
              </div>
              <div class="arch-feature">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#00f0ff" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                <span>Verified ID3 / Vorbis metadata embedding & canonical renaming</span>
              </div>
              <div class="arch-feature">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#00f0ff" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                <span>Zero CPU burn: entire libraries become cataloged and playable in seconds</span>
              </div>
            </div>
            <div class="arch-status-flow">
              <span class="flow-step">uploaded</span>
              <span class="flow-arrow">➔</span>
              <span class="flow-step active">tagged</span>
            </div>
          </div>

          <!-- Tier 2 Card -->
          <div class="arch-card tier-2">
            <div class="arch-header">
              <span class="arch-badge tier-2-badge">Tier 2 • Neural Multimodal</span>
              <h3 class="arch-title">64-D Multimodal Fusion & pgvector</h3>
            </div>
            <p class="arch-desc">
              Triggered on-demand for single tracks, selections, or entire libraries. Librosa extracts 36-D acoustic features
              (tempo, MFCCs, chroma, energy, brightness), while RoBERTa computes 28-D GoEmotions continuous probabilities from synchronized LRCLIB lyrics.
            </p>
            <div class="arch-features-list">
              <div class="arch-feature">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                <span>Fused into a continuous <strong>64-D unit normalized vector</strong></span>
              </div>
              <div class="arch-feature">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                <span>Synchronized karaoke <code class="mono-code">.lrc</code> timestamps line-by-line</span>
              </div>
              <div class="arch-feature">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                <span>Stored in PostgreSQL using <strong>pgvector</strong> for instant cosine search</span>
              </div>
            </div>
            <div class="arch-status-flow">
              <span class="flow-step">tagged</span>
              <span class="flow-arrow">➔</span>
              <span class="flow-step analyzing">analyzing</span>
              <span class="flow-arrow">➔</span>
              <span class="flow-step ready">ready</span>
            </div>
          </div>
        </div>
      </section>

      <!-- ── Vector Hyperspace & Seed Chaining Showcase ──────── -->
      <section class="section-container" id="hyperspace">
        <div class="section-header">
          <span class="section-tag">Vector Intelligence</span>
          <h2 class="section-title">Seed-Driven Nearest Neighbor Playlists</h2>
          <p class="section-desc">
            Unlike rigid genre buckets, Moodify calculates continuous cosine distance from any anchor track.
            Songs are dynamically sequenced with monotonic tempo smoothing so transitions flow seamlessly.
          </p>
        </div>

        <div class="glass-panel vector-flow-card">
          <div class="vector-flow-grid">
            <div class="vector-flow-step">
              <div class="step-badge mono-num">01</div>
              <h4 class="step-heading">Anchor Track Selection</h4>
              <p class="step-body">Select any song in your library as the anchor. If not yet analyzed, Tier-2 analysis runs instantly on-the-fly.</p>
            </div>

            <div class="vector-flow-step">
              <div class="step-badge mono-num">02</div>
              <h4 class="step-heading">pgvector Cosine Sweep</h4>
              <p class="step-body">PostgreSQL scans 64-D hyperspace, filtering candidates that meet your suitability floor (between 50% and 95%).</p>
            </div>

            <div class="vector-flow-step">
              <div class="step-badge mono-num">03</div>
              <h4 class="step-heading">Monotonic BPM Pacing</h4>
              <p class="step-body">Matched tracks are ordered with smooth tempo progressions (ΔBPM ≤ 12) to eliminate jarring sonic hops.</p>
            </div>

            <div class="vector-flow-step">
              <div class="step-badge mono-num">04</div>
              <h4 class="step-heading">Native .M3U8 Export</h4>
              <p class="step-body">Export playlists directly as industry-standard .m3u8 files ready for any media player or DJ rig.</p>
            </div>
          </div>
        </div>
      </section>

      <!-- ── Bottom Conversion Banner ───────────────────────── -->
      <section class="landing-cta-banner">
        <div class="cta-banner-content">
          <h2 class="cta-banner-title">Ready to Moodify Your Library?</h2>
          <p class="cta-banner-subtitle">
            Upload your audio files to the Moodify Workshop, review metadata tags, and initiate neural 64-D hyperspace analysis.
          </p>
          <button class="btn-hero-primary" id="btn-banner-launch">
            <span>Launch Moodify Workshop</span>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="9 18 15 12 9 6"/></svg>
          </button>
        </div>
      </section>
    </div>
  `;

  // Bind Workshop navigation triggers
  const launchBtns = [
    container.querySelector('#btn-hero-launch'),
    container.querySelector('#btn-banner-launch')
  ];

  launchBtns.forEach(btn => {
    btn?.addEventListener('click', () => {
      if (onNavigate) {
        onNavigate('workshop');
      }
    });
  });
}
