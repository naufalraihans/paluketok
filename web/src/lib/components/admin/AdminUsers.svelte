<script lang="ts">
	import { onMount } from 'svelte';
	import { adminListRuns, adminPatchUser, adminGrantCredits, type Run } from '#lib/api';

	// Tab Users: GET /v1/admin/users TIDAK ada di kontrak (openapi.md:34-40),
	// daftar user diturunkan dari /v1/admin/runs (response pakai user_id).
	let users = $state<string[]>([]);
	let loading = $state(true);
	let err = $state('');
	let msg = $state('');
	let targetId = $state('');
	let tier = $state('free');
	let credits = $state(10);
	let reason = $state('');
	let busy = $state(false);

	async function load(): Promise<void> {
		loading = true;
		err = '';
		try {
			const d = await adminListRuns({});
			const ids = new Set<string>();
			for (const r of (d.runs ?? []) as Run[]) if (r.user_id) ids.add(r.user_id);
			users = [...ids];
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal memuat';
		} finally {
			loading = false;
		}
	}
	onMount(load);

	async function saveTier(): Promise<void> {
		if (!targetId.trim() || busy) return;
		busy = true;
		err = '';
		msg = '';
		try {
			const r = await adminPatchUser(targetId.trim(), tier);
			msg = `Plan tier ${r.id} diubah ke ${r.plan_tier}.`;
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal ubah tier';
		} finally {
			busy = false;
		}
	}

	async function grant(): Promise<void> {
		if (!targetId.trim() || credits <= 0 || !reason.trim() || busy) {
			err = 'Isi UUID, jumlah kredit > 0, dan alasan.';
			return;
		}
		busy = true;
		err = '';
		msg = '';
		try {
			const r = await adminGrantCredits(targetId.trim(), credits, reason.trim());
			msg = `${r.credits_granted} kredit diberikan ke ${r.id}.`;
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal grant kredit';
		} finally {
			busy = false;
		}
	}

	const TIERS = ['free', 'starter', 'pro'];
</script>

<section class="space-y-4">
	<div class="rounded-xl border border-slate-200 p-5 space-y-4">
		<h2 class="text-sm font-bold">Aksi user (UUID manual)</h2>
		<p class="text-xs text-[#64748B]">
			Kontrak tak menyediakan <code class="font-mono">GET /v1/admin/users</code>. UUID diambil dari tab
			Runs di bawah. Catat di plan/gaps.md.
		</p>
		<div class="grid gap-3 md:grid-cols-2">
			<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
				User ID
				<input bind:value={targetId} placeholder="UUID user" class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs font-mono" />
			</label>
			<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
				Plan tier
				<div class="flex gap-2">
					<select bind:value={tier} class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs flex-1 bg-white">
						{#each TIERS as t (t)}<option value={t}>{t}</option>{/each}
					</select>
					<button type="button" onclick={saveTier} disabled={busy}
						class="px-3 py-1.5 rounded-lg text-xs font-semibold bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50">Simpan</button>
				</div>
			</label>
			<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
				Jumlah kredit
				<input type="number" min="1" bind:value={credits} class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs" />
			</label>
			<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
				Alasan (wajib)
				<input bind:value={reason} placeholder="mis. kompensasi uji" class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs" />
			</label>
		</div>
		<button type="button" onclick={grant} disabled={busy}
			class="px-4 py-2 rounded-lg text-xs font-semibold bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50">Grant kredit</button>
		{#if msg}<p class="text-xs text-emerald-700">{msg}</p>{/if}
		{#if err}<p class="text-xs text-rose-700">{err}</p>{/if}
	</div>

	<div class="space-y-2">
		<h2 class="text-sm font-bold">User terlihat di runs</h2>
		{#if loading}<p class="text-sm text-[#475569]">Memuat...</p>{/if}
		{#if !loading && users.length === 0}
			<p class="text-sm text-[#475569]">Belum ada user dengan run.</p>
		{:else if !loading}
			<ul class="rounded-xl border border-slate-200 divide-y divide-slate-100">
				{#each users as u (u)}
					<li class="px-4 py-2.5 flex items-center justify-between gap-3">
						<span class="font-mono text-xs">{u}</span>
						<button type="button" class="text-xs font-semibold text-[#2563EB] hover:underline transition-all"
							onclick={() => (targetId = u)}>Pakai</button>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</section>