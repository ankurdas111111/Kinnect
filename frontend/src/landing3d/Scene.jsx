import { useMemo, useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import { Environment, Lightformer, ContactShadows, RoundedBox } from '@react-three/drei';
import * as THREE from 'three';
import { BOARD, PALETTE, BUILDINGS, ROADS, PARK, WATER, PERSON_R, MARKER_H } from './world.js';
import CityProps from './CityProps.jsx';

/** Extrude a closed polygon (park, water) into a slab with a soft edge. */
function polyGeometry(points, depth) {
  const shape = new THREE.Shape();
  shape.moveTo(points[0][0], points[0][1]);
  for (let i = 1; i < points.length; i++) shape.lineTo(points[i][0], points[i][1]);
  shape.closePath();
  const geo = new THREE.ExtrudeGeometry(shape, {
    depth, bevelEnabled: true, bevelSize: 0.7, bevelThickness: 0.35, bevelSegments: 2,
  });
  geo.rotateX(-Math.PI / 2);
  return geo;
}

/** A road as a ribbon: casing underneath, surface on top, so edges read. */
function roadGeometry(pts, width) {
  const curve = new THREE.CatmullRomCurve3(pts.map(([x, z]) => new THREE.Vector3(x, 0, z)));
  const samples = curve.getPoints(80);
  const positions = [];
  const index = [];
  for (let i = 0; i < samples.length; i++) {
    const p = samples[i];
    const next = samples[Math.min(i + 1, samples.length - 1)];
    const prev = samples[Math.max(i - 1, 0)];
    const dir = new THREE.Vector3().subVectors(next, prev).normalize();
    const side = new THREE.Vector3(-dir.z, 0, dir.x).multiplyScalar(width / 2);
    positions.push(p.x + side.x, 0, p.z + side.z, p.x - side.x, 0, p.z - side.z);
    if (i < samples.length - 1) {
      const a = i * 2;
      index.push(a, a + 1, a + 2, a + 1, a + 3, a + 2);
    }
  }
  const geo = new THREE.BufferGeometry();
  geo.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
  geo.setIndex(index);
  geo.computeVertexNormals();
  return geo;
}

/**
 * The environment map, built in-process from a handful of emissive panels.
 *
 * This used to be `<Environment preset="city" />`, which downloads a 1k HDRI
 * from raw.githubusercontent.com at runtime. Suspense blanks the whole scene
 * while that is in flight, so a slow or blocked network left the landing page
 * showing nothing at all — for ~20s on a normal connection, forever behind a
 * firewall. A hero that depends on a third-party CDN is not a hero.
 *
 * Lightformers bake to a cubemap once (`frames={1}`) with no network at all,
 * and they are art-directable, which a stock HDRI is not.
 */
function EnvRig({ theme }) {
  const night = theme === 'dark';
  return (
    <Environment
      key={theme}
      resolution={128}
      frames={1}
      background={false}
      environmentIntensity={night ? 0.55 : 0.4}
    >
      {night ? (
        <>
          {/* cool night sky overhead */}
          <Lightformer form="rect" intensity={0.55} color="#5d7096" scale={[70, 70, 1]}
            position={[0, 26, 0]} rotation={[-Math.PI / 2, 0, 0]} />
          {/* the city's own sodium glow, bouncing off haze at street level */}
          <Lightformer form="rect" intensity={2.2} color="#ff9a4d" scale={[70, 12, 1]}
            position={[0, -7, 26]} rotation={[Math.PI / 2, 0, 0]} />
          <Lightformer form="rect" intensity={1.4} color="#ff8c3c" scale={[70, 12, 1]}
            position={[0, -7, -26]} rotation={[Math.PI / 2, 0, 0]} />
          {/* moon side, for a cool rim on one face of every building */}
          <Lightformer form="rect" intensity={1.1} color="#a9c0e4" scale={[10, 40, 1]}
            position={[-30, 12, -14]} rotation={[0, Math.PI / 2, 0]} />
        </>
      ) : (
        <>
          {/* sky dome */}
          <Lightformer form="rect" intensity={1.5} color="#e8f1ff" scale={[80, 80, 1]}
            position={[0, 30, 0]} rotation={[-Math.PI / 2, 0, 0]} />
          {/* warm late-afternoon sun side */}
          <Lightformer form="rect" intensity={2.4} color="#fff0d8" scale={[26, 40, 1]}
            position={[30, 14, 16]} rotation={[0, -Math.PI / 2, 0]} />
          {/* ground bounce — warm, and the reason shadowed walls are not grey */}
          <Lightformer form="rect" intensity={0.9} color="#e3d2b4" scale={[80, 80, 1]}
            position={[0, -14, 0]} rotation={[Math.PI / 2, 0, 0]} />
        </>
      )}
    </Environment>
  );
}

export default function Scene({ theme = 'light', tone = null, people = [] }) {
  const pal = PALETTE[theme];
  const sosLight = useRef();
  const sosRing = useRef();

  const parkGeo = useMemo(() => polyGeometry(PARK, 0.5), []);
  const waterGeo = useMemo(() => polyGeometry(WATER, 0.3), []);
  const roadGeos = useMemo(() => ROADS.map(r => ({
    surface: roadGeometry(r.pts, r.w),
    casing: roadGeometry(r.pts, r.w + 2.2),
  })), []);

  // The SOS pulse is the only animated light. Everything else is static so the
  // frame cost stays flat.
  useFrame((state, dt) => {
    const want = tone === 'sos' ? 900 : 0;
    if (sosLight.current) {
      sosLight.current.intensity += (want - sosLight.current.intensity) * (1 - Math.exp(-dt * 4));
    }
    if (sosRing.current) {
      const t = state.clock.elapsedTime;
      const on = tone === 'sos' ? 1 : 0;
      const s = 1 + (Math.sin(t * 2.4) * 0.5 + 0.5) * 1.6;
      sosRing.current.scale.setScalar(s * on + 0.001);
      sosRing.current.material.opacity = on * (0.5 - (s - 1) / 1.6 * 0.5);
    }
  });

  const sosPerson = people.find(p => p.status === 'SOS');

  return (
    <group>
      {/* LIGHTING
          Day is a warm late-afternoon key with a bright sky bounce.
          Night is NOT "the same scene, dimmer" — that is what produced a black
          screen. It is a low cool moon for silhouette, a raised ambient floor so
          nothing crushes to pure black, and the city's own lit windows doing the
          actual work. */}
      {theme === 'dark' ? (
        <>
          {/* moon: low intensity, cool, steep — enough for edges, not for fill */}
          <directionalLight
            position={[-70, 150, -40]}
            intensity={0.75}
            color="#9fb6d8"
            castShadow
            shadow-mapSize={[2048, 2048]}
            shadow-bias={-0.0006}
            shadow-normalBias={0.02}
          >
            <orthographicCamera attach="shadow-camera" args={[-170, 170, 150, -150, 1, 600]} />
          </directionalLight>
          {/* sky/ground bounce — stops shadowed faces going to zero */}
          <hemisphereLight args={['#4a5a78', '#2a221a', 0.85]} />
          <ambientLight intensity={0.5} color="#6b7a94" />
          {/* the warm city glow sitting over the block */}
          <pointLight position={[20, 45, 0]} color="#ffb877" intensity={2600} distance={340} decay={2} />
        </>
      ) : (
        <>
          <directionalLight
            position={[90, 140, 70]}
            intensity={tone === 'sos' ? 1.25 : 1.65}
            color="#fff3e2"
            castShadow
            shadow-mapSize={[2048, 2048]}
            shadow-bias={-0.0006}
            shadow-normalBias={0.02}
          >
            <orthographicCamera attach="shadow-camera" args={[-170, 170, 150, -150, 1, 600]} />
          </directionalLight>
          <hemisphereLight args={['#cfe0f2', '#c9bda6', 0.42]} />
          <ambientLight intensity={0.22} color={pal.fog} />
        </>
      )}

      <EnvRig theme={theme} />

      {/* Ground */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} receiveShadow position={[0, -0.02, 0]}>
        <planeGeometry args={[BOARD.w * 9, BOARD.d * 12]} />
        <meshStandardMaterial color={pal.ground} roughness={1} metalness={0} />
      </mesh>

      {/* Roads: casing first (darker, wider), then the surface on top. */}
      {roadGeos.map((r, i) => (
        <group key={i}>
          <mesh geometry={r.casing} position={[0, 0.03, 0]} receiveShadow>
            <meshStandardMaterial color={pal.roadEdge} roughness={1} />
          </mesh>
          <mesh geometry={r.surface} position={[0, 0.06, 0]} receiveShadow>
            <meshStandardMaterial color={pal.road} roughness={0.98} />
          </mesh>
        </group>
      ))}

      <mesh geometry={parkGeo} position={[0, 0.05, 0]} receiveShadow castShadow>
        <meshStandardMaterial color={pal.park} roughness={0.95} />
      </mesh>
      <mesh geometry={waterGeo} position={[0, 0.07, 0]} receiveShadow>
        <meshStandardMaterial color={pal.water} roughness={0.25} metalness={0.15} />
      </mesh>

      {/* Buildings: rounded so the silhouettes catch light on the corners, with
          a separate roof slab. Sharp untextured cubes were the single loudest
          "generated" tell in the previous version. */}
      {BUILDINGS.map((b, i) => (
        <group key={i} position={[b.x, 0, b.z]} rotation={[0, b.rot, 0]}>
          <RoundedBox
            args={[b.w, b.h, b.d]}
            radius={0.5}
            smoothness={3}
            position={[0, b.h / 2, 0]}
            castShadow
            receiveShadow
          >
            <meshStandardMaterial color={b.alt ? pal.wallAlt : pal.wall} roughness={0.88} metalness={0} />
          </RoundedBox>
          <mesh position={[0, b.h + 0.35, 0]} castShadow>
            <boxGeometry args={[b.w + 0.35, 0.55, b.d + 0.35]} />
            <meshStandardMaterial color={b.alt ? pal.roofAlt : pal.roof} roughness={0.9} />
          </mesh>
        </group>
      ))}

      <CityProps theme={theme} />

      {/* People. A pebble is 2.4m across against a 9-34m building, so the scale
          finally reads: these are people standing in a city. */}
      {people.map((p) => (
        <group key={p.id} position={[p.pos[0], 0, p.pos[1]]}>
          {/* stem — connects the marker to a real spot on the ground */}
          <mesh position={[0, MARKER_H / 2, 0]}>
            <cylinderGeometry args={[0.18, 0.85, MARKER_H, 10]} />
            <meshStandardMaterial color={PALETTE[theme][p.color]} roughness={0.5} transparent opacity={0.85} />
          </mesh>
          <mesh position={[0, MARKER_H, 0]} castShadow>
            <sphereGeometry args={[PERSON_R, 28, 22]} />
            <meshStandardMaterial
              color={PALETTE[theme][p.color]}
              roughness={0.35}
              metalness={0.05}
              emissive={PALETTE[theme][p.color]}
              emissiveIntensity={p.status === 'SOS' ? 0.75 : 0.18}
            />
          </mesh>
          {/* footprint ring on the ground, so the marker reads as anchored */}
          <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, 0.14, 0]}>
            <ringGeometry args={[PERSON_R * 0.9, PERSON_R * 1.25, 32]} />
            <meshBasicMaterial color={PALETTE[theme][p.color]} transparent opacity={0.4} depthWrite={false} />
          </mesh>
        </group>
      ))}

      {/* SOS: a ground pulse plus a real point light, so the whole block warms
          rather than just one marker changing colour. */}
      {sosPerson && (
        <group position={[sosPerson.pos[0], 0, sosPerson.pos[1]]}>
          <mesh ref={sosRing} rotation={[-Math.PI / 2, 0, 0]} position={[0, 0.14, 0]}>
            <ringGeometry args={[6, 9, 48]} />
            <meshBasicMaterial color={pal.vermilion} transparent opacity={0} depthWrite={false} />
          </mesh>
          <pointLight ref={sosLight} position={[0, 16, 0]} color={pal.vermilion} intensity={0} distance={120} decay={2} />
        </group>
      )}

      {/* Grounding shadow under the whole block. This is the difference between
          objects sitting ON a plane and objects floating above one. */}
      <ContactShadows
        key={theme}
        position={[0, 0.02, 0]}
        scale={BOARD.w * 1.4}
        far={60}
        blur={2.4}
        opacity={theme === 'dark' ? 0.5 : 0.42}
        color={theme === 'dark' ? '#000000' : '#4a3c2a'}
        resolution={1024}
        frames={1}
      />

      <fog attach="fog" args={[pal.fog, 420, 1000]} />
    </group>
  );
}
