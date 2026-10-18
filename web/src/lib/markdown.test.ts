/**
 * Uji mdToHtml. Jalankan: node --experimental-strip-types src/lib/markdown.test.ts
 *
 * Fokus: (1) markdown yang dipakai model benar-benar jadi tag, (2) tidak ada
 * payload yang bisa lolos jadi elemen/atribut berbahaya.
 */
import { mdToHtml } from './markdown.ts';

let gagal = 0;
function cek(nama: string, dapat: string, harus: (h: string) => boolean) {
	const ok = harus(dapat);
	if (!ok) gagal++;
	console.log(`${ok ? 'PASS' : 'FAIL'}  ${nama}`);
	if (!ok) console.log(`      dapat: ${dapat}`);
}

// --- markdown normal ---
cek('bold jadi <strong>', mdToHtml('ini **tebal** ya'), (h) => h.includes('<strong>tebal</strong>'));
cek('italic jadi <em>', mdToHtml('ini *miring* ya'), (h) => h.includes('<em>miring</em>'));
cek('heading jadi <h3>', mdToHtml('### Tahap'), (h) => h.includes('<h3>'));
cek('list jadi <ul><li>', mdToHtml('- satu\n- dua'), (h) => h.includes('<ul>') && h.includes('<li>'));
cek('kode inline jadi <code>', mdToHtml('pakai `npm i` dulu'), (h) => h.includes('<code>npm i</code>'));
cek('blok kode jadi <pre><code>', mdToHtml('```js\nlet a=1\n```'), (h) =>
	h.includes('<pre>') && h.includes('<code>')
);
cek('garis baru jadi <br>', mdToHtml('baris satu\nbaris dua'), (h) => h.includes('<br>'));
cek('tabel jadi <table>', mdToHtml('| a | b |\n|---|---|\n| 1 | 2 |'), (h) => h.includes('<table>'));

// --- XSS: harus DINETRALKAN ---
cek('script tag dibuang', mdToHtml('<script>alert(1)</script>'), (h) => !h.includes('<script'));
cek('img onerror dibuang', mdToHtml('<img src=x onerror=alert(1)>'), (h) => !/<img/i.test(h));
cek('svg onload dibuang', mdToHtml('<svg onload=alert(1)></svg>'), (h) => !/<svg/i.test(h));
cek('iframe dibuang', mdToHtml('<iframe src="//evil"></iframe>'), (h) => !/<iframe/i.test(h));
cek('javascript: href dibuang', mdToHtml('[klik](javascript:alert(1))'), (h) => !/javascript:/i.test(h));
cek('data: href dibuang', mdToHtml('[klik](data:text/html;base64,PHNjcmlwdD4=)'), (h) =>
	!/data:/i.test(h)
);
cek('onclick atribut dibuang', mdToHtml('<p onclick="alert(1)">x</p>'), (h) => !/onclick/i.test(h));
cek('style atribut dibuang', mdToHtml('<p style="x:y">x</p>'), (h) => !/style=/i.test(h));

// --- tautan sah tetap hidup ---
cek('https href dipertahankan', mdToHtml('[dok](https://example.com)'), (h) =>
	h.includes('href="https://example.com"') && h.includes('rel="noopener noreferrer nofollow"')
);
cek('anchor # dipertahankan', mdToHtml('[ke atas](#atas)'), (h) => h.includes('href="#atas"'));

// --- HTML mentah tidak pernah jadi elemen ---
// Tag di luar daftar putih DIBUANG, bukan ditampilkan sebagai teks: balasan
// model tidak butuh tag HTML mentah, dan membuangnya lebih aman daripada
// menampilkannya. Yang penting: tidak ada elemen asing yang lolos ke DOM.
cek('html mentah dibuang tanpa jadi elemen', mdToHtml('pakai <details> ini'), (h) =>
	!/<details/i.test(h) && h.includes('pakai')
);
cek('atribut pada tag yang diizinkan ikut dibuang', mdToHtml('<p class="x" id="y">halo</p>'), (h) =>
	h.includes('<p>halo</p>')
);

console.log(gagal === 0 ? '\nSEMUA LULUS' : `\n${gagal} GAGAL`);
// Dipanggil manual lewat `node --experimental-strip-types`; bukan bagian build.
// `process` tidak ada di tipe browser, jadi lewat globalThis.
(globalThis as { process?: { exit(code: number): void } }).process?.exit(gagal === 0 ? 0 : 1);
