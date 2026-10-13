<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { LandingContent } from '#lib/mock/types';

	let { personas }: { personas: LandingContent['personas'] } = $props();
</script>

<section class="py-24 px-6 bg-slate-50/50" id="untuk-siapa">
	<div class="max-w-5xl mx-auto">
		<!-- Section Header -->
		<div class="text-center max-w-2xl mx-auto mb-16">
			<h2 class="text-3xl sm:text-4xl font-extrabold text-slate-950 tracking-tight mb-3">
				{personas.heading}
			</h2>
			<p class="text-slate-600 text-sm sm:text-base">{personas.body}</p>
		</div>
		<!-- 3 Persona Cards -->
		<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
			{#each personas.items as p (p.kicker)}
				<div
					class="bg-white rounded-2xl border border-slate-200 p-6 flex flex-col justify-between shadow-subtle hover:border-slate-300 transition-all"
				>
					<div>
						<div class="flex items-center justify-between mb-4">
							<span class="text-3xs font-mono uppercase tracking-wider font-bold text-brand-600"
								>{p.kicker}</span
							>
							<div class="w-7 h-7 rounded-lg bg-blue-50 text-brand-600 flex items-center justify-center">
								<Icon name={p.icon} class="w-4 h-4" />
							</div>
						</div>
						<h3 class="text-base font-bold text-slate-900 mb-2">{p.title}</h3>
						<p class="text-xs text-slate-600 leading-relaxed mb-6">{p.body}</p>
					</div>
					<div class="p-3.5 rounded-xl bg-slate-50 border border-slate-100">
						<div class="text-3xs font-mono text-slate-400 uppercase font-semibold mb-1">
							CONTOH SKENARIO
						</div>
						<div class="text-xs font-bold text-slate-800 mb-1">{p.scenarioTitle}</div>
						<p class="text-2xs text-slate-500 leading-normal">{p.scenarioBody}</p>
					</div>
				</div>
			{/each}
		</div>
	</div>
</section>
