<script>
  /**
   * Ask the Map — floating AI copilot bar.
   *
   * Collapsed: a pill button over the map. Expanded: question input,
   * live tool-call chips as the agent works, streamed narrative, and an
   * audit line ("3 tool calls: ..."). Map annotations render via the
   * aiDirectives store consumed by Map.svelte.
   */
  import { tick } from 'svelte';
  import { askAI } from '../lib/aiClient.js';
  import { aiDirectives } from '../lib/stores/map.js';

  let open = $state(false);
  let question = $state('');
  let busy = $state(false);
  let stage = $state('');          // '', 'thinking', 'answering'
  let toolCalls = $state([]);      // {tool, ok, duration_ms}
  let narrative = $state('');
  let toolSummary = $state('');
  let errorMsg = $state('');
  let hasAnnotations = $state(false);
  let inflight = null;
  let inputEl = $state(null);

  const SUGGESTIONS = [
    'Where is everyone right now?',
    'Where did everyone stop today?',
    'Who arrived home in the last 3 hours?',
    "Show me everyone's movement this afternoon"
  ];

  async function toggleOpen() {
    open = !open;
    if (open) {
      await tick();
      inputEl?.focus();
    }
  }

  function resetOutput() {
    toolCalls = [];
    narrative = '';
    toolSummary = '';
    errorMsg = '';
    stage = '';
  }

  function ask(q) {
    const text = (q ?? question).trim();
    if (!text || busy) return;
    question = text;
    resetOutput();
    busy = true;
    stage = 'thinking';

    inflight = askAI(text, {
      onStatus: (s) => { stage = s.stage || stage; },
      onToolCall: (t) => { toolCalls = [...toolCalls, t]; },
      onNarrative: (delta) => { stage = 'answering'; narrative += delta; },
      onDirectives: (directives) => {
        // Always forward — an empty set must clear the previous answer's pins,
        // otherwise stale annotations sit under a new "no data" narrative.
        aiDirectives.set({ directives });
        hasAnnotations = directives.length > 0;
      },
      onDone: (d) => {
        busy = false;
        stage = '';
        toolSummary = d.tool_summary || '';
        // The done event carries the authoritative narrative; streamed deltas can
        // drop characters at chunk boundaries, so prefer it when present.
        if (d.narrative) narrative = d.narrative;
      },
      onError: (message) => {
        busy = false;
        stage = '';
        errorMsg = message;
      }
    });
  }

  function stop() {
    inflight?.abort();
    busy = false;
    stage = '';
  }

  function clearMap() {
    aiDirectives.set({ clear: true });
    hasAnnotations = false;
    resetOutput();
    question = '';
  }

  function onKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      ask();
    } else if (e.key === 'Escape') {
      open = false;
    }
  }

  const TOOL_ICONS = {
    get_current_positions: '📍',
    get_position_history: '🛤️',
    get_dwell_stops: '⏸️',
    get_geofence_events: '🚪',
    get_saved_places: '🏠',
    distance_m: '📏',
    get_daily_activity: '📊'
  };
</script>

