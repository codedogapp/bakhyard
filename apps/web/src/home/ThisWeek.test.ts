import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import ThisWeek from './ThisWeek.svelte';

test("shows the week's card under its head, which links to the day report", () => {
  render(ThisWeek, { props: { week: { loggedHours: 30, targetHours: 40 } } });

  expect(screen.getByRole('heading', { level: 2, name: 'This week' })).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Day report' })).toHaveAttribute('href', '/time');
  expect(screen.getByText('30h / 40h')).toBeInTheDocument();
});
