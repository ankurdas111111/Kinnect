/**
 * landingStory — data spine for the Landing scrollytelling section
 * ("A day with your family, told by scroll").
 *
 * Coordinate space: 300×200 — the story-stage SVG viewBox inside the single
 * hero/story phone mockup. Scene 0 is the hero's resting frame; scenes 1–4
 * belong to the four story beats (morning dispersal → geofence arrival →
 * SOS-that-resolves-safe → evening all-home).
 *
 * HARD CAP: ≤ 6 concurrently animated nodes per scene
 * (3 pins + 1 geofence ring + 1 SOS ring + 1 chip). Do not add more —
 * mid-range Android WebViews jank beyond this.
 *
 * All hues are CSS token reference strings (raw hex here is a lint violation).
 */

export const STAGE_W = 300;
export const STAGE_H = 200;

/** Named places on the miniature map. `ring: true` places own a geofence circle. */
export const PLACES = [
  { id: 'home',   label: 'Home',   x: 78,  y: 64,  ring: true },
  { id: 'office', label: 'Office', x: 212, y: 40,  ring: false },
  { id: 'school', label: 'School', x: 228, y: 146, ring: true },
];

/** The three demo family members. Member wheel tokens, never --primary-*. */
export const PINS = [
  { id: 'mom',  label: 'Mom',  hue: 'var(--member-1)' },
  { id: 'dad',  label: 'Dad',  hue: 'var(--member-2)' },
  { id: 'zara', label: 'Zara', hue: 'var(--member-3)' },
];

/** Faint commute routes, revealed during the travel beats. */
export const ROUTES = [
  { id: 'dad',  hue: 'var(--member-2)', points: '78,64 128,52 172,44 212,40' },
  { id: 'zara', hue: 'var(--member-3)', points: '78,64 120,98 178,126 228,146' },
];

/** Story beat copy — the scrolling left column. Times are the narrative clock. */
export const BEATS = [
  {
    time: '8:00 AM',
    title: 'Everyone heads out',
    desc: 'Dad rolls toward the office, Zara walks to school. Their pins glide across the map on their usual routes — no “did you leave yet?” texts.',
  },
  {
    time: '9:02 AM',
    title: 'Zara arrives at school',
    desc: 'The school geofence lights up green and a quiet arrival chip confirms it. You never had to ask, and she never had to answer.',
  },
  {
    time: '3:40 PM',
    title: 'One tap when it matters',
    desc: 'A hard stop on the way home. Zara holds SOS — every pin turns toward her, live. Two minutes later: marked safe, family notified.',
  },
  {
    time: '8:14 PM',
    title: 'Everyone home',
    desc: 'Pins converge on Home. All safe. You just lived your day — Kinnect kept the watch so you didn’t have to.',
  },
];

/**
 * Scene states, index-aligned with `activeBeat` (0 = hero resting frame).
 * pos — pin coordinates; geofence — which ring is lit and its tone;
 * routes — commute lines visible; chip — floating status chip;
 * rows — mockup member-list sublines, index-aligned with PINS.
 */
export const SCENES = [
  { // 0 — hero resting frame
    pos: { mom: [78, 64], dad: [150, 52], zara: [148, 118] },
    geofence: { id: 'home', tone: 'idle' },
    routes: false,
    chip: { icon: '✓', tone: 'safe', text: 'All safe' },
    rows: [
      { loc: 'Home',     ago: 'just now' },
      { loc: 'Nearby',   ago: '1 min ago' },
      { loc: 'Nearby',   ago: 'just now' },
    ],
  },
  { // 1 — 8:00 AM, dispersal along faint routes
    pos: { mom: [78, 64], dad: [172, 44], zara: [178, 126] },
    geofence: { id: 'home', tone: 'idle' },
    routes: true,
    chip: { icon: '→', tone: 'info', text: '2 heading out' },
    rows: [
      { loc: 'Home',     ago: 'just now' },
      { loc: 'En route', ago: 'just now' },
      { loc: 'En route', ago: 'just now' },
    ],
  },
  { // 2 — 9:02 AM, school geofence ignites
    pos: { mom: [78, 64], dad: [212, 40], zara: [228, 146] },
    geofence: { id: 'school', tone: 'safe' },
    routes: true,
    chip: { icon: '📍', tone: 'safe', text: 'Zara arrived · School' },
    rows: [
      { loc: 'Home',   ago: '4 min ago' },
      { loc: 'Office', ago: '2 min ago' },
      { loc: 'School', ago: 'just now' },
    ],
  },
  { // 3 — 3:40 PM, SOS beat (chip comes from SOS_CHIPS, phase-driven)
    pos: { mom: [124, 96], dad: [196, 84], zara: [182, 122] },
    geofence: null,
    routes: false,
    chip: null,
    rows: [
      { loc: 'Moving to Zara', ago: 'live' },
      { loc: 'Moving to Zara', ago: 'live' },
      { loc: 'Maple Ave',      ago: 'live' },
    ],
  },
  { // 4 — 8:14 PM, convergence home
    pos: { mom: [72, 60], dad: [88, 72], zara: [68, 76] },
    geofence: { id: 'home', tone: 'safe' },
    routes: false,
    chip: { icon: '✓', tone: 'safe', text: 'Everyone home' },
    rows: [
      { loc: 'Home', ago: 'just now' },
      { loc: 'Home', ago: 'just now' },
      { loc: 'Home', ago: 'just now' },
    ],
  },
];

/**
 * SOS beat chips. The alert chip may only ever show transiently — the beat
 * MUST resolve to `safe` so the page never rests on an alarming frame.
 */
export const SOS_CHIPS = {
  alert: { icon: '⚠', tone: 'alert', text: 'SOS — Zara' },
  safe:  { icon: '✓', tone: 'safe',  text: 'Marked safe · Family notified' },
};
