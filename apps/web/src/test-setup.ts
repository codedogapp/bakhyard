import '@testing-library/jest-dom/vitest';
import { configure } from '@testing-library/svelte';

// A note's page imports its renderer the first time it shows. On a busy CI runner, loading the renderer's packages
// takes longer than a findBy query's default wait of a second.
configure({ asyncUtilTimeout: 3000 });
