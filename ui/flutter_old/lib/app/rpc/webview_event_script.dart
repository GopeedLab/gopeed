import 'dart:convert';

/// Runs before page scripts and reports only the main document's events.
String buildWebViewEventScript(String channel) => '''
(() => {
  if (window !== window.top) return;
  // Browser error documents are reported by native load-error callbacks.
  if (location.protocol === 'chrome-error:') return;
  const pending = [];
  const flush = () => {
    const bridge = window.flutter_inappwebview;
    if (!bridge || typeof bridge.callHandler !== 'function') return;
    while (pending.length) bridge.callHandler(${jsonEncode(channel)}, JSON.stringify(pending.shift()));
  };
  const send = (event, data) => { pending.push({event, data}); flush(); };
  window.addEventListener('flutterInAppWebViewPlatformReady', flush);
  let lastURL = String(location.href);
  const changed = () => {
    const url = String(location.href);
    if (url === lastURL) return;
    lastURL = url;
    send('url-changed', {url, sameDocument: true});
  };
  send('url-changed', {url: lastURL, sameDocument: false});
  for (const name of ['pushState', 'replaceState']) {
    const original = history[name];
    history[name] = function(...args) {
      const result = Reflect.apply(original, this, args);
      changed();
      return result;
    };
  }
  window.addEventListener('hashchange', changed);
  window.addEventListener('popstate', changed);
  window.addEventListener('pageshow', (event) => {
    if (event.persisted) send('url-changed', {url: String(location.href), sameDocument: false});
    else changed();
  });
  window.addEventListener('load', () => send('load', {url: String(location.href)}), {once: true});
})();
''';
