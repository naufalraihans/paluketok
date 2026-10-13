<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { DashboardContent, PageState } from '#lib/mock/dashboard-types';

	let { states }: { states: DashboardContent['states'] } = $props();

	// Kelas label per tone, disalin kartu EMPTY/LOADING/ERROR mockup.
	const badge: Record<PageState['tone'], string> = {
		slate: 'text-slate-500 bg-slate-100',
		blue: 'text-blue-600 bg-blue-50',
		rose: 'text-rose-600 bg-rose-50'
	};

	// Lebar baris skeleton (mockup: penuh, 3/4, penuh).
	const skeleton = ['flex-1', 'w-3/4', 'flex-1'];
</script>

<section class="pt-6 border-t border-slate-200/80 space-y-4" data-purpose="page-states-preview">
	<div class="flex items-center justify-between">
		<h3 class="text-base font-bold text-slate-900">{states.heading}</h3>
		<span class="text-xs font-medium text-slate-400">{states.note}</span>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-3 gap-5">
		{#each states.items as state (state.badge)}
			<div
				class="bg-white border border-slate-200 rounded-xl p-6 flex flex-col justify-between shadow-xs"
				data-purpose="state-{state.tone}"
			>
				<div>
					<div class="mb-4">
						<span
							class="text-3xs font-bold px-2 py-0.5 rounded tracking-wider uppercase {badge[state.tone]}"
							>{state.badge}</span
						>
					</div>

					{#if state.tone === 'slate'}
						<div
							class="w-9 h-9 rounded-lg bg-slate-50 border border-slate-200 flex items-center justify-center text-slate-400 mb-3"
						>
							<Icon name="folder" class="w-4 h-4" />
						</div>
						<h4 class="text-sm font-bold text-slate-900 mb-1">{state.title}</h4>
						<p class="text-xs text-slate-500 leading-relaxed mb-6">{state.body}</p>
					{:else if state.tone === 'blue'}
						<div class="flex items-center gap-2 mb-4">
							<svg class="animate-spin h-3.5 w-3.5 text-blue-600" fill="none" viewBox="0 0 24 24" aria-hidden="true">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path
									class="opacity-75"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
									fill="currentColor"
								></path>
							</svg>
							<span class="text-xs font-semibold text-slate-800">{state.title}</span>
						</div>
						<div class="space-y-3 mb-6">
							{#each skeleton as width, i (i)}
								<div class="flex items-center gap-2">
									<div class="w-5 h-5 rounded bg-slate-100 flex-shrink-0"></div>
									<div class="h-2 rounded bg-slate-100 {width}"></div>
								</div>
							{/each}
						</div>
					{:else}
						<div class="w-7 h-7 rounded-full bg-rose-50 text-rose-600 flex items-center justify-center mb-3">
							<Icon name="alert-triangle" class="w-4 h-4" width={2.5} />
						</div>
						<h4 class="text-sm font-bold text-slate-900 mb-1">{state.title}</h4>
						<p class="text-xs text-slate-500 leading-relaxed mb-6">{state.body}</p>
					{/if}
				</div>

				<div>
					{#if state.tone === 'blue'}
						<span class="text-2xs text-slate-400 block">{states.loadingNote}</span>
					{:else}
						<button
							class="text-xs font-semibold px-3 py-1.5 rounded-lg border border-slate-200 text-slate-800 hover:bg-slate-50 transition-colors"
							type="button"
						>
							{state.action}
						</button>
					{/if}
				</div>
			</div>
		{/each}
	</div>
</section>