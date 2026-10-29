import './styles.css';
import { createRoot } from '@yukino.js/lit-jsx';
import { App } from './app';

createRoot(document.getElementById('app')!).render(<App />);
