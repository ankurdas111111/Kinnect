/**
 * emergencyProfile.js — Pure constants, validation, and localStorage
 * migration/back-sync logic for the Emergency Profile surface.
 *
 * Pure module: no DOM, no stores, no Svelte, no side effects. Unit-testable.
 * The functions here are the ONLY authority on the localStorage payload shape,
 * so the byte-compatibility guarantee (legacy single-contact ↔ modern
 * multi-contact) lives in one place with a test file proving it.
 *
 * Extracted verbatim from EmergencyProfile.svelte:
 *   - STORAGE_KEY / DEFAULTS / MAX_CONTACTS / BLOOD_TYPES   (was :17–41)
 *   - isValidPhone / isValidDob                              (was :71–83)
 *   - legacy single-contact migration                        (was :123–131)
 *   - emergencyName / emergencyPhone back-sync on save       (was :171–173)
 *
 * @module emergencyProfile
 */

// ── Storage key ──────────────────────────────────────────────────────────────
export const STORAGE_KEY = 'kinnect_emergency_profile';

// ── Default profile shape ────────────────────────────────────────────────────
export const DEFAULTS = {
  fullName: '',
  dob: '',
  bloodType: '',
  // Legacy single-contact fields — kept for backward compat, synced from contacts[0] on save
  emergencyName: '',
  emergencyPhone: '',
  // Multiple emergency contacts: [{ name, relation, phone, address }]
  emergencyContacts: [],
  conditions: '',
  allergies: '',
  medications: '',
  doctorName: '',
  doctorPhone: '',
  responderNotes: '',
  language: '',
  updatedAt: null,
};

export const MAX_CONTACTS = 5;

export const BLOOD_TYPES = ['A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-'];

// ── Phone validation ─────────────────────────────────────────────────────────
/**
 * @param {string} val
 * @returns {boolean} true when empty (optional) or a plausible phone number
 */
export function isValidPhone(val) {
  if (!val) return true; // optional
  return /^[+\d\s\-().]{7,20}$/.test(val.trim());
}

// ── Date-of-birth validation ─────────────────────────────────────────────────
/**
 * @param {string} val ISO date string (YYYY-MM-DD) or ''
 * @returns {boolean} true when empty or a real past date after 1900
 */
export function isValidDob(val) {
  if (!val) return true;
  const d = new Date(val);
  if (isNaN(d.getTime())) return false;
  const now = new Date();
  return d < now && d.getFullYear() > 1900;
}

// ── Legacy single-contact migration ──────────────────────────────────────────
/**
 * Given a parsed profile object, return a NEW object merged over DEFAULTS with
 * the one-time legacy single-contact migration applied: if there are no
 * emergencyContacts yet but the old flat emergencyName/emergencyPhone fields
 * hold a value, seed emergencyContacts[0] from them.
 *
 * Byte-compatible with the inline logic previously at EmergencyProfile.svelte
 * lines 122–131. Does NOT mutate its argument.
 *
 * @param {object|null|undefined} parsed
 * @returns {object} normalized profile
 */
export function migrateProfile(parsed) {
  const profile = { ...DEFAULTS, ...(parsed || {}) };
  if (!profile.emergencyContacts?.length && (profile.emergencyName || profile.emergencyPhone)) {
    profile.emergencyContacts = [{
      name: profile.emergencyName || '',
      relation: '',
      phone: profile.emergencyPhone || '',
      address: '',
    }];
  }
  return profile;
}

// ── Legacy back-sync on save ─────────────────────────────────────────────────
/**
 * Return a NEW profile object with the legacy flat emergencyName/emergencyPhone
 * fields synced from the first entry of emergencyContacts, mirroring the write
 * previously done inline at EmergencyProfile.svelte lines 170–173. Empty array
 * clears both legacy fields to ''. Does NOT mutate its argument or set updatedAt.
 *
 * @param {object} profile
 * @returns {object} profile with legacy fields back-synced
 */
export function syncLegacyContact(profile) {
  const fc = profile.emergencyContacts?.[0];
  return {
    ...profile,
    emergencyName: fc?.name || '',
    emergencyPhone: fc?.phone || '',
  };
}

// ── Tracked fields for completeness meter ────────────────────────────────────
export const TRACKED_FIELDS = [
  'fullName', 'dob', 'bloodType', 'emergencyContacts',
  'conditions', 'allergies', 'medications', 'doctorName', 'doctorPhone',
];

/**
 * Count how many tracked fields are filled on a profile.
 * @param {object} profile
 * @returns {number}
 */
export function countFilled(profile) {
  return TRACKED_FIELDS.filter((f) => {
    if (f === 'emergencyContacts') return profile.emergencyContacts?.length > 0;
    return profile[f] && String(profile[f]).trim() !== '';
  }).length;
}
