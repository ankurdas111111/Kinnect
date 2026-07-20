/**
 * Ask-the-Map client: POSTs a question to /api/ai/ask and consumes the SSE
 * stream via fetch + ReadableStream (EventSource cannot POST).
 *
 * Callbacks (all optional):
 *   onStatus({stage, step})       — agent progress ("thinking" / "answering")
 *   onToolCall({tool, ok, ...})   — one audited tool execution
 *   onNarrative(text)             — streamed narrative delta
 *   onDirectives([...])           — validated map directives
 *   onDone({model, audit, ...})   — terminal success summary
 *   onError(message)              — terminal failure
 *
 * Returns { abort() } to cancel the in-flight request.
 */
import { buildApiUrl, fetchCsrf, getCsrf } from './api.js';

export function askAI(question, callbacks = {}) {
  const controller = new AbortController();

  (async () => {
    try {
      if (!getCsrf()) await fetchCsrf();

      // Send the browser's IANA timezone so "this morning" and rendered times
      // match the user, not the server's zone (UTC in production).
      let timezone = '';
      try { timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || ''; } catch { /* older browsers */ }

      const res = await fetch(buildApiUrl('/api/ai/ask'), {
        method: 'POST',
        credentials: 'include',
        signal: controller.signal,
        headers: {
          'Content-Type': 'application/json',
          'x-csrf-token': getCsrf() || '',
          'Accept': 'text/event-stream'
        },
        body: JSON.stringify({ question, timezone, _csrf: getCsrf() })
      });

      const ct = res.headers.get('content-type') || '';
      if (!ct.includes('text/event-stream')) {
        // Pre-stream rejection (validation, cooldown, disabled) arrives as JSON.
        const data = await res.json().catch(() => ({}));
        callbacks.onError?.(data.error || `Request failed (${res.status})`);
        return;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      let sawTerminal = false;
      const dispatch = (eventName, payload) => {
        let data = {};
        try { data = JSON.parse(payload); } catch { return; }
        switch (eventName) {
          case 'status': callbacks.onStatus?.(data); break;
          case 'tool_call': callbacks.onToolCall?.(data); break;
          case 'narrative_delta': callbacks.onNarrative?.(data.text || ''); break;
          case 'directives': callbacks.onDirectives?.(data.map_directives || []); break;
          case 'done': sawTerminal = true; callbacks.onDone?.(data); break;
          case 'error': sawTerminal = true; callbacks.onError?.(data.message || 'Something went wrong'); break;
        }
      };

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        // SSE frames are separated by a blank line.
        let sep;
        while ((sep = buffer.indexOf('\n\n')) >= 0) {
          const frame = buffer.slice(0, sep);
          buffer = buffer.slice(sep + 2);
          let eventName = 'message';
          const dataLines = [];
          for (const line of frame.split('\n')) {
            if (line.startsWith('event: ')) eventName = line.slice(7).trim();
            else if (line.startsWith('data: ')) dataLines.push(line.slice(6));
          }
          if (dataLines.length) dispatch(eventName, dataLines.join('\n'));
        }
      }

      if (!sawTerminal) callbacks.onError?.('Connection closed before the answer finished');
    } catch (err) {
      if (err?.name !== 'AbortError') callbacks.onError?.('Network error');
    }
  })();

  return { abort: () => controller.abort() };
}

/** Probe whether the AI copilot is enabled server-side. */
export async function aiStatus() {
  try {
    const res = await fetch(buildApiUrl('/api/ai/status'), { credentials: 'include' });
    if (!res.ok) return false;
    const data = await res.json();
    return !!data.enabled;
  } catch {
    return false;
  }
}
