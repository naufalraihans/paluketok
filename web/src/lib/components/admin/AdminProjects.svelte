<script lang="ts">
	import { onMount } from 'svelte';
	import { adminListProjects, type Project } from '#lib/api';

	// Tab Projects admin: GET /v1/admin/projects (kontrak openapi.md).
	let rows = $state<Project[]>([]);
	let loading = $state(true);
	let err = $state('');

	onMount(async () => {
		try {
			const d = await adminListProjects();
			rows = d.projects ?? [];
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
			<p class="text-sm text-[#475569]">Belum ada proyek.</p>
		{:else}
			<div class="rounded-xl border border-slate-200 overflow-x-auto">
				<table class="w-full text-sm">
					<thead class="bg-[#F8F9FA] text-[#64748B]">
						<tr>
							<th class="text-left px-4 py-2.5 font-semibold">Proyek</th>
							<th class="text-left px-4 py-2.5 font-semibold">Pemilik</th>
							<th class="text-left px-4 py-2.5 font-semibold">Status</th>
							<th class="text-left px-4 py-2.5 font-semibold">Dibuat</th>
						</tr>
					</thead>
					<tbody>
						{#each rows as p (p.id)}
							<tr class="border-t border-slate-100">
								<td class="px-4 py-3">
									<p class="font-semibold">{p.name}</p>
									<p class="text-xs text-[#64748B] line-clamp-1 max-w-[36ch]">{p.brief}</p>
								</td>
								<td class="px-4 py-3 font-mono text-xs">{p.owner_id}</td>
								<td class="px-4 py-3"><span class="inline-flex px-2 py-0.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-600">{p.status}</span></td>
								<td class="px-4 py-3 text-xs">{fmt(p.created_at)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</section>