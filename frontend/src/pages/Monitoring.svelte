<script>
	import { onMount, onDestroy } from 'svelte';
	import { writable } from 'svelte/store';
	import Card from '../components/primitives/Card.svelte';
	import Skeleton from '../components/primitives/Skeleton.svelte';
	import SectionHeader from '../components/primitives/SectionHeader.svelte';
	import StatusBadge from '../components/primitives/StatusBadge.svelte';
	import EmptyState from '../components/primitives/EmptyState.svelte';
	import { formatAge } from '../lib/presence.js';

	let healthData = writable(null);
	let diagnosticsData = writable(null);
	let metricsData = writable(null);
	let loading = writable(true);
	let error = writable(null);

	// Honesty fix: per-feed last-success timestamps
	let lastSuccessHealth = writable(null);
	let lastSuccessDiagnostics = writable(null);
	let lastSuccessMetrics = writable(null);
	let staleDiagnostics = writable(false);
	let staleMetrics = writable(false);

	// Ticking now for formatAge
	let now = Date.now();
	let tickInterval;

	let healthInterval;
	let diagnosticsInterval;
	let metricsInterval;

	// Get monitoring endpoint (works for both local and render)
	const getMonitoringUrl = () => {
		const isDev = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
		if (isDev) {
			// Local: separate monitoring port
			return 'http://localhost:9090';
		}
		// For Render/production: use same domain and port (monitoring merged into main port)
		return `${window.location.origin}/api`;
	};

	const monitoringUrl = getMonitoringUrl();

	const fetchHealth = async () => {
		try {
			const isDev = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
			const url = isDev ? `${monitoringUrl}/health` : `${monitoringUrl}/health`;
			const response = await fetch(url);
			if (!response.ok) throw new Error(`HTTP ${response.status}`);
			const data = await response.json();
			healthData.set(data);
			error.set(null);
			lastSuccessHealth.set(Date.now());
		} catch (e) {
			error.set(`Health check failed: ${e.message}`);
			// lastSuccessHealth not updated — keeps stale stamp visible
		}
	};

	const fetchDiagnostics = async () => {
		try {
			const isDev = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
			const url = isDev ? `${monitoringUrl}/diagnostics` : `${monitoringUrl}/diagnostics`;
			const response = await fetch(url);
			if (!response.ok) throw new Error(`HTTP ${response.status}`);
			const data = await response.json();
			diagnosticsData.set(data);
			lastSuccessDiagnostics.set(Date.now());
			staleDiagnostics.set(false);
		} catch (e) {
			// Keep stale data visible; flag staleness
			staleDiagnostics.set(true);
		}
	};

	const fetchMetrics = async () => {
		try {
			const isDev = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
			const url = isDev ? `${monitoringUrl}/metrics` : `${monitoringUrl}/metrics`;
			const response = await fetch(url);
			if (!response.ok) throw new Error(`HTTP ${response.status}`);
			const text = await response.text();
			const lines = text.split('\n').filter(line => !line.startsWith('#') && line.trim());
			const metrics = {};
			lines.forEach(line => {
				const match = line.match(/^([a-z_]+(?:_\d+)?)\s+(.+)$/);
				if (match) {
					metrics[match[1]] = parseFloat(match[2]);
				}
			});
			metricsData.set(metrics);
			lastSuccessMetrics.set(Date.now());
			staleMetrics.set(false);
		} catch (e) {
			// Keep stale data visible; flag staleness
			staleMetrics.set(true);
		}
	};

	onMount(() => {
		// Initial fetch
		fetchHealth();
		fetchDiagnostics();
		fetchMetrics();
		loading.set(false);

		// Set up intervals
		healthInterval = setInterval(fetchHealth, 5000); // Every 5 seconds
		diagnosticsInterval = setInterval(fetchDiagnostics, 10000); // Every 10 seconds
		metricsInterval = setInterval(fetchMetrics, 15000); // Every 15 seconds

		// Tick for formatAge to update stale chips
		tickInterval = setInterval(() => { now = Date.now(); }, 5000);
	});

	onDestroy(() => {
		clearInterval(healthInterval);
		clearInterval(diagnosticsInterval);
		clearInterval(metricsInterval);
		clearInterval(tickInterval);
	});

	// Map server health status → StatusBadge state
	const healthStateMap = {
		ok: 'live',
		warning: 'issue',
		error: 'offline',
	};
	const healthLabelMap = {
		ok: 'OK',
		warning: 'Degraded',
		error: 'Down',
	};

	// Token-based status classes (map to .status-* styles below) — no Tailwind
	const getStatusColor = (status) => {
		if (status === 'ok') return 'status-ok';
		if (status === 'error') return 'status-error';
		return 'status-warning';
	};

	const getMemoryWarning = (mb) => {
		if (mb > 800) return 'status-error';
		if (mb > 500) return 'status-warning';
		return 'status-ok';
	};

	const getGoroutineWarning = (count) => {
		if (count > 10000) return 'status-error';
		if (count > 5000) return 'status-warning';
		return 'status-ok';
	};

	const formatBytes = (bytes) => {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
	};

	// Derived stale age strings — recalculate when `now` ticks
	$: staleAgeDiagnostics = $lastSuccessDiagnostics ? formatAge(now - $lastSuccessDiagnostics) : '';
	$: staleAgeMetrics = $lastSuccessMetrics ? formatAge(now - $lastSuccessMetrics) : '';
	$: lastUpdateAge = $lastSuccessHealth ? formatAge(now - $lastSuccessHealth) : '';

	// Metrics table: rows matching active/total/queue filter
	$: filteredMetrics = $metricsData
		? Object.entries($metricsData)
			.filter(([key]) => key.includes('active') || key.includes('total') || key.includes('queue'))
			.sort((a, b) => a[0].localeCompare(b[0]))
		: [];
