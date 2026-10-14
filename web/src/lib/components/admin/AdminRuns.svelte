<script lang="ts">
	import { onMount } from 'svelte';
	import { adminListRuns, type Run } from '#lib/api';

	// Tab Runs admin: GET /v1/admin/runs + filter status & user_id (kontrak).
	let rows = $state<Run[]>([]);
	let loading = $state(true);
	let err = $state('');
	let status = $state('');
	let userId = $state('');

	async function load(): Promise<void> {
		loading = true;
		err = '';
		try {
			const d = await adminListRuns({
				status: status || undefined,
				user_id: userId.trim() || undefined
			});
			rows = d.runs ?? [];
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal memuat';
		} finally {
			loading = false;
		}
	}
	onMount(load);

	const STATUSES = ['', 'queued', 'running', 'done', 'failed'];
</script>

<section class="space-y-4">
	<div class="flex gap-3 items-end flex-wrap">
		<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
			Status
			<select bind:value={status} onchange={load}
				class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs font-medium bg-white"> 
				{#each STATUSES as s (s)}<option value={s}>{s === '' ? 'semua' : s}</option>{/each}
			</select>
		</label>
		<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
			User ID
			<input bind:value={userId} placeholder="UUID (kosong = semua)"
				class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs font-medium w-72" />
		</label>
		<button type="button" onclick={load}
			class="px-3 py-1.5 rounded-lg text-xs font-semibold bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm">Terapkan</button>
	</div>

	{#if loading}<p class="text-sm text-[#475569]">Memuat...</p>{/if}
	{#if err}<div class="rounded-xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700">{err}</div>{/if}
	{#if !loading && !err}
		{#if rows.length === 0}
			<p class="text-sm text-[#475569]">Tidak ada run untuk filter ini.</p>
		{:else}
			<div class="rounded-xl border border-slate-200 overflow-x-auto">
				<table class="w-full text-sm">
					<thead class="bg-[#F8F9FA] text-[#64748B]">
						<tr>
							<th class="text-left px-4 py-2.5 font-semibold">Run</th>
							<th class="text-left px-4 py-2.5 font-semibold">Proyek</th>
							<th class="text-left px-4 py-2.5 font-semibold">User</th>
							<th class="text-left px-4 py-2.5 font-semibold">Status</th>
							<th class="text-left px-4 py-2.5 font-semibold">Progres</th>
						</tr>
					</thead>
					<tbody>
						{#each rows as r (r.id)}
							<tr class="border-t border-slate-100">
								<td class="px-4 py-3 font-mono text-xs">{r.id}</td>
								<td class="px-4 py-3 font-mono text-xs">{r.project_id}</td>
								<td class="px-4 py-3 font-mono text-xs">{r.user_id}</td>
								<td class="px-4 py-3"><span class="inline-flex px-2 py-0.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-600">{r.status}</span></td>
								<td class="px-4 py-3 text-xs">{r.progress}/{r.total}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</section>