/**
 * Rute dinamis /projects/[id]: id-nya berasal dari API saat runtime, tak
 * diketahui saat build. Karena itu jangan diprerender — SvelteKit akan
 * menghasilkan satu file HTML per entry, dan id asli tak punya file sehingga
 * nginx jatuh ke /index.html (landing page). Dengan prerender=false, rute ini
 * dilayani SPA fallback 200.html dan render sepenuhnya di klien.
 */
export const prerender = false;
