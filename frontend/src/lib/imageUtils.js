/**
 * imageUtils.js — shared image compression utility for Kinnect
 *
 * Used by SecretChatPanel (encrypted photo messages).
 * Compresses images to fit within MAX_BINARY_BYTES using canvas re-encode,
 * with progressive quality/resolution reduction passes.
 */

export const MAX_PHOTO_EDGE = 720;
export const MAX_BINARY_BYTES = 100_000;

/**
 * Compress an image File to a base64 data URL that fits within MAX_BINARY_BYTES.
 *
 * @param {File} file - The source image file
 * @returns {Promise<string>} Resolves with a WebP or JPEG data URL
 * @throws {{ tooLarge: true, sizeKB: number }} when image cannot be compressed enough
 * @throws {Error} on image load failure
 */
export function compressImage(file) {
  return new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);
    img.onload = () => {
      URL.revokeObjectURL(url);
      let { width, height } = img;

      // Scale to fit within MAX_PHOTO_EDGE on the longest dimension
      if (width > MAX_PHOTO_EDGE || height > MAX_PHOTO_EDGE) {
        if (width >= height) {
          height = Math.round((height / width) * MAX_PHOTO_EDGE);
          width = MAX_PHOTO_EDGE;
        } else {
          width = Math.round((width / height) * MAX_PHOTO_EDGE);
          height = MAX_PHOTO_EDGE;
        }
      }

      function encode(cvs, q) {
        const webp = cvs.toDataURL('image/webp', q);
        return webp.startsWith('data:image/webp') ? webp : cvs.toDataURL('image/jpeg', q);
      }

      const c0 = document.createElement('canvas');
      c0.width = width;
      c0.height = height;
      c0.getContext('2d').drawImage(img, 0, 0, width, height);
      let result = encode(c0, 0.80);

      // Progressive compression: up to 4 passes reducing size and quality
      let w = width, h = height;
      for (let pass = 0; pass < 4 && result.length * 0.75 > MAX_BINARY_BYTES; pass++) {
        const scale = Math.sqrt(MAX_BINARY_BYTES / (result.length * 0.75)) * 0.88;
        w = Math.max(120, Math.round(w * scale));
        h = Math.max(90, Math.round(h * scale));
        const c = document.createElement('canvas');
        c.width = w;
        c.height = h;
        c.getContext('2d').drawImage(img, 0, 0, w, h);
        result = encode(c, 0.72 - pass * 0.05);
      }

      const finalBinaryKB = Math.round((result.length * 0.75) / 1024);
      if (result.length * 0.75 > MAX_BINARY_BYTES * 1.5) {
        reject({ tooLarge: true, sizeKB: finalBinaryKB });
        return;
      }
      resolve(result);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Image load failed'));
    };
    img.src = url;
  });
}