</script>

<div class="dashboard">
	<header class="header">
		<div class="header-lead">
			<div class="title-row">
				<span class="title-icon" aria-hidden="true">
					<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"/><path d="M7 15l3-4 3 2 4-6"/></svg>
				</span>
				<h1 class="title">Backend Monitoring</h1>
			</div>
			<div class="last-update">
				<span
					class="refresh-indicator"
					class:is-stale={!!$error}
					aria-hidden="true"
				></span>
				{#if $lastSuccessHealth}
					Last update: {lastUpdateAge || 'just now'}
				{:else}
					Loading…
				{/if}
			</div>
		</div>
		<div class="endpoint">Endpoint: {monitoringUrl}</div>
	</header>

	{#if $error}
		<div class="error-message" role="alert">
			<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
			<span>{$error}</span>
		</div>
	{/if}

	{#if $loading}
		<div class="bento-grid" style="--bento-cols:3;">
			{#each Array(5) as _, i (i)}
				<Card variant="glass" hover={false}>
					<div class="metric-head">
						<Skeleton variant="avatar" width="36px" height="36px" radius="10px" />
						<Skeleton variant="title" width="60%" />
					</div>
					<Skeleton variant="text" count={3} />
				</Card>
			{/each}
		</div>
	{:else if $healthData}
		<div class="bento-grid" style="--bento-cols:3;">
			<!-- System Health -->
			<Card variant="glass" hover={false}>
				<div class="metric-head">
					<SectionHeader
						title="System Health"
						level={2}
					>
						{#snippet icon()}
							<span class="metric-icon" aria-hidden="true">
								<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
							</span>
						{/snippet}
					</SectionHeader>
				</div>
				<div class="metric-hero">
					<StatusBadge
						state={healthStateMap[$healthData.status] || 'connecting'}
						label={healthLabelMap[$healthData.status] || $healthData.status.toUpperCase()}
					/>
					<span class="metric-hero-label">Overall status</span>
				</div>
				<div class="stat">
					<span class="stat-label">Database</span>
					<span class="stat-value {getStatusColor($healthData.db)}">{$healthData.db}</span>
				</div>
				<div class="stat">
					<span class="stat-label">DB Connections</span>
					<span class="stat-value">{$healthData.connections}</span>
				</div>
				<div class="stat">
					<span class="stat-label">GC Runs</span>
					<span class="stat-value">{$healthData.memory.num_gc}</span>
				</div>
			</Card>

			{#if $diagnosticsData}
				<!-- Memory -->
				<Card variant="glass" hover={false}>
					<div class="metric-head">
						<SectionHeader title="Memory Usage" level={2}>
							{#snippet icon()}
								<span class="metric-icon" aria-hidden="true">
									<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="4" width="16" height="16" rx="2"/><rect x="9" y="9" width="6" height="6"/><path d="M9 2v2M15 2v2M9 20v2M15 20v2M2 9h2M2 15h2M20 9h2M20 15h2"/></svg>
								</span>
							{/snippet}
							{#if $staleDiagnostics && staleAgeDiagnostics}
								{#snippet action()}
									<span class="stale-chip" aria-label="Data is stale">stale · {staleAgeDiagnostics}</span>
								{/snippet}
							{/if}
						</SectionHeader>
					</div>
					<div class="metric-hero">
						<span class="metric-hero-value {getMemoryWarning($diagnosticsData.runtime.memory_mb.alloc)}">{$diagnosticsData.runtime.memory_mb.alloc}<span class="unit">MB</span></span>
						<span class="metric-hero-label">Allocated</span>
					</div>
					<div class="stat">
						<span class="stat-label">System</span>
						<span class="stat-value">{$diagnosticsData.runtime.memory_mb.sys} MB</span>
					</div>
					<div class="stat">
						<span class="stat-label">Heap Alloc</span>
						<span class="stat-value">{$diagnosticsData.runtime.memory_mb.heap_alloc} MB</span>
					</div>
					<div class="progress-bar" role="presentation">
						<div class="progress-fill" style="transform: scaleX({Math.min($diagnosticsData.runtime.memory_mb.alloc / 1000, 1)});"></div>
					</div>
				</Card>

				<!-- Goroutines -->
				<Card variant="glass" hover={false}>
					<div class="metric-head">
						<SectionHeader title="Goroutines" level={2}>
							{#snippet icon()}
								<span class="metric-icon" aria-hidden="true">
									<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-3-6.7"/><polyline points="21 3 21 9 15 9"/></svg>
								</span>
							{/snippet}
						</SectionHeader>
					</div>
					<div class="metric-hero">
						<span class="metric-hero-value {getGoroutineWarning($diagnosticsData.runtime.goroutines)}">{$diagnosticsData.runtime.goroutines}</span>
						<span class="metric-hero-label">Active</span>
					</div>
					<div class="stat">
						<span class="stat-label">GC Pause</span>
						<span class="stat-value">{($diagnosticsData.runtime.gc.pause_ns / 1_000_000).toFixed(2)} ms</span>
					</div>
					<div class="progress-bar" role="presentation">
						<div class="progress-fill" style="transform: scaleX({Math.min($diagnosticsData.runtime.goroutines / 10000, 1)});"></div>
					</div>
				</Card>

				<!-- Database Pool -->
				<Card variant="glass" hover={false}>
					<div class="metric-head">
						<SectionHeader title="Database Pool" level={2}>
							{#snippet icon()}
								<span class="metric-icon" aria-hidden="true">
									<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14a9 3 0 0 0 18 0V5"/><path d="M3 12a9 3 0 0 0 18 0"/></svg>
								</span>
							{/snippet}
						</SectionHeader>
					</div>
					<div class="metric-hero">
						<span class="metric-hero-value">{$diagnosticsData.database.open_connections}</span>
						<span class="metric-hero-label">Open connections</span>
					</div>
					<div class="stat">
						<span class="stat-label">In Use</span>
						<span class="stat-value">{$diagnosticsData.database.in_use}</span>
					</div>
					<div class="stat">
						<span class="stat-label">Idle</span>
						<span class="stat-value">{$diagnosticsData.database.idle}</span>
					</div>
					<div class="stat">
						<span class="stat-label">Wait Count</span>
						<span class="stat-value">{$diagnosticsData.database.wait_count}</span>
					</div>
				</Card>

				<!-- Cache -->
				<Card variant="glass" hover={false}>
					<div class="metric-head">
						<SectionHeader title="Cache" level={2}>
							{#snippet icon()}
								<span class="metric-icon" aria-hidden="true">
									<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
								</span>
							{/snippet}
						</SectionHeader>
					</div>
					<div class="metric-hero">
						<span class="metric-hero-value">{formatBytes($diagnosticsData.cache.size_bytes)}</span>
						<span class="metric-hero-label">In-memory size</span>
					</div>
					<div class="progress-bar" role="presentation">
						<div class="progress-fill" style="transform: scaleX({Math.min($diagnosticsData.cache.size_bytes / (50 * 1024 * 1024), 1)});"></div>
					</div>
				</Card>
			{/if}
		</div>

		<!-- Key Metrics Table -->
		<Card variant="glass" hover={false}>
			<div class="metric-head">
				<SectionHeader title="Key Prometheus Metrics" level={2}>
					{#snippet icon()}
						<span class="metric-icon" aria-hidden="true">
							<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
						</span>
					{/snippet}
					{#if $staleMetrics && staleAgeMetrics}
						{#snippet action()}
							<span class="stale-chip" aria-label="Data is stale">stale · {staleAgeMetrics}</span>
						{/snippet}
					{/if}
				</SectionHeader>
			</div>

			{#if filteredMetrics.length > 0}
				<table class="metric-table">
					<thead>
						<tr>
							<th>Metric</th>
							<th>Value</th>
						</tr>
					</thead>
					<tbody>
						{#each filteredMetrics as [key, value]}
							<tr>
								<td>{key}</td>
								<td class="metric-num">{typeof value === 'number' ? value.toFixed(0) : value}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{:else}
				<EmptyState
					title="No metrics matched"
					body="Endpoint may be starting up or no active/total/queue metrics registered yet."
					tone="neutral"
				>
					{#snippet icon()}
						<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
					{/snippet}
					{#snippet action()}
						<button class="retry-btn" onclick={fetchMetrics}>Retry</button>
					{/snippet}
				</EmptyState>
			{/if}
		</Card>
	{/if}
</div>

<style>
	.dashboard {
		background: var(--bg-base, #0a0a14); /* raw-color-ok — bg-base fallback */
		color: var(--text-primary);
		font-family: var(--font-sans);
		min-height: 100vh;
		padding: calc(var(--safe-top, env(safe-area-inset-top, 0px)) + var(--space-5)) var(--space-5)
			calc(var(--space-6) + var(--safe-bottom, env(safe-area-inset-bottom, 0px)));
	}

	.header {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: var(--space-4);
		flex-wrap: wrap;
		margin-bottom: var(--space-6);
		padding-bottom: var(--space-4);
		border-bottom: 1px solid var(--border-default);
	}

	.title-row { display: flex; align-items: center; gap: var(--space-2); }
	.title-icon {
		display: grid; place-items: center;
		width: 36px; height: 36px;
		border-radius: var(--radius-md, 10px);
		background: var(--primary-500-12);
		color: var(--primary-400);
	}
	.title-icon :global(svg) { width: 20px; height: 20px; }

	.title {
		margin: 0;
		font-family: var(--font-display);
		font-size: var(--text-xl, 22px);
		font-weight: 700;
		letter-spacing: -0.02em;
		color: var(--text-primary);
	}

	.last-update {
		display: flex;
		align-items: center;
		gap: 6px;
		margin-top: var(--space-2);
		font-size: var(--text-xs, 12px);
		color: var(--text-tertiary);
	}

	.endpoint {
		font-size: var(--text-xs, 12px);
		color: var(--text-tertiary);
		font-family: var(--font-mono, monospace);
	}

	/* ── Card internals ─────────────────────────────────────────────── */
	.metric-head {
		margin-bottom: var(--space-4);
	}

	/* SectionHeader's .sh-icon slot — we wrap metric-icon inside it */
	.metric-icon {
		display: grid; place-items: center;
		width: 36px; height: 36px;
		flex-shrink: 0;
		border-radius: var(--radius-md, 10px);
		background: var(--primary-500-12);
		color: var(--primary-400);
	}
	.metric-icon :global(svg) { width: 19px; height: 19px; }

	.metric-hero {
		display: flex;
		flex-direction: column;
		gap: 2px;
		margin-bottom: var(--space-3);
	}
	.metric-hero-value {
		font-family: var(--font-display);
		font-size: var(--text-2xl, 28px);
		font-weight: 800;
		line-height: 1.05;
		letter-spacing: -0.02em;
		color: var(--text-primary);
		font-variant-numeric: tabular-nums;
	}
	.metric-hero-value .unit {
		font-size: var(--text-sm, 14px);
		font-weight: 600;
		color: var(--text-tertiary);
		margin-left: 4px;
	}
	.metric-hero-label {
		font-size: var(--text-xs, 12px);
		color: var(--text-tertiary);
	}

	.stat {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border-default);
	}
	.stat:last-child { border-bottom: none; }

	.stat-label {
		color: var(--text-tertiary);
		font-size: var(--text-sm, 13px);
	}
	.stat-value {
		font-weight: 700;
		font-size: var(--text-sm, 14px);
		color: var(--text-primary);
		font-variant-numeric: tabular-nums;
	}

	/* Status colors — token driven (replaces Tailwind text-*-600) */
	.status-ok      { color: var(--success-500); }
	.status-error   { color: var(--danger-500); }
	.status-warning { color: var(--warning-500); }

	/* ── Stale chip ─────────────────────────────────────────────────── */
	.stale-chip {
		display: inline-flex;
		align-items: center;
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius-full, 9999px);
		font-size: var(--text-xs, 11px);
		font-weight: 600;
		font-family: var(--font-mono, monospace);
		color: var(--warning-500);
		background: color-mix(in oklch, var(--warning-500) 12%, transparent);
		white-space: nowrap;
	}

	/* ── Metrics table ──────────────────────────────────────────────── */
	.metric-table {
		width: 100%;
		font-size: var(--text-xs, 12px);
		border-collapse: collapse;
	}
	.metric-table th,
	.metric-table td {
		padding: var(--space-2) var(--space-3);
		text-align: left;
		border-bottom: 1px solid var(--border-default);
	}
	.metric-table th {
		color: var(--text-secondary);
		font-weight: 700;
		text-transform: uppercase;
		font-size: 11px;
		letter-spacing: 0.05em;
	}
	.metric-table td { color: var(--text-secondary); font-family: var(--font-mono, monospace); }
	.metric-num { color: var(--text-primary); font-weight: 700; font-variant-numeric: tabular-nums; }
	.metric-table tbody tr:hover { background: var(--surface-hover); }

	/* ── Error banner ───────────────────────────────────────────────── */
	.error-message {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		background: var(--danger-500-12);
		border: 1px solid var(--danger-500-20);
		border-radius: var(--radius-md, 10px);
		padding: var(--space-3) var(--space-4);
		margin-bottom: var(--space-5);
		color: var(--danger-500);
		font-size: var(--text-sm, 13px);
	}
	.error-message :global(svg) { flex-shrink: 0; }

	/* ── Progress bar (transform-only fill) ─────────────────────────── */
	.progress-bar {
		width: 100%;
		height: 6px;
		background: var(--surface-inset);
		border-radius: var(--radius-full, 999px);
		overflow: hidden;
		margin-top: var(--space-2);
	}
	.progress-fill {
		height: 100%;
		width: 100%;
		transform-origin: left center;
		background: linear-gradient(90deg, var(--success-500), var(--primary-500));
		border-radius: var(--radius-full, 999px);
		transition: transform 300ms var(--ease-out, cubic-bezier(0.16,1,0.3,1));
	}

	/* ── Refresh indicator ──────────────────────────────────────────── */
	.refresh-indicator {
		display: inline-block;
		width: 8px; height: 8px;
		background: var(--success-500);
		border-radius: 50%;
		box-shadow: var(--glow-success-sm, 0 0 8px color-mix(in oklch, var(--success-500) 20%, transparent));
		animation: pulse 2s infinite;
	}
	/* Stale state: amber dot, no pulse */
	.refresh-indicator.is-stale {
		background: var(--warning-500);
		box-shadow: none;
		animation: none;
	}

	/* ── Retry button ───────────────────────────────────────────────── */
	.retry-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-height: 44px;
		padding: 0 var(--space-5);
		border-radius: var(--radius-md, 10px);
		background: var(--primary-500-12);
		color: var(--primary-400);
		font-size: var(--text-sm, 13px);
		font-weight: 600;
		border: 1px solid var(--primary-500-20);
		cursor: pointer;
		transition: background var(--duration-fast, 120ms) var(--ease-out, cubic-bezier(0.4,0,0.2,1));
		touch-action: manipulation;
	}
	.retry-btn:hover { background: var(--primary-500-20); }
	.retry-btn:active { transform: scale(0.97); }

	/* ── Bento grid ─────────────────────────────────────────────────── */
	:global(.bento-grid) {
		display: grid;
		grid-template-columns: repeat(var(--bento-cols, 3), 1fr);
		gap: var(--space-4);
		margin-bottom: var(--space-4);
	}

	@media (max-width: 900px) {
		:global(.bento-grid) {
			grid-template-columns: 1fr;
		}
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50%      { opacity: 0.5; }
	}

	@media (prefers-reduced-motion: reduce) {
		.refresh-indicator { animation: none; }
		.progress-fill { transition: none; }
		.retry-btn:active { transform: none; }
	}
</style>
