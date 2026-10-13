<script lang="ts">
	import SiteNav from '#lib/components/SiteNav.svelte';
	import HeroSection from '#lib/components/HeroSection.svelte';
	import ProblemSection from '#lib/components/ProblemSection.svelte';
	import FeatureSection from '#lib/components/FeatureSection.svelte';
	import TraceabilitySection from '#lib/components/TraceabilitySection.svelte';
	import ProcessSection from '#lib/components/ProcessSection.svelte';
	import PersonaSection from '#lib/components/PersonaSection.svelte';
	import SecuritySection from '#lib/components/SecuritySection.svelte';
	import FinalCtaSection from '#lib/components/FinalCtaSection.svelte';
	import SiteFooter from '#lib/components/SiteFooter.svelte';
	import { landing } from '#lib/mock/landing';

	// Fase 9 langkah 34: / = design/web/landingPage.html.
	// Urutan blok mengikuti mockup: nav, hero, problem, feature, traceability,
	// process, persona, security, final CTA, footer.
</script>

<svelte:head>
	<title>{landing.meta.title}</title>
	<meta name="description" content={landing.meta.description} />
</svelte:head>

<SiteNav nav={landing.nav} />

<main>
	<HeroSection hero={landing.hero} />
	<ProblemSection problem={landing.problem} />
	<FeatureSection feature={landing.feature} />
	<TraceabilitySection traceability={landing.traceability} />
	<ProcessSection process={landing.process} />
	<PersonaSection personas={landing.personas} />
	<SecuritySection security={landing.security} />
	<FinalCtaSection finalCta={landing.finalCta} />
</main>

<SiteFooter footer={landing.footer} />