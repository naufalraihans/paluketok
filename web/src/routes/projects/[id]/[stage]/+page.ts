/**
 * Rute dinamis /projects/[id]/[stage]: keduanya berasal dari API saat runtime,
 * tak diketahui saat build. Sama seperti rute /projects/[id], jangan
 * diprerender: kalau diprerender, SvelteKit membuat satu file HTML per entry dan
 * slug asli tak punya file sehingga nginx jatuh ke /index.html (landing page).
 */
export const prerender = false;
