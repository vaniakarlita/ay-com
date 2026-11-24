import { mount } from 'svelte';
import './styles/global.scss';
import './lib/display';
import App from './App.svelte';

const app = mount(App, {
  target: document.getElementById('app')!,
});

export default app;