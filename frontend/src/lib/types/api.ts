/**
 * api.ts — Shared TypeScript interfaces for all new WebSocket event payloads
 * and HTTP request/response shapes introduced by the 9-feature batch.
 *
 * Naming conventions:
 *   Payload  = shape the CLIENT sends to the server (client → server)
 *   Event    = shape the SERVER sends to the client (server → client)
 *   Request  = HTTP request body
 *   Response = HTTP response body
 */

// ─────────────────────────────────────────────────────────────────────────────
// Primitives
// ─────────────────────────────────────────────────────────────────────────────

export type UserID   = string; // UUID v4
export type SocketID = string; // server-assigned connection id
export type RoomCode = string; // 6-char alphanumeric

// ─────────────────────────────────────────────────────────────────────────────
// F1 — Battery Level in Position Updates
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Extended position payload the frontend emits on each GPS fix.
 * Augments the existing position object — only the new field is declared here.
 */
export interface PositionBatteryExtension {
  /** Integer 0–100, or omitted when the Battery API is unavailable. */
  batteryPct?: number;
}

/**
 * "userMoved" broadcast shape (lean payload used by positionPayload struct).
 * batteryPct was already in the struct; documenting the full shape for frontend
 * consumers to reference.
 */
export interface UserMovedEvent {
  socketId: SocketID;
  userId: UserID;
  lat: number | null;
  lng: number | null;
  speed: number;
  lastUpdate: number;       // unix ms
  batteryPct: number | null;
  online: boolean;
  motionClass: "still" | "walk" | "run" | "vehicle";
  safetyScore: number;
  activityContext: string;
  sosActive: boolean;
}

// ─────────────────────────────────────────────────────────────────────────────
// F2 — "I'm Safe" Broadcast
// ─────────────────────────────────────────────────────────────────────────────

/** Client → Server: "iAmSafe". No payload fields required. */
export type IAmSafePayload = Record<string, never>;

/** Server → Client: "iAmSafe" — broadcast to all visible users. */
export interface IAmSafeEvent {
  userId: UserID;
  displayName: string;
  at: number; // unix ms
}

// ─────────────────────────────────────────────────────────────────────────────
// F3 — Meeting Point Pin Per Room
// ─────────────────────────────────────────────────────────────────────────────

/** Client → Server: "setMeetingPoint". */
export interface SetMeetingPointPayload {
  roomCode: RoomCode;
  lat: number;    // -90 to 90
  lng: number;    // -180 to 180
  label: string;  // max 80 chars; sanitized server-side
}

/** Client → Server: "clearMeetingPoint". */
export interface ClearMeetingPointPayload {
  roomCode: RoomCode;
}

/**
 * Server → Client: "meetingPointUpdated" — broadcast to all room members
 * on both set and clear. On clear, lat/lng/label/setBy are null/""/0.
 */
export interface MeetingPointUpdatedEvent {
  roomCode: RoomCode;
  lat: number | null;
  lng: number | null;
  label: string;
  setBy: UserID | "";     // empty string when cleared
  setAt: number;          // unix ms; 0 when cleared
}

/** Shape stored per room in the frontend rooms store. */
export interface MeetingPoint {
  lat: number;
  lng: number;
  label: string;
  setBy: UserID;
  setAt: number; // unix ms
}

// ─────────────────────────────────────────────────────────────────────────────
// F4 — Inferred Location Label
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Added to the "userUpdate" / "existingUsers" / "visibilityRefresh" broadcast
 * payload by SanitizeUser(). Empty string means no saved place matched.
 * No new WS event — this is a new field on the existing user snapshot.
 */
export interface LocationLabelField {
  locationLabel: string;
}

// ─────────────────────────────────────────────────────────────────────────────
// F5 — Speed Alert Threshold
// ─────────────────────────────────────────────────────────────────────────────

/** Client → Server: "setSpeedAlert". thresholdKmh = 0 disables the alert. */
export interface SetSpeedAlertPayload {
  thresholdKmh: number; // non-negative; 0 = disable
}

/** Server → Client: "speedAlertSet" — ack sent to the configuring user only. */
export interface SpeedAlertSetEvent {
  /** Stored value in m/s. 0 = disabled. */
  thresholdMs: number;
}

