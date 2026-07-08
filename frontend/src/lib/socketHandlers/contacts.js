import { myContacts } from '../stores/contacts.js';
import { myLiveLinks } from '../stores/sos.js';
import { bumpHubBadge } from '../stores/hubBadge.js';
import { getShareOrigin } from '../env.js';

/**
 * Contact-domain socket handlers: contact list, contact action results,
 * and live sharing links.
 */
export function register(socket, ctx) {
  const { setBanner } = ctx;

  // Contacts
  socket.on('myContacts', (data) => myContacts.set(data || []));

  // Live links
  socket.on('myLiveLinks', (links) => myLiveLinks.set(links || []));

  // Contact action results
  socket.on('contactError', (data) => {
    setBanner({ type: 'info', text: data?.message || 'Could not update this contact', actions: [] }, 2500);
  });
  socket.on('contactAdded', (data) => {
    setBanner({ type: 'info', text: `${data?.displayName || 'Contact'} added to your family`, actions: [] }, 2000);
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
