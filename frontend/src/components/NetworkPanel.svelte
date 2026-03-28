<script>
  import { onMount } from 'svelte';
  import { socket } from '../lib/socket.js';
  import { networkGraph } from '../lib/stores/network.js';
  import { focusUser } from '../lib/stores/map.js';
  import { getUserColor } from '../lib/getUserColor.js';

  export let embedded = false;

  const edgeColors = { contact: '#818cf8', guardian: '#f59e0b', ward: '#f59e0b', room: '#14b8a6' };

  function refresh() {
    socket.emit('getNetworkGraph');
  }

  onMount(refresh);

  // Simple spring-force layout — runs once when graph data arrives
  let layoutNodes = [];
  let layoutEdges = [];
  const W = 320, H = 320;
  const CX = W / 2, CY = H / 2;

  $: if ($networkGraph) {
    const nodes = $networkGraph.nodes || [];
    const edges = $networkGraph.edges || [];

    // Place nodes in a circle initially
    const count = nodes.length;
    const r = Math.min(CX, CY) * 0.72;
    layoutNodes = nodes.map((n, i) => {
      const angle = (2 * Math.PI * i) / count - Math.PI / 2;
      // Self node goes to center
      return {
        ...n,
        x: n.role === 'self' ? CX : CX + r * Math.cos(angle),
        y: n.role === 'self' ? CY : CY + r * Math.sin(angle),
      };
    });

    // Run a few iterations of force-directed layout
    const nodeMap = new Map(layoutNodes.map(n => [n.id, n]));
    for (let iter = 0; iter < 60; iter++) {
      // Repulsion
      for (let i = 0; i < layoutNodes.length; i++) {
        for (let j = i + 1; j < layoutNodes.length; j++) {
          const a = layoutNodes[i], b = layoutNodes[j];
          const dx = b.x - a.x, dy = b.y - a.y;
          const dist = Math.sqrt(dx * dx + dy * dy) || 1;
          const f = 1200 / (dist * dist);
          const fx = (dx / dist) * f, fy = (dy / dist) * f;
          a.x -= fx * 0.15; a.y -= fy * 0.15;
          b.x += fx * 0.15; b.y += fy * 0.15;
        }
      }
      // Attraction along edges
      for (const e of edges) {
        const s = nodeMap.get(e.source), t = nodeMap.get(e.target);
        if (!s || !t) continue;
        const dx = t.x - s.x, dy = t.y - s.y;
        const dist = Math.sqrt(dx * dx + dy * dy) || 1;
        const f = (dist - 80) * 0.04;
        const fx = (dx / dist) * f, fy = (dy / dist) * f;
        if (s.role !== 'self') { s.x += fx; s.y += fy; }
        if (t.role !== 'self') { t.x -= fx; t.y -= fy; }
      }
      // Clamp
      for (const n of layoutNodes) {
        n.x = Math.max(24, Math.min(W - 24, n.x));
        n.y = Math.max(24, Math.min(H - 24, n.y));
      }
    }

    const nodeMapFinal = new Map(layoutNodes.map(n => [n.id, n]));
    layoutEdges = edges.map(e => ({
      ...e,
      x1: nodeMapFinal.get(e.source)?.x ?? CX,
      y1: nodeMapFinal.get(e.source)?.y ?? CY,
      x2: nodeMapFinal.get(e.target)?.x ?? CX,
      y2: nodeMapFinal.get(e.target)?.y ?? CY,
    }));
  }

  function clickNode(node) {
    // Find the socket ID for this user and focus on map
    focusUser.set(node.id);
  }
</script>

<div class="panel-body network-panel" class:embedded>
  <div class="network-header">
    <span class="card-eyebrow">Your Network</span>
    <button class="refresh-btn" on:click={refresh} aria-label="Refresh network graph">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-.07-8.14"/></svg>
    </button>
  </div>

  {#if !$networkGraph}
    <p class="empty-state">Loading network…</p>
  {:else if layoutNodes.length <= 1}
    <p class="empty-state">Add contacts or join rooms to see your network.</p>
  {:else}
    <!-- Legend -->
    <div class="legend-row">
      <span class="legend-item"><span class="legend-dot" style="background:#818cf8"></span>Contact</span>
      <span class="legend-item"><span class="legend-dot" style="background:#f59e0b"></span>Guardian</span>
      <span class="legend-item"><span class="legend-dot" style="background:#14b8a6"></span>Room</span>
    </div>

    <svg width={W} height={H} viewBox="0 0 {W} {H}" class="network-svg">
      <!-- Edges -->
      {#each layoutEdges as e}
        <line
          x1={e.x1} y1={e.y1} x2={e.x2} y2={e.y2}
          stroke={edgeColors[e.kind] || '#6b7280'}
          stroke-width="1.5"
          stroke-opacity="0.5"
        />
      {/each}
      <!-- Nodes -->
      {#each layoutNodes as n}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <g
          class="network-node"
          transform="translate({n.x},{n.y})"
          on:click={() => clickNode(n)}
          role="button"
          tabindex="0"
          aria-label="{n.name} — {n.role}"
        >
          <circle
            r={n.role === 'self' ? 18 : 14}
            fill={n.role === 'self' ? '#818cf8' : getUserColor(n.id)}
            fill-opacity={n.online ? 1 : 0.4}
            stroke={n.online ? 'rgba(255,255,255,0.7)' : 'rgba(255,255,255,0.2)'}
            stroke-width="1.5"
          />
          {#if !n.online}
            <circle r={n.role === 'self' ? 18 : 14} fill="none" stroke="#6b7280" stroke-width="1.5" stroke-dasharray="3 2"/>
          {/if}
          <text
            text-anchor="middle"
            dominant-baseline="central"
            font-size={n.role === 'self' ? '10' : '8'}
            font-weight="700"
            fill="white"
            style="user-select:none;pointer-events:none;"
          >
            {(n.name || '?').split(' ').map(w => w[0] || '').join('').toUpperCase().slice(0, 2)}
          </text>
          <text
            y={n.role === 'self' ? 26 : 22}
            text-anchor="middle"
            font-size="8"
            fill="var(--text-secondary)"
            style="user-select:none;pointer-events:none;"
          >
            {(n.name || '').split(' ')[0]}
          </text>
        </g>
      {/each}
    </svg>

    <p class="network-hint">Tap a node to locate on map</p>
  {/if}
</div>

<style>
  .network-panel { padding: var(--space-4); }
  .network-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-2); }
  .refresh-btn { background: none; border: none; cursor: pointer; color: var(--text-secondary); padding: 4px; border-radius: var(--radius-md); }
  .refresh-btn:hover { color: var(--text-primary); background: var(--surface-2); }
  .network-svg { display: block; margin: 0 auto; border-radius: var(--radius-xl); background: var(--surface-1); }
  .network-node { cursor: pointer; }
  .network-node:hover circle:first-child { stroke-width: 2.5; }
  .legend-row { display: flex; gap: var(--space-3); margin-bottom: var(--space-2); flex-wrap: wrap; }
  .legend-item { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--text-secondary); }
  .legend-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
  .network-hint { text-align: center; font-size: 11px; color: var(--text-tertiary); margin-top: var(--space-2); }
  .empty-state { text-align: center; color: var(--text-secondary); font-size: 13px; padding: var(--space-8) 0; }
</style>
