import { mount } from 'svelte';
import '@fontsource-variable/space-grotesk';
import '@fontsource/instrument-serif/400-italic.css';
import '@fontsource-variable/jetbrains-mono';
import './styles/tokens.css';
import './styles/base.css';
import './styles/form.css';
import './styles/list.css';
import './styles/chip.css';
import './styles/icon-button.css';

import App from './App.svelte';
import { applyChoice, loadChoice } from './shell/theme';

// public/theme.js already set the scheme; this also points the browser's bar at it, on every screen.
applyChoice(loadChoice());
mount(App, { target: document.getElementById('app')! });
