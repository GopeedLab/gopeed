import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

import '../../core/common/start_config.dart';
import '../../util/log_util.dart';
import '../../util/util.dart';
import 'server.dart';

// Keep the execute wrapper syntax conservative for cross-platform WebView
// compatibility. WKWebView is the strictest target, so the shared wrapper
// follows its constraints.
String buildWebViewExecuteScript({
  required String channelName,
  required String requestId,
  required String expression,
  required List<dynamic> args,
}) {
  return '''
(() => {
  var __channel = window[${jsonEncode(channelName)}];
  var __id = ${jsonEncode(requestId)};
  var __expr = ${jsonEncode(expression)};
  var __args = ${jsonEncode(args)};
  var __serializeError = function(error) {
    return [
      error && error.name ? error.name : 'Error',
      error && error.message ? error.message : String(error),
      error && error.stack ? error.stack : ''
    ].filter(Boolean).join(': ').replace(': @', '\\n@');
  };
  var __post = function(message) {
    if (__channel && typeof __channel.postMessage === 'function') {
      __channel.postMessage(JSON.stringify(message));
    }
  };
  try {
    Promise.resolve()
      .then(function() {
        var __target = (0, eval)(__expr);
        if (typeof __target === 'function') {
          return __target.apply(null, __args || []);
        }
        return __target;
      })
      .then(function(__value) {
        __post({ id: __id, ok: true, value: __value == null ? null : __value });
      })
      .catch(function(error) {
        __post({
          id: __id,
          ok: false,
          error: __serializeError(error)
        });
      });
  } catch (error) {
    __post({
      id: __id,
      ok: false,
      error: __serializeError(error)
    });
  }
})();
''';
}

class WebViewRpcService {
  WebViewRpcService._();

  static final WebViewRpcService instance = WebViewRpcService._();

  final ValueNotifier<List<WebViewRpcPageSession>> pages =
      ValueNotifier<List<WebViewRpcPageSession>>([]);

  final Map<String, WebViewRpcPageSession> _pagesById = {};
  final Completer<void> _overlayReady = Completer<void>();
  final WebViewCookieManager _cookieManager = WebViewCookieManager();

  RpcServerHandle? _server;
  int _pageSeq = 0;

  bool get supported => Util.isMobile() || Util.isMacos();

  void markOverlayReady() {
    if (!_overlayReady.isCompleted) {
      _overlayReady.complete();
    }
  }

  Future<WebViewRpcConfig?> start() async {
    if (!supported) {
      return null;
    }
    if (_server != null) {
      return _toConfig(_server!.binding);
    }
    final binding = await defaultWebViewRpcBinding();
    _server = await startRpcServer(
      binding: binding,
      routes: {
        '/webview': _handleWebViewRequest,
      },
    );
    return _toConfig(_server!.binding);
  }

  Future<void> stop() async {
    for (final page in _pagesById.values.toList()) {
      await page.dispose();
    }
    _pagesById.clear();
    pages.value = const [];
    final server = _server;
    _server = null;
    if (server != null) {
      await server.close();
    }
  }

  Future<void> _handleWebViewRequest(RpcContext ctx) async {
    await _overlayReady.future;
    final body = await ctx.readJSON();
    final method = body['method'] as String? ?? '';
    final params = (body['params'] as Map?)?.cast<String, dynamic>() ??
        const <String, dynamic>{};
    try {
      final result = await _dispatch(method, params);
      await ctx.writeJSON({
        'result': result,
        'error': null,
      });
    } catch (e, stackTrace) {
      logger.w('webview rpc request failed: $method', e, stackTrace);
      await ctx.writeJSON({
        'result': null,
        'error': _toRpcError(method, e),
      });
    }
  }

  Future<dynamic> _dispatch(String method, Map<String, dynamic> params) async {
    switch (method) {
      case 'webview.isAvailable':
        return {'available': true};
      case 'page.open':
        return _openPage(params);
      case 'page.addInitScript':
        return _page(params).addInitScript(_string(params, 'script'));
      case 'page.navigate':
        await _page(params).navigate(
          _string(params, 'url'),
          timeoutMs: _int(params, 'timeoutMs'),
        );
        return {};
      case 'page.execute':
        return _page(params).execute(
          _string(params, 'expression'),
          (_list(params, 'args') ?? const []),
        );
      case 'page.getCookies':
        return _page(params).getCookies();
      case 'page.setCookie':
        await _page(params).setCookie(_cookie(params));
        return {};
      case 'page.deleteCookie':
        await _page(params).deleteCookie(_cookie(params));
        return {};
      case 'page.clearCookies':
        await _page(params).clearCookies();
        return {};
      case 'page.close':
        await _closePage(_pageId(params));
        return {};
      default:
        throw WebViewRpcException(
          code: 'UNKNOWN_METHOD',
          message: 'unknown method: $method',
        );
    }
  }

