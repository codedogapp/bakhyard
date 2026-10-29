import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import WeekCard from './WeekCard.svelte';

test('shows the hours logged against the target, and fills the bar as far', () => {
  const { container } = render(WeekCard, { props: { week: { loggedHours: 30, targetHours: 40 } } });

  expect(screen.getByText('30h / 40h')).toBeInTheDocument();
  expect(screen.getByText('Logged on Jira')).toBeInTheDocument();
  expect(container.querySelector('.track')).toHaveAttribute('aria-hidden', 'true');
  expect(container.querySelector('.fill')).toHaveStyle({ width: '75%' });
});
