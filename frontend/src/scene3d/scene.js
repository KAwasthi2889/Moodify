/**
 * Moodify Procedural Three.js 3D Canvas
 * Interactive kinetic soundwave & constellation engine.
 * Synthesized from @frontend skill THREEJS-PATTERNS.
 */

import * as THREE from 'three';

let renderer = null;
let scene = null;
let camera = null;
let pointsGroup = null;
let isAudioPlaying = false;
let waveFrequency = 1.0;

/**
 * Initializes the Three.js background canvas with mouse parallax & kinetic waves.
 * @param {HTMLCanvasElement} canvas - Target canvas element
 */
export function initScene(canvas) {
  if (!canvas) return;

  // 1. WebGL Renderer with retina clamp
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
  camera = new THREE.PerspectiveCamera(55, window.innerWidth / window.innerHeight, 0.1, 1000);
  camera.position.set(0, 10, 32);
  camera.lookAt(0, 0, 0);

  // 3. Procedural Kinetic Wave Points
  const cols = 60;
  const rows = 60;
  const count = cols * rows;
  const geometry = new THREE.BufferGeometry();
  const positions = new Float32Array(count * 3);
  const colors = new Float32Array(count * 3);

  const colorCyan = new THREE.Color(0x00f0ff);
  const colorViolet = new THREE.Color(0x8b5cf6);
  const tempColor = new THREE.Color();

  let idx = 0;
  for (let i = 0; i < cols; i++) {
    for (let j = 0; j < rows; j++) {
      const u = (i / cols - 0.5) * 55;
      const v = (j / rows - 0.5) * 55;
      positions[idx * 3] = u;
      positions[idx * 3 + 1] = 0;
      positions[idx * 3 + 2] = v;

      // Color gradient from center outwards
      const distFromCenter = Math.sqrt(u * u + v * v) / 38;
      tempColor.copy(colorCyan).lerp(colorViolet, Math.min(distFromCenter, 1.0));
      colors[idx * 3] = tempColor.r;
      colors[idx * 3 + 1] = tempColor.g;
      colors[idx * 3 + 2] = tempColor.b;

      idx++;
    }
  }

  geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
  geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));

  const material = new THREE.PointsMaterial({
    size: 0.22,
    vertexColors: true,
    transparent: true,
    opacity: 0.65,
    blending: THREE.AdditiveBlending,
  });

  pointsGroup = new THREE.Points(geometry, material);
  scene.add(pointsGroup);

  // 4. Mouse Parallax Physics
  const mouse = { x: 0, y: 0, targetX: 0, targetY: 0 };
  window.addEventListener('mousemove', (e) => {
    mouse.targetX = (e.clientX / window.innerWidth - 0.5) * 2;
    mouse.targetY = -(e.clientY / window.innerHeight - 0.5) * 2;
  });

  // 5. Responsive Window Resize
  window.addEventListener('resize', () => {
    if (!camera || !renderer) return;
    camera.aspect = window.innerWidth / window.innerHeight;
    camera.updateProjectionMatrix();
    renderer.setSize(window.innerWidth, window.innerHeight);
  });

  // 6. Battery & Tab-switching pause safety
  let isVisible = true;
  document.addEventListener('visibilitychange', () => {
    isVisible = !document.hidden;
  });

  // Check reduced motion
  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  // 7. Animation Loop with Kinetic Undulation
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
      camera.position.y = 10 + mouse.y * 2;
      camera.lookAt(0, 0, 0);

      // Undulate point wave heights
      const posAttr = pointsGroup.geometry.attributes.position;
      const posArray = posAttr.array;
      const speed = isAudioPlaying ? 2.4 : 0.8;
      const amp = isAudioPlaying ? 2.5 : 1.0;

      for (let k = 0; k < count; k++) {
        const x = posArray[k * 3];
        const z = posArray[k * 3 + 2];
        const dist = Math.sqrt(x * x + z * z);
        posArray[k * 3 + 1] = Math.sin(dist * 0.25 - elapsedTime * speed * waveFrequency) * amp;
      }
      posAttr.needsUpdate = true;

      // Subtle scene drift
      pointsGroup.rotation.y = elapsedTime * 0.03;
    }

    renderer.render(scene, camera);
  }

  animate();
}

/**
 * Dynamically reacts to playback state to amplify wave kinetics.
 * @param {boolean} playing - True when music is actively playing
 * @param {number} bpm - Song tempo to modulate wave speed
 */
export function setPlaybackKinetics(playing, bpm = 120) {
  isAudioPlaying = playing;
  waveFrequency = Math.max(0.6, Math.min(2.0, bpm / 120));
}
