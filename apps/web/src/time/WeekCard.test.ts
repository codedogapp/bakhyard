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

test('without a week, says the hours are unavailable in their place, with no label or bar', () => {
  const { container } = render(WeekCard);

  expect(screen.getByText('Hours unavailable')).toBeInTheDocument();
  expect(screen.queryByText('Logged on Jira')).not.toBeInTheDocument();
  expect(container.querySelector('.track')).not.toBeInTheDocument();
});
