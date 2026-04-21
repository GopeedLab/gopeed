import 'dart:async';
import 'dart:collection';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';

import '../../core/common/start_config.dart';
import '../../util/log_util.dart';
import '../../util/util.dart';
import 'server.dart';

String buildWebViewExecuteScript({
  required String channelName,
  required String requestId,
  required String expression,
  required List<dynamic> args,
}) {
  return '''
(() => {
  var __bridge = window.flutter_inappwebview;
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
    if (__bridge && typeof __bridge.callHandler === 'function') {
      __bridge.callHandler(${jsonEncode(channelName)}, JSON.stringify(message));
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
  final CookieManager _cookieManager = CookieManager.instance();

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
      case 'page.goto':
        await _page(params).goto(
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
    _pagesById[pageId] = session;
    _syncPages();
    try {
      await session.init();
    } catch (e) {
      _pagesById.remove(pageId);
      _syncPages();
      rethrow;
    }
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
    _syncPages();
    await page.dispose();
  }

  void _syncPages() {
    pages.value = _pagesById.values
        .where((page) => !page.headless)
        .toList(growable: false);
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
      'page.goto' => 'NAVIGATION_FAILED',
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
  final CookieManager cookieManager;
  final bool headless;
  final bool debug;
  final String title;
  final int width;
  final int height;
  final String userAgent;
  late final String callbackChannelName;

  final List<UserScript> _initScripts = [];
  final Map<String, Completer<dynamic>> _pendingExecutions =
      <String, Completer<dynamic>>{};

  final Completer<void> _ready = Completer<void>();
  HeadlessInAppWebView? _headlessWebView;
  InAppWebViewController? _controller;
  Completer<void>? _navigation;
  String _currentUrl = '';
  int _executeSeq = 0;
  bool _disposed = false;

  Future<void> init() async {
    callbackChannelName =
        '__gopeedWebViewCallback_${pageId.replaceAll('-', '_')}';
    if (headless) {
      _headlessWebView = HeadlessInAppWebView(
        initialSize: Size(
          width > 0 ? width.toDouble() : 1,
          height > 0 ? height.toDouble() : 1,
        ),
        initialSettings: _settings,
        initialUserScripts: UnmodifiableListView(_initScripts),
        onWebViewCreated: _attachController,
        onLoadStart: _handleLoadStart,
        onLoadStop: _handleLoadStop,
        onReceivedError: _handleReceivedError,
        onReceivedHttpError: _handleReceivedHttpError,
      );
      await _headlessWebView!.run();
    }
    await _ready.future;
  }

  Future<void> addInitScript(String script) async {
    final userScript = UserScript(
      source: script,
      injectionTime: UserScriptInjectionTime.AT_DOCUMENT_START,
    );
    _initScripts.add(userScript);
    final controller = await _controllerOrThrow();
    await controller.addUserScript(userScript: userScript);
    if (_currentUrl.isNotEmpty && !_disposed) {
      try {
        await controller.evaluateJavascript(source: script);
      } catch (_) {
        // Best-effort only for the current page. The user script persists
        // for future navigations through the native user script registry.
      }
    }
  }

  Future<void> goto(String url, {int? timeoutMs}) async {
    final controller = await _controllerOrThrow();
    final navigation = _ensureNavigationCompleter();
    try {
      await controller.loadUrl(urlRequest: URLRequest(url: WebUri(url)));
    } catch (e) {
      if (identical(_navigation, navigation)) {
        _navigation = null;
      }
      throw WebViewRpcException(
        code: 'NAVIGATION_FAILED',
        message: e.toString(),
      );
    }
    final future = navigation.future;
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
    final controller = await _controllerOrThrow();
    final requestId = 'exec-$pageId-${++_executeSeq}';
    final completer = Completer<dynamic>();
    _pendingExecutions[requestId] = completer;

    try {
      await controller.evaluateJavascript(
        source: buildWebViewExecuteScript(
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
    if (_currentUrl.isEmpty) {
      return const [];
    }
    final cookies = await cookieManager.getCookies(url: WebUri(_currentUrl));
    return cookies.map((cookie) {
      return <String, dynamic>{
        'name': cookie.name,
        'value': cookie.value,
        'domain': cookie.domain ?? '',
        'path': cookie.path ?? '/',
        'secure': cookie.isSecure,
        'httpOnly': cookie.isHttpOnly,
      };
    }).toList(growable: false);
  }

  Future<void> setCookie(Map<String, dynamic> cookie) async {
    final targetUrl = _cookieUrl(cookie);
    await cookieManager.setCookie(
      url: targetUrl,
      name: cookie['name'] as String? ?? '',
      value: cookie['value'] as String? ?? '',
      domain: cookie['domain'] as String?,
      path: cookie['path'] as String? ?? '/',
      expiresDate: _cookieExpires(cookie),
      isSecure: cookie['secure'] as bool? ?? false,
      isHttpOnly: cookie['httpOnly'] as bool? ?? false,
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
    await cookieManager.deleteCookie(
      url: _cookieUrl(cookie),
      name: name,
      domain: domain.isNotEmpty ? domain : null,
      path: path,
    );
  }

  Future<void> clearCookies() async {
    await cookieManager.deleteAllCookies();
  }

  Future<void> dispose() async {
    _disposed = true;
    if (!_ready.isCompleted) {
      _ready.completeError(
        WebViewRpcException(
          code: 'PAGE_NOT_FOUND',
          message: 'webview page closed',
        ),
      );
    }
    if (_navigation != null) {
      _completeNavigationError(
        WebViewRpcException(
          code: 'NAVIGATION_FAILED',
          message: 'webview page closed',
        ),
      );
    }
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
    await Future.sync(() => _headlessWebView?.dispose());
    _headlessWebView = null;
    _controller = null;
  }

  Completer<void> _ensureNavigationCompleter() {
    return _navigation ??= Completer<void>();
  }

  Widget buildWebView() {
    return InAppWebView(
      key: ValueKey<String>('webview-$pageId'),
      initialSettings: _settings,
      initialUserScripts: UnmodifiableListView(_initScripts),
      onWebViewCreated: _attachController,
      onLoadStart: _handleLoadStart,
      onLoadStop: _handleLoadStop,
      onReceivedError: _handleReceivedError,
      onReceivedHttpError: _handleReceivedHttpError,
    );
  }

  InAppWebViewSettings get _settings => InAppWebViewSettings(
        javaScriptEnabled: true,
        transparentBackground: true,
        isInspectable: debug,
        userAgent: userAgent.isNotEmpty ? userAgent : null,
      );

  Future<InAppWebViewController> _controllerOrThrow() async {
    await _ready.future;
    final controller = _controller;
    if (controller != null && !_disposed) {
      return controller;
    }
    throw WebViewRpcException(
      code: 'PAGE_NOT_FOUND',
      message: 'page not found',
    );
  }

  void _attachController(InAppWebViewController controller) {
    _controller = controller;
    controller.addJavaScriptHandler(
      handlerName: callbackChannelName,
      callback: (args) {
        if (args.isNotEmpty) {
          _handleExecuteMessage(args.first?.toString() ?? '');
        }
        return null;
      },
    );
    if (!_ready.isCompleted) {
      _ready.complete();
    }
  }

  void _handleLoadStart(InAppWebViewController controller, WebUri? url) {
    _currentUrl = url?.toString() ?? _currentUrl;
    _ensureNavigationCompleter();
  }

  Future<void> _handleLoadStop(
    InAppWebViewController controller,
    WebUri? url,
  ) async {
    _currentUrl = url?.toString() ?? _currentUrl;
    _completeNavigation();
  }

  void _handleReceivedError(
    InAppWebViewController controller,
    WebResourceRequest request,
    WebResourceError error,
  ) {
    _completeNavigationError(
      WebViewRpcException(
        code: 'NAVIGATION_FAILED',
        message: error.description,
      ),
    );
  }

  void _handleReceivedHttpError(
    InAppWebViewController controller,
    WebResourceRequest request,
    WebResourceResponse errorResponse,
  ) {
    _completeNavigationError(
      WebViewRpcException(
        code: 'NAVIGATION_FAILED',
        message:
            'HTTP ${errorResponse.statusCode}: ${errorResponse.reasonPhrase ?? 'navigation failed'}',
      ),
    );
  }

  void _completeNavigation() {
    if (_navigation != null && !_navigation!.isCompleted) {
      _navigation!.complete();
    }
    _navigation = null;
  }

  void _completeNavigationError(Object error) {
    if (_navigation != null && !_navigation!.isCompleted) {
      _navigation!.completeError(error);
    }
    _navigation = null;
  }

  WebUri _cookieUrl(Map<String, dynamic> cookie) {
    final domain = (cookie['domain'] as String? ?? '').trim();
    final path = cookie['path'] as String? ?? '/';
    if (domain.isNotEmpty) {
      final host = domain.startsWith('.') ? domain.substring(1) : domain;
      final scheme = (cookie['secure'] as bool? ?? false) ? 'https' : 'http';
      return WebUri('$scheme://$host$path');
    }
    if (_currentUrl.isNotEmpty) {
      return WebUri(_currentUrl);
    }
    throw WebViewRpcException(
      code: 'INVALID_REQUEST',
      message: 'cookie target url is unavailable',
    );
  }

  int? _cookieExpires(Map<String, dynamic> cookie) {
    final raw = cookie['expires'];
    if (raw is int) {
      return raw;
    }
    if (raw is String && raw.isNotEmpty) {
      return DateTime.tryParse(raw)?.millisecondsSinceEpoch;
    }
    return null;
  }

  void _handleExecuteMessage(String payload) {
    if (payload.isEmpty) {
      return;
    }
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
