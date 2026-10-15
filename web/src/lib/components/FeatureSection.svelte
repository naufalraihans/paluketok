<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import AccessTree from '#lib/components/AccessTree.svelte';
	import type { LandingContent } from '#lib/mock/types';

	let { feature }: { feature: LandingContent['feature'] } = $props();
</script>

<section class="py-20 px-6 bg-slate-50/60 border-t border-slate-100" id="produk">
	<div class="max-w-5xl mx-auto">
		<!-- Section Header -->
		<div class="text-center max-w-2xl mx-auto mb-14">
			<h2 class="text-3xl sm:text-4xl font-extrabold text-slate-950 tracking-tight mb-3">
				{feature.heading}
			</h2>
			<p class="text-sm sm:text-base text-slate-600">{feature.body}</p>
			<div class="mt-2 text-2xs uppercase tracking-wider font-mono text-slate-400">
				{feature.disclaimer}
			</div>
		</div>
		<!-- Flow Visual Card Structure -->
		<div class="bg-white rounded-2xl border border-slate-200/90 shadow-app overflow-hidden">
			<!-- Mockup Toolbar Header -->
			<div
				class="px-5 py-3.5 border-b border-slate-100 flex flex-wrap items-center justify-between gap-3 bg-white"
			>
				<div class="flex items-center gap-3">
					<span
						class="w-6 h-6 rounded bg-slate-900 text-white text-3xs font-bold flex items-center justify-center"
						>{feature.toolbar.mark}</span
					>
					<span class="font-bold text-xs text-slate-800">{feature.toolbar.name}</span>
					<span class="text-slate-300">/</span>
					<span class="text-xs text-slate-500">{feature.toolbar.section}</span>
				</div>
				<div class="flex items-center gap-2">
					<span class="text-2xs font-mono px-2 py-0.5 rounded bg-slate-100 text-slate-600"
						>{feature.toolbar.status}</span
					>
				</div>
			</div>
			<!-- Secondary Sub-Navigation -->
			<div
				class="px-5 py-2.5 bg-slate-50/60 border-b border-slate-100 flex items-center justify-between text-xs"
			>
				<div class="flex items-center gap-4">
					{#each feature.tabs as tab (tab.label)}
						<button
							class={tab.active
								? 'font-semibold text-brand-600 border-b-2 border-brand-600 pb-1 -mb-3'
								: 'text-slate-500 hover:text-slate-800 pb-1'}>{tab.label}</button
						>
					{/each}
				</div>
				<span class="text-2xs text-slate-400 font-mono">{feature.meta}</span>
			</div>
			<!-- Interactive Flow Workspace Body -->
			<div class="p-6 grid grid-cols-1 lg:grid-cols-12 gap-6 bg-canvas">
				<!-- Left & Center Flow Diagram -->
				<div class="lg:col-span-8 space-y-6">
					<!-- Tree hierarchy -->
					<div class="flex flex-col items-center">
						<div
							class="px-4 py-2 rounded-lg bg-white border border-slate-200 shadow-2xs text-xs font-bold text-slate-800 flex items-center gap-2 mb-4"
						>
							<span class="w-2 h-2 rounded-full bg-brand-500"></span>
							{feature.rootName}
						</div>
						<AccessTree groups={feature.groups} variant="diagram" />
					</div>
					<!-- Flow Strip preview -->
					<div class="bg-white rounded-xl border border-slate-200 p-4">
						<div class="text-xs font-bold text-slate-800 mb-3 flex items-center justify-between">
							<span>{feature.flowTitle}</span>
							<span class="text-3xs text-slate-400 font-normal">{feature.flowNote}</span>
						</div>
						<div class="grid grid-cols-3 gap-3 text-center">
							{#each feature.flowSteps as step (step.code)}
								{#if step.active}
									<div class="p-3 rounded-lg bg-blue-50/60 border border-blue-200">
										<div class="text-2xs font-bold text-brand-900">{step.title}</div>
										<div class="text-4xs text-brand-600 mt-1">{step.state}</div>
										<div class="text-4xs font-mono text-brand-600 mt-0.5">{step.code}</div>
									</div>
								{:else}
									<div class="p-3 rounded-lg bg-slate-50 border border-slate-200/80">
										<div class="text-2xs font-bold text-slate-800">{step.title}</div>
										<div class="text-4xs text-slate-400 mt-1">{step.state}</div>
										<div class="text-4xs font-mono text-brand-600 mt-0.5">{step.code}</div>
									</div>
								{/if}
							{/each}
						</div>
					</div>
				</div>
				<!-- Right Detail Drawer -->
				<div
					class="lg:col-span-4 bg-white rounded-xl border border-slate-200 p-4 flex flex-col justify-between"
				>
					<div>
						<div
							class="flex items-center justify-between text-2xs font-mono text-slate-400 mb-1"
						>
							<span>{feature.detail.ref}</span>
							<span>{feature.detail.code}</span>
						</div>
						<h4 class="font-bold text-slate-900 text-sm mb-1">{feature.detail.title}</h4>
						<div
							class="inline-block px-2 py-0.5 rounded text-3xs font-medium bg-emerald-50 text-emerald-700 border border-emerald-200 mb-3"
						>
							{feature.detail.badge}
						</div>
						<p class="text-xs text-slate-600 leading-relaxed mb-4">{feature.detail.body}</p>
						<!-- Benang Merah Traceability Card -->
						<div
							class="p-3 rounded-lg bg-slate-50 border border-slate-200 space-y-2 text-xs"
						>
							<div
								class="font-mono text-3xs font-bold uppercase tracking-wider text-slate-400"
							>
								{feature.detail.traceLabel}
							</div>
							{#each feature.detail.trace as line (line.label)}
								<div class="flex items-center gap-2">
									<span
										class="w-1.5 h-1.5 rounded-full {line.tone === 'brand'
											? 'bg-brand-500'
											: line.tone === 'emerald'
												? 'bg-emerald-500'
												: 'bg-amber-500'}"
									></span>
									<span class="text-slate-800 text-2xs"
										><strong class="font-semibold">{line.label}</strong>
										{line.text}</span
									>
								</div>
							{/each}
						</div>
					</div>
					<div class="pt-4 mt-4 border-t border-slate-100 text-2xs text-slate-400">
						{feature.detail.footerNote}
					</div>
				</div>
			</div>
			<!-- Bottom Footer Strip -->
			<div
				class="px-5 py-3 bg-slate-50/80 border-t border-slate-100 flex flex-wrap items-center justify-between gap-2 text-xs text-slate-600"
			>
				<span>{feature.footerStrip}</span>
				<a
					class="text-brand-600 font-semibold hover:underline inline-flex items-center gap-1"
					href="/projects"
				>
					{feature.footerCta} <span>↗</span>
				</a>
			</div>
		</div>
	</div>
</section>