  Future<Map<String, dynamic>> _openPage(Map<String, dynamic> params) async {
    final pageId = 'page-${++_pageSeq}';
    final session = WebViewRpcPageSession(
      pageId: pageId,
      cookieManager: _cookieManager,
      headless: params['headless'] as bool? ?? false,
      debug: params['debug'] as bool? ?? false,
      title: params['title'] as String? ?? '',
      width: (params['width'] as num?)?.toInt() ?? 1280,
      height: (params['height'] as num?)?.toInt() ?? 800,
      userAgent: params['userAgent'] as String? ?? '',
    );
    await session.init();
    _pagesById[pageId] = session;
    pages.value = _pagesById.values.toList(growable: false);
    return {'pageId': pageId};
  }

  WebViewRpcPageSession _page(Map<String, dynamic> params) {
    final page = _pagesById[_pageId(params)];
    if (page == null) {
      throw WebViewRpcException(
        code: 'PAGE_NOT_FOUND',
        message: 'page not found',
      );
    }
    return page;
  }

  String _pageId(Map<String, dynamic> params) => _string(params, 'pageId');

  String _string(Map<String, dynamic> params, String key) {
    final value = params[key];
    if (value is String && value.isNotEmpty) {
      return value;
    }
    throw WebViewRpcException(
      code: 'INVALID_REQUEST',
      message: 'missing or invalid "$key"',
    );
  }

  int? _int(Map<String, dynamic> params, String key) {
    final value = params[key];
    if (value == null) {
      return null;
    }
    if (value is num) {
      return value.toInt();
    }
    throw WebViewRpcException(
      code: 'INVALID_REQUEST',
      message: 'invalid "$key"',
    );
  }

  List<dynamic>? _list(Map<String, dynamic> params, String key) {
    final value = params[key];
    if (value == null) {
      return null;
    }
    if (value is List<dynamic>) {
      return value;
    }
    throw WebViewRpcException(
      code: 'INVALID_REQUEST',
      message: 'invalid "$key"',
    );
  }

  Map<String, dynamic> _cookie(Map<String, dynamic> params) {
    final value = params['cookie'];
    if (value is Map<String, dynamic>) {
      return value;
    }
    if (value is Map) {
      return value.cast<String, dynamic>();
    }
    throw WebViewRpcException(
      code: 'INVALID_REQUEST',
      message: 'missing or invalid "cookie"',
    );
  }

  Future<void> _closePage(String pageId) async {
    final page = _pagesById.remove(pageId);
    if (page == null) {
      throw WebViewRpcException(
        code: 'PAGE_NOT_FOUND',
        message: 'page not found',
      );
    }
    pages.value = _pagesById.values.toList(growable: false);
    await page.dispose();
  }

  WebViewRpcConfig _toConfig(RpcBinding binding) {
    final config = WebViewRpcConfig()
      ..network = binding.network
      ..address = binding.address;
    return config;
  }

  Map<String, dynamic> _toRpcError(String method, Object error) {
    if (error is WebViewRpcException) {
      return {
        'code': error.code,
        'message': error.message,
      };
    }
    final code = switch (method) {
      'page.navigate' => 'NAVIGATION_FAILED',
      'page.execute' => 'EVALUATION_FAILED',
      _ => 'INTERNAL_ERROR',
    };
    return {
      'code': code,
      'message': error.toString(),
    };
  }
}

class WebViewRpcPageSession {
  WebViewRpcPageSession({
    required this.pageId,
    required this.cookieManager,
    required this.headless,
    required this.debug,
    required this.title,
    required this.width,
    required this.height,
    required this.userAgent,
  });

  final String pageId;
  final WebViewCookieManager cookieManager;
  final bool headless;
  final bool debug;
  final String title;
  final int width;
  final int height;
  final String userAgent;
  late final String callbackChannelName;

  final List<String> _initScripts = [];
  final Map<String, Completer<dynamic>> _pendingExecutions =
      <String, Completer<dynamic>>{};

  late final WebViewController controller;
  Completer<void>? _navigation;
  String _currentUrl = '';
  int _executeSeq = 0;

  Future<void> init() async {
    callbackChannelName =
        '__gopeedWebViewCallback_${pageId.replaceAll('-', '_')}';
    controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setNavigationDelegate(
        NavigationDelegate(
          onPageStarted: (url) {
            _currentUrl = url;
          },
          onPageFinished: (url) async {
            _currentUrl = url;
            await _runInitScripts();
            _navigation?.complete();
            _navigation = null;
          },
          onWebResourceError: (error) {
            _navigation?.completeError(
              WebViewRpcException(
                code: 'NAVIGATION_FAILED',
                message: error.description,
              ),
            );
            _navigation = null;
          },
        ),
      );
    await controller.addJavaScriptChannel(
      callbackChannelName,
      onMessageReceived: (message) {
        _handleExecuteMessage(message.message);
      },
    );
    if (!Util.isMacos()) {
      controller.setBackgroundColor(const Color(0x00000000));
    }
    if (userAgent.isNotEmpty) {
      await controller.setUserAgent(userAgent);
    }
  }

  Future<void> addInitScript(String script) async {
    _initScripts.add(script);
    if (_currentUrl.isNotEmpty) {
      try {
        await controller.runJavaScript(script);
      } catch (_) {
        // Best-effort only: webview_flutter does not expose a true
        // evaluate-on-new-document hook. Future navigations still re-run it.
      }
    }
  }

