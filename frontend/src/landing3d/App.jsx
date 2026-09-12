import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ViewCanvas from './ViewCanvas.jsx';
import { BEATS, PEOPLE, PLACES, personPos } from './world.js';

gsap.registerPlugin(ScrollTrigger);

const prefersReduced = () =>
  typeof matchMedia !== 'undefined' && matchMedia('(prefers-reduced-motion: reduce)').matches;

/**
 * Camera framing per beat. These are AERIAL shots: the board is 220m across,
 * so the camera has to sit well outside it or the nearest buildings loom and
 * the scene reads as a pile of boxes rather than a neighbourhood.
 * Rule of thumb here: height ≈ 1.4× the distance you want to see across.
 */
const SHOTS = [
  { pos: [30, 250, 300], look: [10, -6, 4], fov: 30, parallax: 10 },     // establishing
  { pos: [-46, 225, 275], look: [-30, -6, 20], fov: 29, parallax: 9 },  // school walk
  { pos: [78, 235, 255], look: [58, -6, -28], fov: 29, parallax: 9 },   // office
  { pos: [20, 275, 330], look: [10, -6, 4], fov: 31, parallax: 11 },     // quiet noon
  { pos: [14, 215, 258], look: [4, -6, 8], fov: 29, parallax: 8 },      // Nani
  { pos: [-38, 195, 235], look: [-30, -6, 38], fov: 28, parallax: 7 },  // SOS
  { pos: [-38, 205, 245], look: [-30, -6, 38], fov: 28, parallax: 7 },  // safe
  { pos: [26, 290, 350], look: [10, -6, 4], fov: 32, parallax: 12 },     // settled
];

function peopleAt(beatIndex) {
  const beat = BEATS[Math.max(0, Math.min(BEATS.length - 1, beatIndex))];
  return PEOPLE.map(p => {
    const [placeId, status] = beat.p[p.id];
    const [x, z] = personPos(p.id, placeId);
    return { ...p, pos: [x, z], status };
  });
}

export default function App({ onExit }) {
  const reduced = useMemo(prefersReduced, []);
  const [theme, setTheme] = useState(
    () => (document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'light')
  );
  const [beat, setBeat] = useState(0);
  const [ready, setReady] = useState(false);
  const rootRef = useRef(null);
  const target = useRef({ ...SHOTS[0] });

  // Theme is owned by the Svelte app; mirror it rather than duplicating it.
  useEffect(() => {
    const mo = new MutationObserver(() =>
      setTheme(document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'light')
    );
    mo.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
    return () => mo.disconnect();
  }, []);

  // One ScrollTrigger per beat. Each sets the camera target; the rig damps to
  // it. React state only changes on a beat boundary, never per frame.
  useLayoutEffect(() => {
    if (!rootRef.current) return;
    const ctx = gsap.context(() => {
      const sections = gsap.utils.toArray('.l3d-beat');
      sections.forEach((el, i) => {
        ScrollTrigger.create({
          trigger: el,
          start: 'top 62%',
          end: 'bottom 38%',
          onToggle: (self) => {
            if (!self.isActive) return;
            setBeat(i);
            Object.assign(target.current, SHOTS[i] || SHOTS[SHOTS.length - 1]);
          },
        });
      });
    }, rootRef);
    const raf = requestAnimationFrame(() => ScrollTrigger.refresh());
    return () => { cancelAnimationFrame(raf); ctx.revert(); };
  }, []);

  useEffect(() => {
    const t = setTimeout(() => setReady(true), 120);
    return () => clearTimeout(t);
  }, []);

  const people = useMemo(() => peopleAt(beat), [beat]);
  const tone = BEATS[beat]?.tone ?? null;

  return (
    <div className="l3d" ref={rootRef} data-tone={tone || 'calm'}>
      <div className={`l3d-stage ${ready ? 'is-ready' : ''}`} aria-hidden="true">
        <ViewCanvas theme={theme} tone={tone} people={people} target={target} reduced={reduced} />
      </div>

      <header className="l3d-nav">
        <span className="l3d-word">Kinnect</span>
        <nav className="l3d-nav-actions">
          <a className="l3d-link" href="#/login" onClick={onExit}>Sign in</a>
          <a className="l3d-pill" href="#/register" onClick={onExit}>Create your family</a>
        </nav>
      </header>

      <section className="l3d-hero">
        <p className="l3d-eyebrow">Family location sharing, without the noise</p>
        <h1 className="l3d-h1">A quiet map for the people you love.</h1>
        <p className="l3d-sub">
          Your family as a few pebbles, one honest sentence about how everyone’s
          doing, and nothing else. No feeds. No pings. No dashboard.
        </p>
        <div className="l3d-cta-row">
          <a className="l3d-cta" href="#/register" onClick={onExit}>Create your family — free</a>
          <a className="l3d-cta-quiet" href="#story">Walk one Tuesday with the Nairs ↓</a>
        </div>
      </section>

      {/* Each beat is a real DOM section: the narration is readable with the
          canvas off, by a screen reader, and by a crawler. */}
      <div id="story" className="l3d-story">
        {BEATS.map((b, i) => (
          <section className="l3d-beat" key={b.t} aria-label={`${b.t} — ${b.line}`}>
            <div className={`l3d-card ${beat === i ? 'is-live' : ''}`} data-tone={b.tone || 'calm'}>
              <span className="l3d-time">{b.t}</span>
              <p className="l3d-line">{b.line}</p>
              <p className="l3d-where">
                {PEOPLE.map(p => `${p.name} · ${b.p[p.id][1]}`).join('   ')}
              </p>
            </div>
          </section>
        ))}
      </div>

      <section className="l3d-close">
        <p className="l3d-eyebrow">That’s the whole app</p>
        <h2 className="l3d-h2">Everyone’s settled.</h2>
        <p className="l3d-sub">
          One sentence, a few pebbles, and a hold-to-send SOS for the one evening
          in a thousand. Only people you invite can see anyone — ever.
        </p>
        <a className="l3d-cta" href="#/register" onClick={onExit}>Create your family — free</a>
        <footer className="l3d-foot">
          <span className="l3d-word">Kinnect</span>
          <span>Privacy-first</span><span>No ads, ever</span><span>Bengaluru</span>
        </footer>
      </section>
    </div>
  );
}
