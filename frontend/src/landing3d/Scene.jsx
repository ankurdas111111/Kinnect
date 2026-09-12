import { useMemo, useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import { Environment, ContactShadows, RoundedBox } from '@react-three/drei';
import * as THREE from 'three';
import { BOARD, PALETTE, BUILDINGS, ROADS, PARK, WATER, PERSON_R, MARKER_H } from './world.js';

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
      {/* Key light casts the shadows that give the block its form. A single
          directional source reads as late afternoon; two would flatten it. */}
      <directionalLight
        position={[90, 140, 70]}
        intensity={tone === 'sos' ? 1.05 : 1.45}
        castShadow
        shadow-mapSize={[2048, 2048]}
        shadow-bias={-0.0006}
        shadow-normalBias={0.02}
      >
        <orthographicCamera attach="shadow-camera" args={[-170, 170, 150, -150, 1, 600]} />
      </directionalLight>
      <ambientLight intensity={theme === 'dark' ? 0.22 : 0.34} color={pal.fog} />

      {/* An HDRI environment is what makes the materials stop looking like flat
          paint. It is the cheapest realism available and costs no shadow map. */}
      <Environment preset={theme === 'dark' ? 'night' : 'city'} environmentIntensity={theme === 'dark' ? 0.18 : 0.3} />

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
