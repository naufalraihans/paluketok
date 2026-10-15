<script lang="ts">
	import SiteNav from '#lib/components/SiteNav.svelte';
	import SiteFooter from '#lib/components/SiteFooter.svelte';
	import { landing } from '#lib/mock/landing';
	import { auth, refreshMe } from '#lib/auth.svelte';
	import { createOrder, getOrder, ApiError, type Order } from '#lib/api';
	import { goto } from '$app/navigation';

	// /billing: saldo + riwayat order sesi + beli paket Starter (openapi.md 12).
	// Riwayat order = dari sessionStorage (BE tak sediakan list-order per user,
	// dilapor ke Master). Status order lalu via GET /v1/orders/{id}.
	let orders = $state<Order[]>([]);
	let busy = $state(false);
	let msg = $state('');
	let err = $state('');
	let checking = $state(false);

	const ORDERS_KEY = 'pk-orders';

	function loadLocal(): void {
		try {
			orders = JSON.parse(sessionStorage.getItem(ORDERS_KEY) ?? '[]') as Order[];
		} catch {
			orders = [];
		}
	}

	$effect(() => {
		if (auth.user && !auth.loading) {
			loadLocal();
		} else if (!auth.loading && !auth.user) {
			goto('/login', { replaceState: true });
		}
	});

	async function buy(): Promise<void> {
		if (busy) return;
		busy = true;
		err = '';
		msg = '';
		try {
			const o = await createOrder('starter');
			const fresh = [o, ...orders.filter((x) => x.order_id !== o.order_id)];
			orders = fresh;
			sessionStorage.setItem(ORDERS_KEY, JSON.stringify(fresh));
			if (o.payment_url) {
				window.location.href = o.payment_url;
				return;
			}
			msg = 'Order dibuat, tapi payment_url belum tersedia. Coba cek status.';
		} catch (e) {
			err = e instanceof Error ? e.message : 'Gagal membuat order';
		} finally {
			busy = false;
		}
	}

	async function check(o: Order): Promise<void> {
		checking = true;
		err = '';
		msg = '';
		try {
			const fresh = await getOrder(o.order_id);
			orders = orders.map((x) => (x.order_id === fresh.order_id ? fresh : x));
			sessionStorage.setItem(ORDERS_KEY, JSON.stringify(orders));
			if (fresh.status === 'settlement') {
				msg = `Invoice ${fresh.invoice_number} LUNAS. Kredit bertambah ${fresh.credits_granted}.`;
				await refreshMe();
			} else if (fresh.status === 'expire') {
				msg = `Invoice ${fresh.invoice_number} kedaluwarsa.`;
			} else {
				msg = `Invoice ${fresh.invoice_number} masih menunggu pembayaran.`;
			}
		} catch (e) {
			err = e instanceof ApiError ? e.message : 'Gagal cek status order';
		} finally {
			checking = false;
		}
	}

	const rupiah = (n: number): string => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n);
</script>

<svelte:head>
	<title>Billing · PaluKetok</title>
</svelte:head>
<div class="min-h-screen flex flex-col bg-white text-[#0f172a] font-sans antialiased">
	<SiteNav nav={landing.nav} />
	<main class="flex-1 max-w-4xl w-full mx-auto px-6 py-10 space-y-8">
		{#if !auth.user}
			<p class="text-sm text-[#475569]">Memuat…</p>
		{:else}
			<section class="rounded-xl border border-slate-200 bg-white p-8 shadow-sm flex items-center justify-between" data-purpose="billing-balance">
				<div>
					<p class="text-xs font-semibold uppercase tracking-wider text-[#64748B]">Saldo kredit</p>
					<p class="text-3xl font-bold mt-1">{auth.me?.credits ?? '…'} <span class="text-base font-semibold text-[#64748B]">kredit</span></p>
					<p class="text-sm text-[#475569] mt-1">Paket {auth.me?.plan_tier ?? 'free'}</p>
				</div>
				<div class="text-right">
					<p class="text-sm font-semibold">Paket Starter</p>
					<p class="text-sm text-[#475569] mt-1">{rupiah(25000)} · 10 kredit</p>
					<button
						type="button"
						disabled={busy}
						class="mt-3 inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50"
						onclick={buy}>{busy ? 'Memproses…' : 'Beli paket Starter'}</button
					>
				</div>
			</section>

			{#if msg}<div class="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800" data-purpose="billing-msg">{msg}</div>{/if}
			{#if err}<div class="rounded-xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700" data-purpose="billing-err">{err}</div>{/if}

			<section class="space-y-3" data-purpose="billing-orders">
				<h2 class="text-sm font-bold text-[#0f172a]">Riwayat order (sesi ini)</h2>
				{#if orders.length === 0}
					<p class="text-sm text-[#475569]">Belum ada order. Beli paket Starter untuk mulai.</p>
				{:else}
					<div class="rounded-xl border border-slate-200 overflow-hidden">
						<table class="w-full text-sm">
							<thead class="bg-[#F8F9FA] text-[#64748B]">
								<tr>
									<th class="text-left px-4 py-2.5 font-semibold">Invoice</th>
									<th class="text-left px-4 py-2.5 font-semibold">Nominal</th>
									<th class="text-left px-4 py-2.5 font-semibold">Kredit</th>
									<th class="text-left px-4 py-2.5 font-semibold">Status</th>
									<th class="px-4 py-2.5"></th>
								</tr>
							</thead>
							<tbody>
								{#each orders as o (o.order_id)}
									<tr class="border-t border-slate-100">
										<td class="px-4 py-3 font-mono text-xs">{o.invoice_number}</td>
										<td class="px-4 py-3">{rupiah(o.amount_idr)}</td>
										<td class="px-4 py-3">{o.credits_granted}</td>
										<td class="px-4 py-3">
											<span
												class="inline-flex px-2 py-0.5 rounded-full text-xs font-semibold {o.status === 'settlement'
													? 'bg-emerald-50 text-emerald-700'
													: o.status === 'expire'
														? 'bg-slate-100 text-slate-500'
														: 'bg-amber-50 text-amber-700'}"
											>
												{o.status}
											</span>
										</td>
										<td class="px-4 py-3 text-right whitespace-nowrap">
											{#if o.status === 'pending' && o.payment_url}
												<a
													href={o.payment_url}
													target="_blank"
													rel="noopener noreferrer"
													class="text-xs font-semibold text-[#2563EB] hover:underline transition-all">Bayar</a
												>
												<span class="text-slate-300 mx-1.5">·</span>
											{/if}
											<button
												type="button"
												disabled={checking}
												class="text-xs font-semibold text-[#2563EB] hover:underline disabled:opacity-50 transition-all"
												onclick={() => check(o)}>Cek status</button
											>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					{#if auth.me}
						<p class="text-xs text-[#64748B]">
							Kredit di navbar ter-update otomatis setelah Mayar mengabarkan settlement.
						</p>
					{/if}
				{/if}
			</section>
		{/if}
	</main>
	<SiteFooter footer={landing.footer} />
</div>
