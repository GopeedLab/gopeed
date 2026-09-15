import 'dart:ui';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';
import '../find_interaction/find_interaction_controller.dart';
import '../pull_to_refresh/pull_to_refresh_controller.dart';
import 'in_app_webview_controller.dart';

/// Object specifying creation parameters for creating a [AndroidHeadlessInAppWebView].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformHeadlessInAppWebViewCreationParams] for
/// more information.
@immutable
class AndroidHeadlessInAppWebViewCreationParams
    extends PlatformHeadlessInAppWebViewCreationParams {
  /// Creates a new [AndroidHeadlessInAppWebViewCreationParams] instance.
  AndroidHeadlessInAppWebViewCreationParams(
      {super.controllerFromPlatform,
      super.initialSize,
      super.windowId,
      super.onWebViewCreated,
      super.onLoadStart,
      super.onLoadStop,
      @Deprecated('Use onReceivedError instead') super.onLoadError,
      super.onReceivedError,
      @Deprecated("Use onReceivedHttpError instead") super.onLoadHttpError,
      super.onReceivedHttpError,
      super.onProgressChanged,
      super.onConsoleMessage,
      super.shouldOverrideUrlLoading,
      super.onLoadResource,
      super.onScrollChanged,
      @Deprecated('Use onDownloadStartRequest instead') super.onDownloadStart,
      super.onDownloadStartRequest,
      @Deprecated('Use onLoadResourceWithCustomScheme instead')
      super.onLoadResourceCustomScheme,
      super.onLoadResourceWithCustomScheme,
      super.onCreateWindow,
      super.onCloseWindow,
      super.onJsAlert,
      super.onJsConfirm,
      super.onJsPrompt,
      super.onReceivedHttpAuthRequest,
      super.onReceivedServerTrustAuthRequest,
      super.onReceivedClientCertRequest,
      @Deprecated('Use FindInteractionController.onFindResultReceived instead')
      super.onFindResultReceived,
      super.shouldInterceptAjaxRequest,
      super.onAjaxReadyStateChange,
      super.onAjaxProgress,
      super.shouldInterceptFetchRequest,
      super.onUpdateVisitedHistory,
      @Deprecated("Use onPrintRequest instead") super.onPrint,
      super.onPrintRequest,
      super.onLongPressHitTestResult,
      super.onEnterFullscreen,
      super.onExitFullscreen,
      super.onPageCommitVisible,
      super.onTitleChanged,
      super.onWindowFocus,
      super.onWindowBlur,
      super.onOverScrolled,
      super.onZoomScaleChanged,
      @Deprecated('Use onSafeBrowsingHit instead')
      super.androidOnSafeBrowsingHit,
      super.onSafeBrowsingHit,
      @Deprecated('Use onPermissionRequest instead')
      super.androidOnPermissionRequest,
      super.onPermissionRequest,
      @Deprecated('Use onGeolocationPermissionsShowPrompt instead')
      super.androidOnGeolocationPermissionsShowPrompt,
      super.onGeolocationPermissionsShowPrompt,
      @Deprecated('Use onGeolocationPermissionsHidePrompt instead')
      super.androidOnGeolocationPermissionsHidePrompt,
      super.onGeolocationPermissionsHidePrompt,
      @Deprecated('Use shouldInterceptRequest instead')
      super.androidShouldInterceptRequest,
      super.shouldInterceptRequest,
      @Deprecated('Use onRenderProcessGone instead')
      super.androidOnRenderProcessGone,
      super.onRenderProcessGone,
      @Deprecated('Use onRenderProcessResponsive instead')
      super.androidOnRenderProcessResponsive,
      super.onRenderProcessResponsive,
      @Deprecated('Use onRenderProcessUnresponsive instead')
      super.androidOnRenderProcessUnresponsive,
      super.onRenderProcessUnresponsive,
      @Deprecated('Use onFormResubmission instead')
      super.androidOnFormResubmission,
      super.onFormResubmission,
      @Deprecated('Use onZoomScaleChanged instead') super.androidOnScaleChanged,
      @Deprecated('Use onReceivedIcon instead') super.androidOnReceivedIcon,
      super.onReceivedIcon,
      @Deprecated('Use onReceivedTouchIconUrl instead')
      super.androidOnReceivedTouchIconUrl,
      super.onReceivedTouchIconUrl,
      @Deprecated('Use onJsBeforeUnload instead') super.androidOnJsBeforeUnload,
      super.onJsBeforeUnload,
      @Deprecated('Use onReceivedLoginRequest instead')
      super.androidOnReceivedLoginRequest,
      super.onReceivedLoginRequest,
      super.onPermissionRequestCanceled,
      super.onRequestFocus,
      @Deprecated('Use onWebContentProcessDidTerminate instead')
      super.iosOnWebContentProcessDidTerminate,
      super.onWebContentProcessDidTerminate,
      @Deprecated(
          'Use onDidReceiveServerRedirectForProvisionalNavigation instead')
      super.iosOnDidReceiveServerRedirectForProvisionalNavigation,
      super.onDidReceiveServerRedirectForProvisionalNavigation,
      @Deprecated('Use onNavigationResponse instead')
      super.iosOnNavigationResponse,
      super.onNavigationResponse,
      @Deprecated('Use shouldAllowDeprecatedTLS instead')
      super.iosShouldAllowDeprecatedTLS,
      super.shouldAllowDeprecatedTLS,
      super.onCameraCaptureStateChanged,
      super.onMicrophoneCaptureStateChanged,
      super.onContentSizeChanged,
      super.initialUrlRequest,
      super.initialFile,
      super.initialData,
      @Deprecated('Use initialSettings instead') super.initialOptions,
      super.initialSettings,
      super.contextMenu,
      super.initialUserScripts,
      this.pullToRefreshController,
      this.findInteractionController});

  /// Creates a [AndroidHeadlessInAppWebViewCreationParams] instance based on [PlatformHeadlessInAppWebViewCreationParams].
  AndroidHeadlessInAppWebViewCreationParams.fromPlatformHeadlessInAppWebViewCreationParams(
      PlatformHeadlessInAppWebViewCreationParams params)
      : this(
            controllerFromPlatform: params.controllerFromPlatform,
            initialSize: params.initialSize,
            windowId: params.windowId,
            onWebViewCreated: params.onWebViewCreated,
            onLoadStart: params.onLoadStart,
            onLoadStop: params.onLoadStop,
            onLoadError: params.onLoadError,
            onReceivedError: params.onReceivedError,
            onLoadHttpError: params.onLoadHttpError,
            onReceivedHttpError: params.onReceivedHttpError,
            onProgressChanged: params.onProgressChanged,
            onConsoleMessage: params.onConsoleMessage,
            shouldOverrideUrlLoading: params.shouldOverrideUrlLoading,
            onLoadResource: params.onLoadResource,
            onScrollChanged: params.onScrollChanged,
            onDownloadStart: params.onDownloadStart,
            onDownloadStartRequest: params.onDownloadStartRequest,
            onLoadResourceCustomScheme: params.onLoadResourceCustomScheme,
            onLoadResourceWithCustomScheme:
                params.onLoadResourceWithCustomScheme,
            onCreateWindow: params.onCreateWindow,
            onCloseWindow: params.onCloseWindow,
            onJsAlert: params.onJsAlert,
            onJsConfirm: params.onJsConfirm,
            onJsPrompt: params.onJsPrompt,
            onReceivedHttpAuthRequest: params.onReceivedHttpAuthRequest,
            onReceivedServerTrustAuthRequest:
                params.onReceivedServerTrustAuthRequest,
            onReceivedClientCertRequest: params.onReceivedClientCertRequest,
            onFindResultReceived: params.onFindResultReceived,
            shouldInterceptAjaxRequest: params.shouldInterceptAjaxRequest,
            onAjaxReadyStateChange: params.onAjaxReadyStateChange,
            onAjaxProgress: params.onAjaxProgress,
            shouldInterceptFetchRequest: params.shouldInterceptFetchRequest,
            onUpdateVisitedHistory: params.onUpdateVisitedHistory,
            onPrint: params.onPrint,
            onPrintRequest: params.onPrintRequest,
            onLongPressHitTestResult: params.onLongPressHitTestResult,
            onEnterFullscreen: params.onEnterFullscreen,
            onExitFullscreen: params.onExitFullscreen,
            onPageCommitVisible: params.onPageCommitVisible,
            onTitleChanged: params.onTitleChanged,
            onWindowFocus: params.onWindowFocus,
            onWindowBlur: params.onWindowBlur,
            onOverScrolled: params.onOverScrolled,
            onZoomScaleChanged: params.onZoomScaleChanged,
            androidOnSafeBrowsingHit: params.androidOnSafeBrowsingHit,
            onSafeBrowsingHit: params.onSafeBrowsingHit,
            androidOnPermissionRequest: params.androidOnPermissionRequest,
            onPermissionRequest: params.onPermissionRequest,
            androidOnGeolocationPermissionsShowPrompt:
                params.androidOnGeolocationPermissionsShowPrompt,
            onGeolocationPermissionsShowPrompt:
                params.onGeolocationPermissionsShowPrompt,
            androidOnGeolocationPermissionsHidePrompt:
                params.androidOnGeolocationPermissionsHidePrompt,
            onGeolocationPermissionsHidePrompt:
                params.onGeolocationPermissionsHidePrompt,
            androidShouldInterceptRequest: params.androidShouldInterceptRequest,
            shouldInterceptRequest: params.shouldInterceptRequest,
            androidOnRenderProcessGone: params.androidOnRenderProcessGone,
            onRenderProcessGone: params.onRenderProcessGone,
            androidOnRenderProcessResponsive:
                params.androidOnRenderProcessResponsive,
            onRenderProcessResponsive: params.onRenderProcessResponsive,
            androidOnRenderProcessUnresponsive:
                params.androidOnRenderProcessUnresponsive,
            onRenderProcessUnresponsive: params.onRenderProcessUnresponsive,
            androidOnFormResubmission: params.androidOnFormResubmission,
            onFormResubmission: params.onFormResubmission,
            androidOnScaleChanged: params.androidOnScaleChanged,
            androidOnReceivedIcon: params.androidOnReceivedIcon,
            onReceivedIcon: params.onReceivedIcon,
            androidOnReceivedTouchIconUrl: params.androidOnReceivedTouchIconUrl,
            onReceivedTouchIconUrl: params.onReceivedTouchIconUrl,
            androidOnJsBeforeUnload: params.androidOnJsBeforeUnload,
            onJsBeforeUnload: params.onJsBeforeUnload,
            androidOnReceivedLoginRequest: params.androidOnReceivedLoginRequest,
            onReceivedLoginRequest: params.onReceivedLoginRequest,
            onPermissionRequestCanceled: params.onPermissionRequestCanceled,
            onRequestFocus: params.onRequestFocus,
            iosOnWebContentProcessDidTerminate:
                params.iosOnWebContentProcessDidTerminate,
            onWebContentProcessDidTerminate:
                params.onWebContentProcessDidTerminate,
            iosOnDidReceiveServerRedirectForProvisionalNavigation:
                params.iosOnDidReceiveServerRedirectForProvisionalNavigation,
            onDidReceiveServerRedirectForProvisionalNavigation:
                params.onDidReceiveServerRedirectForProvisionalNavigation,
            iosOnNavigationResponse: params.iosOnNavigationResponse,
            onNavigationResponse: params.onNavigationResponse,
            iosShouldAllowDeprecatedTLS: params.iosShouldAllowDeprecatedTLS,
            shouldAllowDeprecatedTLS: params.shouldAllowDeprecatedTLS,
            onCameraCaptureStateChanged: params.onCameraCaptureStateChanged,
            onMicrophoneCaptureStateChanged:
                params.onMicrophoneCaptureStateChanged,
            onContentSizeChanged: params.onContentSizeChanged,
            initialUrlRequest: params.initialUrlRequest,
            initialFile: params.initialFile,
            initialData: params.initialData,
            initialOptions: params.initialOptions,
            initialSettings: params.initialSettings,
            contextMenu: params.contextMenu,
            initialUserScripts: params.initialUserScripts,
            pullToRefreshController: params.pullToRefreshController
                as AndroidPullToRefreshController?,
            findInteractionController: params.findInteractionController
                as AndroidFindInteractionController?);

  @override
  final AndroidFindInteractionController? findInteractionController;

  @override
  final AndroidPullToRefreshController? pullToRefreshController;
}

