import { render, screen } from '@testing-library/svelte';
import { expect, test } from 'vitest';

import SignIn from './SignIn.svelte';

test('asks the user to sign in with Google', () => {
  render(SignIn);

  expect(screen.getByRole('heading', { level: 1, name: 'Signed out' })).toBeInTheDocument();
  expect(screen.getByText('Sign in with your Google account to see your day.')).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/auth/google/login');
});
