# Bug Report — 2026-05-16

## Summary
- P0 (Production-breaking): 0 found
- P1 (Reliability): 2 found, 2 fixed
- P2 (Hygiene): 1 found, documented

---

## P1 Bugs

### [frontend/src/lib/socket.js:412-415] Sender identity broken when authUser not yet resolved at socket event time

**Pattern**: `secretMsgSent` handler calls `get(authUser)?.userId` — if the auth store has not loaded yet (race on cold start / slow network), this returns `undefined`. The message is stored as `{ ...msg, senderId: undefined }`.

**Root cause**: The WS socket is initialised and begins receiving events before the `/api/me` response comes back. On the `secretMsgSent` ack (the server's confirmation that a sent message was persisted), `get(authUser)` is still `null`, so `myId` is `undefined`. The message is written to the store with `senderId: undefined`. In `SecretChatPanel`, `isOwn = msg.senderId === myId` evaluates to `undefined === <real-uuid>` = `false` — the sent message renders as a received bubble (left-aligned, locked UI) instead of the own encrypted stub (right-aligned, cipher-text gibberish). All messages sent during the auth resolution window appear on the wrong side.

**Fix**: Added an early-return guard when `myId` is falsy. The `secretMsgsHistory` backfill (emitted on `onMount`) will correctly populate the message once auth has resolved, so no message is permanently lost.

**File**: `frontend/src/lib/socket.js` lines 412-416

```diff
-  socket.on('secretMsgSent', (msg) => {
-    if (!msg || !msg.receiverId) return;
-    const myId = get(authUser)?.userId;
-    addSecretMessage(msg.receiverId, { ...msg, senderId: myId });
-  });
+  socket.on('secretMsgSent', (msg) => {
+    if (!msg || !msg.receiverId) return;
+    const myId = get(authUser)?.userId;
+    // Guard: if auth store hasn't resolved yet, myId is undefined.
+    // A message stored with senderId:undefined would be indistinguishable
+    // from a received message (msg.senderId !== myId → always true).
+    // Drop and rely on the secretMsgsHistory fetch to backfill instead.
+    if (!myId) return;
+    addSecretMessage(msg.receiverId, { ...msg, senderId: myId });
+  });
```

**Status**: FIXED

---

### [frontend/src/components/SecretChatPanel.svelte:177] `myId` yields `undefined` before auth resolves, corrupting unread count and bubble sides

**Pattern**: `$: myId = $authUser?.userId;` — no nullish fallback. `myId` is `undefined` (not `''`) during the first render cycle while `authUser` store is `null`.

**Root cause**: `myId` is passed as a prop to `SecretChatMessage` which declares `export let myId = ''`. When the parent passes `{myId}` where `myId` is `undefined`, Svelte passes `undefined`, overriding the default `''`. Two downstream expressions break:

1. `$: unread = msg.senderId !== myId && !msg.seenAt` — `'some-uuid' !== undefined` is always `true`, so every message shows the unread teal dot during the ~100ms window before auth resolves.
2. `$: unreadCount = sortedMsgs.filter(m => m.senderId !== myId && !m.seenAt).length` — same: the header badge shows an inflated count.
3. `{@const isOwn = msg.senderId === myId}` — `'some-uuid' === undefined` is always `false`, meaning all messages initially render as "received" before Svelte re-runs the reactive after auth resolves. On a slow device this flash is visible.

**Fix**: Coalesce `undefined` to `''` so comparisons are never against `undefined`.

**File**: `frontend/src/components/SecretChatPanel.svelte` line 177

```diff
-  $: myId = $authUser?.userId;
+  $: myId = $authUser?.userId ?? '';
```

**Status**: FIXED

---

## P2 Bugs (documented, not fixed)

### [frontend/src/pages/SecretChatViewer.svelte:172-173] Validator skipped silently when conversation has no `fromOwner` messages

In `SecretChatViewer.unlock()`, PIN correctness is checked by attempting to decrypt the first `fromOwner` message. If the conversation has no such message (only peer-originated messages), `validatorMsg` is `undefined`, the `try/catch` block is skipped, and any PIN unlocks the chat. The ciphertext is still unreadable without the correct PIN so there is no security regression — but it means a wrong PIN will appear to "succeed", rendering locked bubbles that silently fail to decrypt. No CTA or error is shown in that case.

---

## False Positives

- `SecretChatViewer.adaptMsg()` hardcodes `senderId: '_sender_'` — correct. That function is only called for the peer's messages (`isOwn={false}` hardcoded on the call site). `senderId` in the adapted shape is never compared to `myId` in the viewer path.
- `SecretChatPanel.svelte` line 651 `{@const isOwn = msg.senderId === myId}` — the P1 timing issue above only triggers if `secretMsgSent` fires before auth resolves. By the time messages are rendered (`gateOpen === true` requires a PIN submission, which requires auth), `authUser` has always resolved. The template itself is safe; the bug was only in the WS event handler.

---

## WS Contract Audit

**Backend `secretMsgSent` ack payload** (`chat_events.go` lines 60-67):
```
{ id, receiverId, ciphertext, iv, salt, createdAt }
```
No `senderId` field — frontend reconstructs it as `senderId: myId`. This is the source of the P1 race.

**Backend `secretMsgReceived` payload** (`chat_events.go` lines 71-78):
```
{ id, senderId, ciphertext, iv, salt, createdAt }
```
Has `senderId` from the server. Frontend uses directly. No issue.

**Backend `secretMsgsHistory` message shape** (Go struct at `chat_events.go` line 125, JSON tags):
```
{ id, senderId, receiverId, ciphertext, iv, salt, seenAt, createdAt }
```
camelCase JSON tags match frontend usage throughout. No mismatch.

**Mismatches**: None beyond the timing gap fixed in P1 bug 1.

---

## Playwright Test Results

Run after both fixes applied, across all three configured Playwright projects:

| Project | Viewport | Tests | Passed | Skipped | Failed |
|---|---|---|---|---|---|
| Desktop Chrome | 1280×720 | 29 | 25 | 4 | 0 |
| iPhone 14 Pro | 390×844 | 29 | 27 | 2 | 0 |
| iPhone SE | 375×667 | 29 | 27 | 2 | 0 |

**Total: 79 non-skipped tests, 79 passed, 0 failed.**

Skipped tests are project-scoped screenshot captures (e.g. "iPhone 14 Pro" screenshots skipped when running under iPhone SE project) — correct behaviour.

Coverage verified:
- Message bubble alignment (sent = right, received = left)
- PIN gate render and dot visualizer
- iOS auto-zoom prevention (all inputs >= 16px font-size)
- Touch targets >= 44px across all interactive elements
- Panel clipping / compose footer visibility
- Keyboard safe-area padding (visualViewport resize)
- Scroll-to-bottom FAB
- Panic mode (blank screen + restore)
- Orientation change stability
- ARIA roles and live regions
