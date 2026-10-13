<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { LandingContent } from '#lib/mock/types';

	let { security }: { security: LandingContent['security'] } = $props();
</script>

<section class="py-24 px-6 bg-white border-t border-slate-100">
	<div class="max-w-5xl mx-auto">
		<div class="grid grid-cols-1 lg:grid-cols-12 gap-12 items-center">
			<!-- Left Text Content -->
			<div class="lg:col-span-5 space-y-6">
				<h2 class="text-3xl sm:text-4xl font-extrabold text-slate-950 tracking-tight">
					{security.heading}
				</h2>
				<p class="text-sm text-slate-600 leading-relaxed">{security.body}</p>
				<!-- Checklist Points -->
				<div class="space-y-4 pt-2">
					{#each security.points as point (point.title)}
						<div class="flex items-start gap-3">
							<div
								class="w-5 h-5 rounded-full bg-emerald-50 text-emerald-600 flex items-center justify-center shrink-0 mt-0.5"
							>
								<Icon name="check" class="w-3 h-3" width={3} />
							</div>
							<div>
								<div class="text-xs font-bold text-slate-900">{point.title}</div>
								<div class="text-xs text-slate-500">{point.body}</div>
							</div>
						</div>
					{/each}
				</div>
				<!-- Disclaimer Notice -->
				<div class="pt-4 border-t border-slate-100 text-2xs text-slate-400 leading-relaxed">
					{security.disclaimer}
				</div>
			</div>
			<!-- Right Technical Schema Visual -->
			<div class="lg:col-span-7">
				<div class="bg-canvas border border-slate-200/90 rounded-2xl p-6 shadow-subtle">
					<div
						class="flex items-center justify-between pb-4 border-b border-slate-200/60 mb-6 text-xs"
					>
						<span class="font-bold text-slate-800">{security.panel.title}</span>
						<span class="font-mono text-slate-400 text-3xs">{security.panel.ref}</span>
					</div>
					<!-- Mini ERD Diagram with Relationship -->
					<div class="grid grid-cols-2 gap-6 relative mb-6">
						{#each security.panel.tables as t (t.name)}
							<div
								class="bg-white rounded-xl border border-slate-200 overflow-hidden shadow-2xs"
							>
								<div
									class="bg-slate-50 px-3 py-1.5 border-b border-slate-100 flex justify-between items-center text-2xs font-mono font-bold text-slate-700"
								>
									<span>{t.name}</span>
									<span class="text-slate-400 font-normal text-4xs">{t.ref}</span>
								</div>
								<div class="p-3 text-2xs font-mono space-y-1 text-slate-600">
									{#each t.columns as col (col.name)}
										<div class="flex justify-between">
											{#if col.key}
												<span class="text-brand-600 font-semibold">{col.name}</span>
											{:else}
												<span>{col.name}</span>
											{/if}
											{#if col.sensitive}
												<span class="text-amber-600 text-4xs bg-amber-50 px-1 rounded">sensitif</span>
											{:else}
												<span class="text-slate-400 text-3xs">{col.type}</span>
											{/if}
										</div>
									{/each}
								</div>
							</div>
						{/each}
					</div>
					<!-- Policy Control Card -->
					<div class="p-4 rounded-xl bg-emerald-50/50 border border-emerald-200/80 mb-4 space-y-2">
						<div class="flex items-center gap-2 text-xs font-bold text-emerald-950">
							<Icon name="shield" class="w-4 h-4 text-emerald-600" />
							<span>{security.panel.policy.code}</span>
						</div>
						<div class="text-2xs space-y-1 pl-6">
							{#each security.panel.policy.lines as line (line.actor)}
								<div class="text-slate-700">
									<strong class="font-medium text-slate-900">{line.actor}</strong>
									{line.text}
								</div>
							{/each}
						</div>
					</div>
					<!-- Decision Pending Card -->
					<div
						class="p-3 rounded-xl bg-white border border-slate-200 flex items-center justify-between text-xs"
					>
						<div class="space-y-0.5">
							<div class="font-mono text-3xs text-slate-400">{security.panel.pending.code}</div>
							<div class="text-2xs text-slate-600 font-medium">{security.panel.pending.title}</div>
						</div>
						<span
							class="text-3xs font-medium px-2 py-0.5 rounded bg-amber-50 text-amber-700 border border-amber-200"
						>
							{security.panel.pending.badge}
						</span>
					</div>
				</div>
			</div>
		</div>
	</div>
</section>
