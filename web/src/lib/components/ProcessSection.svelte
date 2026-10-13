<script lang="ts">
	import type { LandingContent } from '#lib/mock/types';

	let { process }: { process: LandingContent['process'] } = $props();
</script>

<section class="py-24 px-6 bg-white border-b border-slate-100">
	<div class="max-w-5xl mx-auto">
		<!-- Section Header -->
		<div class="text-center max-w-2xl mx-auto mb-16">
			<h2 class="text-3xl sm:text-4xl font-extrabold text-slate-950 tracking-tight mb-3">
				{process.heading}
			</h2>
			<p class="text-slate-500 text-sm">{process.body}</p>
		</div>
		<!-- 4 Steps Columns -->
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-8">
			{#each process.steps as step (step.no)}
				<div class="flex flex-col justify-between">
					<div>
						<div class="font-mono text-3xl font-extrabold italic text-brand-600 mb-3">
							{step.no}
						</div>
						<h3 class="font-bold text-slate-900 text-base mb-2">{step.title}</h3>
						<p class="text-xs text-slate-600 leading-relaxed mb-4">{step.body}</p>
					</div>
					<div class="pt-3 border-t border-slate-100 text-2xs font-mono text-slate-400">
						Hasil / <span class="text-slate-700">{step.outcome}</span>
					</div>
				</div>
			{/each}
		</div>
		<!-- Footer Guidance -->
		<p class="text-center text-xs text-slate-400 mt-14">{process.footnote}</p>
	</div>
</section>
