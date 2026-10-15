<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import { summaryNav, summarySidebar } from '#lib/mock/summary';
	import { auth } from '#lib/auth.svelte';

	// Sidebar /projects/[id]. Kerangka & urutan blok dari mockup (design/web/
	// ringkasanProyek.html), tapi isi live: nama proyek + user dari API.
	// Nav mengarah ke kartu tahap di halaman ini; hitungan "count" dari mock
	// dibuang karena tak ada sumber angkanya di BE.
	let {
		projectName = '',
		ready = 0,
		total = 0
	}: { projectName?: string; ready?: number; total?: number } = $props();

	const user = $derived({
		initials: (auth.me?.username ?? auth.me?.full_name ?? auth.me?.email ?? 'P')
			.slice(0, 1)
			.toUpperCase(),
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
			{#each summaryNav as item (item.label)}
				<a
					class="flex items-center justify-between px-3 py-2 rounded-xl {item.active
						? 'bg-[#EFF6FF] border border-[#BFDBFE]/60 text-[#2563EB] font-medium'
						: 'text-[#475569] hover:bg-slate-50 font-medium'} text-xs {item.label ===
					'Aktivitas proyek'
						? 'mt-2'
						: ''} transition-colors"
					href="#tahap"
				>
					<div class="flex items-center gap-2.5">
						<Icon name={item.icon} class="w-4 h-4 {item.active ? '' : 'text-[#94A3B8]'}" />
						<span>{item.label}</span>
					</div>
				</a>
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
