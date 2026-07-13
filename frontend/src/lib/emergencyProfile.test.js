/**
 * emergencyProfile.test.js — plain node:test, no framework.
 *
 * Run: node --test frontend/src/lib/emergencyProfile.test.js
 *
 * Proves the localStorage payload stays byte-compatible across the god-file
 * split for the three shapes that matter:
 *   1. legacy single-contact (old flat emergencyName/emergencyPhone)
 *   2. modern multi-contact
 *   3. empty profile
 *
 * "Byte-compatible" here means: the serialized JSON the page writes after a
 * load→save round trip is IDENTICAL to what the pre-split inline logic produced,
 * and re-loading that JSON is idempotent (no drift on the second save).
 */

import test from 'node:test';
import assert from 'node:assert/strict';
import {
  DEFAULTS,
  STORAGE_KEY,
  MAX_CONTACTS,
  BLOOD_TYPES,
  isValidPhone,
  isValidDob,
  migrateProfile,
  syncLegacyContact,
  countFilled,
} from './emergencyProfile.js';

// ── Reference implementations copied VERBATIM from the pre-split inline code ──
// These reproduce EmergencyProfile.svelte:122–131 (load/migrate) and :170–175
// (save/back-sync) so the test asserts the pure functions match them byte-for-byte.

function legacyMigrateInline(parsed) {
  const profile = { ...DEFAULTS, ...parsed };
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

function legacySaveInline(profile, fixedIso) {
  // mutates like the original: sets emergencyName/emergencyPhone from contacts[0]
  const p = { ...profile };
  const fc = p.emergencyContacts?.[0];
  p.emergencyName = fc?.name || '';
  p.emergencyPhone = fc?.phone || '';
  return { ...p, updatedAt: fixedIso };
}

const FIXED_ISO = '2026-07-13T00:00:00.000Z';

// Helper: run the NEW pure-function pipeline the same way the split page does.
function newSave(profile, fixedIso) {
  return { ...syncLegacyContact(profile), updatedAt: fixedIso };
}

// ── 1. Legacy single-contact ─────────────────────────────────────────────────
test('legacy single-contact migrates + saves byte-identically', () => {
  const legacyRaw = {
    fullName: 'Ada Lovelace',
    emergencyName: 'Charles Babbage',
    emergencyPhone: '+44 20 7946 0000',
    bloodType: 'O+',
  };

  const oldMigrated = legacyMigrateInline(legacyRaw);
  const newMigrated = migrateProfile(legacyRaw);
  assert.equal(JSON.stringify(newMigrated), JSON.stringify(oldMigrated));

  // migration seeded emergencyContacts[0] from the flat fields
  assert.equal(newMigrated.emergencyContacts.length, 1);
  assert.equal(newMigrated.emergencyContacts[0].name, 'Charles Babbage');
  assert.equal(newMigrated.emergencyContacts[0].phone, '+44 20 7946 0000');

  const oldSaved = legacySaveInline(oldMigrated, FIXED_ISO);
  const newSaved = newSave(newMigrated, FIXED_ISO);
  assert.equal(JSON.stringify(newSaved), JSON.stringify(oldSaved));

  // idempotency: reload the saved payload and save again → no drift
  const reMigrated = migrateProfile(JSON.parse(JSON.stringify(newSaved)));
  const reSaved = newSave(reMigrated, FIXED_ISO);
  assert.equal(JSON.stringify(reSaved), JSON.stringify(newSaved));
});

// ── 2. Modern multi-contact ──────────────────────────────────────────────────
test('modern multi-contact migrates + saves byte-identically', () => {
  const modernRaw = {
    fullName: 'Grace Hopper',
    emergencyContacts: [
      { name: 'Jean Sammet', relation: 'Colleague', phone: '+1 555 100 2000', address: 'DC' },
      { name: 'John Mauchly', relation: 'Mentor', phone: '+1 555 300 4000', address: '' },
    ],
    allergies: 'Penicillin',
  };

  const oldMigrated = legacyMigrateInline(modernRaw);
  const newMigrated = migrateProfile(modernRaw);
  assert.equal(JSON.stringify(newMigrated), JSON.stringify(oldMigrated));

  // migration must NOT touch existing contacts
  assert.equal(newMigrated.emergencyContacts.length, 2);

  const oldSaved = legacySaveInline(oldMigrated, FIXED_ISO);
  const newSaved = newSave(newMigrated, FIXED_ISO);
  assert.equal(JSON.stringify(newSaved), JSON.stringify(oldSaved));

  // back-sync pulls legacy fields from contacts[0]
  assert.equal(newSaved.emergencyName, 'Jean Sammet');
  assert.equal(newSaved.emergencyPhone, '+1 555 100 2000');
});

// ── 3. Empty profile ─────────────────────────────────────────────────────────
test('empty profile migrates + saves byte-identically', () => {
  const oldMigrated = legacyMigrateInline({});
  const newMigrated = migrateProfile({});
  assert.equal(JSON.stringify(newMigrated), JSON.stringify(oldMigrated));
  assert.equal(newMigrated.emergencyContacts.length, 0);

  const oldSaved = legacySaveInline(oldMigrated, FIXED_ISO);
  const newSaved = newSave(newMigrated, FIXED_ISO);
  assert.equal(JSON.stringify(newSaved), JSON.stringify(oldSaved));
  assert.equal(newSaved.emergencyName, '');
  assert.equal(newSaved.emergencyPhone, '');
});

// migrateProfile must not mutate its argument (defensive copy)
test('migrateProfile does not mutate input', () => {
  const input = { emergencyName: 'X', emergencyPhone: '123456789' };
  const snapshot = JSON.stringify(input);
  migrateProfile(input);
  assert.equal(JSON.stringify(input), snapshot);
});

test('migrateProfile tolerates null/undefined', () => {
  assert.equal(JSON.stringify(migrateProfile(null)), JSON.stringify({ ...DEFAULTS }));
  assert.equal(JSON.stringify(migrateProfile(undefined)), JSON.stringify({ ...DEFAULTS }));
});

// ── Validation ───────────────────────────────────────────────────────────────
test('isValidPhone', () => {
  assert.equal(isValidPhone(''), true);          // optional
  assert.equal(isValidPhone('+1 555 000 0000'), true);
  assert.equal(isValidPhone('(020) 7946-0000'), true);
  assert.equal(isValidPhone('123'), false);      // too short
  assert.equal(isValidPhone('abc1234'), false);  // letters
});

test('isValidDob', () => {
  assert.equal(isValidDob(''), true);
  assert.equal(isValidDob('1990-05-04'), true);
  assert.equal(isValidDob('1850-01-01'), false); // before 1900
  assert.equal(isValidDob('not-a-date'), false);
  const future = new Date(Date.now() + 86_400_000).toISOString().split('T')[0];
  assert.equal(isValidDob(future), false);       // future
});

// ── countFilled ──────────────────────────────────────────────────────────────
test('countFilled counts tracked fields', () => {
  assert.equal(countFilled({ ...DEFAULTS }), 0);
  assert.equal(countFilled({ ...DEFAULTS, fullName: 'A', bloodType: 'O+' }), 2);
  assert.equal(countFilled({ ...DEFAULTS, emergencyContacts: [{ name: 'x' }] }), 1);
  assert.equal(countFilled({ ...DEFAULTS, emergencyContacts: [] }), 0);
});

// ── Constants sanity ─────────────────────────────────────────────────────────
test('constants unchanged', () => {
  assert.equal(STORAGE_KEY, 'kinnect_emergency_profile');
  assert.equal(MAX_CONTACTS, 5);
  assert.deepEqual(BLOOD_TYPES, ['A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-']);
});
