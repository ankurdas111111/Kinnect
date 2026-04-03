import { writable } from 'svelte/store';

// rideShare holds the active ride-share state.
// Populated when the server emits 'rideShareStarted'.
export const rideShare = writable({
  active: false,
  token: '',
  vehicle: '',
  vehicleType: '', // 'car' | 'scooter' | 'auto' | 'cab' | 'train' | 'walk' | ''
  dest: '',
  startedAt: 0,
  eta: 0,          // Unix ms timestamp, 0 = not set
});
