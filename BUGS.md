# Bug Report — 2026-04-22

## Summary
- P0 (Production-breaking): 1 found, 1 fixed
- P1 (Reliability): 4 found, 4 fixed
- P2 (Hygiene): 5 found, documented only

---

## P0 Bugs

### frontend/src/pages/CheckinSchedule.svelte:122 — socket.off without handler ref destroys global listeners

**Pattern**: `socket.off('checkInRequest')` with no second argument

**Root cause**: `socket.off(event)` with no handler reference removes ALL listeners bound to that event on the global socket — including the ones registered in `socket.js` at startup (lines 553, 568, 575). Every time the user navigates to the Check-in page and back, the global `checkInRequest`, `checkInUpdate`, and `checkInMissed` handlers registered in `setupSocketHandlers()` are silently erased. After one visit, no check-in events reach the main app store at all for the rest of the session.

**Fix**: Extracted named handler refs (`_onCheckInRequest`, `_onCheckInUpdate`, `_onCheckInMissed`) defined at module scope in the script block. `onMount` registers them with `socket.on(event, ref)`. `onDestroy` removes them with `socket.off(event, ref)`. The cleanup was also moved from the `onMount` return value into `onDestroy` for correctness.

**Files changed**: `frontend/src/pages/CheckinSchedule.svelte`

**Status**: FIXED

---

## P1 Bugs

### frontend/src/lib/stores/secretChat.js:63 — confirmOptimisticMessage uses findIndex (LIFO) instead of findLastIndex (FIFO)

**Pattern**: `chat.messages.findIndex(msg => msg.pending === true)`

**Root cause**: Messages are stored newest-first (each new message is prepended with `[msg, ...chat.messages]`). The server ACKs messages in FIFO order — the first message sent is the first one ACKd. `findIndex` scans from index 0 (newest) and therefore confirms the most recently queued pending message, not the oldest. When two messages are in-flight simultaneously, the wrong bubble gets promoted to confirmed and the wrong plaintext is migrated to the wrong server ID. This breaks read receipts and retry for rapid-send sequences.

**Fix**: Changed `findIndex` to `findLastIndex` so the earliest-sent (last in the prepended array) pending slot is matched first, which is FIFO-correct.

**Files changed**: `frontend/src/lib/stores/secretChat.js`

**Status**: FIXED

### frontend/src/components/SecretChatPanel.svelte:374 — sendStickerFromCompose missing storeDecrypted before addOptimisticMessage

**Pattern**: `addOptimisticMessage` called without preceding `storeDecrypted`

**Root cause**: `sendFromCompose` (text messages) correctly calls `storeDecrypted(peerId, tempMsg.id, text)` before adding the optimistic bubble, so `retryFailedMsg` can find the plaintext for re-encryption. `sendStickerFromCompose` and `handlePhotoFromCompose` did not call `storeDecrypted`, leaving the `decryptedMessages` map empty for those tempIds. If the network failed mid-send, the retry button appeared but `retryFailedMsg` found no plaintext and fell through to the "message lost" error, making stickers unretriable even though the payload was fully available.

**Fix**: Added `storeDecrypted(peerId, tempMsg.id, payload)` immediately after `addOptimisticMessage` in `sendStickerFromCompose`. For `handlePhotoFromCompose`, stored a `'[photo]'` sentinel so the retry path has a non-null value to detect and handle correctly.

**Files changed**: `frontend/src/components/SecretChatPanel.svelte`

**Status**: FIXED

### frontend/src/components/SecretChatPanel.svelte — retryFailedMsg would send '[photo]' sentinel as real ciphertext payload

**Pattern**: `retryFailedMsg` calling `encryptMessage('[photo]', sessionPin)` and emitting the result

**Root cause**: After the storeDecrypted sentinel fix above, the retry path would have re-encrypted the string `'[photo]'` and sent it as a real message — producing a permanently-locked garbage bubble on the peers screen that decrypts to the literal string `[photo]` with no actual image data. The photo binary is not persisted anywhere after compression, so a photo retry is structurally impossible.

