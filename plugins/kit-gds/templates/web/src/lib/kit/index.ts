// A starting point, then yours: one import for the kit.
//   import { Button, Field } from '../lib/kit';
// standard shell exports, the same names and props in every kit: AppShell, Footer
// and SignInPage. Compose against these, not against a kit's own pieces.
export { default as AppShell } from './AppShell.svelte';
export { default as Button } from './Button.svelte';
export { default as Dialog } from './Dialog.svelte';
export { default as ErrorSummary } from './ErrorSummary.svelte';
export { default as Field } from './Field.svelte';
export { default as Footer } from './Footer.svelte';
export type { FooterLink } from './Footer.svelte';
export { default as SignInPage } from './SignInPage.svelte';
export { default as Table } from './Table.svelte';
export type { Column } from './Table.svelte';

// More GOV.UK pieces: additive, not part of the component-kit@2 contract.
export { default as BackLink } from './BackLink.svelte';
export { default as InsetText } from './InsetText.svelte';
export { default as NotificationBanner } from './NotificationBanner.svelte';
export { default as ServiceNavigation } from './ServiceNavigation.svelte';
export type { ServiceNavItem } from './ServiceNavigation.svelte';
export { default as SummaryCard } from './SummaryCard.svelte';
export { default as SummaryList } from './SummaryList.svelte';
export type { SummaryRow } from './SummaryList.svelte';
export { default as Tag } from './Tag.svelte';
export { default as TaskList } from './TaskList.svelte';
export type { Task, TaskStatus } from './TaskList.svelte';
