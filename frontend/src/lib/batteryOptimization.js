/**
 * Wrapper around the native BatteryOptimizationPlugin.
 * Safe to import on web — all calls are no-ops when not on a native platform.
 */
import { registerPlugin } from '@capacitor/core';
import { isNativePlatform } from './geoProvider.js';

// registerPlugin is safe to call at module load time; it lazily resolves the
// native implementation and is a no-op on platforms where the plugin isn't present.
const _plugin = registerPlugin('BatteryOptimization');

/**
 * Returns true if the app is already excluded from Android battery optimization
 * (i.e. "Unrestricted" in battery settings). Always returns true on web.
 */
export async function isIgnoringBatteryOptimizations() {
  if (!isNativePlatform()) return true;
  try {
    const { isIgnoring } = await _plugin.isIgnoring();
    return !!isIgnoring;
  } catch (_) {
    return true; // Assume OK if check fails (e.g. pre-Android 6)
  }
}

/**
 * Opens the Android system dialog that lets the user allow Kinnect to run
 * unrestricted in the background. Falls back to the full battery optimization
 * settings list if the direct dialog is blocked.
 */
export async function requestIgnoreBatteryOptimizations() {
  if (!isNativePlatform()) return;
  try {
    await _plugin.openSettings();
  } catch (_) {}
}
