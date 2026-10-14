<script lang="ts">
	import AuthShell from '#lib/components/AuthShell.svelte';
	import { auth, register } from '#lib/auth.svelte';
	import { goto } from '$app/navigation';

	// /register: email+password; Supabase mengirim email konfirmasi
	// (auth_billing.md 1.3). Sesi muncul setelah konfirmasi via /auth/callback.
	let email = $state('');
	let password = $state('');
	let busy = $state(false);
	let err = $state('');
	let done = $state(false);

	$effect(() => {
		if (auth.user && !auth.loading) goto('/projects', { replaceState: true });
	});

	async function submit() {
		if (!email || !password || busy) return;
		busy = true;
		err = '';
		try {
			await register(email, password);
			done = true;
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal mendaftar';
		} finally {
			busy = false;
		}
	}
</script>

<AuthShell title="Daftar">
	{#if done}
		<p class="text-sm font-semibold text-[#0f172a]">Cek emailmu</p>
		<p class="text-sm text-[#475569] mt-2">
			Tautan konfirmasi dikirim ke {email}. Konfirmasi dulu untuk masuk.
		</p>
	{:else}
		<h1 class="text-lg font-bold text-[#0f172a]">Daftar</h1>
		<p class="text-sm text-[#475569] mt-1">Buat akun dengan email dan password.</p>
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
				placeholder="Password, minimal 8 karakter"
				class="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-slate-300 transition-all"
			/>
			{#if err}<p class="text-sm text-rose-600">{err}</p>{/if}
			<button
				type="submit"
				disabled={busy}
				class="w-full inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50"
			>
				{busy ? 'Memproses…' : 'Daftar'}
			</button>
		</form>
		<p class="text-sm text-[#475569] mt-6">
			Sudah punya akun?
			<a href="/login" class="font-semibold text-[#2563EB] hover:underline">Masuk</a>
		</p>
	{/if}
</AuthShell>
