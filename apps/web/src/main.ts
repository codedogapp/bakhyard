import { mount } from 'svelte';
import '@fontsource-variable/space-grotesk';
import '@fontsource/instrument-serif/400-italic.css';
import '@fontsource-variable/jetbrains-mono';
import './styles/tokens.css';
import './styles/base.css';

import App from './App.svelte';

mount(App, { target: document.getElementById('app')! });
