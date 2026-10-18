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
		listProjectMessages,
		sendProjectMessage,
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
	let draft = $state('');
	let busy = $state(false);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Riwayat dimuat ulang saat proyek berganti (mis. UUID -> slug di URL).
	$effect(() => {
		const id = projectId;
		if (!id) return;
		loading = true;
		error = null;
		listProjectMessages(id)
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

	async function kirim(): Promise<void> {
		const text = draft.trim();
		if (!text || busy || tooLong) return;
		busy = true;
		error = null;
		draft = '';
		try {
			const reply = await sendProjectMessage(projectId, text);
			// Muat ulang riwayat: pertanyaan + jawaban tersimpan di server, jadi
			// tampilan selalu sama dengan isi DB (bukan salinan lokal).
			const res = await listProjectMessages(projectId);
			messages = res.messages ?? [];
			if (reply.suggested_stage && onsuggest) onsuggest(reply.suggested_stage);
		} catch (e: unknown) {
			const msg = e instanceof Error ? e.message : 'gagal mengirim pesan';
			// Slot LLM kosong / provider menolak -> 503. Jelaskan, jangan diam.
			error = msg.includes('503')
				? 'Slot LLM belum siap. Isi dulu di panel admin → settings/ai.'
				: msg;
			draft = text; // jangan buang ketikan pengguna
		} finally {
			busy = false;
		}
	}

	function onkeydown(e: KeyboardEvent): void {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			void kirim();
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
				<h2 class="font-semibold text-xs text-[#0F172A] leading-tight">Tanya perancangan</h2>
				<p class="text-[11px] text-[#64748B] leading-tight">Hanya soal rancangan proyek ini</p>
			</div>
		</div>
		{#if busy}
			<span class="text-[11px] text-[#64748B] shrink-0">menyusun…</span>
		{/if}
	</header>

	<div class="flex flex-col gap-3 px-4 py-3 max-h-[380px] overflow-y-auto" data-purpose="chat-thread">
		{#if loading}
			<p class="text-xs text-[#94A3B8]">memuat percakapan…</p>
		{:else if messages.length === 0}
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
								onclick={() => onsuggest?.(m.suggested_stage ?? '')}
								class="mt-2 inline-flex items-center gap-1.5 rounded-full border border-[#E2E8F0] bg-white px-2.5 py-1 text-[11px] font-medium text-[#475569] transition-colors hover:border-[#CBD5E1] hover:bg-[#F8FAFC]"
								data-purpose="chat-suggested-stage"
							>
								<Icon name="sparkles" class="w-3 h-3 text-[#2563EB]" />
								Lanjut ke {m.suggested_stage}
							</button>
						{/if}
					</div>
				{/if}
			</div>
		{/each}
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
