<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import { summarySidebar } from '#lib/mock/summary';
	import { auth } from '#lib/auth.svelte';
	import type { Artifact } from '#lib/api';

	// Sidebar /projects/[id]. Kerangka & urutan blok dari mockup (design/web/
	// ringkasanProyek.html), tapi nav-nya LIVE dan PROGRESIF.
	//
	// Progresif: menu tahap muncul satu per satu mengikuti artefak yang sudah
	// jadi. Proyek yang baru dibuat belum punya artefak -> sidebar hanya
	// menampilkan "Ringkasan". Begitu AI selesai menyusun PRD, menu "Dokumen ·
	// PRD" muncul, dan seterusnya. Menu yang halamannya belum ada ditandai
	// "segera" dan tidak bisa dipencet.
	let {
		projectName = '',
		ready = 0,
		total = 0,
		artifacts = []
	}: {
		projectName?: string;
		ready?: number;
		total?: number;
		artifacts?: Artifact[];
	} = $props();

	/** Pemetaan tahap backend -> label & ikon menu sidebar. */
	const STAGE_MENU: Record<string, { label: string; icon: string }> = {
		prd: { label: 'Dokumen · PRD', icon: 'document' },
		erd: { label: 'Model data · ERD', icon: 'db-simple' },
		openapi: { label: 'Kontrak OpenAPI', icon: 'code' },
		roles: { label: 'Role access', icon: 'users' },
		security: { label: 'Proteksi', icon: 'shield' },
		structure: { label: 'Struktur & prototipe', icon: 'layers-row' }
	};

	// Menu yang halamannya belum dibangun: tampil tapi tak bisa dipencet.
	const SOON_MENU = [
		{ label: 'Aktivitas proyek', icon: 'clock' },
		{ label: 'Referensi & aset', icon: 'folder-open' }
	];

	const kinds = $derived(new Set(artifacts.map((a) => a.kind)));

	// Menu tahap hanya untuk artefak yang sudah ada, urut tetap ala planner.md.
	const stageMenu = $derived(
		Object.keys(STAGE_MENU)
			.filter((k) => kinds.has(k as Artifact['kind']))
			.map((k) => ({ key: k, ...STAGE_MENU[k] }))
	);

	const user = $derived({
		initials: (auth.me?.username ?? auth.me?.full_name ?? 'P').slice(0, 1).toUpperCase(),
		name: auth.me?.username ?? auth.me?.full_name ?? 'Pengguna',
		role: auth.me ? `${auth.me.plan_tier} · ${auth.me.credits} kredit` : 'memuat…'
	});
</script>

<aside
	class="w-[260px] bg-white border-r border-[#E2E8F0] flex flex-col justify-between shrink-0 h-full select-none"
	data-purpose="sidebar"
