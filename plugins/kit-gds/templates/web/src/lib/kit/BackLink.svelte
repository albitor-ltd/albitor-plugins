<!-- A starting point, then yours: the GOV.UK back link (Frontend 6.4.0 markup). Put it at the top of
     the page, before the main content and above the h1, on every question page of a flow: it goes
     to the page the person came from, so give the real previous page (not '#'). Do not use it with
     breadcrumbs. -->
<script lang="ts">
	let { href, text = 'Back' }: { href: string; text?: string } = $props();
</script>

<a class="govuk-back-link" {href}>{text}</a>
