(function () {
  var script = document.currentScript;
  if (!script) return;

  var token = script.getAttribute('data-token');
  if (!token) return;

  var src = script.src || '';
  var endpoint = src
    ? src.replace(/\/collect\.js(\?.*)?$/, '') + '/api/collect'
    : '/api/collect';

  var KEY_VID = 'yu_vid';
  var KEY_SID = 'yu_sid';
  var KEY_SEEN = 'yu_sid_seen';
  var SESSION_IDLE = 30 * 60 * 1000;

  function read(key) {
    try { return localStorage.getItem(key); } catch (e) { return null; }
  }

  function write(key, value) {
    try { localStorage.setItem(key, value); } catch (e) {}
  }

  function uuid() {
    if (window.crypto && crypto.randomUUID) return crypto.randomUUID();

    var bytes = new Uint8Array(16);
    if (window.crypto && crypto.getRandomValues) {
      crypto.getRandomValues(bytes);
    } else {
      for (var i = 0; i < 16; i++) bytes[i] = Math.floor(Math.random() * 256);
    }
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;

    var hex = [];
    for (var j = 0; j < 16; j++) hex.push((bytes[j] + 0x100).toString(16).slice(1));

    return hex.slice(0, 4).join('') + '-' + hex.slice(4, 6).join('') + '-' +
      hex.slice(6, 8).join('') + '-' + hex.slice(8, 10).join('') + '-' +
      hex.slice(10).join('');
  }

  var visitorID = read(KEY_VID);
  if (!visitorID) {
    visitorID = uuid();
    write(KEY_VID, visitorID);
  }

  function sessionID() {
    var now = Date.now();
    var sid = read(KEY_SID);
    var seen = parseInt(read(KEY_SEEN) || '0', 10);

    if (!sid || !seen || now - seen > SESSION_IDLE) {
      sid = uuid();
      write(KEY_SID, sid);
    }

    write(KEY_SEEN, String(now));
    return sid;
  }

  function payload(type, extra) {
    var data = {
      t: token,
      type: type,
      vid: visitorID,
      sid: sessionID(),
      url: location.href,
      ref: document.referrer || '',
      title: document.title || '',
      screen: (window.screen.width || 0) + 'x' + (window.screen.height || 0),
      lang: navigator.language || ''
    };

    if (extra) {
      for (var key in extra) {
        if (Object.prototype.hasOwnProperty.call(extra, key)) data[key] = extra[key];
      }
    }

    return JSON.stringify(data);
  }

  function send(type, extra) {
    var body = payload(type, extra);

    if (navigator.sendBeacon) {
      try {
        var blob = new Blob([body], { type: 'text/plain' });
        if (navigator.sendBeacon(endpoint, blob)) return;
      } catch (e) {}
    }

    try {
      fetch(endpoint, {
        method: 'POST',
        body: body,
        headers: { 'Content-Type': 'text/plain' },
        keepalive: true,
        credentials: 'omit'
      });
    } catch (e) {}
  }

  var lastURL = '';
  var lastSearch = '';
  var scrollMarks = {};
  var lastScrollAt = 0;
  var pendingTimer = null;

  var DOWNLOAD_EXT = {
    pdf: 1, zip: 1, rar: 1, '7z': 1, tar: 1, gz: 1,
    doc: 1, docx: 1, xls: 1, xlsx: 1, ppt: 1, pptx: 1,
    csv: 1, txt: 1, mp3: 1, mp4: 1, dmg: 1, exe: 1, apk: 1
  };

  function currentURL() {
    return location.pathname + location.search;
  }

  function reportSiteSearch() {
    var params = new URLSearchParams(location.search);
    var keys = ['q', 'query', 's', 'search', 'keyword'];
    var term = '';

    for (var i = 0; i < keys.length; i++) {
      var value = params.get(keys[i]);
      if (value && value.trim()) {
        term = value.trim();
        break;
      }
    }

    if (!term) return;

    var key = location.pathname + '|' + term;
    if (key === lastSearch) return;
    lastSearch = key;
    send('custom', { name: 'site_search', props: { term: term } });
  }

  function reportPageview(force) {
    var current = currentURL();
    if (!force && current === lastURL) return;
    lastURL = current;
    scrollMarks = {};
    send('pageview');
    reportSiteSearch();
  }

  // 延迟到本轮渲染结束再上报：SvelteKit 等框架的路由与 <svelte:head>
  // 在 history 变更后的微任务里更新，立即读取会拿到上一页的标题。
  function schedulePageview() {
    if (pendingTimer) return;
    pendingTimer = setTimeout(function () {
      pendingTimer = null;
      reportPageview(false);
    }, 0);
  }

  schedulePageview();

  var pushState = history.pushState;
  history.pushState = function () {
    pushState.apply(history, arguments);
    schedulePageview();
  };

  var replaceState = history.replaceState;
  history.replaceState = function () {
    replaceState.apply(history, arguments);
    schedulePageview();
  };

  window.addEventListener('popstate', schedulePageview);

  document.addEventListener('visibilitychange', function () {
    if (document.visibilityState === 'hidden') send('ping');
  });

  document.addEventListener('click', function (event) {
    var target = event.target;
    var anchor = target && target.closest ? target.closest('a[href]') : null;
    if (!anchor) return;

    var href = anchor.getAttribute('href') || '';
    if (!href || href.charAt(0) === '#') return;

    var url;
    try {
      url = new URL(anchor.href, location.href);
    } catch (e) {
      return;
    }

    if (url.protocol !== 'http:' && url.protocol !== 'https:') return;

    var fileName = url.pathname.split('/').pop() || '';
    var ext = fileName.indexOf('.') >= 0 ? fileName.split('.').pop().toLowerCase() : '';

    if (DOWNLOAD_EXT[ext]) {
      send('custom', {
        name: 'file_download',
        props: { url: url.href, file: fileName, ext: ext }
      });
      return;
    }

    if (url.hostname !== location.hostname) {
      send('custom', {
        name: 'outbound_click',
        props: { url: url.href, host: url.hostname }
      });
    }
  }, true);

  window.addEventListener('scroll', function () {
    var now = Date.now();
    if (now - lastScrollAt < 500) return;
    lastScrollAt = now;

    var doc = document.documentElement;
    var body = document.body;
    var height = Math.max(doc.scrollHeight, body ? body.scrollHeight : 0) - window.innerHeight;
    if (height <= 0) return;

    var top = window.scrollY || doc.scrollTop || 0;
    var percent = Math.min(100, Math.max(0, Math.round((top / height) * 100)));

    [25, 50, 75, 100].forEach(function (mark) {
      if (percent >= mark && !scrollMarks[mark]) {
        scrollMarks[mark] = true;
        send('custom', { name: 'scroll_depth', props: { percent: mark } });
      }
    });
  }, { passive: true });

  window.yu = {
    // 手动补报一次页面浏览（自动去重），适合在框架的 afterNavigate 中调用。
    pageview: function () {
      reportPageview(false);
    },
    event: function (name, props) {
      if (!name) return;
      send('custom', { name: String(name), props: props || {} });
    }
  };
})();
