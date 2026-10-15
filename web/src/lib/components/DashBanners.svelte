<script lang="ts">
	import { goto } from '$app/navigation';
	import type { DashboardContent } from '#lib/mock/dashboard-types';

	let {
		pipeline,
		balance,
		onCreate
	}: {
		pipeline: DashboardContent['pipeline'];
		balance: DashboardContent['balance'];
		onCreate: () => void;
	} = $props();
</script>

<section class="grid grid-cols-1 lg:grid-cols-3 gap-5" data-purpose="secondary-action-banners">
	<div class="lg:col-span-2 bg-white border border-slate-200 rounded-xl p-6 flex flex-col justify-between shadow-xs">
		<div>
			<h4 class="text-base font-bold text-slate-900 mb-1.5">{pipeline.heading}</h4>
			<p class="text-xs text-slate-500 leading-relaxed mb-4">{pipeline.body}</p>
			<div
				class="text-2xs font-medium text-slate-600 bg-slate-50 border border-slate-100 rounded px-2.5 py-1.5 w-fit mb-5"
			>
				{pipeline.chain}
			</div>
		</div>
		<div>
			<button
				class="inline-flex items-center px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-xs font-semibold rounded-lg shadow-sm transition-colors"
				type="button"
				onclick={onCreate}
			>
				{pipeline.action}
			</button>
		</div>
	</div>

	<div class="bg-white border border-slate-200 rounded-xl p-6 flex flex-col justify-between shadow-xs">
		<div>
			<h4 class="text-base font-bold text-slate-900 mb-2">{balance.heading}</h4>
			<div class="text-3xl font-extrabold text-slate-900 leading-none mb-1">{balance.value}</div>
			<p class="text-xs text-slate-500 mb-3">{balance.meta}</p>
			<p class="text-2xs text-slate-400 leading-relaxed mb-4">{balance.note}</p>
		</div>
		<div>
			<button
				class="w-full text-center text-xs font-semibold px-4 py-2 rounded-lg border border-slate-200 text-slate-800 hover:bg-slate-50 transition-colors"
				type="button"
				onclick={() => goto('/billing')}
			>
				{balance.action}
			</button>
		</div>
	</div>
</section>