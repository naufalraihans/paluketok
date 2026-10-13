<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { LandingContent } from '#lib/mock/types';

	let { traceability }: { traceability: LandingContent['traceability'] } = $props();
</script>

<section class="py-24 px-6 bg-[#0B0F17] text-white relative">
	<div class="max-w-5xl mx-auto">
		<!-- Section Header -->
		<div class="text-center max-w-2xl mx-auto mb-16">
			<h2 class="text-3xl sm:text-4xl font-extrabold tracking-tight mb-4 text-white">
				{traceability.headingLine1}<br />{traceability.headingLine2}
			</h2>
			<p class="text-slate-400 text-sm sm:text-base">{traceability.body}</p>
		</div>
		<!-- 4 Step Traceability Nodes Connected Horizontal -->
		<div class="grid grid-cols-1 md:grid-cols-4 gap-4 relative mb-12">
			{#each traceability.nodes as node (node.ref)}
				<div
					class="bg-slate-900/90 border border-slate-800 rounded-xl p-5 hover:border-slate-700 transition-colors"
				>
					<div
						class="w-8 h-8 rounded-lg bg-blue-500/10 border border-blue-500/20 text-brand-400 flex items-center justify-center mb-4"
					>
						<Icon name={node.icon} class="w-4 h-4" />
					</div>
					<div class="text-2xs font-mono text-slate-400 mb-1">{node.ref}</div>
					<div class="text-xs font-semibold text-slate-300 mb-1">{node.question}</div>
					<div
						class="text-sm font-bold text-white mb-2 {node.mono ? 'font-mono' : ''}"
					>
						{node.answer}
					</div>
					<p
						class="text-xs text-slate-400 leading-relaxed {node.mono
							? 'font-mono text-2xs'
							: ''}"
					>
						{node.detail}
					</p>
				</div>
			{/each}
		</div>
		<!-- Dynamic Context Shift Callout Bar -->
		<div
			class="p-6 rounded-2xl bg-gradient-to-r from-slate-900 via-slate-900/90 to-slate-900 border border-slate-800 flex flex-col md:flex-row items-start md:items-center justify-between gap-5 text-sm"
		>
			<div class="space-y-1">
				<div class="flex items-center gap-2">
					<span class="text-3xs font-mono uppercase font-bold text-brand-400"
						>{traceability.callout.label}</span
					>
					<span class="text-white font-bold text-sm">{traceability.callout.quote}</span>
				</div>
				<p class="text-slate-400 text-xs sm:text-sm">{traceability.callout.body}</p>
			</div>
			<span
				class="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-brand-500/10 text-brand-400 border border-brand-500/20 whitespace-nowrap"
			>
				{traceability.callout.badge}
			</span>
		</div>
	</div>
</section>
