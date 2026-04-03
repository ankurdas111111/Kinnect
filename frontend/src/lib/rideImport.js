/**
 * Smart Ride Import — OCR + clipboard parsing for ride-hailing details.
 * Extracts Indian vehicle plates, vehicle type, and ETA from text.
 */

// ── Indian vehicle plate regex ────────────────────────────────────────
// Matches: KA 01 AB 1234, MH02CD5678, DL 3C AU 1234, etc.
const PLATE_RE = /\b([A-Z]{2})\s?[-]?\s?(\d{1,2})\s?[-]?\s?([A-Z]{1,3})\s?[-]?\s?(\d{4})\b/i;

// ── Vehicle type keyword map (ordered by specificity) ─────────────────
const VEHICLE_KEYWORDS = [
  [/\b(rapido|bike)\b/i,                                'scooter'],
  [/\b(auto[- ]?rickshaw|three\s*wheeler)\b/i,          'auto'],
  [/\bauto\b/i,                                          'auto'],
  [/\b(scooter|scooty)\b/i,                             'scooter'],
  [/\b(sedan|hatchback|suv|swift|ertiga|innova|dzire|wagon\s*r|i10|i20|creta|nexon|baleno|alto|verna|city)\b/i, 'car'],
  [/\b(uber|ola|cab|taxi|prime\s*sedan|prime\s*suv|mini|uber\s*go|ola\s*mini|lux)\b/i, 'cab'],
];

// ── ETA extraction patterns ──────────────────────────────────────────
const ETA_PATTERNS = [
  /arriving\s+in\s+(\d+)\s*min/i,
  /(\d+)\s*min(?:ute)?s?\s*(?:away|left)/i,
  /eta\s*:?\s*(\d+)\s*min/i,
  /in\s+(\d+)\s*min/i,
];

/**
 * Parse ride-hailing text and return extracted fields.
 * @param {string} text
 * @returns {{ plate: string|null, vehicleType: string|null, etaMins: number|null }}
 */
export function parseRideText(text) {
  if (!text) return { plate: null, vehicleType: null, etaMins: null };

  // Extract plate
  let plate = null;
  const plateMatch = text.match(PLATE_RE);
  if (plateMatch) {
    plate = `${plateMatch[1].toUpperCase()} ${plateMatch[2]} ${plateMatch[3].toUpperCase()} ${plateMatch[4]}`;
  }

  // Extract vehicle type
  let vehicleType = null;
  for (const [re, type] of VEHICLE_KEYWORDS) {
    if (re.test(text)) {
      vehicleType = type;
      break;
    }
  }

  // Extract ETA minutes
  let etaMins = null;
  for (const re of ETA_PATTERNS) {
    const m = text.match(re);
    if (m) {
      etaMins = parseInt(m[1], 10);
      break;
    }
  }

  return { plate, vehicleType, etaMins };
}

// ── Tesseract.js lazy loading ─────────────────────────────────────────
let _tesseractWorker = null;

async function getWorker() {
  if (_tesseractWorker) return _tesseractWorker;
  const Tesseract = await import('tesseract.js');
  _tesseractWorker = await Tesseract.createWorker('eng');
  return _tesseractWorker;
}

/**
 * Run on-device OCR on an image file.
 * @param {File|Blob} imageFile
 * @returns {Promise<string>} recognized text
 */
export async function ocrImage(imageFile) {
  const worker = await getWorker();
  const { data } = await worker.recognize(imageFile);
  return data.text;
}

/**
 * Read text from clipboard. Throws on permission denial.
 * @returns {Promise<string>}
 */
export async function readClipboardText() {
  return await navigator.clipboard.readText();
}
