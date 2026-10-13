<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import AccessTree from '#lib/components/AccessTree.svelte';
	import type { LandingContent } from '#lib/mock/types';

	let { hero }: { hero: LandingContent['hero'] } = $props();
	const w = $derived(hero.window);
</script>

<section
	class="pt-20 pb-24 text-center px-6 relative overflow-hidden bg-gradient-to-b from-white via-slate-50/40 to-white"
>
	<div class="max-w-4xl mx-auto">
		<!-- Main Hero Title -->
		<h1
			class="text-4xl sm:text-6xl font-extrabold tracking-tight text-slate-950 mb-6"
		>
			{hero.titleLead}<br />
			Bangun dengan <span
				class="relative inline-block text-brand-600 bg-brand-50 px-3.5 py-0.5 rounded-full border border-brand-100"
				>·benang merah·</span
			>
		</h1>
		<!-- Subheading Description -->
		<p class="text-base sm:text-lg text-slate-600 max-w-2xl mx-auto mb-10">
			{hero.subtitle}
		</p>
		<!-- Call to Action Buttons -->
		<div class="flex flex-col sm:flex-row items-center justify-center gap-4 mb-16">
			<a
				class="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-6 py-3.5 rounded-xl font-medium text-sm text-white bg-brand-500 hover:bg-brand-600 transition-all shadow-sm"
				href={hero.ctaPrimary.href}
			>
				<span>{hero.ctaPrimary.label}</span>
				<span class="text-brand-200">→</span>
			</a>
			<a
				class="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-6 py-3.5 rounded-xl font-medium text-sm text-slate-700 bg-white border border-slate-200 hover:border-slate-300 hover:bg-slate-50 transition-all shadow-subtle"
				href={hero.ctaSecondary.href}
			>
				<span>{hero.ctaSecondary.label}</span>
				<span class="text-slate-400">↗</span>
			</a>
		</div>
	</div>
	<!-- Hero Interactive Application Window Mockup -->
	<div class="max-w-5xl mx-auto text-left" data-purpose="hero-app-window">
		<div class="bg-white rounded-2xl border border-slate-200/90 shadow-app overflow-hidden">
			<!-- Window Header / Browser Bar -->
			<div
				class="bg-slate-50/80 px-4 py-3 border-b border-slate-200 flex items-center justify-between text-xs text-slate-500"
			>
				<div class="flex items-center gap-2">
					<span class="w-2.5 h-2.5 rounded-full bg-red-400 inline-block"></span>
					<span class="w-2.5 h-2.5 rounded-full bg-amber-400 inline-block"></span>
					<span class="w-2.5 h-2.5 rounded-full bg-emerald-400 inline-block"></span>
					<span
						class="ml-4 font-mono text-slate-400 text-2xs bg-slate-100/80 px-2.5 py-0.5 rounded border border-slate-200/60"
						>{w.url}</span
					>
				</div>
				<div class="flex items-center gap-2 font-mono text-2xs text-slate-400">
					<span
						class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded bg-blue-50/60 text-brand-600 border border-blue-100/60"
					>
						<span class="w-1.5 h-1.5 rounded-full bg-brand-500"></span> {w.status}
					</span>
				</div>
			</div>
			<!-- Application Workspace UI -->
			<div class="grid grid-cols-1 md:grid-cols-12 min-h-[460px] bg-canvas">
				<!-- Sidebar Explorer -->
				<div class="md:col-span-3 border-r border-slate-200/80 p-4 bg-white/70">
					<div
						class="text-xs font-semibold uppercase tracking-wider text-slate-400 mb-3 px-2"
					>
						{w.sidebarTitle}
					</div>
					<nav class="space-y-1 text-xs font-medium">
						{#each w.sidebar as item (item.label)}
							{#if item.active}
								<button
									type="button"
									class="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg bg-blue-50 text-brand-600 border border-blue-100/80 font-semibold shadow-xs"
								>
									<Icon name={item.icon} class="w-3.5 h-3.5 text-brand-500" />
									{item.label}
								</button>
							{:else}
								<button
									type="button"
									class="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-slate-600 hover:bg-slate-100 transition-colors"
								>
									<Icon name={item.icon} class="w-3.5 h-3.5 text-slate-400" />
									{item.label}
								</button>
							{/if}
						{/each}
					</nav>
				</div>
				<!-- Visual Canvas Stage -->
				<div class="md:col-span-9 p-6 flex flex-col justify-between">
					<div>
						<div class="flex items-center justify-between pb-5 border-b border-slate-200/60 mb-6">
							<div>
								<h3 class="font-bold text-slate-900 text-sm">{w.canvasTitle}</h3>
								<p class="text-xs text-slate-400 mt-0.5">{w.canvasSubtitle}</p>
							</div>
							<span
								class="text-2xs font-medium text-slate-500 bg-white border border-slate-200 px-2 py-0.5 rounded shadow-2xs"
							>
								{w.canvasBadge}
							</span>
						</div>
						<!-- Tree Diagram Visual -->
						<div class="max-w-xl mx-auto flex flex-col items-center">
							<!-- Root Node -->
							<div
								class="bg-white border-2 border-brand-200 shadow-sm rounded-xl px-5 py-2.5 text-center flex items-center gap-2.5 mb-6"
							>
								<div class="w-2.5 h-2.5 rounded-full bg-brand-500"></div>
								<div>
									<div class="font-bold text-xs text-slate-800">{w.rootName}</div>
									<div class="text-3xs text-slate-400 font-mono">{w.rootMeta}</div>
								</div>
							</div>
							<!-- 3 Access Groups -->
							<AccessTree groups={w.groups} variant="window" />
						</div>
					</div>
					<!-- Footer Tabs Bar inside mockup -->
					<div
						class="mt-6 pt-3 border-t border-slate-200/80 flex items-center justify-between text-2xs text-slate-400"
					>
						<span class="font-mono">{w.footerStrip}</span>
						<div class="flex items-center gap-4">
							{#each w.tabs as tab (tab.label)}
								<span
									class={tab.active
										? 'text-brand-600 font-semibold cursor-pointer'
										: 'hover:text-slate-700 cursor-pointer'}>{tab.label}</span
								>
							{/each}
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</section>
