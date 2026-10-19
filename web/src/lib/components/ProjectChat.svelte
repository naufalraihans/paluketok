<script lang="ts">
	/**
	 * Chat konteks proyek — cakupannya HANYA perancangan.
	 *
	 * Beda dengan chatbot umum: percakapan ini terikat satu proyek, jadi model
	 * sudah tahu brief dan dokumen yang sudah jadi. Balasan model boleh
	 * mengusulkan tahap berikutnya (`suggested_stage`); usulan itu diteruskan ke
	 * halaman lewat callback `onsuggest` supaya sidebar bisa menyorot menunya.
	 */
	import Icon from '#lib/components/Icon.svelte';
	import { mdToHtml } from '#lib/markdown';
	import {
		createChatSession,
		deleteChatSession,
		listChatSessions,
		listProjectMessages,
		sendProjectMessage,
		type ChatSession,
		type ProjectMessage
	} from '#lib/api';

	let {
		projectId,
		onsuggest
	}: {
		projectId: string;
		onsuggest?: (stage: string) => void;
	} = $props();

	let messages = $state<ProjectMessage[]>([]);
	/**
	 * Pesan pengguna yang belum dikonfirmasi server. Ditampilkan segera setelah
	 * kirim supaya tidak ada jeda kosong, lalu DIBUANG begitu riwayat dari
	 * server datang (server yang jadi sumber kebenaran, bukan salinan lokal).
	 */
	let optimistis = $state<ProjectMessage[]>([]);
	let draft = $state('');
	let busy = $state(false);
	let loading = $state(true);
	let error = $state<string | null>(null);

	/**
	 * Sesi percakapan. Satu proyek boleh punya banyak percakapan terpisah,
	 * masing-masing dengan riwayatnya sendiri. `aktif` null = server yang
	 * memilih (sesi terakhir, atau sesi baru bila proyek belum punya).
	 */
	let sesi = $state<ChatSession[]>([]);
	let aktif = $state<string | null>(null);

	// Muat daftar sesi saat proyek berganti (mis. UUID -> slug di URL).
	$effect(() => {
		const id = projectId;
		if (!id) return;
		error = null;
		listChatSessions(id)
			.then((res) => {
				sesi = res.sessions ?? [];
				// Sesi terbaru jadi pilihan awal; daftar sudah urut terbaru dulu.
				if (!aktif && sesi.length > 0) aktif = sesi[0].id;
			})
			.catch((e: unknown) => {
				error = e instanceof Error ? e.message : 'gagal memuat daftar sesi';
			});
	});

	// Riwayat dimuat ulang saat proyek ATAU sesi aktif berganti.
	$effect(() => {
		const id = projectId;
		const sid = aktif;
		if (!id) return;
		loading = true;
		error = null;
		listProjectMessages(id, sid ?? undefined)
			.then((res) => {
				messages = res.messages ?? [];
			})
			.catch((e: unknown) => {
				error = e instanceof Error ? e.message : 'gagal memuat percakapan';
			})
			.finally(() => {
				loading = false;
			});
	});

	const tooLong = $derived(draft.trim().length > 4000);
	const sesiAktif = $derived(sesi.find((s) => s.id === aktif) ?? null);
	const bisaHapus = $derived(sesi.length > 1);

	/** Pindah sesi: riwayat lama dibuang dulu supaya tidak terlihat tercampur. */
	function pilihSesi(id: string): void {
		if (id === aktif || busy) return;
		messages = [];
		optimistis = [];
		aktif = id;
	}

	async function sesiBaru(): Promise<void> {
		if (busy || !projectId) return;
		error = null;
		try {
			const s = await createChatSession(projectId);
			sesi = [s, ...sesi];
			messages = [];
			optimistis = [];
			aktif = s.id;
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'gagal membuat sesi baru';
		}
	}

	async function hapusSesi(id: string): Promise<void> {
		if (busy || !bisaHapus) return;
		error = null;
		try {
			await deleteChatSession(projectId, id);
			sesi = sesi.filter((s) => s.id !== id);
			if (aktif === id) {
				messages = [];
				aktif = sesi[0]?.id ?? null;
			}
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'gagal menghapus sesi';
		}
	}

	async function kirim(teks?: string): Promise<void> {
		const text = (teks ?? draft).trim();
		if (!text || busy || tooLong) return;
		busy = true;
		error = null;
		draft = '';
		// Pesan pengguna langsung tampil, jangan menunggu server. Sebelumnya
		// teks baru muncul setelah balasan datang, jadi terasa seperti pesan
		// hilang. Balasan pertama akan MENGGANTI pesan sementara ini dengan
		// versi dari server (yang punya id asli), lihat `optimistis`.
		const sementara: ProjectMessage = {
			id: `lokal-${Date.now()}`,
			project_id: projectId,
			role: 'user',
			content: text,
			created_at: new Date().toISOString()
		};
		optimistis = [sementara];
		try {
			const reply = await sendProjectMessage(projectId, text, aktif ?? undefined);
			// Muat ulang riwayat: pertanyaan + jawaban tersimpan di server, jadi
			// tampilan selalu sama dengan isi DB (bukan salinan lokal).
			const res = await listProjectMessages(projectId, aktif ?? undefined);
			messages = res.messages ?? [];
			// Sesi pertama yang lahir dari pesan ini belum ada di daftar lokal.
			if (!aktif) {
				const daftar = await listChatSessions(projectId);
				sesi = daftar.sessions ?? [];
				if (sesi.length > 0) aktif = sesi[0].id;
			}
			if (reply.suggested_stage && onsuggest) onsuggest(reply.suggested_stage);
		} catch (e: unknown) {
			const msg = e instanceof Error ? e.message : 'gagal mengirim pesan';
			// Slot LLM kosong / provider menolak -> 503. Jelaskan, jangan diam.
			error = msg.includes('503')
				? 'Slot LLM belum siap. Isi dulu di panel admin → settings/ai.'
				: msg;
			draft = text; // jangan buang ketikan pengguna
		} finally {
			// Selalu bersihkan pesan sementara: kalau berhasil, riwayat server
			// sudah memuatnya; kalau gagal, ketikan dikembalikan ke kotak input.
			optimistis = [];
			busy = false;
		}
	}

	/**
	 * Chip usul tahap dari balasan model: kirim perintah supaya tahap itu disusun.
	 *
	 * Sebelumnya chip ini cuma menyorot menu sidebar lewat `onsuggest`. Menu
	 * tahap baru ada kalau artefaknya sudah jadi, sedangkan usul tahap justru
	 * muncul SEBELUM dokumennya ada, jadi kliknya berakhir tanpa hasil apa pun
	 * (elemen `#tahap-<stage>` belum ada di DOM). Sekarang chip mengirim
	 * perintah nyata, dan `onsuggest` tetap dipanggil supaya sidebar ikut
	 * menyorot begitu menunya muncul.
	 */
	async function susunTahap(stage: string): Promise<void> {
		if (busy || !stage) return;
		onsuggest?.(stage);
		await kirim(`Susun ${stage.toUpperCase()} untuk proyek ini sekarang.`);
	}

	function onkeydown(e: KeyboardEvent): void {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			void kirim(draft);
		}
	}
