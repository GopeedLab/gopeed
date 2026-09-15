import 'dart:io';
import 'dart:collection';
import 'dart:convert';
import 'dart:core';
import 'dart:developer' as developer;
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

import '../web_message/main.dart';

import '../in_app_browser/in_app_browser.dart';
import '../web_storage/web_storage.dart';

import 'headless_in_app_webview.dart';
import '_static_channel.dart';

import '../print_job/main.dart';

///List of forbidden names for JavaScript handlers.
// ignore: non_constant_identifier_names
final _JAVASCRIPT_HANDLER_FORBIDDEN_NAMES = UnmodifiableListView<String>([
  "onLoadResource",
  "shouldInterceptAjaxRequest",
  "onAjaxReadyStateChange",
  "onAjaxProgress",
  "shouldInterceptFetchRequest",
  "onPrintRequest",
  "onWindowFocus",
  "onWindowBlur",
  "callAsyncJavaScript",
  "evaluateJavaScriptWithContentWorld"
]);

/// Object specifying creation parameters for creating a [AndroidInAppWebViewController].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformInAppWebViewControllerCreationParams] for
/// more information.
@immutable
class AndroidInAppWebViewControllerCreationParams
    extends PlatformInAppWebViewControllerCreationParams {
  /// Creates a new [AndroidInAppWebViewControllerCreationParams] instance.
  const AndroidInAppWebViewControllerCreationParams(
      {required super.id, super.webviewParams});

  /// Creates a [AndroidInAppWebViewControllerCreationParams] instance based on [PlatformInAppWebViewControllerCreationParams].
  factory AndroidInAppWebViewControllerCreationParams.fromPlatformInAppWebViewControllerCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformInAppWebViewControllerCreationParams params) {
    return AndroidInAppWebViewControllerCreationParams(
        id: params.id, webviewParams: params.webviewParams);
  }
}