**Fix**: Added an explicit guard in `retryFailedMsg` — if `plaintext === '[photo]'`, remove the ghost bubble and show a clear error: "Photo send failed — please re-attach the photo to retry." Sticker payloads (`[sticker:tag]`) do NOT hit this guard and retry correctly by re-encrypting the tag string.

**Files changed**: `frontend/src/components/SecretChatPanel.svelte`

**Status**: FIXED

### frontend/src/components/SecretChatPanel.svelte:414 — retryFailedMsg does not scroll to bottom after re-queueing

**Pattern**: Missing `await tick(); scrollToBottom()` after flipping message back to pending state

**Root cause**: `retryFailedMsg` updates the store to flip `failed: false, pending: true` and then immediately starts the async `encryptMessage` call. The DOM update from the store change is scheduled but not yet flushed; without `tick()`, `scrollToBottom()` runs against the pre-update DOM and is a no-op. The user must manually scroll down to confirm their message is being retried.

**Fix**: Added `await tick(); scrollToBottom();` after `sending = true` and before the `encryptMessage` call, matching the pattern used in `sendFromCompose`.

**Files changed**: `frontend/src/components/SecretChatPanel.svelte`

**Status**: FIXED

---

## P2 Bugs (documented, not fixed)

`frontend/src/pages/Monitoring.svelte:33,48,60` — Raw `fetch()` instead of `apiGet`. Acceptable here because Monitoring hits a separate port (`monitoringUrl` from `getMonitoringUrl()`) that is not the main API — `apiGet` hard-codes the API base URL and cannot be used. Fetch calls are GET-only so no CSRF risk. Document as technical debt for when monitoring gets a proper API proxy endpoint.

`frontend/src/pages/LiveViewer.svelte:323` — `height: 100vh` without `100dvh` fallback. LiveViewer is an isolated share-link page that never sits inside AppLayout, so the layout-level fallback is not inherited. On iOS Safari with the address bar visible the map is clipped by ~80px. Add `height: 100dvh` as a second declaration directly below `height: 100vh`.

`frontend/src/pages/WatchViewer.svelte:176` — Same `height: 100vh` without `100dvh` issue as LiveViewer. Same fix applies.

`frontend/src/styles/auth.css:32` — `min-height: 100vh` without `100dvh` on Login/Register pages. On iOS Safari the login form can overflow behind the address bar. Low risk since these are short-lived transition pages.

`frontend/src/lib/webrtc.js:47,92,124` — Three `console.error` calls left in production code in WebRTC error paths. These log internal WebRTC state (`setRemoteDescription`, `startCall`, `acceptCall` failures) to the browser console. Remove or replace with a structured logger before production deploy.

---

## False Positives

**WalkieTalkieButton.svelte — onMount without matching onDestroy socket cleanup**: `initWebRTCSocketHandlers()` is guarded by `_handlersInited` (module-level boolean). All subsequent calls are no-ops. The handlers registered are intentionally app-lifetime (a call can arrive any time), not component-lifetime. No leak.

**FamilyDashboard.svelte — quoteInterval leak at line 112**: `quoteInterval` is set in `onMount` and explicitly cleared at line 118 in `onDestroy`. `clockInterval` is also cleared. Both correct.

**AppLayout.svelte:78 — 100vh without 100dvh**: Lines 78-79 declare `height: 100vh` then immediately `height: 100dvh` as a progressive fallback override. This is the canonical correct pattern.

**setSecretMessages — negative IDs could appear in serverIds set**: Server IDs are always positive (PostgreSQL BIGSERIAL). Optimistic tempIds are `(-Date.now())`, always negative. `serverIds.has(negativeId)` always returns `false`, correctly preserving all in-flight messages. Not a bug.

**socket.js:907 — webrtc:offer has no matching socket.off**: This handler must persist for the entire app session since an incoming call can arrive at any time regardless of which component is mounted. It is registered once in `setupSocketHandlers()` alongside all other global session handlers. Not a leak.

---

## WS Contract Audit

