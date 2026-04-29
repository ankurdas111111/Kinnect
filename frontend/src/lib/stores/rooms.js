import { writable } from 'svelte/store';

export const myRooms = writable([]);
export const myShareCode = writable('');
export const myContactInfo = writable({ email: '', mobile: '' });

// F8: room bulletin board — roomCode → note[]
export const roomNotes = writable(new Map());
