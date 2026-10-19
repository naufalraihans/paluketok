<script lang="ts">
	/**
	 * Panel skill perancangan.
	 *
	 * Skill = instruksi markdown tersimpan yang bisa dipanggil di chat dengan
	 * `/nama-skill`. Cakupannya global, jadi panel ini menulis satu daftar yang
	 * dipakai semua orang; itu sebabnya tulisannya tebal soal dampaknya.
	 */
	import { onMount } from 'svelte';
	import Icon from '#lib/components/Icon.svelte';
	import { createSkill, deleteSkill, listSkills, updateSkill, type Skill } from '#lib/api';

	let daftar = $state<Skill[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let busy = $state(false);

	// Form: `editId` null = mode tambah, terisi = mode ubah.
	let editId = $state<string | null>(null);
	let nama = $state('');
	let ringkasan = $state('');
	let isi = $state('');
	let konfirmasiHapus = $state<string | null>(null);

	const isiTerlaluPanjang = $derived(isi.length > 20000);
	const bisaSimpan = $derived(
		!busy && isi.trim().length > 0 && !isiTerlaluPanjang && (editId !== null || nama.trim().length > 0)
	);

	async function muat() {
		loading = true;
		error = null;
		try {
			const res = await listSkills();
			daftar = res.skills ?? [];
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'gagal memuat skill';
		} finally {
			loading = false;
		}
	}

	onMount(muat);

	function reset() {
		editId = null;
		nama = '';
		ringkasan = '';
		isi = '';
	}

	function mulaiUbah(s: Skill) {
		editId = s.id;
		nama = s.nama;
		ringkasan = s.ringkasan;
		isi = s.isi;
		error = null;
	}

	async function simpan() {
		if (!bisaSimpan) return;
		busy = true;
		error = null;
		try {
			if (editId) {
				await updateSkill(editId, { ringkasan, isi });
			} else {
				await createSkill({ nama: nama.trim().toLowerCase(), ringkasan, isi });
			}
			reset();
			await muat();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'gagal menyimpan skill';
		} finally {
			busy = false;
		}
	}

	async function hapus(id: string) {
		if (busy) return;
		busy = true;
		error = null;
		try {
			await deleteSkill(id);
			konfirmasiHapus = null;
			if (editId === id) reset();
			await muat();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'gagal menghapus skill';
		} finally {
			busy = false;
		}
	}
</script>

<div class="flex flex-col gap-4">
	<div class="flex items-start justify-between gap-4">
		<div>
			<h2 class="text-sm font-semibold text-[#0F172A]">Skill perancangan</h2>
			<p class="mt-0.5 text-xs text-[#64748B]">
				Instruksi tersimpan yang dipanggil di chat dengan <code class="rounded bg-[#F1F5F9] px-1 py-0.5 text-[11px]">/nama-skill</code>. Berlaku untuk semua pengguna.
			</p>
		</div>
		{#if editId}
			<button
				type="button"
				onclick={reset}
				class="shrink-0 rounded-lg border border-[#E2E8F0] bg-white px-3 py-1.5 text-xs font-medium text-[#475569] transition-colors hover:bg-[#F8FAFC]"
			>
				Batal ubah
			</button>
		{/if}
	</div>

	{#if error}
		<div class="rounded-lg border border-[#FECACA] bg-[#FEF2F2] px-3 py-2 text-xs text-[#B91C1C]" role="alert">
			{error}
		</div>
	{/if}

	<div class="grid gap-4 lg:grid-cols-[1fr_1.2fr]">
		<!-- Daftar skill -->
		<section class="rounded-xl border border-[#E2E8F0] bg-white">
			<header class="border-b border-[#E2E8F0] px-4 py-2.5">
				<h3 class="text-xs font-semibold text-[#0F172A]">
					Terpasang <span class="font-normal text-[#64748B]">({daftar.length})</span>
				</h3>
			</header>
			{#if loading}
				<p class="px-4 py-6 text-xs text-[#64748B]">memuat…</p>
			{:else if daftar.length === 0}
				<p class="px-4 py-6 text-xs text-[#64748B]">
					Belum ada skill. Tambah di panel sebelah supaya bisa dipanggil di chat.
				</p>
			{:else}
				<ul class="divide-y divide-[#E2E8F0]">
					{#each daftar as s (s.id)}
						<li class="flex items-start gap-2 px-4 py-3">
							<Icon name="puzzle" class="mt-0.5 w-4 h-4 shrink-0 text-[#2563EB]" />
							<div class="min-w-0 flex-1">
								<code class="text-xs font-semibold text-[#0F172A]">/{s.nama}</code>
								{#if s.ringkasan}
									<p class="mt-0.5 text-[11px] leading-snug text-[#64748B]">{s.ringkasan}</p>
								{/if}
							</div>
							<div class="flex shrink-0 items-center gap-1">
								<button
									type="button"
									onclick={() => mulaiUbah(s)}
									class="rounded-md px-2 py-1 text-[11px] font-medium text-[#475569] transition-colors hover:bg-[#F1F5F9]"
									data-purpose="skill-edit"
								>
									Ubah
								</button>
								{#if konfirmasiHapus === s.id}
									<button
										type="button"
										onclick={() => hapus(s.id)}
										disabled={busy}
										class="rounded-md bg-[#B91C1C] px-2 py-1 text-[11px] font-medium text-white transition-colors hover:bg-[#991B1B] disabled:opacity-50"
										data-purpose="skill-delete-confirm"
									>
										Yakin
									</button>
								{:else}
									<button
										type="button"
										onclick={() => (konfirmasiHapus = s.id)}
										class="rounded-md px-2 py-1 text-[11px] font-medium text-[#B91C1C] transition-colors hover:bg-[#FEF2F2]"
										data-purpose="skill-delete"
									>
										Hapus
									</button>
								{/if}
							</div>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<!-- Form tambah/ubah -->
		<section class="rounded-xl border border-[#E2E8F0] bg-white p-4">
			<h3 class="text-xs font-semibold text-[#0F172A]">
				{editId ? 'Ubah skill' : 'Skill baru'}
			</h3>

			<label class="mt-3 block">
				<span class="text-[11px] font-medium text-[#475569]">Nama</span>
				<input
					type="text"
					bind:value={nama}
					disabled={editId !== null}
					placeholder="cek-keamanan"
					class="mt-1 w-full rounded-lg border border-[#E2E8F0] px-2.5 py-1.5 font-mono text-xs text-[#0F172A] outline-none transition-colors focus:border-[#2563EB] disabled:bg-[#F8FAFC] disabled:text-[#64748B]"
					data-purpose="skill-name"
				/>
				<span class="mt-1 block text-[11px] text-[#64748B]">
					{editId
						? 'Nama tidak bisa diubah: itu cara skill dipanggil di chat.'
						: 'Huruf kecil, angka, tanda hubung. Dipanggil sebagai /nama-skill.'}
				</span>
			</label>

			<label class="mt-3 block">
				<span class="text-[11px] font-medium text-[#475569]">Ringkasan</span>
				<input
					type="text"
					bind:value={ringkasan}
					maxlength="200"
					placeholder="Periksa risiko otorisasi lebih dulu"
					class="mt-1 w-full rounded-lg border border-[#E2E8F0] px-2.5 py-1.5 text-xs text-[#0F172A] outline-none transition-colors focus:border-[#2563EB]"
					data-purpose="skill-summary"
				/>
			</label>

			<label class="mt-3 block">
				<span class="flex items-center justify-between text-[11px] font-medium text-[#475569]">
					<span>Instruksi (markdown)</span>
					<span class="font-normal {isiTerlaluPanjang ? 'text-[#B91C1C]' : 'text-[#64748B]'}">
						{isi.length}/20000
					</span>
				</span>
				<textarea
					bind:value={isi}
					rows="10"
					placeholder={'# Aturan\n\n- Periksa izin setiap endpoint\n- Sebut risiko lebih dulu, baru saran'}
					class="mt-1 w-full resize-y rounded-lg border border-[#E2E8F0] px-2.5 py-2 font-mono text-xs leading-relaxed text-[#0F172A] outline-none transition-colors focus:border-[#2563EB]"
					data-purpose="skill-body"
				></textarea>
			</label>

			<div class="mt-3 flex items-center justify-between gap-3">
				<p class="text-[11px] text-[#64748B]">
					Dipakai saat pengguna mengetik <code class="rounded bg-[#F1F5F9] px-1">/{(nama || 'nama').trim().toLowerCase() || 'nama'}</code>
				</p>
				<button
					type="button"
					onclick={simpan}
					disabled={!bisaSimpan}
					class="shrink-0 rounded-lg bg-[#2563EB] px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-[#1D4ED8] disabled:cursor-not-allowed disabled:opacity-50"
					data-purpose="skill-save"
				>
					{editId ? 'Simpan perubahan' : 'Tambah skill'}
				</button>
			</div>
		</section>
	</div>
</div>
