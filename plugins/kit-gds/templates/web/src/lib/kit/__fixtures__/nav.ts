// The nav every fixture's AppShell is given.
import type { NavGroup } from '../../shell/types';

export function fixtureNav(extra?: string): NavGroup[] {
	return [
		{
			id: null,
			items: [
				{ label: 'Home', order: 0, href: '/' },
				{ label: 'Reports', order: 10, href: '/reports' },
				...(extra ? [{ label: extra, order: 20, href: '/extra' }] : [])
			]
		}
	];
}
