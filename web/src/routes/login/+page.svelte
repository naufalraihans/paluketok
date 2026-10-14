<script lang="ts">
	import AuthShell from '#lib/components/AuthShell.svelte';
	import { auth, login, loginMagic } from '#lib/auth.svelte';
	import { goto } from '$app/navigation';

	// /login: email+password (auth_billing.md 1.3), plus jalur magic link.
	let email = $state('');
	let password = $state('');
	let busy = $state(false);
	let err = $state('');
	let magicSent = $state(false);

	$effect(() => {
		if (auth.user && !auth.loading) goto('/projects', { replaceState: true });
	});

	async function submit() {
		if (!email || !password || busy) return;
		busy = true;
		err = '';
		try {
			await login(email, password);
			goto('/projects', { replaceState: true });
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal masuk';
		} finally {
			busy = false;
		}
	}

	async function magic() {
		if (!email || busy) return;
		busy = true;
		err = '';
		try {
			await loginMagic(email);
			magicSent = true;
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal mengirim tautan';
		} finally {
			busy = false;
		}
	}
</script>

<AuthShell title="Masuk">
	{#if magicSent}
		<p class="text-sm font-semibold text-[#0f172a]">Tautan terkirim</p>
		<p class="text-sm text-[#475569] mt-2">Cek {email} untuk tautan masuk.</p>
	{:else}
		<h1 class="text-lg font-bold text-[#0f172a]">Masuk</h1>
		<p class="text-sm text-[#475569] mt-1">Gunakan email dan password akunmu.</p>
		<form class="space-y-3 mt-5" onsubmit={(e) => { e.preventDefault(); submit(); }}>
			<input
				type="email"
				required
				bind:value={email}
				placeholder="nama@email.com"
				class="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-slate-300 transition-all"
			/>
			<input
				type="password"
				required
				minlength={8}
				bind:value={password}
				placeholder="Password"
				class="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-slate-300 transition-all"
			/>
			{#if err}<p class="text-sm text-rose-600">{err}</p>{/if}
			<button
				type="submit"
				disabled={busy}
				class="w-full inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50"
			>
				{busy ? 'Memproses…' : 'Masuk'}
			</button>
		</form>
		<button
			type="button"
			class="mt-3 w-full text-sm font-medium text-slate-600 hover:text-slate-900 transition-colors"
			onclick={magic}>atau kirim tautan masuk via email</button
		>
		<p class="text-sm text-[#475569] mt-6">
			Belum punya akun?
			<a href="/register" class="font-semibold text-[#2563EB] hover:underline">Daftar</a>
			<span class="mx-1 text-slate-300">·</span>
			<a href="/reset" class="font-semibold text-[#2563EB] hover:underline">Lupa password</a>
		</p>
	{/if}
</AuthShell>
