// Real native smoke test: flutter run -d macos -t tool/webview_profile_check.dart
import 'dart:async';
import 'dart:io';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:gopeed/app/rpc/webview_profile.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const MaterialApp(home: SizedBox()));
  try {
    await checkProfiles();
    stdout.writeln('PASS: native proxy, concurrent profiles, cookies, LocalStorage, IndexedDB and reopening');
    exit(0);
  } catch (error, stack) {
    stderr.writeln('FAIL: $error\n$stack');
    exit(1);
  }
}

void expectValue(dynamic actual, dynamic expected) {
  if (actual != expected) throw StateError('Expected $expected, got $actual');
}

Future<void> checkProfiles() async {
  final context = SecurityContext()
    ..useCertificateChainBytes(utf8.encode(fixtureCertificate))
    ..usePrivateKeyBytes(utf8.encode(fixtureKey));
  final origin = await HttpServer.bindSecure(InternetAddress.loopbackIPv4, 0, context);
  origin.listen((request) async {
    request.response.headers.contentType = ContentType.html;
    request.response.write('<html><body>profile fixture</body></html>');
    await request.response.close();
  });
  final plainOrigin = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
  plainOrigin.listen((request) async {
    request.response.headers.contentType = ContentType.html;
    request.response.write('<html><body>profile fixture</body></html>');
    await request.response.close();
  });
  var proxyHits = 0;
  final proxy = await ServerSocket.bind(InternetAddress.loopbackIPv4, 0);
  proxy.listen((socket) async {
    final input = StreamIterator(socket);
    final buffer = <int>[];
    Future<List<int>> read(int count) async {
      while (buffer.length < count) {
        if (!await input.moveNext()) throw StateError('Incomplete SOCKS handshake');
        buffer.addAll(input.current);
      }
      final value = buffer.sublist(0, count);
      buffer.removeRange(0, count);
      return value;
    }

    try {
      final hello = await read(2);
      await read(hello[1]);
      socket.add([5, 0]);
      await socket.flush();
      final header = await read(4);
      if (header[3] == 1) {
        await read(4);
      } else if (header[3] == 4) {
        await read(16);
      } else {
        final size = await read(1);
        await read(size[0]);
      }
      final port = await read(2);
      final targetPort = port[0] * 256 + port[1];
      proxyHits++;
      final remote = await Socket.connect(
        InternetAddress.loopbackIPv4,
        targetPort == 443 ? origin.port : plainOrigin.port,
      );
      socket.add([5, 0, 0, 1, 127, 0, 0, 1, 0, 0]);
      await socket.flush();
      remote.listen(socket.add, onDone: socket.destroy, onError: (_) => socket.destroy());
      if (buffer.isNotEmpty) remote.add(buffer);
      while (await input.moveNext()) {
        remote.add(input.current);
      }
      remote.destroy();
    } finally {
      await input.cancel();
      socket.destroy();
    }
  });
  final suffix = DateTime.now().microsecondsSinceEpoch.toRadixString(16).padLeft(12, '0');
  final a = WebViewProfile('11111111-2222-3333-4444-${suffix.substring(suffix.length - 12)}');
  final b = WebViewProfile('22222222-2222-3333-4444-${suffix.substring(suffix.length - 12)}');
  final proxyUrl = 'socks5://127.0.0.1:${proxy.port}';
  await a.prepare(proxyUrl);
  await b.prepare(proxyUrl);
  final target = WebUri('https://profiles.invalid/');
  final pages = <HeadlessInAppWebView>[];
  Future<InAppWebViewController> open(WebViewProfile profile) async {
    final ready = Completer<InAppWebViewController>();
    final page = HeadlessInAppWebView(
      onReceivedServerTrustAuthRequest: (_, _) async =>
          ServerTrustAuthResponse(action: ServerTrustAuthResponseAction.PROCEED),
      initialSettings: WebViewProfileSettings(profileId: profile.id, debug: true),
      onWebViewCreated: (controller) {
        controller.loadUrl(urlRequest: URLRequest(url: target));
      },
      onLoadStop: (controller, url) {
        if (url?.host == target.host && !ready.isCompleted) ready.complete(controller);
      },
      onReceivedError: (_, _, error) {
        if (!ready.isCompleted) ready.completeError(StateError(error.description));
      },
    );
    pages.add(page);
    await page.run();
    return ready.future.timeout(const Duration(seconds: 20));
  }

  Future<dynamic> js(InAppWebViewController c, String source) => c.evaluateJavascript(source: source);
  Future<dynamic> database(InAppWebViewController c, bool write) async {
    final result = await c.callAsyncJavaScript(
      functionBody:
          '''
      return await new Promise((resolve,reject)=>{
        const request=indexedDB.open('profile-check',1);
        request.onupgradeneeded=()=>request.result.createObjectStore('state');
        request.onerror=()=>reject(String(request.error));
        request.onsuccess=()=>{
          const db=request.result;
          const tx=db.transaction('state','readwrite');
          const store=tx.objectStore('state');
          const op=${write ? "store.put('saved','owner')" : "store.get('owner')"};
          let value;
          op.onsuccess=()=>value=op.result;
          tx.oncomplete=()=>{db.close();resolve(value===undefined?null:value);};
          tx.onerror=()=>{db.close();reject(String(tx.error));};
        };
      });
    ''',
    );
    if (result?.error != null) throw StateError(result!.error.toString());
    return result?.value;
  }

  try {
    final plainReady = Completer<void>();
    final plain = HeadlessInAppWebView(
      initialSettings: WebViewProfileSettings(profileId: a.id, debug: true),
      initialUrlRequest: URLRequest(url: WebUri('http://profiles.invalid/')),
      onLoadStop: (_, url) {
        if (url?.host == 'profiles.invalid' && !plainReady.isCompleted) plainReady.complete();
      },
      onReceivedError: (_, _, error) {
        if (!plainReady.isCompleted) plainReady.completeError(StateError(error.description));
      },
    );
    pages.add(plain);
    await plain.run();
    await plainReady.future.timeout(const Duration(seconds: 20));
    if (proxyHits == 0) throw StateError('HTTP bypassed proxy');
    final first = await open(a);
    await js(first, "localStorage.setItem('owner','saved')");
    await database(first, true);
    await a.setCookie(
      url: target,
      name: 'session',
      value: 'a',
      domain: 'profiles.invalid',
      isHttpOnly: true,
      expiresDate: DateTime.now().add(const Duration(days: 1)).millisecondsSinceEpoch,
    );
    final second = await open(b);
    expectValue(await js(second, "localStorage.getItem('owner')"), null);
    expectValue(await database(second, false), null);
    expectValue((await b.getCookies(url: target)).length, 0);
    final shared = await open(a);
    expectValue(await js(shared, "localStorage.getItem('owner')"), 'saved');
    expectValue(await database(shared, false), 'saved');
    expectValue((await a.getCookies(url: target)).single.value, 'a');
    await b.deleteAllCookies();
    expectValue((await a.getCookies(url: target)).single.value, 'a');
    for (final page in pages) {
      await page.dispose();
    }
    pages.clear();
    final reopened = await open(a);
    expectValue(await js(reopened, "localStorage.getItem('owner')"), 'saved');
    expectValue((await a.getCookies(url: target)).single.isHttpOnly, true);
    await a.deleteCookie(url: target, name: 'session', domain: 'profiles.invalid');
    expectValue((await a.getCookies(url: target)).length, 0);
    if (proxyHits == 0) throw StateError('Browser bypassed proxy');
  } finally {
    for (final page in pages) {
      await page.dispose();
    }
    await proxy.close();
    await plainOrigin.close(force: true);
    await origin.close(force: true);
  }
}