</script>

<section
	class="bg-white border border-[#E2E8F0] rounded-xl flex flex-col overflow-hidden"
	data-purpose="project-chat"
>
	<header
		class="flex items-center justify-between gap-3 px-4 py-3 border-b border-[#E2E8F0] bg-[#F8FAFC]"
	>
		<div class="flex items-center gap-2 min-w-0">
			<Icon name="sparkles" class="w-4 h-4 text-[#2563EB] shrink-0" />
			<div class="min-w-0">
				<h2 class="font-semibold text-xs text-[#0F172A] leading-tight">
					{sesiAktif?.judul ?? 'Tanya perancangan'}
				</h2>
				<p class="text-[11px] text-[#64748B] leading-tight">
					{sesi.length > 1 ? `${sesi.length} sesi percakapan` : 'Hanya soal rancangan proyek ini'}
				</p>
			</div>
		</div>
		<div class="flex items-center gap-2 shrink-0">
			{#if busy}
				<span class="text-[11px] text-[#64748B]">menyusun…</span>
			{/if}
			<button
				type="button"
				onclick={() => void sesiBaru()}
				disabled={busy}
				title="Mulai percakapan baru"
				class="inline-flex items-center gap-1 rounded-lg border border-[#E2E8F0] bg-white px-2 py-1 text-[11px] font-medium text-[#475569] transition-colors hover:border-[#CBD5E1] hover:bg-[#F8FAFC] disabled:opacity-50 disabled:cursor-not-allowed"
				data-purpose="session-new"
			>
				<Icon name="refresh" class="w-3 h-3" />
				Sesi baru
			</button>
			{#if bisaHapus && aktif}
				<button
					type="button"
					onclick={() => void hapusSesi(aktif ?? '')}
					disabled={busy}
					title="Hapus sesi ini"
					class="rounded-lg border border-[#E2E8F0] bg-white px-2 py-1 text-[11px] font-medium text-[#B91C1C] transition-colors hover:border-[#FCA5A5] hover:bg-[#FEF2F2] disabled:opacity-50 disabled:cursor-not-allowed"
					data-purpose="session-delete"
				>
					Hapus
				</button>
			{/if}
		</div>
	</header>

	<!-- Daftar sesi: satu proyek boleh punya banyak percakapan terpisah. -->
	{#if sesi.length > 1}
		<div
			class="flex gap-1.5 px-3 py-2 overflow-x-auto border-b border-[#E2E8F0] bg-white"
			data-purpose="session-list"
		>
			{#each sesi as s (s.id)}
				<button
					type="button"
					onclick={() => pilihSesi(s.id)}
					disabled={busy}
					class="shrink-0 max-w-[180px] truncate rounded-full border px-2.5 py-1 text-[11px] transition-colors disabled:cursor-not-allowed {s.id ===
					aktif
						? 'border-[#2563EB] bg-[#EFF6FF] text-[#1D4ED8] font-medium'
						: 'border-[#E2E8F0] bg-white text-[#475569] hover:border-[#CBD5E1] hover:bg-[#F8FAFC]'}"
					data-purpose="session-tab"
				>
					{s.judul}
				</button>
			{/each}
		</div>
	{/if}

	<div class="flex flex-col gap-3 px-4 py-3 max-h-[380px] overflow-y-auto" data-purpose="chat-thread">
		{#if loading}
			<p class="text-xs text-[#94A3B8]">memuat percakapan…</p>
		{:else if messages.length === 0 && optimistis.length === 0}
			<p class="text-xs text-[#64748B]">
				Belum ada percakapan. Tanya apa saja soal rancangan proyek ini, misalnya
				<span class="font-medium text-[#0F172A]">"mulai dari mana?"</span>
			</p>
		{/if}

		{#each messages as m (m.id)}
			<div class="flex {m.role === 'user' ? 'justify-end' : 'justify-start'}">
				{#if m.role === 'user'}
					<div
						class="rounded-xl px-3 py-2 text-xs leading-relaxed max-w-[85%] whitespace-pre-wrap bg-[#2563EB] text-white"
					>
						{m.content}
					</div>
				{:else}
					{@const suggested = m.suggested_stage ? `TAHAP BERIKUTNYA: ${m.suggested_stage}` : ''}
					<div class="max-w-[85%] min-w-0">
						<!-- Isi balasan = markdown dari model. Disaring mdToHtml (lihat
						     lib/markdown.ts): tanpa itu, teks dari brief bisa menyelipkan
						     HTML dan dieksekusi di browser pembaca. -->
						<div
							class="md text-xs leading-relaxed text-[#0F172A]"
							data-purpose="chat-markdown"
						>
							{@html mdToHtml(suggested ? m.content.replace(suggested, '') : m.content)}
						</div>
						{#if m.suggested_stage}
							<button
								type="button"
								onclick={() => void susunTahap(m.suggested_stage ?? '')}
								disabled={busy}
								class="mt-2 inline-flex items-center gap-1.5 rounded-full border border-[#E2E8F0] bg-white px-2.5 py-1 text-[11px] font-medium text-[#475569] transition-colors hover:border-[#CBD5E1] hover:bg-[#F8FAFC] disabled:opacity-50 disabled:cursor-not-allowed"
								data-purpose="chat-suggested-stage"
							>
								<Icon name="sparkles" class="w-3 h-3 text-[#2563EB]" />
								Susun {m.suggested_stage.toUpperCase()}
							</button>
						{/if}
					</div>
				{/if}
			</div>
		{/each}

		<!-- Pesan pengguna yang baru dikirim, belum dikonfirmasi server. -->
		{#each optimistis as m (m.id)}
			<div class="flex justify-end">
				<div
					class="rounded-xl px-3 py-2 text-xs leading-relaxed max-w-[85%] whitespace-pre-wrap bg-[#2563EB] text-white opacity-70"
					data-purpose="chat-pending-user"
				>
					{m.content}
				</div>
			</div>
		{/each}

		<!--
			Indikator mengetik selama menunggu balasan. Titiknya berdenyut
			bergantian (animation-delay bertingkat), jadi terlihat hidup tanpa
			perlu JavaScript tambahan. Selama ini ada, pengguna tahu permintaan
			sedang diproses, bukan macet.
		-->
		{#if busy}
			<div class="flex justify-start" data-purpose="chat-typing">
				<div class="flex items-center gap-1 py-1" aria-label="Ketok Magic sedang menyusun balasan">
					<span class="typing-dot"></span>
					<span class="typing-dot"></span>
					<span class="typing-dot"></span>
				</div>
			</div>
		{/if}
	</div>

	{#if error}
		<p class="px-4 pb-2 text-[11px] text-[#B91C1C]" data-purpose="chat-error">{error}</p>
	{/if}

	<footer class="border-t border-[#E2E8F0] p-3 flex flex-col gap-2">
		<textarea
			bind:value={draft}
			onkeydown={onkeydown}
			rows="2"
			placeholder={busy ? 'menunggu balasan…' : 'Tulis pertanyaan perancangan… (Enter kirim)'}
			class="w-full resize-none border border-[#E2E8F0] rounded-lg px-3 py-2 text-xs text-[#0F172A] placeholder:text-[#94A3B8] focus:outline-none focus:border-[#2563EB] transition-colors"
			data-purpose="chat-input"
		></textarea>
		<div class="flex items-center justify-between gap-3">
			<span class="text-[11px] {tooLong ? 'text-[#B91C1C]' : 'text-[#94A3B8]'}">
				{draft.trim().length}/4000 · gratis
			</span>
			<button
				type="button"
				onclick={() => void kirim()}
				disabled={busy || !draft.trim() || tooLong}
				class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors {busy ||
				!draft.trim() ||
				tooLong
					? 'bg-[#E2E8F0] text-[#94A3B8] cursor-not-allowed'
					: 'bg-[#2563EB] text-white hover:bg-[#1D4ED8]'}"
				data-purpose="chat-send"
			>
				{busy ? 'Mengirim…' : 'Kirim'}
			</button>
		</div>
	</footer>
</section>
