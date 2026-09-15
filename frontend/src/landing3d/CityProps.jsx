import { useLayoutEffect, useMemo, useRef } from 'react';
import * as THREE from 'three';
import { PALETTE } from './world.js';
import { buildWindows, buildTanks, buildTrees, buildLamps } from './props.js';

const dummy = new THREE.Object3D();

/**
 * Windows. At night these ARE the lighting — a city reads as a city because
 * its windows are lit, not because its walls are painted dark. By day they are
 * recessed glass and simply add facade detail.
 *
 * ~1,100 quads in one InstancedMesh, one draw call.
 */
function Windows({ theme }) {
  const lit = useRef();
  const unlit = useRef();
  const data = useMemo(buildWindows, []);
  const night = theme === 'dark';

  const groups = useMemo(() => ({
    on: data.filter(w => w.lit),
    off: data.filter(w => !w.lit),
  }), [data]);

  const place = (mesh, items) => {
    if (!mesh) return;
    items.forEach((w, i) => {
      dummy.position.set(w.p[0], w.p[1], w.p[2]);
      dummy.rotation.set(0, w.ry, 0);
      dummy.updateMatrix();
      mesh.setMatrixAt(i, dummy.matrix);
    });
    mesh.instanceMatrix.needsUpdate = true;
  };
  useLayoutEffect(() => { place(lit.current, groups.on); }, [groups]);
  useLayoutEffect(() => { place(unlit.current, groups.off); }, [groups]);

  return (
    <group>
      <instancedMesh ref={lit} args={[undefined, undefined, groups.on.length]} frustumCulled={false}>
        <planeGeometry args={[1.5, 2.0]} />
        <meshStandardMaterial
          color={night ? '#ffd9a3' : '#8c8378'}
          emissive={night ? '#ffc379' : '#000000'}
          emissiveIntensity={night ? 2.6 : 0}
          roughness={0.35}
          metalness={0.1}
          side={THREE.DoubleSide}
        />
      </instancedMesh>
      <instancedMesh ref={unlit} args={[undefined, undefined, groups.off.length]} frustumCulled={false}>
        <planeGeometry args={[1.5, 2.0]} />
        <meshStandardMaterial
          color={night ? '#1b1712' : '#7d746a'}
          roughness={0.2}
          metalness={0.3}
          side={THREE.DoubleSide}
        />
      </instancedMesh>
    </group>
  );
}

/** Rooftop water tanks — breaks the flat-roof silhouette, and it is what is
 *  actually up there on every roof in Bengaluru. */
function Tanks({ theme }) {
  const ref = useRef();
  const data = useMemo(buildTanks, []);
  useLayoutEffect(() => {
    if (!ref.current) return;
    data.forEach((t, i) => {
      dummy.position.set(t.p[0], t.p[1], t.p[2]);
      dummy.scale.setScalar(t.s);
      dummy.rotation.set(0, 0, 0);
      dummy.updateMatrix();
      ref.current.setMatrixAt(i, dummy.matrix);
    });
    ref.current.instanceMatrix.needsUpdate = true;
  }, [data]);
  return (
    <instancedMesh ref={ref} args={[undefined, undefined, data.length]} castShadow receiveShadow frustumCulled={false}>
      <cylinderGeometry args={[1.5, 1.5, 3, 12]} />
      <meshStandardMaterial color={theme === 'dark' ? '#241f1a' : '#3b352e'} roughness={0.85} />
    </instancedMesh>
  );
}

/** Trees. The park was a flat green polygon; this gives it a canopy. */
function Trees({ theme }) {
  const canopy = useRef();
  const trunk = useRef();
  const data = useMemo(buildTrees, []);
  const pal = PALETTE[theme];

  useLayoutEffect(() => {
    data.forEach((t, i) => {
      dummy.position.set(t.p[0], 4.2 * t.s, t.p[2]);
      dummy.scale.setScalar(t.s);
      dummy.rotation.set(0, i * 1.7, 0);
      dummy.updateMatrix();
      canopy.current?.setMatrixAt(i, dummy.matrix);

      dummy.position.set(t.p[0], 1.6 * t.s, t.p[2]);
      dummy.scale.set(t.s, t.s, t.s);
      dummy.updateMatrix();
      trunk.current?.setMatrixAt(i, dummy.matrix);
    });
    if (canopy.current) canopy.current.instanceMatrix.needsUpdate = true;
    if (trunk.current) trunk.current.instanceMatrix.needsUpdate = true;
  }, [data]);

  return (
    <group>
      <instancedMesh ref={trunk} args={[undefined, undefined, data.length]} castShadow frustumCulled={false}>
        <cylinderGeometry args={[0.35, 0.45, 3.2, 6]} />
        <meshStandardMaterial color={theme === 'dark' ? '#2b241c' : '#6b5844'} roughness={1} />
      </instancedMesh>
      <instancedMesh ref={canopy} args={[undefined, undefined, data.length]} castShadow frustumCulled={false}>
        <icosahedronGeometry args={[3.1, 1]} />
        <meshStandardMaterial color={pal.parkDark} roughness={0.95} flatShading />
      </instancedMesh>
    </group>
  );
}

/** Street lamps. Emissive only — real point lights per lamp would be dozens of
 *  shadow-casting sources, and bloom sells the glow for free. */
function Lamps({ theme }) {
  const head = useRef();
  const post = useRef();
  const data = useMemo(buildLamps, []);
  const night = theme === 'dark';

  useLayoutEffect(() => {
    data.forEach((l, i) => {
      dummy.rotation.set(0, 0, 0);
      dummy.scale.setScalar(1);
      dummy.position.set(l.p[0], l.p[1], l.p[2]);
      dummy.updateMatrix();
      head.current?.setMatrixAt(i, dummy.matrix);
      dummy.position.set(l.p[0], l.p[1] / 2, l.p[2]);
      dummy.updateMatrix();
      post.current?.setMatrixAt(i, dummy.matrix);
    });
    if (head.current) head.current.instanceMatrix.needsUpdate = true;
    if (post.current) post.current.instanceMatrix.needsUpdate = true;
  }, [data]);

  return (
    <group>
      <instancedMesh ref={post} args={[undefined, undefined, data.length]} frustumCulled={false}>
        <cylinderGeometry args={[0.16, 0.2, 6.5, 6]} />
        <meshStandardMaterial color={night ? '#2e2820' : '#6f6659'} roughness={0.9} />
      </instancedMesh>
      <instancedMesh ref={head} args={[undefined, undefined, data.length]} frustumCulled={false}>
        <sphereGeometry args={[0.62, 10, 8]} />
        <meshStandardMaterial
          color={night ? '#ffe2b0' : '#9c9286'}
          emissive={night ? '#ffbe6e' : '#000000'}
          emissiveIntensity={night ? 3.4 : 0}
          roughness={0.4}
        />
      </instancedMesh>
    </group>
  );
}

export default function CityProps({ theme }) {
  return (
    <group>
      <Windows theme={theme} />
      <Tanks theme={theme} />
      <Trees theme={theme} />
      {/* Lamps earn their place only at night, when they are light sources.
          By day they read as nails dropped on the board. */}
      {theme === 'dark' && <Lamps theme={theme} />}
    </group>
  );
}