///Controls a WebView, such as an [InAppWebView] widget instance, a [AndroidHeadlessInAppWebView] instance or [AndroidInAppBrowser] WebView instance.
///
///If you are using the [InAppWebView] widget, an [InAppWebViewController] instance can be obtained by setting the [InAppWebView.onWebViewCreated]
///callback. Instead, if you are using an [AndroidInAppBrowser] instance, you can get it through the [AndroidInAppBrowser.webViewController] attribute.
class AndroidInAppWebViewController extends PlatformInAppWebViewController
    with ChannelController {
  static final MethodChannel _staticChannel = IN_APP_WEBVIEW_STATIC_CHANNEL;

  // List of properties to be saved and restored for keep alive feature
  Map<String, JavaScriptHandlerCallback> _javaScriptHandlersMap =
      HashMap<String, JavaScriptHandlerCallback>();
  Map<UserScriptInjectionTime, List<UserScript>> _userScripts = {
    UserScriptInjectionTime.AT_DOCUMENT_START: <UserScript>[],
    UserScriptInjectionTime.AT_DOCUMENT_END: <UserScript>[]
  };
  Set<String> _webMessageListenerObjNames = Set();
  Map<String, ScriptHtmlTagAttributes> _injectedScriptsFromURL = {};
  Set<AndroidWebMessageChannel> _webMessageChannels = Set();
  Set<AndroidWebMessageListener> _webMessageListeners = Set();

  // static map that contains the properties to be saved and restored for keep alive feature
  static final Map<InAppWebViewKeepAlive, InAppWebViewControllerKeepAliveProps?>
      _keepAliveMap = {};

  AndroidInAppBrowser? _inAppBrowser;

  PlatformInAppBrowserEvents? get _inAppBrowserEventHandler =>
      _inAppBrowser?.eventHandler;

  dynamic _controllerFromPlatform;

  @override
  late AndroidWebStorage webStorage;

  AndroidInAppWebViewController(
      PlatformInAppWebViewControllerCreationParams params)
      : super.implementation(params
                is AndroidInAppWebViewControllerCreationParams
            ? params
            : AndroidInAppWebViewControllerCreationParams
                .fromPlatformInAppWebViewControllerCreationParams(params)) {
    channel = MethodChannel('com.pichillilorenzo/flutter_inappwebview_$id');
    handler = handleMethod;
    initMethodCallHandler();

    final initialUserScripts = webviewParams?.initialUserScripts;
    if (initialUserScripts != null) {
      for (final userScript in initialUserScripts) {
        if (userScript.injectionTime ==
            UserScriptInjectionTime.AT_DOCUMENT_START) {
          this
              ._userScripts[UserScriptInjectionTime.AT_DOCUMENT_START]
              ?.add(userScript);
        } else {
          this
              ._userScripts[UserScriptInjectionTime.AT_DOCUMENT_END]
              ?.add(userScript);
        }
      }
    }

    this._init(params);
  }

  static final AndroidInAppWebViewController _staticValue =
      AndroidInAppWebViewController(
          AndroidInAppWebViewControllerCreationParams(id: null));

  factory AndroidInAppWebViewController.static() {
    return _staticValue;
  }

  AndroidInAppWebViewController.fromInAppBrowser(
      PlatformInAppWebViewControllerCreationParams params,
      MethodChannel channel,
      AndroidInAppBrowser inAppBrowser,
      UnmodifiableListView<UserScript>? initialUserScripts)
      : super.implementation(
            params is AndroidInAppWebViewControllerCreationParams
                ? params
                : AndroidInAppWebViewControllerCreationParams
                    .fromPlatformInAppWebViewControllerCreationParams(params)) {
    this.channel = channel;
    this._inAppBrowser = inAppBrowser;

    if (initialUserScripts != null) {
      for (final userScript in initialUserScripts) {
        if (userScript.injectionTime ==
            UserScriptInjectionTime.AT_DOCUMENT_START) {
          this
              ._userScripts[UserScriptInjectionTime.AT_DOCUMENT_START]
              ?.add(userScript);
        } else {
          this
              ._userScripts[UserScriptInjectionTime.AT_DOCUMENT_END]
              ?.add(userScript);
        }
      }
    }
    this._init(params);
  }

  void _init(PlatformInAppWebViewControllerCreationParams params) {
    _controllerFromPlatform =
        params.webviewParams?.controllerFromPlatform?.call(this) ?? this;

    webStorage = AndroidWebStorage(AndroidWebStorageCreationParams(
        localStorage: AndroidLocalStorage.defaultStorage(controller: this),
        sessionStorage:
            AndroidSessionStorage.defaultStorage(controller: this)));

    if (params.webviewParams is PlatformInAppWebViewWidgetCreationParams) {
      final keepAlive =
          (params.webviewParams as PlatformInAppWebViewWidgetCreationParams)
              .keepAlive;
      if (keepAlive != null) {
        InAppWebViewControllerKeepAliveProps? props = _keepAliveMap[keepAlive];
        if (props == null) {
          // save controller properties to restore it later
          _keepAliveMap[keepAlive] = InAppWebViewControllerKeepAliveProps(
              injectedScriptsFromURL: _injectedScriptsFromURL,
              javaScriptHandlersMap: _javaScriptHandlersMap,
              userScripts: _userScripts,
              webMessageListenerObjNames: _webMessageListenerObjNames,
              webMessageChannels: _webMessageChannels,
              webMessageListeners: _webMessageListeners);
        } else {
          // restore controller properties
          _injectedScriptsFromURL = props.injectedScriptsFromURL;
          _javaScriptHandlersMap = props.javaScriptHandlersMap;
          _userScripts = props.userScripts;
          _webMessageListenerObjNames = props.webMessageListenerObjNames;
          _webMessageChannels =
              props.webMessageChannels as Set<AndroidWebMessageChannel>;
          _webMessageListeners =
              props.webMessageListeners as Set<AndroidWebMessageListener>;
        }
      }
    }
  }

  _debugLog(String method, dynamic args) {
    debugLog(
        className: this.runtimeType.toString(),
        name: _inAppBrowser == null
            ? "WebView"
            : _inAppBrowser.runtimeType.toString(),
        id: (getViewId() ?? _inAppBrowser?.id).toString(),
        debugLoggingSettings:
            PlatformInAppWebViewController.debugLoggingSettings,
        method: method,
        args: args);
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    if (PlatformInAppWebViewController.debugLoggingSettings.enabled &&
        call.method != "onCallJsHandler") {
      _debugLog(call.method, call.arguments);
    }

    switch (call.method) {
      case "onLoadStart":
        _injectedScriptsFromURL.clear();
        if ((webviewParams != null && webviewParams!.onLoadStart != null) ||
            _inAppBrowserEventHandler != null) {
          String? url = call.arguments["url"];
          WebUri? uri = url != null ? WebUri(url) : null;
          if (webviewParams != null && webviewParams!.onLoadStart != null)
            webviewParams!.onLoadStart!(_controllerFromPlatform, uri);
          else
            _inAppBrowserEventHandler!.onLoadStart(uri);
        }
        break;
      case "onLoadStop":
        if ((webviewParams != null && webviewParams!.onLoadStop != null) ||
            _inAppBrowserEventHandler != null) {
          String? url = call.arguments["url"];
          WebUri? uri = url != null ? WebUri(url) : null;
          if (webviewParams != null && webviewParams!.onLoadStop != null)
            webviewParams!.onLoadStop!(_controllerFromPlatform, uri);
          else
            _inAppBrowserEventHandler!.onLoadStop(uri);
        }
        break;
      case "onReceivedError":
        if ((webviewParams != null &&
                (webviewParams!.onReceivedError != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.onLoadError != null)) ||
            _inAppBrowserEventHandler != null) {
          WebResourceRequest request = WebResourceRequest.fromMap(
              call.arguments["request"].cast<String, dynamic>())!;
          WebResourceError error = WebResourceError.fromMap(
              call.arguments["error"].cast<String, dynamic>())!;
          var isForMainFrame = request.isForMainFrame ?? false;

          if (webviewParams != null) {
            if (webviewParams!.onReceivedError != null)
              webviewParams!.onReceivedError!(
                  _controllerFromPlatform, request, error);
            else if (isForMainFrame) {
              // ignore: deprecated_member_use_from_same_package
              webviewParams!.onLoadError!(_controllerFromPlatform, request.url,
                  error.type.toNativeValue() ?? -1, error.description);
            }
          } else {
            if (isForMainFrame) {
              _inAppBrowserEventHandler!.onLoadError(request.url,
                  error.type.toNativeValue() ?? -1, error.description);
            }
            _inAppBrowserEventHandler!.onReceivedError(request, error);
          }
        }
        break;
      case "onReceivedHttpError":
        if ((webviewParams != null &&
                (webviewParams!.onReceivedHttpError != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.onLoadHttpError != null)) ||
            _inAppBrowserEventHandler != null) {
          WebResourceRequest request = WebResourceRequest.fromMap(
              call.arguments["request"].cast<String, dynamic>())!;
          WebResourceResponse errorResponse = WebResourceResponse.fromMap(
              call.arguments["errorResponse"].cast<String, dynamic>())!;
          var isForMainFrame = request.isForMainFrame ?? false;

          if (webviewParams != null) {
            if (webviewParams!.onReceivedHttpError != null)
              webviewParams!.onReceivedHttpError!(
                  _controllerFromPlatform, request, errorResponse);
            else if (isForMainFrame) {
              // ignore: deprecated_member_use_from_same_package
              webviewParams!.onLoadHttpError!(
                  _controllerFromPlatform,
                  request.url,
                  errorResponse.statusCode ?? -1,
                  errorResponse.reasonPhrase ?? '');
            }
          } else {
            if (isForMainFrame) {
              _inAppBrowserEventHandler!.onLoadHttpError(
                  request.url,
                  errorResponse.statusCode ?? -1,
                  errorResponse.reasonPhrase ?? '');
            }
            _inAppBrowserEventHandler!
                .onReceivedHttpError(request, errorResponse);
          }
        }
        break;
      case "onProgressChanged":
        if ((webviewParams != null &&
                webviewParams!.onProgressChanged != null) ||
            _inAppBrowserEventHandler != null) {
          int progress = call.arguments["progress"];
          if (webviewParams != null && webviewParams!.onProgressChanged != null)
            webviewParams!.onProgressChanged!(
                _controllerFromPlatform, progress);
          else
            _inAppBrowserEventHandler!.onProgressChanged(progress);
        }
        break;
      case "shouldOverrideUrlLoading":
        if ((webviewParams != null &&
                webviewParams!.shouldOverrideUrlLoading != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          NavigationAction navigationAction =
              NavigationAction.fromMap(arguments)!;

          if (webviewParams != null &&
              webviewParams!.shouldOverrideUrlLoading != null)
            return (await webviewParams!.shouldOverrideUrlLoading!(
                    _controllerFromPlatform, navigationAction))
                ?.toNativeValue();
          return (await _inAppBrowserEventHandler!
                  .shouldOverrideUrlLoading(navigationAction))
              ?.toNativeValue();
        }
        break;
      case "onConsoleMessage":
        if ((webviewParams != null &&
                webviewParams!.onConsoleMessage != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          ConsoleMessage consoleMessage = ConsoleMessage.fromMap(arguments)!;
          if (webviewParams != null && webviewParams!.onConsoleMessage != null)
            webviewParams!.onConsoleMessage!(
                _controllerFromPlatform, consoleMessage);
          else
            _inAppBrowserEventHandler!.onConsoleMessage(consoleMessage);
        }
        break;
      case "onScrollChanged":
        if ((webviewParams != null && webviewParams!.onScrollChanged != null) ||
            _inAppBrowserEventHandler != null) {
          int x = call.arguments["x"];
          int y = call.arguments["y"];
          if (webviewParams != null && webviewParams!.onScrollChanged != null)
            webviewParams!.onScrollChanged!(_controllerFromPlatform, x, y);
          else
            _inAppBrowserEventHandler!.onScrollChanged(x, y);
        }
        break;
      case "onDownloadStartRequest":
        if ((webviewParams != null &&
                // ignore: deprecated_member_use_from_same_package
                (webviewParams!.onDownloadStart != null ||
                    webviewParams!.onDownloadStartRequest != null)) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          DownloadStartRequest downloadStartRequest =
              DownloadStartRequest.fromMap(arguments)!;

          if (webviewParams != null) {
            if (webviewParams!.onDownloadStartRequest != null)
              webviewParams!.onDownloadStartRequest!(
                  _controllerFromPlatform, downloadStartRequest);
            else {
              // ignore: deprecated_member_use_from_same_package
              webviewParams!.onDownloadStart!(
                  _controllerFromPlatform, downloadStartRequest.url);
            }
          } else {
            // ignore: deprecated_member_use_from_same_package
            _inAppBrowserEventHandler!
                .onDownloadStart(downloadStartRequest.url);
            _inAppBrowserEventHandler!
                .onDownloadStartRequest(downloadStartRequest);
          }
        }
        break;
      case "onLoadResourceWithCustomScheme":
        if ((webviewParams != null &&
                (webviewParams!.onLoadResourceWithCustomScheme != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.onLoadResourceCustomScheme != null)) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> requestMap =
              call.arguments["request"].cast<String, dynamic>();
          WebResourceRequest request = WebResourceRequest.fromMap(requestMap)!;

          if (webviewParams != null) {
            if (webviewParams!.onLoadResourceWithCustomScheme != null)
              return (await webviewParams!.onLoadResourceWithCustomScheme!(
                      _controllerFromPlatform, request))
                  ?.toMap();
            else {
              return (await params
                          .webviewParams!
                          // ignore: deprecated_member_use_from_same_package
                          .onLoadResourceCustomScheme!(
                      _controllerFromPlatform, request.url))
                  ?.toMap();
            }
          } else {
            return ((await _inAppBrowserEventHandler!
                        .onLoadResourceWithCustomScheme(request)) ??
                    (await _inAppBrowserEventHandler!
                        .onLoadResourceCustomScheme(request.url)))
                ?.toMap();
          }
        }
        break;
      case "onCreateWindow":
        if ((webviewParams != null && webviewParams!.onCreateWindow != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          CreateWindowAction createWindowAction =
              CreateWindowAction.fromMap(arguments)!;

          if (webviewParams != null && webviewParams!.onCreateWindow != null)
            return await webviewParams!.onCreateWindow!(
                _controllerFromPlatform, createWindowAction);
          else
            return await _inAppBrowserEventHandler!
                .onCreateWindow(createWindowAction);
        }
        break;
      case "onCloseWindow":
        if (webviewParams != null && webviewParams!.onCloseWindow != null)
          webviewParams!.onCloseWindow!(_controllerFromPlatform);
        else if (_inAppBrowserEventHandler != null)
          _inAppBrowserEventHandler!.onCloseWindow();
        break;
      case "onTitleChanged":
        if ((webviewParams != null && webviewParams!.onTitleChanged != null) ||
            _inAppBrowserEventHandler != null) {
          String? title = call.arguments["title"];
          if (webviewParams != null && webviewParams!.onTitleChanged != null)
            webviewParams!.onTitleChanged!(_controllerFromPlatform, title);
          else
            _inAppBrowserEventHandler!.onTitleChanged(title);
        }
        break;
      case "onGeolocationPermissionsShowPrompt":
        if ((webviewParams != null &&
                (webviewParams!.onGeolocationPermissionsShowPrompt != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnGeolocationPermissionsShowPrompt !=
                        null)) ||
            _inAppBrowserEventHandler != null) {
          String origin = call.arguments["origin"];

          if (webviewParams != null) {
            if (webviewParams!.onGeolocationPermissionsShowPrompt != null)
              return (await webviewParams!.onGeolocationPermissionsShowPrompt!(
                      _controllerFromPlatform, origin))
                  ?.toMap();
            else {
              return (await params
                          .webviewParams!
                          // ignore: deprecated_member_use_from_same_package
                          .androidOnGeolocationPermissionsShowPrompt!(
                      _controllerFromPlatform, origin))
                  ?.toMap();
            }
          } else {
            return ((await _inAppBrowserEventHandler!
                        .onGeolocationPermissionsShowPrompt(origin)) ??
                    (await _inAppBrowserEventHandler!
                        .androidOnGeolocationPermissionsShowPrompt(origin)))
                ?.toMap();
          }
        }
        break;
      case "onGeolocationPermissionsHidePrompt":
        if (webviewParams != null &&
            (webviewParams!.onGeolocationPermissionsHidePrompt != null ||
                // ignore: deprecated_member_use_from_same_package
                webviewParams!.androidOnGeolocationPermissionsHidePrompt !=
                    null)) {
          if (webviewParams!.onGeolocationPermissionsHidePrompt != null)
            webviewParams!
                .onGeolocationPermissionsHidePrompt!(_controllerFromPlatform);
          else {
            // ignore: deprecated_member_use_from_same_package
            webviewParams!.androidOnGeolocationPermissionsHidePrompt!(
                _controllerFromPlatform);
          }
        } else if (_inAppBrowserEventHandler != null) {
          _inAppBrowserEventHandler!.onGeolocationPermissionsHidePrompt();
          // ignore: deprecated_member_use_from_same_package
          _inAppBrowserEventHandler!
              .androidOnGeolocationPermissionsHidePrompt();
        }
        break;
      case "shouldInterceptRequest":
        if ((webviewParams != null &&
                (webviewParams!.shouldInterceptRequest != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidShouldInterceptRequest != null)) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          WebResourceRequest request = WebResourceRequest.fromMap(arguments)!;

          if (webviewParams != null) {
            if (webviewParams!.shouldInterceptRequest != null)
              return (await webviewParams!.shouldInterceptRequest!(
                      _controllerFromPlatform, request))
                  ?.toMap();
            else {
              // ignore: deprecated_member_use_from_same_package
              return (await webviewParams!.androidShouldInterceptRequest!(
                      _controllerFromPlatform, request))
                  ?.toMap();
            }
          } else {
            return ((await _inAppBrowserEventHandler!
                        .shouldInterceptRequest(request)) ??
                    (await _inAppBrowserEventHandler!
                        .androidShouldInterceptRequest(request)))
                ?.toMap();
          }
        }
        break;
      case "onRenderProcessUnresponsive":
        if ((webviewParams != null &&
                (webviewParams!.onRenderProcessUnresponsive != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnRenderProcessUnresponsive !=
                        null)) ||
            _inAppBrowserEventHandler != null) {
          String? url = call.arguments["url"];
          WebUri? uri = url != null ? WebUri(url) : null;

          if (webviewParams != null) {
            if (webviewParams!.onRenderProcessUnresponsive != null)
              return (await webviewParams!.onRenderProcessUnresponsive!(
                      _controllerFromPlatform, uri))
                  ?.toNativeValue();
            else {
              // ignore: deprecated_member_use_from_same_package
              return (await webviewParams!.androidOnRenderProcessUnresponsive!(
                      _controllerFromPlatform, uri))
                  ?.toNativeValue();
            }
          } else {
            return ((await _inAppBrowserEventHandler!
                        .onRenderProcessUnresponsive(uri)) ??
                    (await _inAppBrowserEventHandler!
                        .androidOnRenderProcessUnresponsive(uri)))
                ?.toNativeValue();
          }
        }
        break;
      case "onRenderProcessResponsive":
        if ((webviewParams != null &&
                (webviewParams!.onRenderProcessResponsive != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnRenderProcessResponsive != null)) ||
            _inAppBrowserEventHandler != null) {
          String? url = call.arguments["url"];
          WebUri? uri = url != null ? WebUri(url) : null;

          if (webviewParams != null) {
            if (webviewParams!.onRenderProcessResponsive != null)
              return (await webviewParams!.onRenderProcessResponsive!(
                      _controllerFromPlatform, uri))
                  ?.toNativeValue();
            else {
              // ignore: deprecated_member_use_from_same_package
              return (await webviewParams!.androidOnRenderProcessResponsive!(
                      _controllerFromPlatform, uri))
                  ?.toNativeValue();
            }
          } else {
            return ((await _inAppBrowserEventHandler!
                        .onRenderProcessResponsive(uri)) ??
                    (await _inAppBrowserEventHandler!
                        .androidOnRenderProcessResponsive(uri)))
                ?.toNativeValue();
          }
        }
        break;
      case "onRenderProcessGone":
        if ((webviewParams != null &&
                (webviewParams!.onRenderProcessGone != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnRenderProcessGone != null)) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          RenderProcessGoneDetail detail =
              RenderProcessGoneDetail.fromMap(arguments)!;

          if (webviewParams != null) {
            if (webviewParams!.onRenderProcessGone != null)
              webviewParams!.onRenderProcessGone!(
                  _controllerFromPlatform, detail);
            else {
              // ignore: deprecated_member_use_from_same_package
              webviewParams!.androidOnRenderProcessGone!(
                  _controllerFromPlatform, detail);
            }
          } else if (_inAppBrowserEventHandler != null) {
            _inAppBrowserEventHandler!.onRenderProcessGone(detail);
            // ignore: deprecated_member_use_from_same_package
            _inAppBrowserEventHandler!.androidOnRenderProcessGone(detail);
          }
        }
        break;
      case "onFormResubmission":
        if ((webviewParams != null &&
                (webviewParams!.onFormResubmission != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnFormResubmission != null)) ||
            _inAppBrowserEventHandler != null) {
          String? url = call.arguments["url"];
          WebUri? uri = url != null ? WebUri(url) : null;

          if (webviewParams != null) {
            if (webviewParams!.onFormResubmission != null)
              return (await webviewParams!.onFormResubmission!(
                      _controllerFromPlatform, uri))
                  ?.toNativeValue();
            else {
              // ignore: deprecated_member_use_from_same_package
              return (await webviewParams!.androidOnFormResubmission!(
                      _controllerFromPlatform, uri))
                  ?.toNativeValue();
            }
          } else {
            return ((await _inAppBrowserEventHandler!
                        .onFormResubmission(uri)) ??
                    // ignore: deprecated_member_use_from_same_package
                    (await _inAppBrowserEventHandler!
                        .androidOnFormResubmission(uri)))
                ?.toNativeValue();
          }
        }
        break;
      case "onZoomScaleChanged":
        if ((webviewParams != null &&
                // ignore: deprecated_member_use_from_same_package
                (webviewParams!.androidOnScaleChanged != null ||
                    webviewParams!.onZoomScaleChanged != null)) ||
            _inAppBrowserEventHandler != null) {
          double oldScale = call.arguments["oldScale"];
          double newScale = call.arguments["newScale"];

          if (webviewParams != null) {
            if (webviewParams!.onZoomScaleChanged != null)
              webviewParams!.onZoomScaleChanged!(
                  _controllerFromPlatform, oldScale, newScale);
            else {
              // ignore: deprecated_member_use_from_same_package
              webviewParams!.androidOnScaleChanged!(
                  _controllerFromPlatform, oldScale, newScale);
            }
          } else {
            _inAppBrowserEventHandler!.onZoomScaleChanged(oldScale, newScale);
            // ignore: deprecated_member_use_from_same_package
            _inAppBrowserEventHandler!
                .androidOnScaleChanged(oldScale, newScale);
          }
        }
        break;
      case "onReceivedIcon":
        if ((webviewParams != null &&
                (webviewParams!.onReceivedIcon != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnReceivedIcon != null)) ||
            _inAppBrowserEventHandler != null) {
          Uint8List icon =
              Uint8List.fromList(call.arguments["icon"].cast<int>());

          if (webviewParams != null) {
            if (webviewParams!.onReceivedIcon != null)
              webviewParams!.onReceivedIcon!(_controllerFromPlatform, icon);
            else {
              // ignore: deprecated_member_use_from_same_package
              webviewParams!.androidOnReceivedIcon!(
                  _controllerFromPlatform, icon);
            }
          } else {
            _inAppBrowserEventHandler!.onReceivedIcon(icon);
            // ignore: deprecated_member_use_from_same_package
            _inAppBrowserEventHandler!.androidOnReceivedIcon(icon);
          }
        }
        break;
      case "onReceivedTouchIconUrl":
        if ((webviewParams != null &&
                (webviewParams!.onReceivedTouchIconUrl != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnReceivedTouchIconUrl != null)) ||
            _inAppBrowserEventHandler != null) {
          String url = call.arguments["url"];
          bool precomposed = call.arguments["precomposed"];
          WebUri uri = WebUri(url);

          if (webviewParams != null) {
            if (webviewParams!.onReceivedTouchIconUrl != null)
              webviewParams!.onReceivedTouchIconUrl!(
                  _controllerFromPlatform, uri, precomposed);
            else {
              // ignore: deprecated_member_use_from_same_package
              webviewParams!.androidOnReceivedTouchIconUrl!(
                  _controllerFromPlatform, uri, precomposed);
            }
          } else {
            _inAppBrowserEventHandler!.onReceivedTouchIconUrl(uri, precomposed);
            // ignore: deprecated_member_use_from_same_package
            _inAppBrowserEventHandler!
                .androidOnReceivedTouchIconUrl(uri, precomposed);
          }
        }
        break;
      case "onJsAlert":
        if ((webviewParams != null && webviewParams!.onJsAlert != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          JsAlertRequest jsAlertRequest = JsAlertRequest.fromMap(arguments)!;

          if (webviewParams != null && webviewParams!.onJsAlert != null)
            return (await webviewParams!.onJsAlert!(
                    _controllerFromPlatform, jsAlertRequest))
                ?.toMap();
          else
            return (await _inAppBrowserEventHandler!.onJsAlert(jsAlertRequest))
                ?.toMap();
        }
        break;
      case "onJsConfirm":
        if ((webviewParams != null && webviewParams!.onJsConfirm != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          JsConfirmRequest jsConfirmRequest =
              JsConfirmRequest.fromMap(arguments)!;

          if (webviewParams != null && webviewParams!.onJsConfirm != null)
            return (await webviewParams!.onJsConfirm!(
                    _controllerFromPlatform, jsConfirmRequest))
                ?.toMap();
          else
            return (await _inAppBrowserEventHandler!
                    .onJsConfirm(jsConfirmRequest))
                ?.toMap();
        }
        break;
      case "onJsPrompt":
        if ((webviewParams != null && webviewParams!.onJsPrompt != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          JsPromptRequest jsPromptRequest = JsPromptRequest.fromMap(arguments)!;

          if (webviewParams != null && webviewParams!.onJsPrompt != null)
            return (await webviewParams!.onJsPrompt!(
                    _controllerFromPlatform, jsPromptRequest))
                ?.toMap();
          else
            return (await _inAppBrowserEventHandler!
                    .onJsPrompt(jsPromptRequest))
                ?.toMap();
        }
        break;
      case "onJsBeforeUnload":
        if ((webviewParams != null &&
                (webviewParams!.onJsBeforeUnload != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnJsBeforeUnload != null)) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          JsBeforeUnloadRequest jsBeforeUnloadRequest =
              JsBeforeUnloadRequest.fromMap(arguments)!;

          if (webviewParams != null) {
            if (webviewParams!.onJsBeforeUnload != null)
              return (await webviewParams!.onJsBeforeUnload!(
                      _controllerFromPlatform, jsBeforeUnloadRequest))
                  ?.toMap();
            else {
              // ignore: deprecated_member_use_from_same_package
              return (await webviewParams!.androidOnJsBeforeUnload!(
                      _controllerFromPlatform, jsBeforeUnloadRequest))
                  ?.toMap();
            }
          } else {
            return ((await _inAppBrowserEventHandler!
                        .onJsBeforeUnload(jsBeforeUnloadRequest)) ??
                    (await _inAppBrowserEventHandler!
                        .androidOnJsBeforeUnload(jsBeforeUnloadRequest)))
                ?.toMap();
          }
        }
        break;
      case "onSafeBrowsingHit":
        if ((webviewParams != null &&
                (webviewParams!.onSafeBrowsingHit != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnSafeBrowsingHit != null)) ||
            _inAppBrowserEventHandler != null) {
          String url = call.arguments["url"];
          SafeBrowsingThreat? threatType =
              SafeBrowsingThreat.fromNativeValue(call.arguments["threatType"]);
          WebUri uri = WebUri(url);

          if (webviewParams != null) {
            if (webviewParams!.onSafeBrowsingHit != null)
              return (await webviewParams!.onSafeBrowsingHit!(
                      _controllerFromPlatform, uri, threatType))
                  ?.toMap();
            else {
              // ignore: deprecated_member_use_from_same_package
              return (await webviewParams!.androidOnSafeBrowsingHit!(
                      _controllerFromPlatform, uri, threatType))
                  ?.toMap();
            }
          } else {
            return ((await _inAppBrowserEventHandler!
                        .onSafeBrowsingHit(uri, threatType)) ??
                    (await _inAppBrowserEventHandler!
                        .androidOnSafeBrowsingHit(uri, threatType)))
                ?.toMap();
          }
        }
        break;
      case "onReceivedLoginRequest":
        if ((webviewParams != null &&
                (webviewParams!.onReceivedLoginRequest != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnReceivedLoginRequest != null)) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          LoginRequest loginRequest = LoginRequest.fromMap(arguments)!;

          if (webviewParams != null) {
            if (webviewParams!.onReceivedLoginRequest != null)
              webviewParams!.onReceivedLoginRequest!(
                  _controllerFromPlatform, loginRequest);
            else {
              // ignore: deprecated_member_use_from_same_package
              webviewParams!.androidOnReceivedLoginRequest!(
                  _controllerFromPlatform, loginRequest);
            }
          } else {
            _inAppBrowserEventHandler!.onReceivedLoginRequest(loginRequest);
            // ignore: deprecated_member_use_from_same_package
            _inAppBrowserEventHandler!
                .androidOnReceivedLoginRequest(loginRequest);
          }
        }
        break;
      case "onPermissionRequestCanceled":
        if ((webviewParams != null &&
                webviewParams!.onPermissionRequestCanceled != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          PermissionRequest permissionRequest =
              PermissionRequest.fromMap(arguments)!;

          if (webviewParams != null &&
              webviewParams!.onPermissionRequestCanceled != null)
            webviewParams!.onPermissionRequestCanceled!(
                _controllerFromPlatform, permissionRequest);
          else
            _inAppBrowserEventHandler!
                .onPermissionRequestCanceled(permissionRequest);
        }
        break;
      case "onRequestFocus":
        if ((webviewParams != null && webviewParams!.onRequestFocus != null) ||
            _inAppBrowserEventHandler != null) {
          if (webviewParams != null && webviewParams!.onRequestFocus != null)
            webviewParams!.onRequestFocus!(_controllerFromPlatform);
          else
            _inAppBrowserEventHandler!.onRequestFocus();
        }
        break;
      case "onReceivedHttpAuthRequest":
        if ((webviewParams != null &&
                webviewParams!.onReceivedHttpAuthRequest != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          HttpAuthenticationChallenge challenge =
              HttpAuthenticationChallenge.fromMap(arguments)!;

          if (webviewParams != null &&
              webviewParams!.onReceivedHttpAuthRequest != null)
            return (await webviewParams!.onReceivedHttpAuthRequest!(
                    _controllerFromPlatform, challenge))
                ?.toMap();
          else
            return (await _inAppBrowserEventHandler!
                    .onReceivedHttpAuthRequest(challenge))
                ?.toMap();
        }
        break;
      case "onReceivedServerTrustAuthRequest":
        if ((webviewParams != null &&
                webviewParams!.onReceivedServerTrustAuthRequest != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          ServerTrustChallenge challenge =
              ServerTrustChallenge.fromMap(arguments)!;

          if (webviewParams != null &&
              webviewParams!.onReceivedServerTrustAuthRequest != null)
            return (await webviewParams!.onReceivedServerTrustAuthRequest!(
                    _controllerFromPlatform, challenge))
                ?.toMap();
          else
            return (await _inAppBrowserEventHandler!
                    .onReceivedServerTrustAuthRequest(challenge))
                ?.toMap();
        }
        break;
      case "onReceivedClientCertRequest":
        if ((webviewParams != null &&
                webviewParams!.onReceivedClientCertRequest != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          ClientCertChallenge challenge =
              ClientCertChallenge.fromMap(arguments)!;

          if (webviewParams != null &&
              webviewParams!.onReceivedClientCertRequest != null)
            return (await webviewParams!.onReceivedClientCertRequest!(
                    _controllerFromPlatform, challenge))
                ?.toMap();
          else
            return (await _inAppBrowserEventHandler!
                    .onReceivedClientCertRequest(challenge))
                ?.toMap();
        }
        break;
      case "onFindResultReceived":
        if ((webviewParams != null &&
                (webviewParams!.onFindResultReceived != null ||
                    (webviewParams!.findInteractionController != null &&
                        webviewParams!.findInteractionController!.params
                                .onFindResultReceived !=
                            null))) ||
            _inAppBrowserEventHandler != null) {
          int activeMatchOrdinal = call.arguments["activeMatchOrdinal"];
          int numberOfMatches = call.arguments["numberOfMatches"];
          bool isDoneCounting = call.arguments["isDoneCounting"];
          if (webviewParams != null) {
            if (webviewParams!.findInteractionController != null &&
                webviewParams!.findInteractionController!.params
                        .onFindResultReceived !=
                    null)
              webviewParams!
                      .findInteractionController!.params.onFindResultReceived!(
                  webviewParams!.findInteractionController!,
                  activeMatchOrdinal,
                  numberOfMatches,
                  isDoneCounting);
            else
              webviewParams!.onFindResultReceived!(_controllerFromPlatform,
                  activeMatchOrdinal, numberOfMatches, isDoneCounting);
          } else {
            if (_inAppBrowser!.findInteractionController != null &&
                _inAppBrowser!
                        .findInteractionController!.onFindResultReceived !=
                    null)
              _inAppBrowser!.findInteractionController!.onFindResultReceived!(
                  webviewParams!.findInteractionController!,
                  activeMatchOrdinal,
                  numberOfMatches,
                  isDoneCounting);
            else
              _inAppBrowserEventHandler!.onFindResultReceived(
                  activeMatchOrdinal, numberOfMatches, isDoneCounting);
          }
        }
        break;
      case "onPermissionRequest":
        if ((webviewParams != null &&
                (webviewParams!.onPermissionRequest != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.androidOnPermissionRequest != null)) ||
            _inAppBrowserEventHandler != null) {
          String origin = call.arguments["origin"];
          List<String> resources = call.arguments["resources"].cast<String>();

          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          PermissionRequest permissionRequest =
              PermissionRequest.fromMap(arguments)!;

          if (webviewParams != null) {
            if (webviewParams!.onPermissionRequest != null)
              return (await webviewParams!.onPermissionRequest!(
                      _controllerFromPlatform, permissionRequest))
                  ?.toMap();
            else {
              return (await webviewParams!.androidOnPermissionRequest!(
                      _controllerFromPlatform, origin, resources))
                  ?.toMap();
            }
          } else {
            return (await _inAppBrowserEventHandler!
                        .onPermissionRequest(permissionRequest))
                    ?.toMap() ??
                (await _inAppBrowserEventHandler!
                        .androidOnPermissionRequest(origin, resources))
                    ?.toMap();
          }
        }
        break;
      case "onUpdateVisitedHistory":
        if ((webviewParams != null &&
                webviewParams!.onUpdateVisitedHistory != null) ||
            _inAppBrowserEventHandler != null) {
          String? url = call.arguments["url"];
          bool? isReload = call.arguments["isReload"];
          WebUri? uri = url != null ? WebUri(url) : null;
          if (webviewParams != null &&
              webviewParams!.onUpdateVisitedHistory != null)
            webviewParams!.onUpdateVisitedHistory!(
                _controllerFromPlatform, uri, isReload);
          else
            _inAppBrowserEventHandler!.onUpdateVisitedHistory(uri, isReload);
        }
        break;
      case "onWebContentProcessDidTerminate":
        if (webviewParams != null &&
            (webviewParams!.onWebContentProcessDidTerminate != null ||
                // ignore: deprecated_member_use_from_same_package
                webviewParams!.iosOnWebContentProcessDidTerminate != null)) {
          if (webviewParams!.onWebContentProcessDidTerminate != null)
            webviewParams!
                .onWebContentProcessDidTerminate!(_controllerFromPlatform);
          else {
            // ignore: deprecated_member_use_from_same_package
            webviewParams!
                .iosOnWebContentProcessDidTerminate!(_controllerFromPlatform);
          }
        } else if (_inAppBrowserEventHandler != null) {
          _inAppBrowserEventHandler!.onWebContentProcessDidTerminate();
          // ignore: deprecated_member_use_from_same_package
          _inAppBrowserEventHandler!.iosOnWebContentProcessDidTerminate();
        }
        break;
      case "onPageCommitVisible":
        if ((webviewParams != null &&
                webviewParams!.onPageCommitVisible != null) ||
            _inAppBrowserEventHandler != null) {
          String? url = call.arguments["url"];
          WebUri? uri = url != null ? WebUri(url) : null;
          if (webviewParams != null &&
              webviewParams!.onPageCommitVisible != null)
            webviewParams!.onPageCommitVisible!(_controllerFromPlatform, uri);
          else
            _inAppBrowserEventHandler!.onPageCommitVisible(uri);
        }
        break;
      case "onDidReceiveServerRedirectForProvisionalNavigation":
        if (webviewParams != null &&
            (webviewParams!
                        .onDidReceiveServerRedirectForProvisionalNavigation !=
                    null ||
                params
                        .webviewParams!
                        // ignore: deprecated_member_use_from_same_package
                        .iosOnDidReceiveServerRedirectForProvisionalNavigation !=
                    null)) {
          if (webviewParams!
                  .onDidReceiveServerRedirectForProvisionalNavigation !=
              null)
            webviewParams!.onDidReceiveServerRedirectForProvisionalNavigation!(
                _controllerFromPlatform);
          else {
            params
                    .webviewParams!
                    // ignore: deprecated_member_use_from_same_package
                    .iosOnDidReceiveServerRedirectForProvisionalNavigation!(
                _controllerFromPlatform);
          }
        } else if (_inAppBrowserEventHandler != null) {
          _inAppBrowserEventHandler!
              .onDidReceiveServerRedirectForProvisionalNavigation();
          _inAppBrowserEventHandler!
              .iosOnDidReceiveServerRedirectForProvisionalNavigation();
        }
        break;
      case "onNavigationResponse":
        if ((webviewParams != null &&
                (webviewParams!.onNavigationResponse != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.iosOnNavigationResponse != null)) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          // ignore: deprecated_member_use_from_same_package
          IOSWKNavigationResponse iosOnNavigationResponse =
              // ignore: deprecated_member_use_from_same_package
              IOSWKNavigationResponse.fromMap(arguments)!;

          NavigationResponse navigationResponse =
              NavigationResponse.fromMap(arguments)!;

          if (webviewParams != null) {
            if (webviewParams!.onNavigationResponse != null)
              return (await webviewParams!.onNavigationResponse!(
                      _controllerFromPlatform, navigationResponse))
                  ?.toNativeValue();
            else {
              // ignore: deprecated_member_use_from_same_package
              return (await webviewParams!.iosOnNavigationResponse!(
                      _controllerFromPlatform, iosOnNavigationResponse))
                  ?.toNativeValue();
            }
          } else {
            return (await _inAppBrowserEventHandler!
                        .onNavigationResponse(navigationResponse))
                    ?.toNativeValue() ??
                (await _inAppBrowserEventHandler!
                        .iosOnNavigationResponse(iosOnNavigationResponse))
                    ?.toNativeValue();
          }
        }
        break;
      case "shouldAllowDeprecatedTLS":
        if ((webviewParams != null &&
                (webviewParams!.shouldAllowDeprecatedTLS != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.iosShouldAllowDeprecatedTLS != null)) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          URLAuthenticationChallenge challenge =
              URLAuthenticationChallenge.fromMap(arguments)!;

          if (webviewParams != null) {
            if (webviewParams!.shouldAllowDeprecatedTLS != null)
              return (await webviewParams!.shouldAllowDeprecatedTLS!(
                      _controllerFromPlatform, challenge))
                  ?.toNativeValue();
            else {
              // ignore: deprecated_member_use_from_same_package
              return (await webviewParams!.iosShouldAllowDeprecatedTLS!(
                      _controllerFromPlatform, challenge))
                  ?.toNativeValue();
            }
          } else {
            return (await _inAppBrowserEventHandler!
                        .shouldAllowDeprecatedTLS(challenge))
                    ?.toNativeValue() ??
                // ignore: deprecated_member_use_from_same_package
                (await _inAppBrowserEventHandler!
                        .iosShouldAllowDeprecatedTLS(challenge))
                    ?.toNativeValue();
          }
        }
        break;
      case "onLongPressHitTestResult":
        if ((webviewParams != null &&
                webviewParams!.onLongPressHitTestResult != null) ||
            _inAppBrowserEventHandler != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          InAppWebViewHitTestResult hitTestResult =
              InAppWebViewHitTestResult.fromMap(arguments)!;

          if (webviewParams != null &&
              webviewParams!.onLongPressHitTestResult != null)
            webviewParams!.onLongPressHitTestResult!(
                _controllerFromPlatform, hitTestResult);
          else
            _inAppBrowserEventHandler!.onLongPressHitTestResult(hitTestResult);
        }
        break;
      case "onCreateContextMenu":
        ContextMenu? contextMenu;
        if (webviewParams != null && webviewParams!.contextMenu != null) {
          contextMenu = webviewParams!.contextMenu;
        } else if (_inAppBrowserEventHandler != null &&
            _inAppBrowser!.contextMenu != null) {
          contextMenu = _inAppBrowser!.contextMenu;
        }

        if (contextMenu != null && contextMenu.onCreateContextMenu != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          InAppWebViewHitTestResult hitTestResult =
              InAppWebViewHitTestResult.fromMap(arguments)!;

          contextMenu.onCreateContextMenu!(hitTestResult);
        }
        break;
      case "onHideContextMenu":
        ContextMenu? contextMenu;
        if (webviewParams != null && webviewParams!.contextMenu != null) {
          contextMenu = webviewParams!.contextMenu;
        } else if (_inAppBrowserEventHandler != null &&
            _inAppBrowser!.contextMenu != null) {
          contextMenu = _inAppBrowser!.contextMenu;
        }

        if (contextMenu != null && contextMenu.onHideContextMenu != null) {
          contextMenu.onHideContextMenu!();
        }
        break;
      case "onContextMenuActionItemClicked":
        ContextMenu? contextMenu;
        if (webviewParams != null && webviewParams!.contextMenu != null) {
          contextMenu = webviewParams!.contextMenu;
        } else if (_inAppBrowserEventHandler != null &&
            _inAppBrowser!.contextMenu != null) {
          contextMenu = _inAppBrowser!.contextMenu;
        }

        if (contextMenu != null) {
          int? androidId = call.arguments["androidId"];
          String? iosId = call.arguments["iosId"];
          dynamic id = call.arguments["id"];
          String title = call.arguments["title"];

          ContextMenuItem menuItemClicked = ContextMenuItem(
              id: id,
              // ignore: deprecated_member_use_from_same_package
              androidId: androidId,
              // ignore: deprecated_member_use_from_same_package
              iosId: iosId,
              title: title,
              action: null);

          for (var menuItem in contextMenu.menuItems) {
            if (menuItem.id == id) {
              menuItemClicked = menuItem;
              if (menuItem.action != null) {
                menuItem.action!();
              }
              break;
            }
          }

          if (contextMenu.onContextMenuActionItemClicked != null) {
            contextMenu.onContextMenuActionItemClicked!(menuItemClicked);
          }
        }
        break;
      case "onEnterFullscreen":
        if (webviewParams != null && webviewParams!.onEnterFullscreen != null)
          webviewParams!.onEnterFullscreen!(_controllerFromPlatform);
        else if (_inAppBrowserEventHandler != null)
          _inAppBrowserEventHandler!.onEnterFullscreen();
        break;
      case "onExitFullscreen":
        if (webviewParams != null && webviewParams!.onExitFullscreen != null)
          webviewParams!.onExitFullscreen!(_controllerFromPlatform);
        else if (_inAppBrowserEventHandler != null)
          _inAppBrowserEventHandler!.onExitFullscreen();
        break;
      case "onOverScrolled":
        if ((webviewParams != null && webviewParams!.onOverScrolled != null) ||
            _inAppBrowserEventHandler != null) {
          int x = call.arguments["x"];
          int y = call.arguments["y"];
          bool clampedX = call.arguments["clampedX"];
          bool clampedY = call.arguments["clampedY"];

          if (webviewParams != null && webviewParams!.onOverScrolled != null)
            webviewParams!.onOverScrolled!(
                _controllerFromPlatform, x, y, clampedX, clampedY);
          else
            _inAppBrowserEventHandler!.onOverScrolled(x, y, clampedX, clampedY);
        }
        break;
      case "onWindowFocus":
        if (webviewParams != null && webviewParams!.onWindowFocus != null)
          webviewParams!.onWindowFocus!(_controllerFromPlatform);
        else if (_inAppBrowserEventHandler != null)
          _inAppBrowserEventHandler!.onWindowFocus();
        break;
      case "onWindowBlur":
        if (webviewParams != null && webviewParams!.onWindowBlur != null)
          webviewParams!.onWindowBlur!(_controllerFromPlatform);
        else if (_inAppBrowserEventHandler != null)
          _inAppBrowserEventHandler!.onWindowBlur();
        break;
      case "onPrintRequest":
        if ((webviewParams != null &&
                (webviewParams!.onPrintRequest != null ||
                    // ignore: deprecated_member_use_from_same_package
                    webviewParams!.onPrint != null)) ||
            _inAppBrowserEventHandler != null) {
          String? url = call.arguments["url"];
          String? printJobId = call.arguments["printJobId"];
          WebUri? uri = url != null ? WebUri(url) : null;
          AndroidPrintJobController? printJob = printJobId != null
              ? AndroidPrintJobController(
                  AndroidPrintJobControllerCreationParams(id: printJobId))
              : null;

          if (webviewParams != null) {
            if (webviewParams!.onPrintRequest != null)
              return await webviewParams!.onPrintRequest!(
                  _controllerFromPlatform, uri, printJob);
            else {
              // ignore: deprecated_member_use_from_same_package
              webviewParams!.onPrint!(_controllerFromPlatform, uri);
              return false;
            }
          } else {
            // ignore: deprecated_member_use_from_same_package
            _inAppBrowserEventHandler!.onPrint(uri);
            return await _inAppBrowserEventHandler!
                .onPrintRequest(uri, printJob);
          }
        }
        break;
      case "onInjectedScriptLoaded":
        String id = call.arguments[0];
        var onLoadCallback = _injectedScriptsFromURL[id]?.onLoad;
        if ((webviewParams != null || _inAppBrowserEventHandler != null) &&
            onLoadCallback != null) {
          onLoadCallback();
        }
        break;
      case "onInjectedScriptError":
        String id = call.arguments[0];
        var onErrorCallback = _injectedScriptsFromURL[id]?.onError;
        if ((webviewParams != null || _inAppBrowserEventHandler != null) &&
            onErrorCallback != null) {
          onErrorCallback();
        }
        break;
      case "onCameraCaptureStateChanged":
        if ((webviewParams != null &&
                webviewParams!.onCameraCaptureStateChanged != null) ||
            _inAppBrowserEventHandler != null) {
          var oldState =
              MediaCaptureState.fromNativeValue(call.arguments["oldState"]);
          var newState =
              MediaCaptureState.fromNativeValue(call.arguments["newState"]);

          if (webviewParams != null &&
              webviewParams!.onCameraCaptureStateChanged != null)
            webviewParams!.onCameraCaptureStateChanged!(
                _controllerFromPlatform, oldState, newState);
          else
            _inAppBrowserEventHandler!
                .onCameraCaptureStateChanged(oldState, newState);
        }
        break;
      case "onMicrophoneCaptureStateChanged":
        if ((webviewParams != null &&
                webviewParams!.onMicrophoneCaptureStateChanged != null) ||
            _inAppBrowserEventHandler != null) {
          var oldState =
              MediaCaptureState.fromNativeValue(call.arguments["oldState"]);
          var newState =
              MediaCaptureState.fromNativeValue(call.arguments["newState"]);

          if (webviewParams != null &&
              webviewParams!.onMicrophoneCaptureStateChanged != null)
            webviewParams!.onMicrophoneCaptureStateChanged!(
                _controllerFromPlatform, oldState, newState);
          else
            _inAppBrowserEventHandler!
                .onMicrophoneCaptureStateChanged(oldState, newState);
        }
        break;
      case "onContentSizeChanged":
        if ((webviewParams != null &&
                webviewParams!.onContentSizeChanged != null) ||
            _inAppBrowserEventHandler != null) {
          var oldContentSize = MapSize.fromMap(
              call.arguments["oldContentSize"]?.cast<String, dynamic>())!;
          var newContentSize = MapSize.fromMap(
              call.arguments["newContentSize"]?.cast<String, dynamic>())!;

          if (webviewParams != null &&
              webviewParams!.onContentSizeChanged != null)
            webviewParams!.onContentSizeChanged!(
                _controllerFromPlatform, oldContentSize, newContentSize);
          else
            _inAppBrowserEventHandler!
                .onContentSizeChanged(oldContentSize, newContentSize);
        }
        break;
      case "onCallJsHandler":
        String handlerName = call.arguments["handlerName"];
        // decode args to json
        List<dynamic> args = jsonDecode(call.arguments["args"]);

        _debugLog(handlerName, args);

        switch (handlerName) {
          case "onLoadResource":
            if ((webviewParams != null &&
                    webviewParams!.onLoadResource != null) ||
                _inAppBrowserEventHandler != null) {
              Map<String, dynamic> arguments = args[0].cast<String, dynamic>();
              arguments["startTime"] = arguments["startTime"] is int
                  ? arguments["startTime"].toDouble()
                  : arguments["startTime"];
              arguments["duration"] = arguments["duration"] is int
                  ? arguments["duration"].toDouble()
                  : arguments["duration"];

              var response = LoadedResource.fromMap(arguments)!;

              if (webviewParams != null &&
                  webviewParams!.onLoadResource != null)
                webviewParams!.onLoadResource!(
                    _controllerFromPlatform, response);
              else
                _inAppBrowserEventHandler!.onLoadResource(response);
            }
            return null;
          case "shouldInterceptAjaxRequest":
            if ((webviewParams != null &&
                    webviewParams!.shouldInterceptAjaxRequest != null) ||
                _inAppBrowserEventHandler != null) {
              Map<String, dynamic> arguments = args[0].cast<String, dynamic>();
              AjaxRequest request = AjaxRequest.fromMap(arguments)!;

              if (webviewParams != null &&
                  webviewParams!.shouldInterceptAjaxRequest != null)
                return jsonEncode(
                    await params.webviewParams!.shouldInterceptAjaxRequest!(
                        _controllerFromPlatform, request));
              else
                return jsonEncode(await _inAppBrowserEventHandler!
                    .shouldInterceptAjaxRequest(request));
            }
            return null;
          case "onAjaxReadyStateChange":
            if ((webviewParams != null &&
                    webviewParams!.onAjaxReadyStateChange != null) ||
                _inAppBrowserEventHandler != null) {
              Map<String, dynamic> arguments = args[0].cast<String, dynamic>();
              AjaxRequest request = AjaxRequest.fromMap(arguments)!;

              if (webviewParams != null &&
                  webviewParams!.onAjaxReadyStateChange != null)
                return (await webviewParams!.onAjaxReadyStateChange!(
                        _controllerFromPlatform, request))
                    ?.toNativeValue();
              else
                return (await _inAppBrowserEventHandler!
                        .onAjaxReadyStateChange(request))
                    ?.toNativeValue();
            }
            return null;
          case "onAjaxProgress":
            if ((webviewParams != null &&
                    webviewParams!.onAjaxProgress != null) ||
                _inAppBrowserEventHandler != null) {
              Map<String, dynamic> arguments = args[0].cast<String, dynamic>();
              AjaxRequest request = AjaxRequest.fromMap(arguments)!;

              if (webviewParams != null &&
                  webviewParams!.onAjaxProgress != null)
                return (await webviewParams!.onAjaxProgress!(
                        _controllerFromPlatform, request))
                    ?.toNativeValue();
              else
                return (await _inAppBrowserEventHandler!
                        .onAjaxProgress(request))
                    ?.toNativeValue();
            }
            return null;
          case "shouldInterceptFetchRequest":
            if ((webviewParams != null &&
                    webviewParams!.shouldInterceptFetchRequest != null) ||
                _inAppBrowserEventHandler != null) {
              Map<String, dynamic> arguments = args[0].cast<String, dynamic>();
              FetchRequest request = FetchRequest.fromMap(arguments)!;

              if (webviewParams != null &&
                  webviewParams!.shouldInterceptFetchRequest != null)
                return jsonEncode(
                    await webviewParams!.shouldInterceptFetchRequest!(
                        _controllerFromPlatform, request));
              else
                return jsonEncode(await _inAppBrowserEventHandler!
                    .shouldInterceptFetchRequest(request));
            }
            return null;
          case "onWindowFocus":
            if (webviewParams != null && webviewParams!.onWindowFocus != null)
              webviewParams!.onWindowFocus!(_controllerFromPlatform);
            else if (_inAppBrowserEventHandler != null)
              _inAppBrowserEventHandler!.onWindowFocus();
            return null;
          case "onWindowBlur":
            if (webviewParams != null && webviewParams!.onWindowBlur != null)
              webviewParams!.onWindowBlur!(_controllerFromPlatform);
            else if (_inAppBrowserEventHandler != null)
              _inAppBrowserEventHandler!.onWindowBlur();
            return null;
          case "onInjectedScriptLoaded":
            String id = args[0];
            var onLoadCallback = _injectedScriptsFromURL[id]?.onLoad;
            if ((webviewParams != null || _inAppBrowserEventHandler != null) &&
                onLoadCallback != null) {
              onLoadCallback();
            }
            return null;
          case "onInjectedScriptError":
            String id = args[0];
            var onErrorCallback = _injectedScriptsFromURL[id]?.onError;
            if ((webviewParams != null || _inAppBrowserEventHandler != null) &&
                onErrorCallback != null) {
              onErrorCallback();
            }
            return null;
        }

        if (_javaScriptHandlersMap.containsKey(handlerName)) {
          // convert result to json
          try {
            return jsonEncode(await _javaScriptHandlersMap[handlerName]!(args));
          } catch (error, stacktrace) {
            developer.log(error.toString() + '\n' + stacktrace.toString(),
                name: 'JavaScript Handler "$handlerName"');
            throw Exception(error.toString().replaceFirst('Exception: ', ''));
          }
        }
        break;
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }
    return null;
  }

  @override
  Future<WebUri?> getUrl() async {
    Map<String, dynamic> args = <String, dynamic>{};
    String? url = await channel?.invokeMethod<String?>('getUrl', args);
    return url != null ? WebUri(url) : null;
  }

  @override
  Future<String?> getTitle() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<String?>('getTitle', args);
  }

  @override
  Future<int?> getProgress() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<int?>('getProgress', args);
  }

  @override
  Future<String?> getHtml() async {
    String? html;

    InAppWebViewSettings? settings = await getSettings();
    if (settings != null && settings.javaScriptEnabled == true) {
      html = await evaluateJavascript(
          source: "window.document.getElementsByTagName('html')[0].outerHTML;");
      if (html != null && html.isNotEmpty) return html;
    }

    var webviewUrl = await getUrl();
    if (webviewUrl == null) {
      return html;
    }

    if (webviewUrl.isScheme("file")) {
      var assetPathSplit = webviewUrl.toString().split("/flutter_assets/");
      var assetPath = assetPathSplit[assetPathSplit.length - 1];
      try {
        var bytes = await rootBundle.load(assetPath);
        html = utf8.decode(bytes.buffer.asUint8List());
      } catch (e) {}
    } else {
      try {
        HttpClient client = HttpClient();
        var htmlRequest = await client.getUrl(webviewUrl);
        html =
            await (await htmlRequest.close()).transform(Utf8Decoder()).join();
      } catch (e) {
        developer.log(e.toString(), name: this.runtimeType.toString());
      }
    }

    return html;
  }

  @override
  Future<List<Favicon>> getFavicons() async {
    List<Favicon> favicons = [];

    var webviewUrl = await getUrl();

    if (webviewUrl == null) {
      return favicons;
    }

    String? manifestUrl;

    var html = await getHtml();
    if (html == null || html.isEmpty) {
      return favicons;
    }
    var assetPathBase;

    if (webviewUrl.isScheme("file")) {
      var assetPathSplit = webviewUrl.toString().split("/flutter_assets/");
      assetPathBase = assetPathSplit[0] + "/flutter_assets/";
    }

    InAppWebViewSettings? settings = await getSettings();
    if (settings != null && settings.javaScriptEnabled == true) {
      List<Map<dynamic, dynamic>> links = (await evaluateJavascript(source: """
(function() {
  var linkNodes = document.head.getElementsByTagName("link");
  var links = [];
  for (var i = 0; i < linkNodes.length; i++) {
    var linkNode = linkNodes[i];
    if (linkNode.rel === 'manifest') {
      links.push(
        {
          rel: linkNode.rel,
          href: linkNode.href,
          sizes: null
        }
      );
    } else if (linkNode.rel != null && linkNode.rel.indexOf('icon') >= 0) {
      links.push(
        {
          rel: linkNode.rel,
          href: linkNode.href,
          sizes: linkNode.sizes != null && linkNode.sizes.value != "" ? linkNode.sizes.value : null
        }
      );
    }
  }
  return links;
})();
"""))?.cast<Map<dynamic, dynamic>>() ?? [];
      for (var link in links) {
        if (link["rel"] == "manifest") {
          manifestUrl = link["href"];
          if (!_isUrlAbsolute(manifestUrl!)) {
            if (manifestUrl.startsWith("/")) {
              manifestUrl = manifestUrl.substring(1);
            }
            manifestUrl = ((assetPathBase == null)
                    ? webviewUrl.scheme + "://" + webviewUrl.host + "/"
                    : assetPathBase) +
                manifestUrl;
          }
          continue;
        }
        favicons.addAll(_createFavicons(webviewUrl, assetPathBase, link["href"],
            link["rel"], link["sizes"], false));
      }
    }

    // try to get /favicon.ico
    try {
      HttpClient client = HttpClient();
      var faviconUrl =
          webviewUrl.scheme + "://" + webviewUrl.host + "/favicon.ico";
      var faviconUri = WebUri(faviconUrl);
      var headRequest = await client.headUrl(faviconUri);
      var headResponse = await headRequest.close();
      if (headResponse.statusCode == 200) {
        favicons.add(Favicon(url: faviconUri, rel: "shortcut icon"));
      }
    } catch (e) {
      developer.log("/favicon.ico file not found: " + e.toString(),
          name: this.runtimeType.toString());
    }

    // try to get the manifest file
    HttpClientRequest? manifestRequest;
    HttpClientResponse? manifestResponse;
    bool manifestFound = false;
    if (manifestUrl == null) {
      manifestUrl =
          webviewUrl.scheme + "://" + webviewUrl.host + "/manifest.json";
    }
    try {
      HttpClient client = HttpClient();
      manifestRequest = await client.getUrl(Uri.parse(manifestUrl));
      manifestResponse = await manifestRequest.close();
      manifestFound = manifestResponse.statusCode == 200 &&
          manifestResponse.headers.contentType?.mimeType == "application/json";
    } catch (e) {
      developer.log("Manifest file not found: " + e.toString(),
          name: runtimeType.toString());
    }

    if (manifestFound) {
      try {
        Map<String, dynamic> manifest = json
            .decode(await manifestResponse!.transform(Utf8Decoder()).join());
        if (manifest.containsKey("icons")) {
          for (Map<String, dynamic> icon in manifest["icons"]) {
            favicons.addAll(_createFavicons(webviewUrl, assetPathBase,
                icon["src"], icon["rel"], icon["sizes"], true));
          }
        }
      } catch (e) {
        developer.log(
            "Cannot get favicons from Manifest file. It might not have a valid format: " +
                e.toString(),
            error: e,
            name: runtimeType.toString());
      }
    }

    return favicons;
  }

  bool _isUrlAbsolute(String url) {
    return url.startsWith("http://") || url.startsWith("https://");
  }

  List<Favicon> _createFavicons(WebUri url, String? assetPathBase,
      String urlIcon, String? rel, String? sizes, bool isManifest) {
    List<Favicon> favicons = [];

    List<String> urlSplit = urlIcon.split("/");
    if (!_isUrlAbsolute(urlIcon)) {
      if (urlIcon.startsWith("/")) {
        urlIcon = urlIcon.substring(1);
      }
      urlIcon = ((assetPathBase == null)
              ? url.scheme + "://" + url.host + "/"
              : assetPathBase) +
          urlIcon;
    }
    if (isManifest) {
      rel = (sizes != null)
          ? urlSplit[urlSplit.length - 1]
              .replaceFirst("-" + sizes, "")
              .split(" ")[0]
              .split(".")[0]
          : null;
    }
    if (sizes != null && sizes.isNotEmpty && sizes != "any") {
      List<String> sizesSplit = sizes.split(" ");
      for (String size in sizesSplit) {
        int width = int.parse(size.split("x")[0]);
        int height = int.parse(size.split("x")[1]);
        favicons.add(Favicon(
            url: WebUri(urlIcon), rel: rel, width: width, height: height));
      }
    } else {
      favicons.add(
          Favicon(url: WebUri(urlIcon), rel: rel, width: null, height: null));
    }

    return favicons;
  }

  @override
  Future<void> loadUrl(
      {required URLRequest urlRequest,
      @Deprecated('Use allowingReadAccessTo instead')
      Uri? iosAllowingReadAccessTo,
      WebUri? allowingReadAccessTo}) async {
    assert(urlRequest.url != null && urlRequest.url.toString().isNotEmpty);
    assert(
        allowingReadAccessTo == null || allowingReadAccessTo.isScheme("file"));
    assert(iosAllowingReadAccessTo == null ||
        iosAllowingReadAccessTo.isScheme("file"));

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('urlRequest', () => urlRequest.toMap());
    args.putIfAbsent(
        'allowingReadAccessTo',
        () =>
            allowingReadAccessTo?.toString() ??
            iosAllowingReadAccessTo?.toString());
    await channel?.invokeMethod('loadUrl', args);
  }

  @override
  Future<void> postUrl(
      {required WebUri url, required Uint8List postData}) async {
    assert(url.toString().isNotEmpty);
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('url', () => url.toString());
    args.putIfAbsent('postData', () => postData);
    await channel?.invokeMethod('postUrl', args);
  }

  @override
  Future<void> loadData(
      {required String data,
      String mimeType = "text/html",
      String encoding = "utf8",
      WebUri? baseUrl,
      @Deprecated('Use historyUrl instead') Uri? androidHistoryUrl,
      WebUri? historyUrl,
      @Deprecated('Use allowingReadAccessTo instead')
      Uri? iosAllowingReadAccessTo,
      WebUri? allowingReadAccessTo}) async {
    assert(
        allowingReadAccessTo == null || allowingReadAccessTo.isScheme("file"));
    assert(iosAllowingReadAccessTo == null ||
        iosAllowingReadAccessTo.isScheme("file"));

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('data', () => data);
    args.putIfAbsent('mimeType', () => mimeType);
    args.putIfAbsent('encoding', () => encoding);
    args.putIfAbsent('baseUrl', () => baseUrl?.toString() ?? "about:blank");
    args.putIfAbsent(
        'historyUrl',
        () =>
            historyUrl?.toString() ??
            androidHistoryUrl?.toString() ??
            "about:blank");
    args.putIfAbsent(
        'allowingReadAccessTo',
        () =>
            allowingReadAccessTo?.toString() ??
            iosAllowingReadAccessTo?.toString());
    await channel?.invokeMethod('loadData', args);
  }

  @override
  Future<void> loadFile({required String assetFilePath}) async {
    assert(assetFilePath.isNotEmpty);
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('assetFilePath', () => assetFilePath);
    await channel?.invokeMethod('loadFile', args);
  }

  @override
  Future<void> reload() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('reload', args);
  }

  @override
  Future<void> goBack() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('goBack', args);
  }

  @override
  Future<bool> canGoBack() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('canGoBack', args) ?? false;
  }

  @override
  Future<void> goForward() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('goForward', args);
  }

  @override
  Future<bool> canGoForward() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('canGoForward', args) ?? false;
  }

  @override
  Future<void> goBackOrForward({required int steps}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('steps', () => steps);
    await channel?.invokeMethod('goBackOrForward', args);
  }

  @override
  Future<bool> canGoBackOrForward({required int steps}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('steps', () => steps);
    return await channel?.invokeMethod<bool>('canGoBackOrForward', args) ??
        false;
  }

  @override
  Future<void> goTo({required WebHistoryItem historyItem}) async {
    var steps = historyItem.offset;
    if (steps != null) {
      await goBackOrForward(steps: steps);
    }
  }

  @override
  Future<bool> isLoading() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('isLoading', args) ?? false;
  }

  @override
  Future<void> stopLoading() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('stopLoading', args);
  }

  @override
  Future<dynamic> evaluateJavascript(
      {required String source, ContentWorld? contentWorld}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('source', () => source);
    args.putIfAbsent('contentWorld', () => contentWorld?.toMap());
    var data = await channel?.invokeMethod('evaluateJavascript', args);
    if (data != null) {
      try {
        // try to json decode the data coming from JavaScript
        // otherwise return it as it is.
        data = json.decode(data);
      } catch (e) {}
    }
    return data;
  }

  @override
  Future<void> injectJavascriptFileFromUrl(
      {required WebUri urlFile,
      ScriptHtmlTagAttributes? scriptHtmlTagAttributes}) async {
    assert(urlFile.toString().isNotEmpty);
    var id = scriptHtmlTagAttributes?.id;
    if (scriptHtmlTagAttributes != null && id != null) {
      _injectedScriptsFromURL[id] = scriptHtmlTagAttributes;
    }
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('urlFile', () => urlFile.toString());
    args.putIfAbsent(
        'scriptHtmlTagAttributes', () => scriptHtmlTagAttributes?.toMap());
    await channel?.invokeMethod('injectJavascriptFileFromUrl', args);
  }

  @override
  Future<dynamic> injectJavascriptFileFromAsset(
      {required String assetFilePath}) async {
    String source = await rootBundle.loadString(assetFilePath);
    return await evaluateJavascript(source: source);
  }

  @override
  Future<void> injectCSSCode({required String source}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('source', () => source);
    await channel?.invokeMethod('injectCSSCode', args);
  }

  @override
  Future<void> injectCSSFileFromUrl(
      {required WebUri urlFile,
      CSSLinkHtmlTagAttributes? cssLinkHtmlTagAttributes}) async {
    assert(urlFile.toString().isNotEmpty);
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('urlFile', () => urlFile.toString());
    args.putIfAbsent(
        'cssLinkHtmlTagAttributes', () => cssLinkHtmlTagAttributes?.toMap());
    await channel?.invokeMethod('injectCSSFileFromUrl', args);
  }

  @override
  Future<void> injectCSSFileFromAsset({required String assetFilePath}) async {
    String source = await rootBundle.loadString(assetFilePath);
    await injectCSSCode(source: source);
  }

  @override
  void addJavaScriptHandler(
      {required String handlerName,
      required JavaScriptHandlerCallback callback}) {
    assert(!_JAVASCRIPT_HANDLER_FORBIDDEN_NAMES.contains(handlerName),
        '"$handlerName" is a forbidden name!');
    this._javaScriptHandlersMap[handlerName] = (callback);
  }

  @override
  JavaScriptHandlerCallback? removeJavaScriptHandler(
      {required String handlerName}) {
    return this._javaScriptHandlersMap.remove(handlerName);
  }

  @override
  bool hasJavaScriptHandler({required String handlerName}) {
    return this._javaScriptHandlersMap.containsKey(handlerName);
  }

  @override
  Future<Uint8List?> takeScreenshot(
      {ScreenshotConfiguration? screenshotConfiguration}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent(
        'screenshotConfiguration', () => screenshotConfiguration?.toMap());
    return await channel?.invokeMethod<Uint8List?>('takeScreenshot', args);
  }

  @override
  @Deprecated('Use setSettings instead')
  Future<void> setOptions({required InAppWebViewGroupOptions options}) async {
    InAppWebViewSettings settings =
        InAppWebViewSettings.fromMap(options.toMap()) ?? InAppWebViewSettings();
    await setSettings(settings: settings);
  }

  @override
  @Deprecated('Use getSettings instead')
  Future<InAppWebViewGroupOptions?> getOptions() async {
    InAppWebViewSettings? settings = await getSettings();

    Map<dynamic, dynamic>? options = settings?.toMap();
    if (options != null) {
      options = options.cast<String, dynamic>();
      return InAppWebViewGroupOptions.fromMap(options as Map<String, dynamic>);
    }

    return null;
  }

  @override
  Future<void> setSettings({required InAppWebViewSettings settings}) async {
    Map<String, dynamic> args = <String, dynamic>{};

    args.putIfAbsent('settings', () => settings.toMap());
    await channel?.invokeMethod('setSettings', args);
  }

  @override
  Future<InAppWebViewSettings?> getSettings() async {
    Map<String, dynamic> args = <String, dynamic>{};

    Map<dynamic, dynamic>? settings =
        await channel?.invokeMethod('getSettings', args);
    if (settings != null) {
      settings = settings.cast<String, dynamic>();
      return InAppWebViewSettings.fromMap(settings as Map<String, dynamic>);
    }

    return null;
  }

  @override
  Future<WebHistory?> getCopyBackForwardList() async {
    Map<String, dynamic> args = <String, dynamic>{};
    Map<String, dynamic>? result =
        (await channel?.invokeMethod('getCopyBackForwardList', args))
            ?.cast<String, dynamic>();
    return WebHistory.fromMap(result);
  }

  @override
  @Deprecated("Use InAppWebViewController.clearAllCache instead")
  Future<void> clearCache() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('clearCache', args);
  }

  @override
  @Deprecated("Use FindInteractionController.findAll instead")
  Future<void> findAllAsync({required String find}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('find', () => find);
    await channel?.invokeMethod('findAll', args);
  }

  @override
  @Deprecated("Use FindInteractionController.findNext instead")
  Future<void> findNext({required bool forward}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('forward', () => forward);
    await channel?.invokeMethod('findNext', args);
  }

  @override
  @Deprecated("Use FindInteractionController.clearMatches instead")
  Future<void> clearMatches() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('clearMatches', args);
  }

  @override
  @Deprecated("Use tRexRunnerHtml instead")
  Future<String> getTRexRunnerHtml() async {
    return await tRexRunnerHtml;
  }

  @override
  @Deprecated("Use tRexRunnerCss instead")
  Future<String> getTRexRunnerCss() async {
    return await tRexRunnerCss;
  }

  @override
  Future<void> scrollTo(
      {required int x, required int y, bool animated = false}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('x', () => x);
    args.putIfAbsent('y', () => y);
    args.putIfAbsent('animated', () => animated);
    await channel?.invokeMethod('scrollTo', args);
  }

  @override
  Future<void> scrollBy(
      {required int x, required int y, bool animated = false}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('x', () => x);
    args.putIfAbsent('y', () => y);
    args.putIfAbsent('animated', () => animated);
    await channel?.invokeMethod('scrollBy', args);
  }

  @override
  Future<void> pauseTimers() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('pauseTimers', args);
  }

  @override
  Future<void> resumeTimers() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('resumeTimers', args);
  }

  @override
  Future<AndroidPrintJobController?> printCurrentPage(
      {PrintJobSettings? settings}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("settings", () => settings?.toMap());
    String? jobId =
        await channel?.invokeMethod<String?>('printCurrentPage', args);
    if (jobId != null) {
      return AndroidPrintJobController(
          PlatformPrintJobControllerCreationParams(id: jobId));
    }
    return null;
  }

  @override
  Future<int?> getContentHeight() async {
    Map<String, dynamic> args = <String, dynamic>{};
    var height = await channel?.invokeMethod('getContentHeight', args);
    if (height == null || height == 0) {
      // try to use javascript
      var scrollHeight = await evaluateJavascript(
          source: "document.documentElement.scrollHeight;");
      if (scrollHeight != null && scrollHeight is num) {
        height = scrollHeight.toInt();
      }
    }
    return height;
  }

  @override
  Future<int?> getContentWidth() async {
    Map<String, dynamic> args = <String, dynamic>{};
    var height = await channel?.invokeMethod('getContentWidth', args);
    if (height == null || height == 0) {
      // try to use javascript
      var scrollHeight = await evaluateJavascript(
          source: "document.documentElement.scrollWidth;");
      if (scrollHeight != null && scrollHeight is num) {
        height = scrollHeight.toInt();
      }
    }
    return height;
  }

  @override
  Future<void> zoomBy(
      {required double zoomFactor,
      @Deprecated('Use animated instead') bool? iosAnimated,
      bool animated = false}) async {
    assert(zoomFactor > 0.01 && zoomFactor <= 100.0);

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('zoomFactor', () => zoomFactor);
    args.putIfAbsent('animated', () => iosAnimated ?? animated);
    return await channel?.invokeMethod('zoomBy', args);
  }

  @override
  Future<WebUri?> getOriginalUrl() async {
    Map<String, dynamic> args = <String, dynamic>{};
    String? url = await channel?.invokeMethod<String?>('getOriginalUrl', args);
    return url != null ? WebUri(url) : null;
  }

  @override
  Future<double?> getZoomScale() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<double?>('getZoomScale', args);
  }

  @override
  @Deprecated('Use getZoomScale instead')
  Future<double?> getScale() async {
    return await getZoomScale();
  }

  @override
  Future<String?> getSelectedText() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<String?>('getSelectedText', args);
  }

  @override
  Future<InAppWebViewHitTestResult?> getHitTestResult() async {
    Map<String, dynamic> args = <String, dynamic>{};
    Map<dynamic, dynamic>? hitTestResultMap =
        await channel?.invokeMethod('getHitTestResult', args);

    if (hitTestResultMap == null) {
      return null;
    }

    hitTestResultMap = hitTestResultMap.cast<String, dynamic>();

    InAppWebViewHitTestResultType? type =
        InAppWebViewHitTestResultType.fromNativeValue(
            hitTestResultMap["type"]?.toInt());
    String? extra = hitTestResultMap["extra"];
    return InAppWebViewHitTestResult(type: type, extra: extra);
  }

  @override
  Future<void> clearFocus() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod('clearFocus', args);
  }

  @override
  Future<void> setContextMenu(ContextMenu? contextMenu) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("contextMenu", () => contextMenu?.toMap());
    await channel?.invokeMethod('setContextMenu', args);
    _inAppBrowser?.setContextMenu(contextMenu);
  }

  @override
  Future<RequestFocusNodeHrefResult?> requestFocusNodeHref() async {
    Map<String, dynamic> args = <String, dynamic>{};
    Map<dynamic, dynamic>? result =
        await channel?.invokeMethod('requestFocusNodeHref', args);
    return result != null
        ? RequestFocusNodeHrefResult(
            url: result['url'] != null ? WebUri(result['url']) : null,
            title: result['title'],
            src: result['src'],
          )
        : null;
  }

  @override
  Future<RequestImageRefResult?> requestImageRef() async {
    Map<String, dynamic> args = <String, dynamic>{};
    Map<dynamic, dynamic>? result =
        await channel?.invokeMethod('requestImageRef', args);
    return result != null
        ? RequestImageRefResult(
            url: result['url'] != null ? WebUri(result['url']) : null,
          )
        : null;
  }

  @override
  Future<List<MetaTag>> getMetaTags() async {
    List<MetaTag> metaTags = [];

    List<Map<dynamic, dynamic>>? metaTagList =
        (await evaluateJavascript(source: """
(function() {
  var metaTags = [];
  var metaTagNodes = document.head.getElementsByTagName('meta');
  for (var i = 0; i < metaTagNodes.length; i++) {
    var metaTagNode = metaTagNodes[i];
    
    var otherAttributes = metaTagNode.getAttributeNames();
    var nameIndex = otherAttributes.indexOf("name");
    if (nameIndex !== -1) otherAttributes.splice(nameIndex, 1);
    var contentIndex = otherAttributes.indexOf("content");
    if (contentIndex !== -1) otherAttributes.splice(contentIndex, 1);
    
    var attrs = [];
    for (var j = 0; j < otherAttributes.length; j++) {
      var otherAttribute = otherAttributes[j];
      attrs.push(
        {
          name: otherAttribute,
          value: metaTagNode.getAttribute(otherAttribute)
        }
      );
    }

    metaTags.push(
      {
        name: metaTagNode.name,
        content: metaTagNode.content,
        attrs: attrs
      }
    );
  }
  return metaTags;
})();
    """))?.cast<Map<dynamic, dynamic>>();

    if (metaTagList == null) {
      return metaTags;
    }

    for (var metaTag in metaTagList) {
      var attrs = <MetaTagAttribute>[];

      for (var metaTagAttr in metaTag["attrs"]) {
        attrs.add(MetaTagAttribute(
            name: metaTagAttr["name"], value: metaTagAttr["value"]));
      }

      metaTags.add(MetaTag(
          name: metaTag["name"], content: metaTag["content"], attrs: attrs));
    }

    return metaTags;
  }

  @override
  Future<Color?> getMetaThemeColor() async {
    Color? themeColor;

    try {
      Map<String, dynamic> args = <String, dynamic>{};
      themeColor = UtilColor.fromStringRepresentation(
          await channel?.invokeMethod('getMetaThemeColor', args));
      return themeColor;
    } catch (e) {
      // not implemented
    }

    // try using javascript
    var metaTags = await getMetaTags();
    MetaTag? metaTagThemeColor;

    for (var metaTag in metaTags) {
      if (metaTag.name == "theme-color") {
        metaTagThemeColor = metaTag;
        break;
      }
    }

    if (metaTagThemeColor == null) {
      return null;
    }

    var colorValue = metaTagThemeColor.content;

    themeColor = colorValue != null
        ? UtilColor.fromStringRepresentation(colorValue)
        : null;

    return themeColor;
  }

  @override
  Future<int?> getScrollX() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<int?>('getScrollX', args);
  }

  @override
  Future<int?> getScrollY() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<int?>('getScrollY', args);
  }

  @override
  Future<SslCertificate?> getCertificate() async {
    Map<String, dynamic> args = <String, dynamic>{};
    Map<String, dynamic>? sslCertificateMap =
        (await channel?.invokeMethod('getCertificate', args))
            ?.cast<String, dynamic>();
    return SslCertificate.fromMap(sslCertificateMap);
  }

  @override
  Future<void> addUserScript({required UserScript userScript}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('userScript', () => userScript.toMap());
    if (!(_userScripts[userScript.injectionTime]?.contains(userScript) ??
        false)) {
      _userScripts[userScript.injectionTime]?.add(userScript);
      await channel?.invokeMethod('addUserScript', args);
    }
  }

  @override
  Future<void> addUserScripts({required List<UserScript> userScripts}) async {
    for (var i = 0; i < userScripts.length; i++) {
      await addUserScript(userScript: userScripts[i]);
    }
  }

  @override
  Future<bool> removeUserScript({required UserScript userScript}) async {
    var index = _userScripts[userScript.injectionTime]?.indexOf(userScript);
    if (index == null || index == -1) {
      return false;
    }

    _userScripts[userScript.injectionTime]?.remove(userScript);
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('userScript', () => userScript.toMap());
    args.putIfAbsent('index', () => index);
    await channel?.invokeMethod('removeUserScript', args);

    return true;
  }

  @override
  Future<void> removeUserScriptsByGroupName({required String groupName}) async {
    final List<UserScript> userScriptsAtDocumentStart = List.from(
        _userScripts[UserScriptInjectionTime.AT_DOCUMENT_START] ?? []);
    for (final userScript in userScriptsAtDocumentStart) {
      if (userScript.groupName == groupName) {
        _userScripts[userScript.injectionTime]?.remove(userScript);
      }
    }

    final List<UserScript> userScriptsAtDocumentEnd =
        List.from(_userScripts[UserScriptInjectionTime.AT_DOCUMENT_END] ?? []);
    for (final userScript in userScriptsAtDocumentEnd) {
      if (userScript.groupName == groupName) {
        _userScripts[userScript.injectionTime]?.remove(userScript);
      }
    }

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('groupName', () => groupName);
    await channel?.invokeMethod('removeUserScriptsByGroupName', args);
  }

  @override
  Future<void> removeUserScripts(
      {required List<UserScript> userScripts}) async {
    for (final userScript in userScripts) {
      await removeUserScript(userScript: userScript);
    }
  }

  @override
  Future<void> removeAllUserScripts() async {
    _userScripts[UserScriptInjectionTime.AT_DOCUMENT_START]?.clear();
    _userScripts[UserScriptInjectionTime.AT_DOCUMENT_END]?.clear();

    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('removeAllUserScripts', args);
  }

  @override
  bool hasUserScript({required UserScript userScript}) {
    return _userScripts[userScript.injectionTime]?.contains(userScript) ??
        false;
  }

  @override
  Future<CallAsyncJavaScriptResult?> callAsyncJavaScript(
      {required String functionBody,
      Map<String, dynamic> arguments = const <String, dynamic>{},
      ContentWorld? contentWorld}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('functionBody', () => functionBody);
    args.putIfAbsent('arguments', () => arguments);
    args.putIfAbsent('contentWorld', () => contentWorld?.toMap());
    var data = await channel?.invokeMethod('callAsyncJavaScript', args);
    if (data == null) {
      return null;
    }
    data = json.decode(data);
    return CallAsyncJavaScriptResult(
        value: data["value"], error: data["error"]);
  }

  @override
  Future<String?> saveWebArchive(
      {required String filePath, bool autoname = false}) async {
    if (!autoname) {
      assert(filePath.endsWith("." + WebArchiveFormat.MHT.toNativeValue()));
    }

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("filePath", () => filePath);
    args.putIfAbsent("autoname", () => autoname);
    return await channel?.invokeMethod<String?>('saveWebArchive', args);
  }

  @override
  Future<bool> isSecureContext() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('isSecureContext', args) ?? false;
  }

  @override
  Future<AndroidWebMessageChannel?> createWebMessageChannel() async {
    Map<String, dynamic> args = <String, dynamic>{};
    Map<String, dynamic>? result =
        (await channel?.invokeMethod('createWebMessageChannel', args))
            ?.cast<String, dynamic>();
    final webMessageChannel = AndroidWebMessageChannel.static().fromMap(result);
    if (webMessageChannel != null) {
      _webMessageChannels.add(webMessageChannel);
    }
    return webMessageChannel;
  }

  @override
  Future<void> postWebMessage(
      {required WebMessage message, WebUri? targetOrigin}) async {
    if (targetOrigin == null) {
      targetOrigin = WebUri('');
    }
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('message', () => message.toMap());
    args.putIfAbsent('targetOrigin', () => targetOrigin.toString());
    await channel?.invokeMethod('postWebMessage', args);
  }

  @override
  Future<void> addWebMessageListener(
      PlatformWebMessageListener webMessageListener) async {
    assert(!_webMessageListeners.contains(webMessageListener),
        "${webMessageListener} was already added.");
    assert(
        !_webMessageListenerObjNames
            .contains(webMessageListener.params.jsObjectName),
        "jsObjectName ${webMessageListener.params.jsObjectName} was already added.");
    _webMessageListeners.add(webMessageListener as AndroidWebMessageListener);
    _webMessageListenerObjNames.add(webMessageListener.params.jsObjectName);

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('webMessageListener', () => webMessageListener.toMap());
    await channel?.invokeMethod('addWebMessageListener', args);
  }

  @override
  bool hasWebMessageListener(PlatformWebMessageListener webMessageListener) {
    return _webMessageListeners.contains(webMessageListener) ||
        _webMessageListenerObjNames
            .contains(webMessageListener.params.jsObjectName);
  }

  @override
  Future<bool> canScrollVertically() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('canScrollVertically', args) ??
        false;
  }

  @override
  Future<bool> canScrollHorizontally() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('canScrollHorizontally', args) ??
        false;
  }

  @override
  Future<bool> startSafeBrowsing() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('startSafeBrowsing', args) ??
        false;
  }

  @override
  Future<void> clearSslPreferences() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('clearSslPreferences', args);
  }

  @override
  Future<void> pause() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('pause', args);
  }

  @override
  Future<void> resume() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('resume', args);
  }

  @override
  Future<bool> pageDown({required bool bottom}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("bottom", () => bottom);
    return await channel?.invokeMethod<bool>('pageDown', args) ?? false;
  }

  @override
  Future<bool> pageUp({required bool top}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("top", () => top);
    return await channel?.invokeMethod<bool>('pageUp', args) ?? false;
  }

  @override
  Future<bool> zoomIn() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('zoomIn', args) ?? false;
  }

  @override
  Future<bool> zoomOut() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('zoomOut', args) ?? false;
  }

  @override
  Future<void> clearHistory() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod('clearHistory', args);
  }

  @override
  Future<bool> isInFullscreen() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('isInFullscreen', args) ?? false;
  }

  @override
  Future<void> clearFormData() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod('clearFormData', args);
  }

  @override
  Future<String> getDefaultUserAgent() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await _staticChannel.invokeMethod<String>(
            'getDefaultUserAgent', args) ??
        '';
  }

  @override
  Future<void> clearClientCertPreferences() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await _staticChannel.invokeMethod('clearClientCertPreferences', args);
  }

  @override
  Future<WebUri?> getSafeBrowsingPrivacyPolicyUrl() async {
    Map<String, dynamic> args = <String, dynamic>{};
    String? url = await _staticChannel.invokeMethod(
        'getSafeBrowsingPrivacyPolicyUrl', args);
    return url != null ? WebUri(url) : null;
  }

  @override
  @Deprecated("Use setSafeBrowsingAllowlist instead")
  Future<bool> setSafeBrowsingWhitelist({required List<String> hosts}) async {
    return await setSafeBrowsingAllowlist(hosts: hosts);
  }

  @override
  Future<bool> setSafeBrowsingAllowlist({required List<String> hosts}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('hosts', () => hosts);
    return await _staticChannel.invokeMethod<bool>(
            'setSafeBrowsingAllowlist', args) ??
        false;
  }

  @override
  Future<WebViewPackageInfo?> getCurrentWebViewPackage() async {
    Map<String, dynamic> args = <String, dynamic>{};
    Map<String, dynamic>? packageInfo =
        (await _staticChannel.invokeMethod('getCurrentWebViewPackage', args))
            ?.cast<String, dynamic>();
    return WebViewPackageInfo.fromMap(packageInfo);
  }

  @override
  Future<void> setWebContentsDebuggingEnabled(bool debuggingEnabled) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('debuggingEnabled', () => debuggingEnabled);
    return await _staticChannel.invokeMethod(
        'setWebContentsDebuggingEnabled', args);
  }

  @override
  Future<String?> getVariationsHeader() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await _staticChannel.invokeMethod<String?>(
        'getVariationsHeader', args);
  }

  @override
  Future<bool> isMultiProcessEnabled() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await _staticChannel.invokeMethod<bool>(
            'isMultiProcessEnabled', args) ??
        false;
  }

  @override
  Future<void> disableWebView() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await _staticChannel.invokeMethod('disableWebView', args);
  }

  @override
  Future<void> disposeKeepAlive(InAppWebViewKeepAlive keepAlive) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('keepAliveId', () => keepAlive.id);
    await _staticChannel.invokeMethod('disposeKeepAlive', args);
    _keepAliveMap[keepAlive] = null;
  }

  @override
  Future<void> clearAllCache({bool includeDiskFiles = true}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('includeDiskFiles', () => includeDiskFiles);
    await _staticChannel.invokeMethod('clearAllCache', args);
  }

  @override
  Future<String> get tRexRunnerHtml async => await rootBundle.loadString(
      'packages/flutter_inappwebview/assets/t_rex_runner/t-rex.html');

  @override
  Future<String> get tRexRunnerCss async => await rootBundle.loadString(
      'packages/flutter_inappwebview/assets/t_rex_runner/t-rex.css');

  @override
  dynamic getViewId() {
    return id;
  }

  @override
  void dispose({bool isKeepAlive = false}) {
    disposeChannel(removeMethodCallHandler: !isKeepAlive);
    _inAppBrowser = null;
    webStorage.dispose();
    if (!isKeepAlive) {
      _controllerFromPlatform = null;
      _javaScriptHandlersMap.clear();
      _userScripts.clear();
      _webMessageListenerObjNames.clear();
      _injectedScriptsFromURL.clear();
      for (final webMessageChannel in _webMessageChannels) {
        webMessageChannel.dispose();
      }
      _webMessageChannels.clear();
      for (final webMessageListener in _webMessageListeners) {
        webMessageListener.dispose();
      }
      _webMessageListeners.clear();
    }
  }
}

extension InternalInAppWebViewController on AndroidInAppWebViewController {
  get handleMethod => _handleMethod;
}
