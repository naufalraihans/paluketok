<script lang="ts">
	import type { DashboardContent, ProjectStatus } from '#lib/mock/dashboard-types';

	let {
		list,
		onArchive
	}: {
		list: DashboardContent['list'];
		onArchive: (card: DashboardContent['list']['cards'][number]) => void;
	} = $props();

	// Kelas badge per status, disalin dari kartu 1/2/3 mockup.
	const badge: Record<ProjectStatus, string> = {
		ready: 'text-emerald-600 bg-emerald-50 border-emerald-100',
		draft: 'text-blue-600 bg-blue-50 border-blue-100',
		archived: 'text-slate-500 bg-slate-100 border-slate-200'
	};
</script>

<section class="space-y-4" data-purpose="all-projects-list">
	<div class="flex items-center justify-between">
		<h3 class="text-base font-bold text-slate-900">{list.heading}</h3>
		<span class="text-xs font-medium text-slate-400">{list.count}</span>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
		{#each list.cards as card (card.id ?? card.name)}
			<article
				class="bg-white border border-slate-200 rounded-xl p-6 flex flex-col justify-between hover:border-slate-300 transition-all duration-200 shadow-xs"
				data-purpose="project-card"
			>
				<div>
					<div class="flex items-start justify-between mb-4">
						<div class="w-10 h-10 rounded-lg bg-slate-100 flex items-center justify-center">
							<div class="w-4 h-4 rounded bg-slate-300"></div>
						</div>
						<span
							class="text-2xs font-semibold px-2.5 py-0.5 rounded-full border {badge[card.status]}"
							>{card.status}</span
						>
					</div>
					<h4 class="text-base font-bold text-slate-900 mb-1.5">{card.name}</h4>
					<p class="text-xs text-slate-500 leading-relaxed mb-4">{card.brief}</p>
					<div class="space-y-1 mb-5">
						<span class="text-3xs font-bold text-slate-400 uppercase tracking-wider block">STACK</span>
						<p class="text-xs font-medium text-slate-700">{card.stack}</p>
					</div>
				</div>
				<div class="pt-4 border-t border-slate-100">
					<span class="text-2xs text-slate-400 block mb-2">{card.meta}</span>
					<div class="flex items-center justify-end gap-2 pt-1">
						{#if card.status !== 'archived'}
							<button
								class="text-2xs font-semibold px-2.5 py-1.5 rounded-lg text-rose-600 hover:bg-rose-50 transition-colors duration-200 cursor-pointer"
								type="button"
								onclick={() => onArchive(card)}
							>
								Arsipkan
							</button>
						{/if}
						<button
							class="text-xs font-semibold px-3 py-1.5 rounded-lg border border-slate-200 text-slate-800 hover:bg-slate-50 transition-colors duration-200 cursor-pointer"
							type="button"
							onclick={() => {
								// URL proyek memakai slug (enak dibaca). UUID hanya
								// cadangan bila proyek lama belum punya slug.
								const target = card.slug ?? card.id ?? card.name.toLowerCase().replace(/\s+/g, '-');
								window.location.href = `/projects/${target}`;
							}}
						>
							{card.action}
						</button>
					</div>
				</div>
			</article>
		{/each}
	</div>
</section>