/**
 * Server → Client: "speedAlert" — sent to the guardian's socket(s) when
 * the ward's speed exceeds their configured threshold.
 * Rate-limited to one alert per 5 minutes per ward.
 */
export interface SpeedAlertEvent {
  userId: UserID;
  displayName: string;
  /** Current speed in m/s at the moment of alert. */
  speedMs: number;
  /** Configured threshold in m/s. */
  thresholdMs: number;
  timestamp: number; // unix ms
}

// ─────────────────────────────────────────────────────────────────────────────
// F6 — Geofence Entry/Exit Event Log
// ─────────────────────────────────────────────────────────────────────────────

/** Client → Server: "getGeofenceLog". Returns events for the calling user only. */
export type GetGeofenceLogPayload = Record<string, never>;

export interface GeofenceEventEntry {
  id: string;            // UUID
  fenceName: string;     // empty string if the user-geofence has no label
  eventType: "entry" | "exit";
  lat: number;
  lng: number;
  ts: number;            // unix ms
}

/** Server → Client: "geofenceLog" — response to getGeofenceLog. Last 50 entries. */
export interface GeofenceLogEvent {
  events: GeofenceEventEntry[];
}

// ─────────────────────────────────────────────────────────────────────────────
// F7 — Proximity Alerts
// ─────────────────────────────────────────────────────────────────────────────

/** Client → Server: "setProximityAlert". Upserts by (owner, target) pair. */
export interface SetProximityAlertPayload {
  targetUserId: UserID;
  /** Alert fires when target comes within this many metres of the owner. Min 50. */
  radiusM: number;
}

/** Client → Server: "removeProximityAlert". */
export interface RemoveProximityAlertPayload {
  targetUserId: UserID;
}

/** Client → Server: "listProximityAlerts". Returns the caller's configured alerts. */
export type ListProximityAlertsPayload = Record<string, never>;

export interface ProximityAlert {
  id: string;              // UUID
  ownerUserId: UserID;
  targetUserId: UserID;
  targetName: string;
  radiusM: number;
  enabled: boolean;
  lastTriggeredAt: number; // unix ms; 0 = never triggered
  createdAt: number;       // unix ms
}

/** Server → Client: "proximityAlertSet" — ack after setProximityAlert. */
export interface ProximityAlertSetEvent {
  alert: ProximityAlert;
}

/** Server → Client: "proximityAlertRemoved" — ack after removeProximityAlert. */
export interface ProximityAlertRemovedEvent {
  targetUserId: UserID;
}

/** Server → Client: "proximityAlerts" — response to listProximityAlerts. */
export interface ProximityAlertsEvent {
  alerts: ProximityAlert[];
}

/**
 * Server → Client: "proximityAlert" — sent to the alert owner's socket when
 * the target user moves within the configured radius.
 * Rate-limited: one alert per (owner, target) pair per 5 minutes.
 */
export interface ProximityAlertEvent {
  targetUserId: UserID;
  targetName: string;
  /** Distance in metres at the moment of trigger. */
  distanceM: number;
  radiusM: number;
  lat: number;
  lng: number;
  at: number; // unix ms
}

// ─────────────────────────────────────────────────────────────────────────────
// F8 — Family Bulletin Board Per Room
// ─────────────────────────────────────────────────────────────────────────────

/** Client → Server: "postRoomNote". */
export interface PostRoomNotePayload {
  roomCode: RoomCode;
  /** Max 200 chars. Sanitized server-side. */
  body: string;
}

/** Client → Server: "deleteRoomNote". Author or room admin only. */
export interface DeleteRoomNotePayload {
  noteId: string; // UUID
}

/** Client → Server: "getRoomNotes". Returns last 20 notes for the room. */
export interface GetRoomNotesPayload {
  roomCode: RoomCode;
}

export interface RoomNote {
  id: string;              // UUID
  roomCode: RoomCode;
  authorId: UserID;
  authorName: string;
  body: string;
  createdAt: number;       // unix ms
}

/** Server → Client: "roomNotes" — response to getRoomNotes. */
export interface RoomNotesEvent {
  roomCode: RoomCode;
  /** Ordered newest-first, max 20 items. */
  notes: RoomNote[];
}

