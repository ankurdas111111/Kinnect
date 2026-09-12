import { Suspense, useRef, useState } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { EffectComposer, Bloom, Vignette, N8AO, SMAA } from '@react-three/postprocessing';
import { AdaptiveDpr, PerformanceMonitor } from '@react-three/drei';
import { damp3, damp } from 'maath/easing';
import * as THREE from 'three';
import Scene from './Scene.jsx';

/**
 * Camera rig. The scroll timeline sets a TARGET; this damps toward it every
 * frame and adds a small pointer parallax on top. Damping rather than snapping
 * is what makes the scene feel inhabited instead of stepped — it is the highest
 * ratio of feel to effort in the whole genre.
 */
function Rig({ target, reduced }) {
  const { camera, pointer } = useThree();
  const look = useRef(new THREE.Vector3());

  useFrame((_, dt) => {
    const t = target.current;
    const px = reduced ? 0 : pointer.x;
    const py = reduced ? 0 : pointer.y;
    damp3(
      camera.position,
      [t.pos[0] + px * t.parallax, t.pos[1] + py * t.parallax * 0.4, t.pos[2]],
      reduced ? 0.001 : 0.55,
      dt
    );
    damp3(look.current, t.look, reduced ? 0.001 : 0.5, dt);
    damp(camera, 'fov', t.fov, 0.6, dt);
    camera.updateProjectionMatrix();
    camera.lookAt(look.current);
  });
  return null;
}

export default function ViewCanvas({ theme, tone, people, target, reduced }) {
  // Post-processing is the first thing to go when a machine cannot keep up.
  // AO and bloom are what make this look expensive, but a page that looks
  // expensive at 12fps is worse than a plain one at 60. PerformanceMonitor
  // watches real frame times and drops them rather than guessing from the
  // user-agent string.
  const [degraded, setDegraded] = useState(false);
  const effects = !reduced && !degraded;
  return (
    <Canvas
      className="l3d-canvas"
      // Pointer events must reach 3D objects even though the canvas sits behind
      // the whole document. Without these two the scene is inert.
      eventSource={typeof document !== 'undefined' ? document.body : undefined}
      eventPrefix="client"
      shadows
      dpr={[1, 1.6]}
      gl={{ antialias: false, stencil: true, powerPreference: 'high-performance' }}
      camera={{ position: [30, 250, 300], fov: 30, near: 1, far: 1600 }}
      onCreated={({ gl }) => {
        gl.toneMapping = THREE.NeutralToneMapping;
        gl.toneMappingExposure = 0.92;
      }}
    >
      <PerformanceMonitor
        bounds={() => [50, 60]}
        onDecline={() => setDegraded(true)}
        flipflops={2}
        onFallback={() => setDegraded(true)}
      />
      <AdaptiveDpr pixelated />
      <Suspense fallback={null}>
        <Scene theme={theme} tone={tone} people={people} />
        <Rig target={target} reduced={reduced} />
        {effects && (
          <EffectComposer enableNormalPass multisampling={0}>
            {/* Ambient occlusion is the single biggest step from "flat boxes on
                a plane" to a place with depth — it darkens every crevice where
                a wall meets the ground. */}
            <N8AO aoRadius={9} intensity={2.0} distanceFalloff={0.7} quality="low" halfRes color="#2a231c" />
            {/* Bloom only bites on the SOS emissive, which is the one thing on
                the board allowed to glow. */}
            <Bloom intensity={0.5} luminanceThreshold={0.72} luminanceSmoothing={0.25} mipmapBlur />
            <Vignette offset={0.32} darkness={0.42} />
            <SMAA />
          </EffectComposer>
        )}
      </Suspense>
    </Canvas>
  );
}
