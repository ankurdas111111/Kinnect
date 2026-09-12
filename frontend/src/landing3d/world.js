/**
 * The world the landing renders: one Bengaluru neighbourhood, a Tuesday, four
 * people. Pure data — no three.js, no React — so the scene, the scroll beats
 * and the DOM narration all read from the same source.
 *
 * Coordinates are metres on a 220 x 150 board centred on the origin. Real
 * units matter here: the previous version drew buildings the same size as the
 * people standing next to them, which is the single biggest reason it read as
 * a toy rather than a place.
 */

export const BOARD = { w: 220, d: 150 };

// Buildings are true scale (9-34m). People are NOT: at 0.6m they are sub-pixel
// from an aerial camera, and they are the subject of this page. So they render
// as map markers — a head on a stem, raised clear of the rooftops — which is
// the same scale lie every map makes, including Kinnect's own.
export const PERSON_R = 3.6;
export const MARKER_H = 10;

export const PALETTE = {
  light: {
    ground: '#ddd2bf', road: '#b0a288', roadEdge: '#998b6f',
    park: '#c2d6b0', parkDark: '#adc79a', water: '#cbdce4',
    wall: '#f3eee4', wallAlt: '#e7e0d2', roof: '#c2b49d', roofAlt: '#b0a086',
    ink: '#2a231c', fog: '#ded4c2',
    ember: '#b0532c', sage: '#47795b', vermilion: '#b2392e', ochre: '#b9822f',
    m1: '#4a7ba6', m2: '#8d5a8f', m3: '#3f7d78',
  },
  dark: {
    ground: '#221d17', road: '#2b251d', roadEdge: '#332c22',
    park: '#2a3527', parkDark: '#222c20', water: '#243038',
    wall: '#332c23', wallAlt: '#2b2520', roof: '#3d342a', roofAlt: '#332b23',
    ink: '#f2ede4', fog: '#221d17',
    ember: '#e08a5a', sage: '#6fa384', vermilion: '#d45a4a', ochre: '#cf9b45',
    m1: '#6d9fc9', m2: '#b07fb2', m3: '#5da39d',
  },
};

/** Named places. Labels render as DOM, never as 3D text. */
export const PLACES = [
  { id: 'home',    name: 'Home',    x:   4, z:   6 },
  { id: 'school',  name: 'School',  x: -52, z:  30 },
  { id: 'office',  name: 'Office',  x:  62, z: -34 },
  { id: 'market',  name: 'Market',  x:  26, z:  -2 },
  { id: 'tuition', name: 'Tuition', x: -30, z:  40 },
];

/**
 * Buildings. Height and footprint vary on purpose — a street of identical
 * cubes is the tell of a generated scene. `rot` breaks the grid alignment so
 * the block does not read as graph paper.
 */
function block(x, z, w, d, h, rot = 0, alt = false) {
  return { x, z, w, d, h, rot, alt };
}
export const BUILDINGS = [
  // Home cluster — low, domestic
  block(-6, 2, 11, 13, 9, 0.06), block(10, 10, 13, 11, 11, -0.04, true),
  block(-14, 14, 10, 10, 8, 0.1), block(14, -6, 12, 12, 13, 0.02, true),
  // Market strip — taller, denser
  block(28, 4, 16, 12, 16, -0.05), block(30, -12, 12, 14, 21, 0.03, true),
  block(42, -4, 14, 13, 18, 0.07), block(44, 12, 11, 11, 12, -0.02, true),
  // Office district — the tallest things on the board
  block(58, -30, 17, 16, 34, 0.04), block(74, -26, 14, 15, 27, -0.06, true),
  block(60, -48, 15, 14, 30, 0.02), block(78, -44, 13, 13, 23, 0.05, true),
  // School / tuition side — low and wide
  block(-50, 28, 20, 14, 11, -0.03), block(-64, 36, 14, 12, 9, 0.05, true),
  block(-28, 38, 16, 13, 10, 0.02), block(-40, 50, 13, 12, 8, -0.04, true),
  // Scatter so the edges are not empty
  block(-70, -10, 12, 12, 14, 0.03, true), block(-56, -26, 13, 14, 17, -0.05),
  block(6, 44, 12, 11, 10, 0.06, true), block(-18, -30, 14, 13, 19, 0.01),
  block(84, 8, 12, 12, 15, -0.03, true), block(-86, 16, 11, 11, 12, 0.04),
];

