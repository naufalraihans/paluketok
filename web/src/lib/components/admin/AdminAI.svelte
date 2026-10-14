<script lang="ts">
	import { onMount } from 'svelte';
	import {
		adminListAISettings,
		adminSetAISetting,
		type AISetting,
		type AISettingInput
	} from '#lib/api';

	// Tab AI Settings: slot tunggal planner (SlotLLM di BE). PUT tanpa api_key
	// berarti memakai key tersimpan (mask tak boleh dikirim balik).
	let slots = $state<AISetting[]>([]);
	let loading = $state(true);
	let err = $state('');
	let msg = $state('');
	let busy = $state(false);

	// Form edit per slot. apiKey kosong = key tetap. value textarea hanya
	// default max token dari GET.
	let apiKey = $state('');
	let provider = $state('');
	let baseUrl = $state('');
	let model = $state('');
	let maxIn = $state(0);
	let maxOut = $state(0);
	let active = $state(false);

	async function pick(s: AISetting): Promise<void> {
		provider = s.provider;
		baseUrl = s.base_url;
		model = s.model;
		maxIn = s.max_input_tokens;
		maxOut = s.max_output_tokens;
		active = s.is_active;
		apiKey = '';
		msg = '';
	}

	async function save(): Promise<void> {
		if (busy) return;
		busy = true;
		err = '';
		msg = '';
		try {
			const body: AISettingInput = {
				slot: 'planner',
				provider,
				base_url: baseUrl.trim(),
				model: model.trim(),
				max_input_tokens: maxIn,
				max_output_tokens: maxOut,
				is_active: active
			};
			if (apiKey.trim()) body.api_key = apiKey.trim();
			const r = await adminSetAISetting('planner', body);
			msg = `Slot planner tersimpan (aktif: ${r.is_active}).`;
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
		err = '';
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
		if (slots.length > 0) await pick(slots[0]);
	});
</script>

<section class="space-y-4">
	{#if loading}<p class="text-sm text-[#475569]">Memuat...</p>{/if}
	{#if err}<div class="rounded-xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700">{err}</div>{/if}

	{#if !loading}
		{#if slots.length === 0}
			<p class="text-sm text-[#475569]">
				Belum ada slot AI. Isi form di bawah untuk membuat slot planner pertama.
			</p>
		{:else}
			<div class="rounded-xl border border-slate-200 overflow-x-auto">
				<table class="w-full text-sm">
					<thead class="bg-[#F8F9FA] text-[#64748B]">
						<tr>
							<th class="text-left px-4 py-2.5 font-semibold">Slot</th>
							<th class="text-left px-4 py-2.5 font-semibold">Provider</th>
							<th class="text-left px-4 py-2.5 font-semibold">Base URL</th>
							<th class="text-left px-4 py-2.5 font-semibold">Model</th>
							<th class="text-left px-4 py-2.5 font-semibold">API key</th>
							<th class="text-left px-4 py-2.5 font-semibold">Status</th>
							<th class="px-4 py-2.5"></th>
						</tr>
					</thead>
					<tbody>
						{#each slots as s (s.slot)}
							<tr class="border-t border-slate-100">
								<td class="px-4 py-3 font-semibold">{s.slot}</td>
								<td class="px-4 py-3 text-xs">{s.provider}</td>
								<td class="px-4 py-3 text-xs font-mono max-w-[32ch] truncate">{s.base_url}</td>
								<td class="px-4 py-3 text-xs">{s.model}</td>
								<!-- Masked dari server: ditampilkan sebagai teks statis, bukan input. -->
								<td class="px-4 py-3 text-xs font-mono text-[#64748B]">{s.api_key}</td>
								<td class="px-4 py-3">
									<span class="inline-flex px-2 py-0.5 rounded-full text-xs font-semibold {s.is_active
										? 'bg-emerald-50 text-emerald-700'
										: 'bg-slate-100 text-slate-500'}">{s.is_active ? 'aktif' : 'nonaktif'}</span
									>
								</td>
								<td class="px-4 py-3 text-right">
									<button type="button" class="text-xs font-semibold text-[#2563EB] hover:underline transition-all"
										onclick={() => pick(s)}>Edit</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}

	<form class="rounded-xl border border-slate-200 p-5 space-y-4" onsubmit={(e) => { e.preventDefault(); save(); }}>
		<div class="flex items-center justify-between">
			<h2 class="text-sm font-bold">Edit slot planner</h2>
			<label class="flex items-center gap-2 text-xs font-medium text-[#475569]">
				<input type="checkbox" bind:checked={active} class="rounded border-slate-300" />
				Aktif
			</label>
		</div>
		<div class="grid gap-3 md:grid-cols-2">
			<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
				Provider
				<input bind:value={provider} required placeholder="openai-compatible"
					class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs" />
			</label>
			<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
				Model
				<input bind:value={model} required placeholder="gpt-4o-mini"
					class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs" />
			</label>
			<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1 md:col-span-2">
				Base URL (wajib https, host di allowlist keamanan)
				<input bind:value={baseUrl} required placeholder="https://api.example.com/v1"
					class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs font-mono" />
			</label>
			<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
				Max input tokens
				<input type="number" min="1" bind:value={maxIn} required
					class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs" />
			</label>
			<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
				Max output tokens
				<input type="number" min="1" bind:value={maxOut} required
					class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs" />
			</label>
		</div>
		<label class="text-xs font-semibold text-[#475569] flex flex-col gap-1">
			API key (isi ulang untuk mengganti; kosongkan bila tetap)
			<input type="password" bind:value={apiKey} autocomplete="off"
				placeholder="kosong = pakai key tersimpan"
				class="border border-slate-200 rounded-lg px-3 py-1.5 text-xs font-mono" />
		</label>
		<div class="flex items-center gap-3">
			<button type="submit" disabled={busy}
				class="px-4 py-2 rounded-lg text-xs font-semibold bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50">
				{busy ? 'Menyimpan...' : 'Simpan slot'}
			</button>
			{#if msg}<p class="text-xs text-emerald-700">{msg}</p>{/if}
		</div>
	</form>
</section>