/** Server → Client: "roomNoteAdded" — broadcast to all online room members. Flat payload (no wrapper). */
export interface RoomNoteAddedEvent {
  roomCode: RoomCode;
  id: string;
  authorId: UserID;
  authorName: string;
  body: string;
  createdAt: number; // unix ms
}

/** Server → Client: "roomNoteDeleted" — broadcast to all online room members. */
export interface RoomNoteDeletedEvent {
  roomCode: RoomCode;
  noteId: string; // UUID
}

// ─────────────────────────────────────────────────────────────────────────────
// F9 — Daily Activity Summary
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Client → Server: "getDailyActivity".
 * If userId is omitted the server returns the caller's own data.
 * Guardians may supply a ward's userId.
 */
export interface GetDailyActivityPayload {
  userId?: UserID;
}

export interface DailyActivityDay {
  /** Calendar date in UTC: "YYYY-MM-DD". */
  date: string;
  /** Accumulated metres for that calendar day. */
  distanceM: number;
  /** Minutes during which the user's motionClass was not "still". */
  activeMinutes: number;
  /** Unix ms of the last upsert for this row. */
  updatedAt: number;
}

/** Server → Client: "dailyActivity" — response to getDailyActivity. Last 7 days. */
export interface DailyActivityEvent {
  userId: UserID;
  /** Ordered most-recent-first, up to 7 entries. */
  days: DailyActivityDay[];
}

// ─────────────────────────────────────────────────────────────────────────────
// Union helpers — useful for socket.on discriminated dispatch in socket.js
// ─────────────────────────────────────────────────────────────────────────────

/** All new server-emitted event names introduced by the 9-feature batch. */
export type NewServerEventName =
  | "iAmSafe"
  | "meetingPointUpdated"
  | "speedAlertSet"
  | "speedAlert"
  | "geofenceLog"
  | "proximityAlertSet"
  | "proximityAlertRemoved"
  | "proximityAlerts"
  | "proximityAlert"
  | "roomNotes"
  | "roomNoteAdded"
  | "roomNoteDeleted"
  | "dailyActivity";

/** All new client-emitted event names introduced by the 9-feature batch. */
export type NewClientEventName =
  | "iAmSafe"
  | "setMeetingPoint"
  | "clearMeetingPoint"
  | "setSpeedAlert"
  | "getGeofenceLog"
  | "setProximityAlert"
  | "removeProximityAlert"
  | "listProximityAlerts"
  | "postRoomNote"
  | "deleteRoomNote"
  | "getRoomNotes"
  | "getDailyActivity";

/** Map from server event name to its payload type. */
export interface ServerEventMap {
  iAmSafe:                IAmSafeEvent;
  meetingPointUpdated:    MeetingPointUpdatedEvent;
  speedAlertSet:          SpeedAlertSetEvent;
  speedAlert:             SpeedAlertEvent;
  geofenceLog:            GeofenceLogEvent;
  proximityAlertSet:      ProximityAlertSetEvent;
  proximityAlertRemoved:  ProximityAlertRemovedEvent;
  proximityAlerts:        ProximityAlertsEvent;
  proximityAlert:         ProximityAlertEvent;
  roomNotes:              RoomNotesEvent;
  roomNoteAdded:          RoomNoteAddedEvent;
  roomNoteDeleted:        RoomNoteDeletedEvent;
  dailyActivity:          DailyActivityEvent;
}

/** Map from client event name to its payload type. */
export interface ClientEventMap {
  iAmSafe:              IAmSafePayload;
  setMeetingPoint:      SetMeetingPointPayload;
  clearMeetingPoint:    ClearMeetingPointPayload;
  setSpeedAlert:        SetSpeedAlertPayload;
  getGeofenceLog:       GetGeofenceLogPayload;
  setProximityAlert:    SetProximityAlertPayload;
  removeProximityAlert: RemoveProximityAlertPayload;
  listProximityAlerts:  ListProximityAlertsPayload;
  postRoomNote:         PostRoomNotePayload;
  deleteRoomNote:       DeleteRoomNotePayload;
  getRoomNotes:         GetRoomNotesPayload;
  getDailyActivity:     GetDailyActivityPayload;
}
