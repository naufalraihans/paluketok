<script lang="ts">
	import { onMount } from 'svelte';
	import {
		adminListAISettings,
		adminSetAISetting,
		type AISetting,
		type AISettingInput
	} from '#lib/api';

	// Hanya URL yang perlu diisi admin. token, model, dan provider TIDAK
	// ditanyakan: nilainya diisi server saat baris dibuat (lihat defaultSlot),
	// karena admin tidak mungkin tahu batas token tiap penyedia.
	// PUT tanpa api_key berarti memakai key tersimpan (mask tak boleh dikirim balik).
	const SLOTS: { id: string; label: string; hint: string }[] = [
		{
			id: 'planner',
			label: 'Planner LLM',
			hint: 'Penyusun dokumen planning. Isi base URL dan API key penyedia.'
		},
		{
			id: 'judge',
			label: 'AI Judge',
			hint: 'Penilai keputusan, dipanggil saat penyusun ragu. Isi base URL dan API key.'
		}
	];
	// Nilai awal untuk slot judge yang belum pernah diisi: JEV, penyedia yang
	// sudah dipakai. Setelah tersimpan, yang dipakai selalu isi database.
	const JUDGE_DEFAULT_URL = 'https://api.experientiallabs.ai/v1/systemone';

	let slots = $state<AISetting[]>([]);
	let loading = $state(true);
	let err = $state('');
	let msg = $state('');
	let busy = $state(false);

	let selected = $state('planner');
	let apiKey = $state('');
	let baseUrl = $state('');
	let active = $state(true);

	const meta = $derived(SLOTS.find((s) => s.id === selected) ?? SLOTS[0]);
	const existing = $derived(slots.find((s) => s.slot === selected));
	// Slot penilai: model & endpoint bebas diisi admin. "judge" sekarang berarti
	// peran (penilai), bukan produk tertentu; JEV hanya nilai awal.
	const isJudge = $derived(selected === 'judge');

	function pick(slotId: string, s?: AISetting): void {
		selected = slotId;
		baseUrl = s?.base_url ?? (slotId === 'judge' ? JUDGE_DEFAULT_URL : '');
		active = s?.is_active ?? true;
		apiKey = '';
		msg = '';
		err = '';
	}

	async function save(): Promise<void> {
		if (busy) return;
		busy = true;
		err = '';
		msg = '';
		try {
			// provider, model, dan batas token sengaja TIDAK dikirim: server
			// memakai nilai lama atau nilai awal per slot. Mengirim 0 akan
			// ditimpa server, dan mengirim angka karangan lebih buruk.
			const body: AISettingInput = {
				slot: selected,
				base_url: baseUrl.trim(),
				is_active: active
			};
			if (apiKey.trim()) body.api_key = apiKey.trim();
			const r = await adminSetAISetting(selected, body);
			msg = `Slot ${selected} tersimpan (aktif: ${r.is_active}).`;
			if (apiKey.trim()) apiKey = '';
			await load();
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal simpan slot';
		} finally {
			busy = false;
		}
	}

	async function load(): Promise<void> {
		loading = true;
		try {
			slots = (await adminListAISettings()).slots ?? [];
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal memuat';
		} finally {
			loading = false;
		}
	}
	onMount(async () => {
		await load();
		pick('planner', slots.find((s) => s.slot === 'planner'));
	});
</script>

<section class="space-y-4">
	{#if loading}<p class="text-sm text-[#475569]">Memuat...</p>{/if}
	{#if err}<div class="rounded-xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700">{err}</div>{/if}

	{#if !loading}
		<div class="rounded-xl border border-slate-200 overflow-x-auto">
			<table class="w-full text-sm">
				<thead class="bg-[#F8F9FA] text-[#64748B]">
					<tr>
						<th class="text-left px-4 py-2.5 font-semibold">Slot</th>
						<th class="text-left px-4 py-2.5 font-semibold">Kegunaan</th>
						<th class="text-left px-4 py-2.5 font-semibold">Base URL</th>
						<th class="text-left px-4 py-2.5 font-semibold">Model</th>
						<th class="text-left px-4 py-2.5 font-semibold">API key</th>
						<th class="text-left px-4 py-2.5 font-semibold">Status</th>
						<th class="px-4 py-2.5"></th>
					</tr>
				</thead>
				<tbody>
					{#each SLOTS as sm (sm.id)}
						{@const s = slots.find((x) => x.slot === sm.id)}
						<tr class="border-t border-slate-100">
							<td class="px-4 py-3 font-semibold">{sm.label}</td>
							<td class="px-4 py-3 text-xs text-[#64748B]">{sm.hint}</td>
							<td class="px-4 py-3 text-xs font-mono max-w-[28ch] truncate">
								{s?.base_url ?? '— belum diisi'}
							</td>
							<td class="px-4 py-3 text-xs">{s?.model ?? '—'}</td>
							<!-- Masked dari server: teks statis, bukan input. -->
							<td class="px-4 py-3 text-xs font-mono text-[#64748B]">{s?.api_key ?? '—'}</td>
							<td class="px-4 py-3">
								{#if !s}
									<span class="inline-flex px-2 py-0.5 rounded-full text-xs font-semibold bg-rose-50 text-rose-700"
										>belum diisi</span
									>
								{:else}
									<span
										class="inline-flex px-2 py-0.5 rounded-full text-xs font-semibold {s.is_active
											? 'bg-emerald-50 text-emerald-700'
											: 'bg-slate-100 text-slate-500'}">{s.is_active ? 'aktif' : 'nonaktif'}</span
									>
								{/if}
							</td>
							<td class="px-4 py-3 text-right">
								<button
									type="button"
									class="text-xs font-semibold text-[#2563EB] hover:underline transition-all"
									onclick={() => pick(sm.id, s)}>Edit</button
								>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<form
		class="rounded-xl border border-slate-200 p-5 space-y-4"
		onsubmit={(e) => {
			e.preventDefault();
			save();
		}}
	>
		<div class="flex items-center justify-between">
			<h2 class="text-sm font-bold">Edit slot {meta.label}</h2>
			<label class="flex items-center gap-2 text-xs font-medium text-[#475569]">
				<input type="checkbox" bind:checked={active} class="rounded border-slate-300" />
				Aktif
			</label>
		</div>
		<p class="text-xs text-[#64748B]">{meta.hint}</p>
		{#if !existing}
			<p class="text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2">
				Slot ini belum ada di database. Simpan untuk membuatnya.
			</p>
		{/if}
		<div class="grid gap-3 md:grid-cols-2">
			<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1 md:col-span-2">
				Base URL (wajib https, host non-lokal)
				<input
					bind:value={baseUrl}
					required
					placeholder={isJudge ? JUDGE_DEFAULT_URL : 'https://api.example.com/v1'}
					class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs font-mono"
				/>
			</label>
		</div>
		<p class="text-xs text-[#64748B]">
			Model, provider, dan batas token diisi server dengan nilai awal per slot. Tidak perlu
			diisi di sini.
		</p>
		<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
			API key (isi ulang untuk mengganti; kosongkan bila tetap)
			<input
				type="password"
				bind:value={apiKey}
				autocomplete="off"
				placeholder="kosong = pakai key tersimpan"
				class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs font-mono"
			/>
		</label>
		<div class="flex items-center gap-3">
			<button
				type="submit"
				disabled={busy}
				class="px-4 py-2 rounded-lg text-xs font-semibold bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50"
			>
				{busy ? 'Menyimpan...' : 'Simpan slot'}
			</button>
			{#if msg}<p class="text-xs text-emerald-700">{msg}</p>{/if}
		</div>
	</form>
</section>
