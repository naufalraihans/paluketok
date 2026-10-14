<script lang="ts">
	import { onMount } from 'svelte';
	import { adminListDecisions, type Decision } from '#lib/api';

	// Tab Keputusan: GET /v1/admin/decisions (log JEV semua user).
	let rows = $state<Decision[]>([]);
	let loading = $state(true);
	let err = $state('');

	onMount(async () => {
		try {
			rows = (await adminListDecisions()).decisions ?? [];
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal memuat';
		} finally {
			loading = false;
		}
	});
	const fmt = (s: string): string => new Date(s).toLocaleString('id-ID');
</script>

<section class="space-y-3">
	{#if loading}<p class="text-sm text-[#475569]">Memuat...</p>{/if}
	{#if err}<div class="rounded-xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700">{err}</div>{/if}
	{#if !loading && !err}
		{#if rows.length === 0}
			<p class="text-sm text-[#475569]">Belum ada keputusan JEV tercatat.</p>
		{:else}
			<div class="rounded-xl border border-slate-200 overflow-x-auto">
				<table class="w-full text-sm">
					<thead class="bg-[#F8F9FA] text-[#64748B]">
						<tr>
							<th class="text-left px-4 py-2.5 font-semibold">Tahap</th>
							<th class="text-left px-4 py-2.5 font-semibold">Pertanyaan</th>
							<th class="text-left px-4 py-2.5 font-semibold">Gate</th>
							<th class="text-left px-4 py-2.5 font-semibold">Pilihan</th>
							<th class="text-left px-4 py-2.5 font-semibold">Confidence</th>
							<th class="text-left px-4 py-2.5 font-semibold">Waktu</th>
						</tr>
					</thead>
					<tbody>
						{#each rows as d (d.id)}
							<tr class="border-t border-slate-100">
								<td class="px-4 py-3 font-semibold">{d.stage}</td>
								<td class="px-4 py-3 max-w-[40ch]"><span class="line-clamp-2">{d.question}</span></td>
								<td class="px-4 py-3"><span class="inline-flex px-2 py-0.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-600">{d.gate}</span></td>
								<td class="px-4 py-3 text-xs">{d.choice}</td>
								<td class="px-4 py-3 text-xs">{d.confidence.toFixed(2)}</td>
								<td class="px-4 py-3 text-xs">{fmt(d.created_at)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</section>