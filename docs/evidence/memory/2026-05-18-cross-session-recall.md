# Cross-session Memory Recall Evidence - 2026-05-18

## Scope

Section 6 asks for: open FORGE, write a note, close, reopen, ask for the note.

The desktop memory page now exposes a compact note composer that records an
operator note through the governed memory observation API. The chat surface can
then ask against the persisted observation after a remount/reopen boundary.

## Automated Evidence

Backend store-reopen prompt inclusion:

```bash
cd services/core && go test ./internal/api -run TestChatLLMMessagesIncludeMemoryObservationsAfterStoreReopen
```

This test records a memory observation, closes the first store/server, reopens
the same data directory, creates a chat thread, asks what should be remembered,
and verifies that `buildChatLLMMessages` includes the reopened memory
observation marker (`basalt notebook`).

Desktop chat surface remount:

```bash
npm -w @forge/desktop run test -- src/pages/ChatPage.test.tsx
```

`apps/desktop/src/pages/ChatPage.test.tsx` commits a memory observation through
the existing memory API seam in the mocked desktop backend, opens the chat page,
creates a thread, unmounts the page to simulate closing the surface, remounts
the chat page, asks from the composer what should be remembered after reopen,
and verifies the rendered assistant reply recalls the marker.

Desktop memory note write path:

```bash
npm -w @forge/desktop run test -- src/pages/MemoryPage.test.tsx
```

`apps/desktop/src/pages/MemoryPage.test.tsx` renders the Memory page, writes a
note through the new `Write Memory Note` panel, verifies that
`api.memory.createObservation` receives a governed `note` observation payload,
refreshes observations, and renders the newly recorded marker.

## Remaining Limitation

This is not a full Tauri process close/reopen video capture. It now covers the
daily-use path in automated seams: GUI note creation through the Memory page,
backend store-reopen recall, and desktop chat remount/ask/render recall.
