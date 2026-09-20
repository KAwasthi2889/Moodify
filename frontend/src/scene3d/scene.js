/**
 * Moodify Procedural Three.js Kinetic Canvas
 * Interactive soundwave & orbital node field.
 * Synthesized from @frontend skill THREEJS-PATTERNS with dynamic audio tempo modulation.
 */

import * as THREE from 'three';

let renderer = null;
let scene = null;
let camera = null;
let pointsMesh = null;
let isAudioPlaying = false;
let waveFrequency = 1.0;
let targetMoodHex = 0x00f0ff;
let currentMoodColor = new THREE.Color(0x00f0ff);

/**
 * Initializes the Three.js canvas with kinetic soundwave points & mouse parallax
 * @param {HTMLCanvasElement} canvas - Target canvas
 */
export function initScene(canvas) {
  if (!canvas) return;

  // 1. Renderer Setup with DPR clamp
  renderer = new THREE.WebGLRenderer({
    canvas,
    alpha: true,
    antialias: true,
    powerPreference: 'high-performance',
  });
  renderer.setSize(window.innerWidth, window.innerHeight);
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

  // 2. Scene & Perspective Camera
  scene = new THREE.Scene();
  camera = new THREE.PerspectiveCamera(50, window.innerWidth / window.innerHeight, 0.1, 1000);
  camera.position.set(0, 12, 34);
  camera.lookAt(0, 0, 0);

  // 3. Kinetic Undulation Waveform Points
  const cols = 64;
  const rows = 64;
  const count = cols * rows;
  const geometry = new THREE.BufferGeometry();
  const positions = new Float32Array(count * 3);
  const colors = new Float32Array(count * 3);

  const baseCyan = new THREE.Color(0x00f0ff);
  const baseIndigo = new THREE.Color(0x6366f1);
  const tempCol = new THREE.Color();

  let idx = 0;
  for (let i = 0; i < cols; i++) {
    for (let j = 0; j < rows; j++) {
      const u = (i / cols - 0.5) * 58;
      const v = (j / rows - 0.5) * 58;
      positions[idx * 3] = u;
      positions[idx * 3 + 1] = 0;
      positions[idx * 3 + 2] = v;

      const dist = Math.sqrt(u * u + v * v) / 40;
      tempCol.copy(baseCyan).lerp(baseIndigo, Math.min(dist, 1.0));
      colors[idx * 3] = tempCol.r;
      colors[idx * 3 + 1] = tempCol.g;
      colors[idx * 3 + 2] = tempCol.b;

      idx++;
    }
  }

  geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
  geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));

  const material = new THREE.PointsMaterial({
    size: 0.24,
    vertexColors: true,
    transparent: true,
    opacity: 0.65,
    blending: THREE.AdditiveBlending,
  });

  pointsMesh = new THREE.Points(geometry, material);
  scene.add(pointsMesh);

  // 4. Mouse Parallax Physics
  const mouse = { x: 0, y: 0, targetX: 0, targetY: 0 };
  window.addEventListener('mousemove', (e) => {
    mouse.targetX = (e.clientX / window.innerWidth - 0.5) * 2;
    mouse.targetY = -(e.clientY / window.innerHeight - 0.5) * 2;
  });

  // 5. Responsive Resize
  window.addEventListener('resize', () => {
    if (!camera || !renderer) return;
    camera.aspect = window.innerWidth / window.innerHeight;
    camera.updateProjectionMatrix();
    renderer.setSize(window.innerWidth, window.innerHeight);
  });

  // 6. Tab Visibility Pausing (Battery safety)
  let isVisible = true;
  document.addEventListener('visibilitychange', () => {
    isVisible = !document.hidden;
  });

  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  // 7. Render & Animation Loop
  const clock = new THREE.Clock();
  function animate() {
    requestAnimationFrame(animate);
    if (!isVisible) return;

    const elapsedTime = clock.getElapsedTime();

    if (!prefersReducedMotion) {
      // Smooth camera damping towards mouse target
      mouse.x += (mouse.targetX - mouse.x) * 0.04;
      mouse.y += (mouse.targetY - mouse.y) * 0.04;
      camera.position.x = mouse.x * 4;
      camera.position.y = 12 + mouse.y * 2.5;
      camera.lookAt(0, 0, 0);

      // Kinetic soundwave undulation
      const posAttr = pointsMesh.geometry.attributes.position;
      const posArr = posAttr.array;
      const colAttr = pointsMesh.geometry.attributes.color;
      const colArr = colAttr.array;

      const speed = isAudioPlaying ? 2.2 * waveFrequency : 0.75;
      const amp = isAudioPlaying ? 2.4 : 0.9;

      // Lerp current mood color
      currentMoodColor.lerp(new THREE.Color(targetMoodHex), 0.03);

      for (let k = 0; k < count; k++) {
        const x = posArr[k * 3];
        const z = posArr[k * 3 + 2];
        const dist = Math.sqrt(x * x + z * z);
        const elevation = Math.sin(dist * 0.28 - elapsedTime * speed) * amp +
                          Math.cos((x + z) * 0.15 + elapsedTime * 0.6) * (amp * 0.35);

        posArr[k * 3 + 1] = elevation;

        // Dynamic color blend based on height and current mood
        if (isAudioPlaying) {
          const heightRatio = Math.max(0, Math.min(1, (elevation + amp) / (amp * 2)));
          tempCol.copy(currentMoodColor).lerp(baseIndigo, 1 - heightRatio * 0.7);
          colArr[k * 3] = tempCol.r;
          colArr[k * 3 + 1] = tempCol.g;
          colArr[k * 3 + 2] = tempCol.b;
        }
      }

      posAttr.needsUpdate = true;
      if (isAudioPlaying) colAttr.needsUpdate = true;

      // Slow orbital drift
      pointsMesh.rotation.y = elapsedTime * 0.025;
    }

    renderer.render(scene, camera);
  }

  animate();
}

/**
 * Modulates wave kinetics based on audio playback state & song attributes
 * @param {boolean} playing - True when audio is playing
 * @param {number} bpm - Song tempo
 * @param {string} dominantMood - Mood string
 */
export function setPlaybackKinetics(playing, bpm = 120, dominantMood = '') {
  isAudioPlaying = playing;
  waveFrequency = Math.max(0.6, Math.min(2.2, bpm / 120));

  // Map mood string to hex color for points
  const moodMap = {
    'joy & optimism': 0xf59e0b,
    'desire & love': 0xf43f5e,
    'sadness & grief': 0x6366f1,
    'anger & annoyance': 0xef4444,
    'calm & neutral': 0x10b981
  };

  targetMoodHex = moodMap[dominantMood] || 0x00f0ff;
}
