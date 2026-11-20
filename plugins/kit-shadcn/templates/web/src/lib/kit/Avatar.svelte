<!-- A starting point, then yours: a person or team, as a photo with initials when there is none (or
     it fails to load). The name is the accessible name; the image has an empty alt so it is
     not read twice.

       <Avatar name="Ada Lovelace" src={person.photo} size="md" /> -->
<script lang="ts">
	import { Avatar } from 'bits-ui';

	let {
		name,
		src,
		size = 'md'
	}: { name: string; src?: string; size?: 'sm' | 'md' | 'lg' } = $props();

	const initials = $derived(
		name
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((w) => w[0].toUpperCase())
			.join('')
	);
</script>

<Avatar.Root class="kit-avatar kit-avatar-{size}" role="img" aria-label={name} data-kit="avatar">
	{#if src}<Avatar.Image {src} alt="" class="kit-avatar-image" />{/if}
	<Avatar.Fallback class="kit-avatar-fallback" aria-hidden="true">{initials}</Avatar.Fallback>
</Avatar.Root>

<style>
	:global(.kit-avatar) {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex: none;
		overflow: hidden;
		border-radius: 50%;
		background: var(--muted);
		color: var(--foreground);
		font-weight: 600;
		inline-size: 2.5rem;
		block-size: 2.5rem;
		font-size: var(--text-small);
	}
	:global(.kit-avatar-sm) {
		inline-size: 1.75rem;
		block-size: 1.75rem;
		font-size: var(--text-micro);
	}
	:global(.kit-avatar-lg) {
		inline-size: 4rem;
		block-size: 4rem;
		font-size: var(--text-heading);
	}
	:global(.kit-avatar-image) {
		inline-size: 100%;
		block-size: 100%;
		object-fit: cover;
	}
</style>
