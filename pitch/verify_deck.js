// Verifikasi deck: overflow geometris + kontras WCAG + integritas dasar.
// Dipakai lewat Chrome headless --dump-dom (hasil JSON ditulis ke <pre id="out">).
(function () {
  var out = { ok: true, findings: [], slides: [] };
  var stage = document.getElementById('stage');
  var slides = Array.prototype.slice.call(document.querySelectorAll('.slide'));
  out.file_protocol = location.protocol;
  out.font_inline = /data:font\/woff2/.test(document.querySelector('style').textContent);
  out.gradient = /gradient/i.test(document.querySelector('style').textContent);

  function lum(c) {
    var m = c.match(/[\d.]+/g);
    if (!m) return null;
    var rgb = m.slice(0, 3).map(Number).map(function (v) {
      v /= 255;
      return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
    });
    return 0.2126 * rgb[0] + 0.7152 * rgb[1] + 0.0722 * rgb[2];
  }
  function bgOf(el) {
    var n = el;
    while (n && n !== document.documentElement) {
      var b = getComputedStyle(n).backgroundColor;
      if (b && b !== 'rgba(0, 0, 0, 0)' && b !== 'transparent') return b;
      n = n.parentElement;
    }
    return 'rgb(255,255,255)';
  }
  function contrast(a, b) {
    var la = lum(a), lb = lum(b);
    if (la === null || lb === null) return null;
    return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05);
  }

  slides.forEach(function (s, i) {
    var prev = s.style.display;
    s.style.display = 'block';
    var sr = s.getBoundingClientRect();
    var sc = parseFloat(getComputedStyle(stage).transform.split('(')[1]);
    var info = { n: i + 1, overflow: [], low_contrast: [], min_contrast: 99, nodes: 0 };
    var all = s.querySelectorAll('*');
    for (var k = 0; k < all.length; k++) {
      var el = all[k];
      if (el.classList.contains('a')) { el.style.animation = 'none'; el.style.opacity = 1; }
      var cs = getComputedStyle(el);
      if (cs.display === 'none' || cs.visibility === 'hidden') continue;
      var r = el.getBoundingClientRect();
      if (r.width === 0 && r.height === 0) continue;
      info.nodes++;
      // toleransi 1px pasca-skala
      var slack = 1.5 / sc;
      if (r.left < sr.left - slack || r.top < sr.top - slack ||
          r.right > sr.right + slack || r.bottom > sr.bottom + slack) {
        info.overflow.push({
          tag: el.tagName.toLowerCase(),
          cls: (el.className || '').toString().slice(0, 44),
          t: el.textContent.trim().slice(0, 34),
          d: [Math.round(r.left - sr.left), Math.round(r.top - sr.top),
              Math.round(r.right - sr.right), Math.round(r.bottom - sr.bottom)]
        });
      }
      // kontras: hanya elemen berteks langsung
      var own = '';
      for (var c = 0; c < el.childNodes.length; c++) {
        if (el.childNodes[c].nodeType === 3) own += el.childNodes[c].textContent;
      }
      own = own.trim();
      if (own.length > 1 && el.tagName !== 'PRE' && el.tagName !== 'STYLE') {
        var fs = parseFloat(cs.fontSize);
        if (fs <= 26) {
          var ratio = contrast(cs.color, bgOf(el));
          if (ratio !== null) {
            if (ratio < info.min_contrast) info.min_contrast = Math.round(ratio * 100) / 100;
            var need = (fs >= 18.66 || (fs >= 14 && parseInt(cs.fontWeight, 10) >= 700)) ? 3 : 4.5;
            if (ratio < need) {
              info.low_contrast.push({ t: own.slice(0, 40), fs: fs, ratio: Math.round(ratio * 100) / 100, need: need });
            }
          }
        }
      }
    }
    s.style.display = prev;
    info.min_contrast = info.min_contrast === 99 ? null : info.min_contrast;
    if (info.overflow.length || info.low_contrast.length) out.ok = false;
    out.slides.push(info);
  });

  var pre = document.createElement('pre');
  pre.id = 'out';
  pre.textContent = JSON.stringify(out);
  document.body.appendChild(pre);
})();