/** Arterial roads, as centre-lines. Drawn wide with a darker casing. */
export const ROADS = [
  { pts: [[-110, 22], [-40, 16], [10, 8], [52, -18], [110, -40]], w: 9 },
  { pts: [[18, -75], [22, -20], [16, 20], [24, 75]], w: 7.5 },
  { pts: [[-110, 52], [-30, 46], [30, 40], [110, 34]], w: 6 },
];

/** The park: a real polygon, not a blob. */
export const PARK = [
  [-44, -4], [-30, -12], [-12, -10], [-4, 2], [-8, 16], [-22, 24],
  [-40, 22], [-50, 12],
];

export const WATER = [
  [66, 40], [82, 34], [96, 40], [100, 54], [88, 64], [72, 60], [64, 50],
];

/**
 * The day, as eight beats. Each carries where everyone is and what the page
 * says. `tone` drives the lighting: quiet dusk, a vermilion SOS, a sage
 * all-clear.
 */
export const BEATS = [
  { t: '07:10', line: 'Everyone’s still home.', tone: null,
    p: { you: ['home', 'home'], meera: ['home', 'home'], arjun: ['home', 'home'], nani: ['home', 'home'] } },
  { t: '07:40', line: 'Meera walks to school.', tone: null,
    p: { you: ['home', 'home'], meera: ['school', 'walking'], arjun: ['home', 'leaving'], nani: ['home', 'home'] } },
  { t: '09:05', line: 'Arjun’s at work.', tone: null,
    p: { you: ['home', 'home'], meera: ['school', 'school'], arjun: ['office', 'at work'], nani: ['home', 'home'] } },
  { t: '12:30', line: 'Nothing to report.', tone: null,
    p: { you: ['home', 'home'], meera: ['school', 'school'], arjun: ['office', 'at work'], nani: ['home', 'home'] } },
  { t: '16:40', line: 'Nani’s phone goes quiet.', tone: 'quiet',
    p: { you: ['home', 'home'], meera: ['tuition', 'tuition'], arjun: ['office', 'at work'], nani: ['home', 'quiet'] } },
  { t: '18:52', line: 'Meera holds SOS.', tone: 'sos',
    p: { you: ['home', 'home'], meera: ['tuition', 'SOS'], arjun: ['market', 'on his way'], nani: ['home', 'home'] } },
  { t: '19:04', line: 'She’s safe.', tone: 'safe',
    p: { you: ['home', 'home'], meera: ['tuition', 'safe'], arjun: ['tuition', 'with her'], nani: ['home', 'home'] } },
  { t: '19:30', line: 'Everyone’s settled.', tone: null,
    p: { you: ['home', 'home'], meera: ['home', 'home'], arjun: ['home', 'home'], nani: ['home', 'home'] } },
];

export const PEOPLE = [
  { id: 'you',   name: 'You',   color: 'ember' },
  { id: 'meera', name: 'Meera', color: 'm2' },
  { id: 'arjun', name: 'Arjun', color: 'm1' },
  { id: 'nani',  name: 'Nani',  color: 'm3' },
];

const PLACE_BY_ID = Object.fromEntries(PLACES.map(p => [p.id, p]));

/** Where a person stands at a beat, fanned out so pebbles never fully overlap. */
export function personPos(personId, placeId) {
  const place = PLACE_BY_ID[placeId] || PLACE_BY_ID.home;
  const i = PEOPLE.findIndex(p => p.id === personId);
  const a = (i / PEOPLE.length) * Math.PI * 2;
  return [place.x + Math.cos(a) * 5.5, place.z + Math.sin(a) * 5.5];
}
