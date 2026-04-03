import { mount } from 'svelte';
import App from './App.svelte';
import './global.css';
import './styles/components.css';
import './styles/themes.css';

// Always use Vaporwave
localStorage.setItem('theme', 'vapor');
document.documentElement.setAttribute('data-theme', 'vapor');

const app = mount(App, { target: document.getElementById('app') });

export default app;