>
	<div class="p-4 flex flex-col gap-4 overflow-y-auto">
		<div class="flex items-center gap-2.5 px-2 py-1" data-purpose="brand-header">
			<div
				class="w-7 h-7 rounded-lg bg-black text-white flex items-center justify-center font-bold text-base shadow-sm"
			>
				<Icon name="logo-cube" class="w-4 h-4" width={2.5} />
			</div>
			<div>
				<h1 class="font-bold text-sm text-[#0F172A] leading-tight">PaluKetok</h1>
				<p class="text-[11px] text-[#64748B] font-medium leading-tight">Workspace Studio</p>
			</div>
		</div>

		<div class="flex flex-col gap-1.5" data-purpose="project-selector-container">
			<span class="text-[10px] font-bold text-[#94A3B8] tracking-wider uppercase px-2"
				>Proyek Aktif</span
			>
			<div
				class="flex items-center justify-between border border-[#E2E8F0] rounded-xl px-3 py-2 bg-white hover:bg-slate-50 cursor-pointer shadow-xs transition-colors"
			>
				<div class="flex items-center gap-2 min-w-0">
					<span class="w-2 h-2 rounded-full bg-[#0F172A] shrink-0"></span>
					<span class="font-semibold text-xs text-[#0F172A] truncate"
						>{projectName || 'Memuat…'}</span
					>
				</div>
				<Icon name="chevron-up-down" class="w-3.5 h-3.5 text-[#94A3B8] shrink-0" />
			</div>
		</div>

		<nav class="flex flex-col gap-1 pt-2" data-purpose="main-navigation">
			<span class="text-[10px] font-bold text-[#94A3B8] tracking-wider uppercase px-2 mb-1"
				>Rancangan Proyek</span
			>

			<a
				class="flex items-center justify-between px-3 py-2 rounded-xl bg-[#EFF6FF] border border-[#BFDBFE]/60 text-[#2563EB] font-medium text-xs transition-colors"
				href="#atas"
			>
				<div class="flex items-center gap-2.5">
					<Icon name="sitemap" class="w-4 h-4" />
					<span>Ringkasan</span>
				</div>
			</a>

			{#each stageMenu as item (item.key)}
				<a
					class="flex items-center justify-between px-3 py-2 rounded-xl text-[#475569] hover:bg-slate-50 font-medium text-xs transition-colors"
					href="#tahap-{item.key}"
					data-purpose="menu-{item.key}"
				>
					<div class="flex items-center gap-2.5">
						<Icon name={item.icon} class="w-4 h-4 text-[#94A3B8]" />
						<span>{item.label}</span>
					</div>
					<span class="text-[10px] font-semibold text-[#10B981]">siap</span>
				</a>
			{/each}

			{#if stageMenu.length === 0}
				<p class="text-[11px] text-[#94A3B8] px-3 py-2 leading-snug" data-purpose="empty-hint">
					Menu tahap muncul setelah AI menyusun dokumennya.
				</p>
			{/if}

			{#each SOON_MENU as item (item.label)}
				<div
					class="flex items-center justify-between px-3 py-2 rounded-xl text-[#CBD5E1] font-medium text-xs cursor-not-allowed mt-2"
					title="Halaman belum tersedia"
					data-purpose="menu-segera"
				>
					<div class="flex items-center gap-2.5">
						<Icon name={item.icon} class="w-4 h-4 text-[#CBD5E1]" />
						<span>{item.label}</span>
					</div>
					<span class="text-[10px] font-semibold text-[#CBD5E1]">segera</span>
				</div>
			{/each}
		</nav>
	</div>

	<div
		class="p-4 border-t border-[#F1F5F9] flex flex-col gap-3.5 bg-white"
		data-purpose="sidebar-footer"
	>
		<div class="border border-[#E2E8F0] rounded-xl p-3 bg-white shadow-xs">
			<span class="text-[10px] font-bold text-[#94A3B8] tracking-wider uppercase block mb-1"
				>{summarySidebar.milestone.label}</span
			>
			<p class="font-bold text-xs text-[#0F172A]">
				{total > 0 ? `${ready}/${total} tahap selesai` : summarySidebar.milestone.title}
			</p>
			<p class="text-[11px] text-[#64748B] mt-0.5">
				{total > 0 ? 'Enam dokumen planning' : summarySidebar.milestone.when}
			</p>
		</div>

		<div class="flex items-center justify-between pt-1">
			<div class="flex items-center gap-2.5">
				<div
					class="w-8 h-8 rounded-full bg-[#1E293B] text-white flex items-center justify-center font-bold text-xs"
				>
					{user.initials}
				</div>
				<div>
					<p class="text-xs font-bold text-[#0F172A] leading-tight">{user.name}</p>
					<p class="text-[11px] text-[#64748B] leading-tight">{user.role}</p>
				</div>
			</div>
			<button
				aria-label="Pengaturan pengguna"
				class="text-[#94A3B8] hover:text-[#475569] p-1 rounded-md"
				type="button"
			>
				<Icon name="adjustments" class="w-4 h-4" />
			</button>
		</div>
	</div>
</aside>
