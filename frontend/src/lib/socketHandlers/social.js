import { pulseMap } from '../stores/pulses.js';
import { rideShare } from '../stores/rideShare.js';

/**
 * Social/presence socket handlers: pulse check-ins, ride sharing, and
 * crowd mode drift alerts.
 */
export function register(socket, ctx) {
  const { setBanner } = ctx;

  // Pulse Check-In — ephemeral heartbeat from a contact
  socket.on('pulseReceived', (data) => {
    if (!data || !data.userId) return;
    pulseMap.update(m => {
      const nm = new Map(m);
      nm.set(data.userId, data);
      return nm;
    });
    // Auto-expire after 30s
    setTimeout(() => {
      pulseMap.update(m => { const nm = new Map(m); nm.delete(data.userId); return nm; });
    }, 30000);
    if (data.type === 'ok') {
      setBanner({ type: 'info', text: `${data.displayName || 'Someone'} says they're okay`, actions: [] }, 5000);
    } else if (data.type === 'callme') {
      setBanner({ type: 'sos', text: `${data.displayName || 'Someone'} is asking you to call them`, actions: [
        { label: 'Got it', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
      ] }, 8000);
    }
  });

  // Share My Ride — server confirms ride link was created
  socket.on('rideShareStarted', (data) => {
    if (!data?.token) return;
    // Use update() to preserve locally-set vehicleType and eta from the component
    rideShare.update(current => ({
      ...current,
      active: true,
      token: data.token,
      vehicle: data.vehicle || '',
      dest: data.dest || '',
      startedAt: Date.now(),
    }));
  });

  // Ride ended by server (token expired or endRide called from another device)
  socket.on('rideShareError', (data) => {
    setBanner({ type: 'info', text: data?.message || 'Could not share your ride right now', actions: [] }, 3000);
  });

  // Crowd Mode — someone in your festival group has drifted too far
  socket.on('crowdAlert', (data) => {
    if (!data?.fromName) return;
    const dist = data.distanceM != null ? `${data.distanceM}m` : 'far';
    setBanner({
      type: 'info',
      text: `${data.fromName} has drifted ${dist} away from your group`,
      actions: [
        { label: 'OK', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
      ]
    }, 12000);
  });
}
