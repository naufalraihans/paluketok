<script lang="ts">
	import AuthShell from '#lib/components/AuthShell.svelte';
	import { supabase } from '#lib/supabase';
	import { requestReset, updatePassword } from '#lib/auth.svelte';
	import { goto } from '$app/navigation';

	// /reset (auth_billing.md 1.3): link reset balik ke sini.
	// Mode 1: minta link reset (masukkan email).
	// Mode 2: link membawa kode PKCE -> tukar jadi sesi, lalu set password baru.
	let email = $state('');
	let password = $state('');
	let busy = $state(false);
	let err = $state('');
	let sent = $state(false);
	let ready = $state(false); // sesi reset aktif, boleh set password baru

	$effect(() => {
		const u = supabase.auth;
		u.getSession().then(async ({ data }) => {
			if (data.session) {
				ready = true;
				return;
			}
			const code = new URL(location.href).searchParams.get('code');
			if (code) {
				const { error } = await u.exchangeCodeForSession(code);
				if (error) err = error.message;
				else ready = true;
			}
		});
	});

	async function submitRequest() {
		if (!email || busy) return;
		busy = true;
		err = '';
		try {
			await requestReset(email);
			sent = true;
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal mengirim email reset';
		} finally {
			busy = false;
		}
	}

	async function submitNew() {
		if (!password || busy) return;
		busy = true;
		err = '';
		try {
			await updatePassword(password);
			goto('/login', { replaceState: true });
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal menyimpan password';
		} finally {
			busy = false;
		}
	}
</script>

<AuthShell title="Reset password">
	{#if ready}
		<h1 class="text-lg font-bold text-[#0f172a]">Password baru</h1>
		<p class="text-sm text-[#475569] mt-1">Tautan terverifikasi. Set password barumu.</p>
		<form class="space-y-3 mt-5" onsubmit={(e) => { e.preventDefault(); submitNew(); }}>
			<input
				type="password"
				required
				minlength={8}
				bind:value={password}
				placeholder="Password baru, minimal 8 karakter"
				class="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-slate-300 transition-all"
			/>
			{#if err}<p class="text-sm text-rose-600">{err}</p>{/if}
			<button
				type="submit"
				disabled={busy}
				class="w-full inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50"
			>
				{busy ? 'Menyimpan…' : 'Simpan password'}
			</button>
		</form>
	{:else if sent}
		<p class="text-sm font-semibold text-[#0f172a]">Tautan reset terkirim</p>
		<p class="text-sm text-[#475569] mt-2">Cek {email} dan buka tautan resetnya.</p>
	{:else}
		<h1 class="text-lg font-bold text-[#0f172a]">Lupa password</h1>
		<p class="text-sm text-[#475569] mt-1">Masukkan email, kami kirim tautan reset.</p>
		<form class="space-y-3 mt-5" onsubmit={(e) => { e.preventDefault(); submitRequest(); }}>
			<input
				type="email"
				required
				bind:value={email}
				placeholder="nama@email.com"
				class="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-slate-300 transition-all"
			/>
			{#if err}<p class="text-sm text-rose-600">{err}</p>{/if}
			<button
				type="submit"
				disabled={busy}
				class="w-full inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50"
			>
				{busy ? 'Mengirim…' : 'Kirim tautan reset'}
			</button>
		</form>
		<p class="text-sm text-[#475569] mt-6">
			<a href="/login" class="font-semibold text-[#2563EB] hover:underline">Kembali masuk</a>
		</p>
	{/if}
</AuthShell>
