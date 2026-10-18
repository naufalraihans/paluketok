/* PaluKetok Pitch Deck — navigasi, timer, catatan. Tanpa dependensi luar. */
(function () {
  var stage = document.getElementById('stage');
  var slides = Array.prototype.slice.call(document.querySelectorAll('.slide'));
  var notes = document.getElementById('notes');
  var notesBody = document.getElementById('notesBody');
  var notesHint = document.getElementById('notesHint');
  var clock = document.getElementById('clock');
  var cl = document.getElementById('cl');
  var TOTAL = slides.length;
  var cur = 0;

  /* ---- skala panggung ke jendela ---- */
  function fit() {
    var s = Math.min(window.innerWidth / 1920, window.innerHeight / 1080);
    stage.style.transform = 'translate(-50%,-50%) scale(' + s + ')';
  }
  window.addEventListener('resize', fit);
  document.addEventListener('fullscreenchange', function () {
    setTimeout(fit, 60);
  });
  fit();

  /* ---- target waktu kumulatif ---- */
  var target = [];
  var acc = 0;
  slides.forEach(function (s) {
    acc += parseInt(s.getAttribute('data-t') || '60', 10);
    target.push(acc);
  });
  var GRAND = acc;

  function mmss(v) {
    v = Math.max(0, Math.round(v));
    var m = Math.floor(v / 60), s = v % 60;
    return (m < 10 ? '0' : '') + m + ':' + (s < 10 ? '0' : '') + s;
  }

  function stateOf(i) {
    var dt = parseInt(slides[i].getAttribute('data-t') || '60', 10);
    var part = Math.floor(dt / 3);
    return slides[i].getAttribute('data-state') || ('Bagian ' + (1 + (i < part ? 0 : i < 2 * part ? 1 : 2)) + ' dari 3');
  }

  /* ---- tampilkan slide ---- */
  function show(i, silent) {
    i = Math.max(0, Math.min(TOTAL - 1, i));
    if (i === cur && slides[i].classList.contains('on')) {
      paintNotes();
      return;
    }
    slides[cur].classList.remove('on');
    cur = i;
    var el = slides[cur];
    el.classList.add('on');

    /* mulai ulang animasi masuk */
    var anim = el.querySelectorAll('.a');
    for (var k = 0; k < anim.length; k++) {
      anim[k].style.animation = 'none';
      void anim[k].offsetWidth;
      anim[k].style.animation = '';
    }
    if (!silent) {
      try { localStorage.setItem('pk_deck_slide', String(cur)); } catch (e) {}
      if (location.hash !== '#' + (cur + 1)) {
        history.replaceState(null, '', '#' + (cur + 1));
      }
    }
    paintNotes();
  }

  function paintNotes() {
    notesBody.textContent = slides[cur].getAttribute('data-notes') || '(tanpa catatan)';
    notesHint.textContent =
      'Slide ' + (cur + 1) + ' / ' + TOTAL +
      '  ·  ' + stateOf(cur) +
      '  ·  target slide ' + mmss(target[cur] - (cur > 0 ? target[cur - 1] : 0));
  }

  /* ---- timer ---- */
  var el0 = null, paused = false;
  function tick(ts) {
    if (el0 === null) el0 = ts;
    var e = (ts - el0) / 1000;
    cl.textContent = mmss(e) + ' / ' + mmss(GRAND);
    clock.classList.toggle('over', e > GRAND);
    requestAnimationFrame(tick);
  }
  requestAnimationFrame(tick);
  clock.classList.add('paused');

  function toggleTimer() {
    if (paused) {
      paused = false;
      if (el0 === null) { el0 = performance.now(); }
      else { el0 = performance.now() - pausedAt; }
      clock.classList.remove('paused');
    } else {
      paused = true;
      pausedAt = paused ? (performance.now() - el0) : 0;
      clock.classList.add('paused');
    }
  }
  var pausedAt = 0;

  /* ---- input ---- */
  document.addEventListener('keydown', function (e) {
    var k = e.key;
    if (k === 'ArrowRight' || k === ' ' || k === 'PageDown' || k === 'Enter') {
      e.preventDefault(); show(cur + 1);
    } else if (k === 'ArrowLeft' || k === 'PageUp' || k === 'Backspace') {
      e.preventDefault(); show(cur - 1);
    } else if (k === 'Home') { e.preventDefault(); show(0); }
    else if (k === 'End') { e.preventDefault(); show(TOTAL - 1); }
    else if (k === 'n' || k === 'N') { notes.classList.toggle('on'); }
    else if (k === 'T') { toggleTimer(); }
    else if (k === 'R') { el0 = null; paused = true; clock.classList.add('paused'); cl.textContent = mmss(0) + ' / ' + mmss(GRAND); }
    else if (k === 'f' || k === 'F') {
      if (document.fullscreenElement) document.exitFullscreen();
      else document.documentElement.requestFullscreen();
    } else if (/^[1-9]$/.test(k)) { show(parseInt(k, 10) - 1); }
  });

  document.addEventListener('click', function (e) {
    if (e.clientX < window.innerWidth * 0.18) show(cur - 1);
    else show(cur + 1);
  });

  /* ---- titik awal ---- */
  var start = 0;
  var h = (location.hash || '').replace('#', '');
  if (/^[0-9]+$/.test(h)) start = parseInt(h, 10) - 1;
  else {
    try {
      var lv = localStorage.getItem('pk_deck_slide');
      if (lv !== null) start = parseInt(lv, 10);
    } catch (e) {}
  }
  show(start, true);
})();
