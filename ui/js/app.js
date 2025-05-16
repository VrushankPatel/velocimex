// Velocimex Web UI Application
import { VelocimexApp } from './velocimex.js';

// Main application initialization
document.addEventListener('DOMContentLoaded', () => {
    const connectionStatusEl = document.getElementById('connection-status');
    const [indicator, label] = connectionStatusEl.children;

    function updateConnectionUI(isConnected, mode = null) {
        if (isConnected) {
            indicator.classList.remove('bg-gray-400', 'bg-red-500');
            indicator.classList.add('bg-green-500');
            label.textContent = `Connected${mode ? ` (${mode})` : ''}`;
        } else {
            indicator.classList.remove('bg-green-500', 'bg-gray-400');
            indicator.classList.add('bg-red-500');
            label.textContent = 'Disconnected';
        }
    }

    // Initialize app
    const app = new VelocimexApp();

    // Listen for connection status changes
    app.on('websocket:open', () => {
        updateConnectionUI(true);
        console.log('WebSocket connected');
    });

    app.on('websocket:close', () => {
        updateConnectionUI(false);
        console.log('WebSocket disconnected');
    });

    app.on('websocket:status', (status) => {
        updateConnectionUI(true, status.mode);
        console.log('System status:', status);
    });

    // Expose app to window for debugging
    window.app = app;
});