{#if !open}
  <button class="ai-fab" onclick={toggleOpen} aria-label="Ask the Map — AI copilot">
    <span class="ai-fab-spark" aria-hidden="true">✦</span>
    Ask the Map
  </button>
{:else}
  <section class="ai-panel" aria-label="Ask the Map AI copilot">
    <header class="ai-head">
      <span class="ai-title"><span class="ai-fab-spark" aria-hidden="true">✦</span> Ask the Map</span>
      <div class="ai-head-actions">
        {#if hasAnnotations}
          <button class="ai-ghost" onclick={clearMap}>Clear map</button>
        {/if}
        <button class="ai-ghost" onclick={toggleOpen} aria-label="Close">✕</button>
      </div>
    </header>

    <div class="ai-input-row">
      <input
        bind:this={inputEl}
        bind:value={question}
        onkeydown={onKeydown}
        placeholder="Where did Dad stop on his way home?"
        maxlength="500"
        disabled={busy}
        aria-label="Question about your family's locations"
      />
      {#if busy}
        <button class="ai-send ai-stop" onclick={stop} aria-label="Stop">■</button>
      {:else}
        <button class="ai-send" onclick={() => ask()} disabled={!question.trim()} aria-label="Ask">→</button>
      {/if}
    </div>

    {#if !busy && !narrative && !errorMsg}
      <div class="ai-suggestions">
        {#each SUGGESTIONS as s}
          <button class="ai-chip ai-suggestion" onclick={() => ask(s)}>{s}</button>
        {/each}
      </div>
    {/if}

    {#if toolCalls.length || stage}
      <div class="ai-activity" aria-live="polite">
        {#each toolCalls as t}
          <span class="ai-chip ai-tool" class:ai-tool-err={!t.ok} title={t.error || `${t.duration_ms}ms`}>
            {TOOL_ICONS[t.tool] || '🔧'} {t.tool.replaceAll('_', ' ')}
            {#if t.ok}<span class="ai-tool-ms">{t.duration_ms}ms</span>{:else}✗{/if}
          </span>
        {/each}
        {#if stage === 'thinking'}
          <span class="ai-chip ai-pulse">thinking…</span>
        {/if}
      </div>
    {/if}

    {#if narrative}
      <p class="ai-narrative">{narrative}{#if stage === 'answering'}<span class="ai-caret" aria-hidden="true"></span>{/if}</p>
    {/if}

    {#if errorMsg}
      <p class="ai-error" role="alert">{errorMsg}</p>
    {/if}

    {#if toolSummary}
      <footer class="ai-audit">Grounded in {toolSummary}</footer>
    {/if}
  </section>
{/if}

<style>
  .ai-fab {
    position: absolute;
    bottom: calc(24px + var(--tabbar-h, 0px));
    left: 50%;
    transform: translateX(-50%);
    z-index: 30;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 10px 18px;
    border: 1px solid color-mix(in oklab, var(--accent-400, #a78bfa) 45%, transparent);
    border-radius: 999px;
    background: color-mix(in oklab, var(--bg-elevated, #171a2e) 82%, transparent);
    color: var(--text-primary, #e7e9f4);
    font-size: 13.5px;
    font-weight: 650;
    cursor: pointer;
    backdrop-filter: blur(10px);
    box-shadow: 0 6px 24px rgba(0, 0, 0, 0.35), 0 0 0 1px rgba(167, 139, 250, 0.08);
    transition: transform 0.18s ease, box-shadow 0.18s ease;
  }
  .ai-fab:hover {
    transform: translateX(-50%) translateY(-2px);
    box-shadow: 0 10px 32px rgba(124, 58, 237, 0.28);
  }
  .ai-fab-spark { color: var(--accent-400, #a78bfa); }

  .ai-panel {
    position: absolute;
    bottom: calc(24px + var(--tabbar-h, 0px));
    left: 50%;
    transform: translateX(-50%);
    z-index: 30;
    width: min(560px, calc(100vw - 28px));
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 14px;
    border: 1px solid color-mix(in oklab, var(--accent-400, #a78bfa) 30%, transparent);
    border-radius: 18px;
    background: color-mix(in oklab, var(--bg-elevated, #171a2e) 92%, transparent);
    backdrop-filter: blur(14px);
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.45);
    animation: ai-panel-in 0.24s cubic-bezier(0.32, 1.4, 0.6, 1) both;
  }
  @keyframes ai-panel-in {
    from { opacity: 0; transform: translateX(-50%) translateY(14px) scale(0.97); }
    to   { opacity: 1; transform: translateX(-50%) translateY(0) scale(1); }
  }

  .ai-head { display: flex; align-items: center; justify-content: space-between; }
  .ai-title { font-size: 13px; font-weight: 700; letter-spacing: 0.02em; color: var(--text-primary, #e7e9f4); }
  .ai-head-actions { display: flex; gap: 6px; }
  .ai-ghost {
    border: none; background: transparent; cursor: pointer;
    color: var(--text-tertiary, #8b90ad); font-size: 12px; padding: 4px 8px; border-radius: 8px;
  }
  .ai-ghost:hover { color: var(--text-primary, #e7e9f4); background: rgba(255, 255, 255, 0.06); }

  .ai-input-row { display: flex; gap: 8px; }
  .ai-input-row input {
    flex: 1;
    padding: 11px 14px;
    border-radius: 12px;
    border: 1px solid var(--border-default, rgba(255,255,255,0.1));
    background: var(--bg-inset, rgba(0,0,0,0.25));
    color: var(--text-primary, #e7e9f4);
    font-size: 14px;
    outline: none;
  }
  .ai-input-row input:focus { border-color: var(--accent-400, #a78bfa); }
  .ai-send {
    width: 42px; border-radius: 12px; border: none; cursor: pointer;
    background: linear-gradient(135deg, var(--accent-500, #8b5cf6), var(--accent-600, #7c3aed));
    color: #fff; font-size: 17px; font-weight: 700;
    transition: filter 0.15s ease;
  }
  .ai-send:disabled { opacity: 0.4; cursor: default; }
  .ai-send:not(:disabled):hover { filter: brightness(1.12); }
  .ai-stop { background: var(--bg-inset, rgba(0,0,0,0.3)); color: var(--text-secondary, #b9bed6); font-size: 12px; }

  .ai-suggestions { display: flex; flex-wrap: wrap; gap: 6px; }
  .ai-chip {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 5px 11px; border-radius: 999px; font-size: 11.5px; line-height: 1.4;
    border: 1px solid var(--border-default, rgba(255,255,255,0.1));
    background: rgba(255, 255, 255, 0.04);
    color: var(--text-secondary, #b9bed6);
  }
  .ai-suggestion { cursor: pointer; transition: border-color 0.15s ease, color 0.15s ease; border: 1px solid var(--border-default, rgba(255,255,255,0.1)); background: transparent; }
  .ai-suggestion:hover { border-color: var(--accent-400, #a78bfa); color: var(--text-primary, #e7e9f4); }

  .ai-activity { display: flex; flex-wrap: wrap; gap: 6px; }
  .ai-tool { animation: ai-chip-in 0.25s ease both; }
  .ai-tool-ms { color: var(--text-tertiary, #8b90ad); font-size: 10px; }
  .ai-tool-err { border-color: rgba(244, 63, 94, 0.5); color: #fda4af; }
  .ai-pulse { animation: ai-pulse 1.4s ease-in-out infinite; }
  @keyframes ai-chip-in { from { opacity: 0; transform: scale(0.85); } to { opacity: 1; transform: scale(1); } }
  @keyframes ai-pulse { 0%, 100% { opacity: 0.45; } 50% { opacity: 1; } }

  .ai-narrative {
    margin: 0; font-size: 14px; line-height: 1.55;
    color: var(--text-primary, #e7e9f4);
  }
  .ai-caret {
    display: inline-block; width: 7px; height: 15px; margin-left: 3px;
    background: var(--accent-400, #a78bfa); vertical-align: text-bottom;
    animation: ai-pulse 0.9s ease-in-out infinite;
  }
  .ai-error { margin: 0; font-size: 13px; color: #fda4af; }
  .ai-audit {
    font-size: 10.5px; color: var(--text-tertiary, #8b90ad);
    border-top: 1px solid var(--border-default, rgba(255,255,255,0.08));
    padding-top: 8px;
  }

  @media (max-width: 640px) {
    .ai-fab { bottom: calc(88px + env(safe-area-inset-bottom, 0px)); }
    .ai-panel { bottom: calc(80px + env(safe-area-inset-bottom, 0px)); width: calc(100vw - 16px); }
  }
</style>
