// 12 general-purpose map pin icons — used by CustomPinDialog and createCustomPinIcon.
export const PIN_ICONS = [
  { id: 'pin',     emoji: '📍', label: 'Pin' },
  { id: 'home',    emoji: '🏠', label: 'Home' },
  { id: 'work',    emoji: '💼', label: 'Work' },
  { id: 'school',  emoji: '🏫', label: 'School' },
  { id: 'medical', emoji: '🏥', label: 'Medical' },
  { id: 'food',    emoji: '🍽️', label: 'Food' },
  { id: 'shop',    emoji: '🛒', label: 'Shop' },
  { id: 'park',    emoji: '🌳', label: 'Park' },
  { id: 'warning', emoji: '⚠️', label: 'Warning' },
  { id: 'star',    emoji: '⭐', label: 'Favorite' },
  { id: 'flag',    emoji: '🚩', label: 'Flag' },
  { id: 'meetup',  emoji: '🤝', label: 'Meet Here' },
];

export function getPinIcon(id) {
  return PIN_ICONS.find(i => i.id === id) || PIN_ICONS[0];
}