// Self-signed test fixture; trusted only by the smoke-test callback.
const fixtureCertificate = '''-----BEGIN CERTIFICATE-----
MIIDFzCCAf+gAwIBAgIUTVTmWtXNVthZEKsowr8XXq+WPf0wDQYJKoZIhvcNAQEL
BQAwGzEZMBcGA1UEAwwQcHJvZmlsZXMuaW52YWxpZDAeFw0yNjA5MTMxNjE4NTFa
Fw0yNjA5MTYxNjE4NTFaMBsxGTAXBgNVBAMMEHByb2ZpbGVzLmludmFsaWQwggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDM8AJ8i1fy++T4VjH1pyRo96Hw
7yQnjAKhbh4kD4w145bc1RgmWgJLqSSZaegXKsn5Cw2FKxCjA0TyZCTGq+fkDp45
ERWN2TACMWGyz/7ROuAOVfIo92AnDMpdinn+wn9DdWo8z9UEnzCjs613CLCXmGOQ
9R/rTO2VlTlAM5VJygqOpBrtPzrx1ojJhL3fY9e5yYP0yxJqCf7ukCG/lrhHj5fK
7twD+tJMkkGkEMHqavmmTPlD8TsUYKB2Qf9orRPQIxNZkbtG+DEzckRhUJDBprI/
pKVwG680d/40AZ3NvAjbz8WjmWGj2gBkSC1NdQ1k6a6MFJOPGnKfDepMk4S7AgMB
AAGjUzBRMB0GA1UdDgQWBBTwL0eACgwkw0isYAWpZwp1uqULWjAfBgNVHSMEGDAW
gBTwL0eACgwkw0isYAWpZwp1uqULWjAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3
DQEBCwUAA4IBAQCplXaffZy1mYexyq744NFnIDBlnb3FLbBSG7tJt4SAY1dDysr9
lNd8C5udrPxlmezUmlihzv8BqJcUNA0T9iYX6rOlNmD6xaaHs7akAGxvd8upIuM6
iuKt3lXrE+xhVKGLs1XNYVOC+03H+sI1RxSOIpVPY43zSZXRXd8ADQ/+74KLsjIP
uQ1qUPG59C36Tfmp8UBA4pb3inPyRxcpGAREbXeX1zMuyoOs4c80CsV26Jh+sQFZ
VdSqtQrwzCLxb51D1MzeOakD69CWgM1hMIG12wqKvOz0d7uHLCXgIYLIYbaqgsF3
FdpVXgGYktDl/GfVa2i9RLCZKUGOl2OuPFie
-----END CERTIFICATE-----
''';
const fixtureKey = '''-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDM8AJ8i1fy++T4
VjH1pyRo96Hw7yQnjAKhbh4kD4w145bc1RgmWgJLqSSZaegXKsn5Cw2FKxCjA0Ty
ZCTGq+fkDp45ERWN2TACMWGyz/7ROuAOVfIo92AnDMpdinn+wn9DdWo8z9UEnzCj
s613CLCXmGOQ9R/rTO2VlTlAM5VJygqOpBrtPzrx1ojJhL3fY9e5yYP0yxJqCf7u
kCG/lrhHj5fK7twD+tJMkkGkEMHqavmmTPlD8TsUYKB2Qf9orRPQIxNZkbtG+DEz
ckRhUJDBprI/pKVwG680d/40AZ3NvAjbz8WjmWGj2gBkSC1NdQ1k6a6MFJOPGnKf
DepMk4S7AgMBAAECggEACWGTO5BFXmIZgSSw9MXJ7OQpDZZk2UqXdDxONOhhSrY4
WbOHf3nvUPdVLZCjhyv2Qug2njnycQPdTBs5c766xr3EkGgvzGZ8xBzuk/jheUJw
rzdP9oAcbq0vNkj4fKj6mmPtkQsSfTusRlIbRq+nJclcBW6zH1pOJeLG1CGIW/BG
wskEc9/86434Kmmot14f/nIGHPBPEL7MgsoXtMoM7lm/bOPekxd481SOA9j2h8Wl
wLVyFrtzTqDOeNBCZNBPpQeaWgBg4nSeX8TRUMAxDYhlMu0ENmYKp2LDfw6XXEii
yJsO07JNfcHQNyFPkW/E840KAvj+RzEUkQ4VhczLgQKBgQDncicAlvRWhCD//Jd7
1FFMx8zaz6hKRrbdI0zzpzIle3Ebe7zXL84KMzLSP3pwD/ABwktoEhQjTjvo/VCh
pkvnqOgH8k9QUEyuzVhlhwC5RZOVfz3vnKTJMuCjsQ66OY1ljNVDWWLv5MkbVPlu
wmziIWqdnitN7X0F9oHnnshuiQKBgQDireplA0+aZfQFsiWm/YSaQa/QJUVBkA0N
La+C12O+dCr2n3gJsjVPteqVloJRymwpLtOW+3XWD4U4W7JJh3B+TNdtkgjT2jYF
GK/29B/mspB9aCXuMimq1XAyFC27vSR6PhiYPdQJ+dWtACigRXTZvjTdC7EvxSsZ
MNqjdO8oIwKBgHsXbATVQ/frZ72dcldqUR7buGi7Pk7akxDDYH7JclMa9sneIk3z
38nu9t144z5MS6Iz2nTsAIfSaOx+JO8ECaSYYWcwiNw3CMGC9rtwdIMUrOw6cw9H
qSBUjcKhPSrvxvK2VwnttT/O1uJRbCNrBguyKAjSAUf9wZt1QhuUiBa5AoGAc18p
Sbhhr6fsh2mgmFm4P/rmzP3rPe9fdTGnfuS5s5nUtCl+IxE86REiEHjGY3KvklZv
Aw2BcC3+FC03lQ0GSII7s1z8eTc6/2UNSpf+FKEVwX5cr9uAMZ7ot9RlYoRmKIGq
avItFda+1oqaIti3HIwew9LLoBCuWufun/tILhkCgYACBuavVB5q0JW520hb5KeA
ScD4+OLyxfKWYPvBNaIZLBP/lWuXQmxoXHvPRbxNjGQTldj1SbGS6yD5yvHs17vb
cFF+y2NS4hTD6VMMnl5sZT9OQjEEgj5/nnq+TlcHljpQ3q+OQ7rtJJFt832ZAXLd
ts2luZQYoJ+6aosodHGjCA==
-----END PRIVATE KEY-----
''';
