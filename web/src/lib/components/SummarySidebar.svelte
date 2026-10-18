<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import { summarySidebar } from '#lib/mock/summary';
	import { auth } from '#lib/auth.svelte';
	import { STAGES as DEFAULT_STAGES, STAGE_MENU_LABEL, STAGE_LABEL, STAGE_ICON, type PlanStage } from '#lib/stages';
	import type { Artifact } from '#lib/api';

	// Sidebar /projects/[id] dan /projects/[id]/[stage].
	//
	// Menu tahap ditampilkan SELURUHNYA sejak awal, bukan satu per satu mengikuti
	// artefak yang sudah jadi. Dulu menu yang artefaknya belum ada disembunyikan,
	// dan itu justru sebab keluhan "menu dokumen hilang": pengguna tidak tahu
	// tahap apa saja yang akan ada. Sekarang yang belum jadi ditandai "belum".
	//
	// Urutan dan label datang dari skema tahap milik run (`stages`). Sebelum ada
	// run, dipakai susunan awal enam tahap sebagai pratinjau.
	let {
		projectName = '',
		ready = 0,
		total = 0,
		artifacts = [],
		slug = '',
		stages = [],
		activeKey = ''
	}: {
		projectName?: string;
		ready?: number;
		total?: number;
		artifacts?: Artifact[];
		/** Slug proyek untuk href; jatuh ke kosong bila belum diketahui. */
		slug?: string;
		/** Skema tahap run. Kosong = pakai susunan awal sebagai pratinjau. */
		stages?: PlanStage[];
		/** Key tahap yang sedang dibuka, untuk menyorot menunya. */
		activeKey?: string;
	} = $props();

	// Menu yang halamannya belum dibangun: tampil tapi tak bisa dipencet.
	const SOON_MENU = [
		{ label: 'Aktivitas proyek', icon: 'clock' },
		{ label: 'Referensi & aset', icon: 'folder-open' }
	];

	const kinds = $derived(new Set(artifacts.map((a) => a.kind)));

	/**
	 * Skema yang dipakai: dari run bila ada, kalau tidak susunan awal. Tahap
	 * karangan tetap dapat label dan ikon wajar lewat cadangan di stages.ts.
	 */
	const spec = $derived(
		stages.length > 0
			? stages
			: DEFAULT_STAGES.map((key, i) => ({
					key,
					label: STAGE_LABEL[key],
					position: i
				}))
	);

	const stageMenu = $derived(
		[...spec]
			.sort((a, b) => a.position - b.position)
			.map((s) => ({
				key: s.key,
				label: s.label || STAGE_MENU_LABEL[s.key] || STAGE_LABEL[s.key] || s.key,
				icon: STAGE_ICON[s.key] ?? 'document',
				ok: kinds.has(s.key as Artifact['kind'])
			}))
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
				class="flex items-center justify-between px-3 py-2 rounded-xl text-[#475569] hover:bg-slate-50 font-medium text-xs transition-colors"
				href={slug ? `/projects/${slug}` : '/projects'}
			>
				<div class="flex items-center gap-2.5">
					<Icon name="sitemap" class="w-4 h-4" />
					<span>Ringkasan</span>
				</div>
			</a>

			{#each stageMenu as item (item.key)}
				<a
					class="flex items-center justify-between px-3 py-2 rounded-xl font-medium text-xs transition-colors {activeKey ===
					item.key
						? 'bg-[#EFF6FF] border border-[#BFDBFE]/60 text-[#2563EB]'
						: 'text-[#475569] hover:bg-slate-50'}"
					href={slug ? `/projects/${slug}/${item.key}` : `/projects/${item.key}`}
					data-purpose="menu-{item.key}"
				>
					<div class="flex items-center gap-2.5">
						<Icon name={item.icon} class="w-4 h-4 text-[#94A3B8]" />
						<span>{item.label}</span>
					</div>
					<span
						class="text-[10px] font-semibold {item.ok ? 'text-[#10B981]' : 'text-[#CBD5E1]'}"
						>{item.ok ? 'siap' : 'belum'}</span
					>
				</a>
			{/each}

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
				{total > 0 ? `${ready}/${total} tahap selesai` : 'Belum ada dokumen'}
			</p>
			<p class="text-[11px] text-[#64748B] mt-0.5">
				{total > 0
					? `${total} dokumen planning`
					: 'Mulai generate untuk menyusun dokumen planning'}
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
