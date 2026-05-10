// Web Crypto API — available in all modern browsers and Capacitor WebViews.
// No external dependencies. All operations are async.
// PIN is always numbers-only; key derivation via PBKDF2(SHA-256, 100k iterations).

const PBKDF2_ITERATIONS = 100_000;
const KEY_LENGTH = 256;

/** Derive an AES-GCM CryptoKey from a numeric PIN string and raw salt bytes. */
async function deriveKey(pin, saltBytes) {
  const enc = new TextEncoder();
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    enc.encode(pin),
    'PBKDF2',
    false,
    ['deriveKey']
  );
  return crypto.subtle.deriveKey(
    { name: 'PBKDF2', salt: saltBytes, iterations: PBKDF2_ITERATIONS, hash: 'SHA-256' },
    keyMaterial,
    { name: 'AES-GCM', length: KEY_LENGTH },
    false,
    ['encrypt', 'decrypt']
  );
}

const toB64 = (buf) => {
  const bytes = new Uint8Array(buf);
  let binary = '';
  // Process in 8 KB chunks to avoid exceeding V8's max spread argument count (~65 K).
  // Without chunking, large photo buffers (~134 KB) throw RangeError on spread.
  for (let i = 0; i < bytes.length; i += 8192) {
    binary += String.fromCharCode(...bytes.subarray(i, i + 8192));
  }
  return btoa(binary);
};;
const fromB64 = (s) => Uint8Array.from(atob(s), (c) => c.charCodeAt(0));

/**
 * Encrypt plaintext with a numeric PIN.
 * Returns { ciphertext, iv, salt } — all base64 strings.
 * Each call generates fresh random salt (16 bytes) and IV (12 bytes).
 */
export async function encryptMessage(plaintext, pin) {
  const saltBytes = crypto.getRandomValues(new Uint8Array(16));
  const ivBytes = crypto.getRandomValues(new Uint8Array(12));
  const key = await deriveKey(pin, saltBytes);
  const enc = new TextEncoder();
  const ciphertextBuf = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv: ivBytes },
    key,
    enc.encode(plaintext)
  );
  return {
    ciphertext: toB64(ciphertextBuf),
    iv: toB64(ivBytes.buffer),
    salt: toB64(saltBytes.buffer),
  };
}

/**
 * Decrypt a message given its base64 ciphertext, iv, salt, and the PIN.
 * Throws a DOMException ("OperationError") if the PIN is wrong —
 * AES-GCM authentication tag verification fails client-side.
 */
export async function decryptMessage(ciphertext, iv, salt, pin) {
  const saltBytes = fromB64(salt);
  const ivBytes = fromB64(iv);
  const ciphertextBytes = fromB64(ciphertext);
  const key = await deriveKey(pin, saltBytes);
  const plaintextBuf = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: ivBytes },
    key,
    ciphertextBytes
  );
  return new TextDecoder().decode(plaintextBuf);
}