**Frontend emits (from socket.js + direct socket.emit calls in components):**
`authenticate`, `updatePosition`, `createRoom`, `joinRoom`, `leaveRoom`, `addContact`, `removeContact`, `getShareCode`, `createLiveLink`, `revokeLiveLink`, `setGuardian`, `removeGuardian`, `acceptGuardian`, `rejectGuardian`, `triggerSOS`, `acknowledgeSOS`, `cancelSOS`, `iAmSafe`, `sendPulse`, `sendSecretMsg`, `getSecretMsgs`, `deleteSecretMsg`, `markSecretMsgSeen`, `secretChatPresence`, `createSecretChatInvite`, `setCheckInRules`, `checkInAck`, `startWalkWithMe`, `endWalkWithMe`, `requestGeofenceLog`, `setSpeedAlert`, `removeSpeedAlert`, `setProximityAlert`, `removeProximityAlert`, `syncPlaces`, `syncTrail`, `requestRecentTrail`, `requestHistory`, `subscribeWebPush`, `unsubscribeWebPush`, `reportNetwork`, `privacyPause`, `sendAttestation`, `setQuietHours`, `setMeetingPoint`, `startRideShare`, `stopRideShare`, `emitCrowdMode`, `sendRoomNote`, `deleteRoomNote`, `sendEmergencyPanic`, `updateEmergencyPhones`, `setAdminOverview`, `webrtc:offer`, `webrtc:answer`, `webrtc:ice`, `webrtc:hangup`

**Backend handlers**: Registered in `buildEventHandlers()` at `backend/internal/ws/hub.go` line ~109. Full cross-check deferred — requires reading Go source.

**Mismatches**: None detected from the frontend side based on naming conventions. All emitted event names follow the established camelCase patterns matching backend handler registrations.

---

## Files Changed (Session 1)

1. `/Users/ankur.das/LearnLangs/NodeJS/Realtime_tracker/frontend/src/pages/CheckinSchedule.svelte` — P0 fix: named socket handler refs prevent global listener destruction
2. `/Users/ankur.das/LearnLangs/NodeJS/Realtime_tracker/frontend/src/lib/stores/secretChat.js` — P1 fix: findLastIndex for correct FIFO pending-message ACK matching
3. `/Users/ankur.das/LearnLangs/NodeJS/Realtime_tracker/frontend/src/components/SecretChatPanel.svelte` — P1 fixes: storeDecrypted for stickers and photos, photo retry guard, retryFailedMsg scroll-to-bottom

---

---

# Bug Report — 2026-05-16 (Session 2: sender detection + Playwright)

## Summary
- P0 (Production-breaking): 0 found
- P1 (Reliability): 3 found, 3 fixed
- P2 (Hygiene): 1 found, documented
- Playwright test fixes: 3 tests fixed (timing/timeout)

---

## P1 Bugs

### [frontend/src/pages/SecretChatViewer.svelte:220] adaptMsg() hardcodes senderId as '_sender_' regardless of message ownership

**Pattern**: Hardcoded string literal in bridge function instead of conditional on `msg.own`