  Future<void> navigate(String url, {int? timeoutMs}) async {
    _navigation = Completer<void>();
    try {
      await controller.loadRequest(Uri.parse(url));
    } catch (e) {
      _navigation = null;
      throw WebViewRpcException(
        code: 'NAVIGATION_FAILED',
        message: e.toString(),
      );
    }
    final future = _navigation!.future;
    if (timeoutMs == null || timeoutMs <= 0) {
      await future;
      return;
    }
    await future.timeout(
      Duration(milliseconds: timeoutMs),
      onTimeout: () {
        _navigation = null;
        throw WebViewRpcException(
          code: 'TIMEOUT',
          message: 'navigation timeout after ${timeoutMs}ms',
        );
      },
    );
  }

  Future<dynamic> execute(String expression, List<dynamic> args) async {
    final requestId = 'exec-$pageId-${++_executeSeq}';
    final completer = Completer<dynamic>();
    _pendingExecutions[requestId] = completer;

    try {
      await controller.runJavaScript(
        buildWebViewExecuteScript(
          channelName: callbackChannelName,
          requestId: requestId,
          expression: expression,
          args: args,
        ),
      );
    } catch (e) {
      _pendingExecutions.remove(requestId);
      throw WebViewRpcException(
        code: 'EVALUATION_FAILED',
        message: e.toString(),
      );
    }

    try {
      final result = await completer.future.timeout(
        const Duration(seconds: 30),
        onTimeout: () {
          _pendingExecutions.remove(requestId);
          throw WebViewRpcException(
            code: 'TIMEOUT',
            message: 'javascript execution timeout',
          );
        },
      );
      return result;
    } finally {
      _pendingExecutions.remove(requestId);
    }
  }

  Future<List<Map<String, dynamic>>> getCookies() async {
    final raw = await execute('() => document.cookie', const []);
    final cookieText = raw?.toString() ?? '';
    if (cookieText.isEmpty) {
      return const [];
    }
    final uri = _currentUrl.isEmpty ? null : Uri.tryParse(_currentUrl);
    return cookieText
        .split(';')
        .map((entry) => entry.trim())
        .where((entry) => entry.isNotEmpty)
        .map((entry) {
      final separator = entry.indexOf('=');
      final name = separator >= 0 ? entry.substring(0, separator) : entry;
      final value = separator >= 0 ? entry.substring(separator + 1) : '';
      return <String, dynamic>{
        'name': name,
        'value': value,
        'domain': uri?.host ?? '',
        'path': '/',
      };
    }).toList(growable: false);
  }

  Future<void> setCookie(Map<String, dynamic> cookie) async {
    await cookieManager.setCookie(
      WebViewCookie(
        name: cookie['name'] as String? ?? '',
        value: cookie['value'] as String? ?? '',
        domain: cookie['domain'] as String? ?? '',
        path: cookie['path'] as String? ?? '/',
      ),
    );
  }

  Future<void> deleteCookie(Map<String, dynamic> cookie) async {
    final name = cookie['name'] as String? ?? '';
    final path = cookie['path'] as String? ?? '/';
    final domain = cookie['domain'] as String? ?? '';
    if (name.isEmpty) {
      throw WebViewRpcException(
        code: 'INVALID_REQUEST',
        message: 'cookie.name is required',
      );
    }
    final cookieDomain = domain.isNotEmpty ? '; domain=$domain' : '';
    final cookiePath = '; path=$path';
    const expires = '; expires=Thu, 01 Jan 1970 00:00:00 GMT';
    await controller.runJavaScript(
      'document.cookie = ${jsonEncode('$name=$expires$cookiePath$cookieDomain')}',
    );
  }

  Future<void> clearCookies() async {
    await cookieManager.clearCookies();
  }

  Future<void> dispose() async {
    for (final completer in _pendingExecutions.values) {
      if (!completer.isCompleted) {
        completer.completeError(
          WebViewRpcException(
            code: 'EVALUATION_FAILED',
            message: 'webview page closed',
          ),
        );
      }
    }
    _pendingExecutions.clear();
  }

  Future<void> _runInitScripts() async {
    for (final script in _initScripts) {
      await controller.runJavaScript(script);
    }
  }

  void _handleExecuteMessage(String payload) {
    final decoded = jsonDecode(payload);
    if (decoded is! Map) {
      return;
    }
    final message = decoded.cast<String, dynamic>();
    final id = message['id']?.toString();
    if (id == null || id.isEmpty) {
      return;
    }
    final completer = _pendingExecutions[id];
    if (completer == null || completer.isCompleted) {
      return;
    }
    if (message['ok'] == true) {
      completer.complete(message['value']);
      return;
    }
    completer.completeError(
      WebViewRpcException(
        code: 'EVALUATION_FAILED',
        message: message['error']?.toString() ?? 'javascript execution failed',
      ),
    );
  }
}

class WebViewRpcException implements Exception {
  final String code;
  final String message;

  WebViewRpcException({
    required this.code,
    required this.message,
  });

  @override
  String toString() => message;
}
