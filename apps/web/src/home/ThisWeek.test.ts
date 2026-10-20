import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import ThisWeek from './ThisWeek.svelte';

test('shows the hours logged against the target, and links to the day report', () => {
  const { container } = render(ThisWeek, { props: { week: { loggedHours: 30, targetHours: 40 } } });

  expect(screen.getByRole('heading', { level: 2, name: 'This week' })).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Day report' })).toHaveAttribute('href', '/time');
  expect(screen.getByText('30h / 40h')).toBeInTheDocument();
  expect(screen.getByText('Logged on Jira')).toBeInTheDocument();
  expect(container.querySelector('.fill')).toHaveStyle({ width: '75%' });
});
