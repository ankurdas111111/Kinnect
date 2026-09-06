import { myContacts } from '../stores/contacts.js';
import { myLiveLinks, liveViewers } from '../stores/sos.js';
import { bumpHubBadge } from '../stores/hubBadge.js';
import { focusUser } from '../stores/map.js';
import { getShareOrigin } from '../env.js';

/**
 * Contact-domain socket handlers: contact list, contact action results,
 * live sharing links, live-link viewers, and nudges.
 */
export function register(socket, ctx) {
  const { setBanner } = ctx;

  // Contacts
  socket.on('myContacts', (data) => myContacts.set(data || []));

  // Live links
  socket.on('myLiveLinks', (links) => {
    myLiveLinks.set(links || []);
    // Prune viewer lists for links that no longer exist (stopped/expired).
    const alive = new Set((links || []).map((l) => l.token));
    liveViewers.update((m) => {
      let changed = false;
      for (const token of m.keys()) {
        if (!alive.has(token)) { m.delete(token); changed = true; }
      }
      return changed ? new Map(m) : m;
    });
  });

  // Hearth 05a "Watching now": the link owner sees who opened the link.
  socket.on('liveViewerJoined', (data) => {
    if (!data?.token) return;
    liveViewers.update((m) => {
      const list = m.get(data.token) || [];
      m.set(data.token, [...list, { viewerName: data.viewerName || 'Someone', at: data.at || Date.now() }]);
      return new Map(m);
    });
    setBanner({ type: 'info', text: `${data.viewerName || 'Someone'} is watching your live location`, actions: [] }, 3000);
  });
  socket.on('liveViewerLeft', (data) => {
    if (!data?.token) return;
    liveViewers.update((m) => {
      const list = m.get(data.token) || [];
      const i = list.findIndex((v) => v.viewerName === (data.viewerName || 'Someone'));
      if (i === -1 && !list.length) return m;
      const next = i === -1 ? list.slice(0, -1) : [...list.slice(0, i), ...list.slice(i + 1)];
      if (next.length) m.set(data.token, next); else m.delete(data.token);
      return new Map(m);
    });
  });

  // Hearth 06b "Nudge": a family member asked this user to share again.
  socket.on('nudged', (data) => {
    setBanner({
      type: 'info',
      text: `${data?.fromName || 'Someone'} asked you to check in — turn sharing on when you can`,
      actions: [],
    }, 8000);
  });
  socket.on('nudgeSent', () => {
    setBanner({ type: 'info', text: 'Nudge sent — they’ll get a note from you', actions: [] }, 2500);
  });

  // Contact action results
  socket.on('contactError', (data) => {
    setBanner({ type: 'info', text: data?.message || 'Could not update this contact', actions: [] }, 2500);
  });
  socket.on('contactAdded', (data) => {
    // Hearth 03d: joining is a moment — name the person, then glide the map to
    // their pin once their first position lands (focusUser resolves userIds).
    setBanner({ type: 'info', text: `${data?.displayName || 'Someone'} joined your family`, actions: [] }, 4000);
    if (data?.userId) setTimeout(() => focusUser.set(data.userId), 1200);
    bumpHubBadge(false);
  });
  socket.on('contactRemoved', () => {
    setBanner({ type: 'info', text: 'Contact removed from your list', actions: [] }, 2000);
  });
  socket.on('liveLinkCreated', (data) => {
    const url = getShareOrigin() + '/#/live/' + data.token;
    navigator.clipboard.writeText(url).catch(() => {
      setBanner({ type: 'info', text: 'Share this link: ' + url, actions: [] }, 10000);
      return;
    });
    setBanner({ type: 'info', text: 'Live link copied — share it with anyone', actions: [] }, 2500);
  });
}