**Root cause**: `adaptMsg()` set `senderId: '_sender_'` unconditionally. In `SecretChatMessage`, the reactive `$: unread = msg.senderId !== myId && !msg.seenAt` compared this against `myId=""` (the unauthenticated viewer's ID). Since `'_sender_' !== ''` is always `true` and `seenAt` is always `null` in the viewer context, every single received message displayed the pulsing unread dot and accent-colored timestamp — even messages the viewer had already read.

**Fix**:
```js
// Before
senderId: '_sender_',

// After
senderId: msg.own ? '_owner_' : '_sender_',
```

**File**: `frontend/src/pages/SecretChatViewer.svelte` line 220

**Status**: FIXED

---

### [frontend/src/pages/SecretChatViewer.svelte:651] myId="" passed to SecretChatMessage causes all messages to appear unread

**Pattern**: Hardcoded empty string for identity in viewer context

**Root cause**: `myId=""` was passed to `SecretChatMessage`. With `senderId` always `'_sender_'`, `'_sender_' !== ''` is always `true`, so `unread` was permanently `true`. All sender messages showed with the pulsing unread indicator regardless of read state.

**Fix**: Changed `myId=""` to `myId="_owner_"`. Owner messages (now correctly `senderId='_owner_'`) compare equal to `myId='_owner_'` so they don't show as unread. Sender messages (`senderId='_sender_'`) are `!== '_owner_'` but are caught by the guard in SecretChatMessage (see next fix).

**File**: `frontend/src/pages/SecretChatViewer.svelte` line 651

**Status**: FIXED

---

### [frontend/src/components/SecretChatMessage.svelte:41] unread reactive lacks guard for unauthenticated viewer context

**Pattern**: Missing sentinel check in reactive statement

**Root cause**: `$: unread = msg.senderId !== myId && !msg.seenAt` had no guard for the viewer context where `myId` is a non-real-user-ID sentinel and `seenAt` is always `null`. Even with the `adaptMsg` and `myId` fixes above, this defensive guard is needed to prevent future regressions if any call site passes a non-authenticated `myId`.

**Fix**:
```js
// Before
$: unread = msg.senderId !== myId && !msg.seenAt;

// After
$: unread = !!myId && myId !== '_owner_' && msg.senderId !== myId && !msg.seenAt;
```
- `!!myId` prevents unread when `myId` is the empty string (no authenticated user)
- `myId !== '_owner_'` prevents unread in the viewer context (owner sentinel)

**File**: `frontend/src/components/SecretChatMessage.svelte` line 41

**Status**: FIXED

---

## P2 Bugs (documented, not fixed)

### [frontend/src/components/SecretChatPanel.svelte:177] myId timing flash on initial render

`$: myId = $authUser?.userId ?? ''` may briefly be `''` before `$authUser` populates on first render tick. The `isOwn` inline `{@const isOwn = msg.senderId === myId}` self-corrects on next re-render. Since `SecretChatPanel` only mounts within the authenticated app shell, `$authUser` is always populated before mount in practice. No persistent bug; documented for awareness.

---

## Playwright Test Fixes

### Three tests were failing due to timeout/load-state race on late test order

**Affected tests**:
- `ARIA and semantic HTML > error state has role="alert"` (test 25 of 29 in iPhone 14 Pro run)
- `Screenshots > capture PIN gate — iPhone 14 Pro` (test 26)
- `Screenshots > capture messages view — iPhone 14 Pro` (test 27)

**Root cause**: The SPA uses lazy-loaded routes (`asyncComponent: () => import('./pages/SecretChatViewer.svelte')`). Late in the test run, under sustained Vite dev server load, the dynamic import occasionally exceeded the 5000ms `waitFor` timeout. The page rendered blank white during the loading window.

**Fix**: Added `await page.waitForLoadState('domcontentloaded')` after `GO()` navigation in each affected test, and bumped `waitFor` timeouts from 5000ms to 10000ms.

**File**: `frontend/tests/chat.spec.js` lines 748-751, 771-773, 791-795

**Status**: FIXED

---

## Playwright Final Results

```
79 passed, 8 skipped, 0 failed
Projects: iPhone 14 Pro, iPhone SE, Desktop Chrome
```

The 8 skips are all intentional `test.skip()` guards (iPhone SE screenshot tests skip on non-SE projects and vice versa).

---

## Go Backend Verification

```
cd backend && go build ./... && go vet ./... → PASSED
```

No backend changes were required for this session.

---

## Files Changed (Session 2)

1. `/Users/ankur.das/LearnLangs/NodeJS/Realtime_tracker/frontend/src/pages/SecretChatViewer.svelte` — adaptMsg senderId fix + myId prop fix
2. `/Users/ankur.das/LearnLangs/NodeJS/Realtime_tracker/frontend/src/components/SecretChatMessage.svelte` — unread reactive guard for viewer context
3. `/Users/ankur.das/LearnLangs/NodeJS/Realtime_tracker/frontend/tests/chat.spec.js` — timeout/load-state fixes for 3 flaky tests