///{@macro flutter_inappwebview_platform_interface.PlatformHeadlessInAppWebView}
class AndroidHeadlessInAppWebView extends PlatformHeadlessInAppWebView
    with ChannelController {
  @override
  late final String id;

  bool _started = false;
  bool _running = false;

  static const MethodChannel _sharedChannel =
      const MethodChannel('com.pichillilorenzo/flutter_headless_inappwebview');

  AndroidInAppWebViewController? _webViewController;

  /// Constructs a [AndroidHeadlessInAppWebView].
  AndroidHeadlessInAppWebView(PlatformHeadlessInAppWebViewCreationParams params)
      : super.implementation(
          params is AndroidHeadlessInAppWebViewCreationParams
              ? params
              : AndroidHeadlessInAppWebViewCreationParams
                  .fromPlatformHeadlessInAppWebViewCreationParams(params),
        ) {
    id = IdGenerator.generate();
  }

  @override
  AndroidInAppWebViewController? get webViewController => _webViewController;

  dynamic _controllerFromPlatform;

  AndroidHeadlessInAppWebViewCreationParams get _androidParams =>
      params as AndroidHeadlessInAppWebViewCreationParams;

  _init() {
    _webViewController = AndroidInAppWebViewController(
      AndroidInAppWebViewControllerCreationParams(
          id: id, webviewParams: params),
    );
    _controllerFromPlatform =
        params.controllerFromPlatform?.call(_webViewController!) ??
            _webViewController!;
    _androidParams.pullToRefreshController?.init(id);
    _androidParams.findInteractionController?.init(id);
    channel =
        MethodChannel('com.pichillilorenzo/flutter_headless_inappwebview_$id');
    handler = _handleMethod;
    initMethodCallHandler();
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    switch (call.method) {
      case "onWebViewCreated":
        if (params.onWebViewCreated != null && _webViewController != null) {
          params.onWebViewCreated!(_controllerFromPlatform);
        }
        break;
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }
    return null;
  }

  Future<void> run() async {
    if (_started) {
      return;
    }
    _started = true;
    _init();

    final initialSettings = params.initialSettings ?? InAppWebViewSettings();
    _inferInitialSettings(initialSettings);

    Map<String, dynamic> settingsMap =
        (params.initialSettings != null ? initialSettings.toMap() : null) ??
            params.initialOptions?.toMap() ??
            initialSettings.toMap();

    Map<String, dynamic> pullToRefreshSettings =
        _androidParams.pullToRefreshController?.params.settings.toMap() ??
            _androidParams.pullToRefreshController?.params.options.toMap() ??
            PullToRefreshSettings(enabled: false).toMap();

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('id', () => id);
    args.putIfAbsent(
        'params',
        () => <String, dynamic>{
              'initialUrlRequest': params.initialUrlRequest?.toMap(),
              'initialFile': params.initialFile,
              'initialData': params.initialData?.toMap(),
              'initialSettings': settingsMap,
              'contextMenu': params.contextMenu?.toMap() ?? {},
              'windowId': params.windowId,
              'initialUserScripts':
                  params.initialUserScripts?.map((e) => e.toMap()).toList() ??
                      [],
              'pullToRefreshSettings': pullToRefreshSettings,
              'initialSize': params.initialSize.toMap()
            });
    await _sharedChannel.invokeMethod('run', args);
    _running = true;
  }

  void _inferInitialSettings(InAppWebViewSettings settings) {
    if (params.shouldOverrideUrlLoading != null &&
        settings.useShouldOverrideUrlLoading == null) {
      settings.useShouldOverrideUrlLoading = true;
    }
    if (params.onLoadResource != null && settings.useOnLoadResource == null) {
      settings.useOnLoadResource = true;
    }
    if (params.onDownloadStartRequest != null &&
        settings.useOnDownloadStart == null) {
      settings.useOnDownloadStart = true;
    }
    if (params.shouldInterceptAjaxRequest != null &&
        settings.useShouldInterceptAjaxRequest == null) {
      settings.useShouldInterceptAjaxRequest = true;
    }
    if (params.shouldInterceptFetchRequest != null &&
        settings.useShouldInterceptFetchRequest == null) {
      settings.useShouldInterceptFetchRequest = true;
    }
    if (params.shouldInterceptRequest != null &&
        settings.useShouldInterceptRequest == null) {
      settings.useShouldInterceptRequest = true;
    }
    if (params.onRenderProcessGone != null &&
        settings.useOnRenderProcessGone == null) {
      settings.useOnRenderProcessGone = true;
    }
    if (params.onNavigationResponse != null &&
        settings.useOnNavigationResponse == null) {
      settings.useOnNavigationResponse = true;
    }
  }

  @override
  bool isRunning() {
    return _running;
  }

  @override
  Future<void> setSize(Size size) async {
    if (!_running) {
      return;
    }

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('size', () => size.toMap());
    await channel?.invokeMethod('setSize', args);
  }

  @override
  Future<Size?> getSize() async {
    if (!_running) {
      return null;
    }

    Map<String, dynamic> args = <String, dynamic>{};
    Map<String, dynamic> sizeMap =
        (await channel?.invokeMethod('getSize', args))?.cast<String, dynamic>();
    return MapSize.fromMap(sizeMap);
  }

  @override
  Future<void> dispose() async {
    if (!_running) {
      return;
    }
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('dispose', args);
    disposeChannel();
    _started = false;
    _running = false;
    _webViewController?.dispose();
    _webViewController = null;
    _controllerFromPlatform = null;
    _androidParams.pullToRefreshController?.dispose();
    _androidParams.findInteractionController?.dispose();
  }
}

extension InternalHeadlessInAppWebView on AndroidHeadlessInAppWebView {
  Future<void> internalDispose() async {
    _started = false;
    _running = false;
  }
}
