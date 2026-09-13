import 'dart:async';
import 'dart:collection';
import 'dart:typed_data';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [AndroidChromeSafariBrowser].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformChromeSafariBrowserCreationParams] for
/// more information.
@immutable
class AndroidChromeSafariBrowserCreationParams
    extends PlatformChromeSafariBrowserCreationParams {
  /// Creates a new [AndroidChromeSafariBrowserCreationParams] instance.
  const AndroidChromeSafariBrowserCreationParams();

  /// Creates a [AndroidChromeSafariBrowserCreationParams] instance based on [PlatformChromeSafariBrowserCreationParams].
  factory AndroidChromeSafariBrowserCreationParams.fromPlatformChromeSafariBrowserCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformChromeSafariBrowserCreationParams params) {
    return AndroidChromeSafariBrowserCreationParams();
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformChromeSafariBrowser}
class AndroidChromeSafariBrowser extends PlatformChromeSafariBrowser
    with ChannelController {
  @override
  final String id = IdGenerator.generate();

  /// Constructs a [AndroidChromeSafariBrowser].
  AndroidChromeSafariBrowser(PlatformChromeSafariBrowserCreationParams params)
      : super.implementation(
          params is AndroidChromeSafariBrowserCreationParams
              ? params
              : AndroidChromeSafariBrowserCreationParams
                  .fromPlatformChromeSafariBrowserCreationParams(params),
        );

  static final AndroidChromeSafariBrowser _staticValue =
      AndroidChromeSafariBrowser(AndroidChromeSafariBrowserCreationParams());

  /// Provide static access.
  factory AndroidChromeSafariBrowser.static() {
    return _staticValue;
  }

  ChromeSafariBrowserActionButton? _actionButton;
  Map<int, ChromeSafariBrowserMenuItem> _menuItems = new HashMap();
  ChromeSafariBrowserSecondaryToolbar? _secondaryToolbar;
  bool _isOpened = false;
  static const MethodChannel _staticChannel =
      const MethodChannel('com.pichillilorenzo/flutter_chromesafaribrowser');

  _init() {
    channel =
        MethodChannel('com.pichillilorenzo/flutter_chromesafaribrowser_$id');
    handler = _handleMethod;
    initMethodCallHandler();
  }

  _debugLog(String method, dynamic args) {
    debugLog(
        className: this.runtimeType.toString(),
        id: id,
        debugLoggingSettings: PlatformChromeSafariBrowser.debugLoggingSettings,
        method: method,
        args: args);
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    _debugLog(call.method, call.arguments);

    switch (call.method) {
      case "onServiceConnected":
        eventHandler?.onServiceConnected();
        break;
      case "onOpened":
        eventHandler?.onOpened();
        break;
      case "onCompletedInitialLoad":
        final bool? didLoadSuccessfully = call.arguments["didLoadSuccessfully"];
        eventHandler?.onCompletedInitialLoad(didLoadSuccessfully);
        break;
      case "onNavigationEvent":
        final navigationEvent = CustomTabsNavigationEventType.fromNativeValue(
            call.arguments["navigationEvent"]);
        eventHandler?.onNavigationEvent(navigationEvent);
        break;
      case "onRelationshipValidationResult":
        final relation =
            CustomTabsRelationType.fromNativeValue(call.arguments["relation"]);
        final requestedOrigin = call.arguments["requestedOrigin"] != null
            ? WebUri(call.arguments["requestedOrigin"])
            : null;
        final bool result = call.arguments["result"];
        eventHandler?.onRelationshipValidationResult(
            relation, requestedOrigin, result);
        break;
      case "onClosed":
        _isOpened = false;
        final onClosed = eventHandler?.onClosed;
        dispose();
        onClosed?.call();
        break;
      case "onItemActionPerform":
        String url = call.arguments["url"];
        String title = call.arguments["title"];
        int id = call.arguments["id"].toInt();
        if (this._actionButton?.id == id) {
          if (this._actionButton?.action != null) {
            this._actionButton?.action!(url, title);
          }
          if (this._actionButton?.onClick != null) {
            this._actionButton?.onClick!(WebUri(url), title);
          }
        } else if (this._menuItems[id] != null) {
          if (this._menuItems[id]?.action != null) {
            this._menuItems[id]?.action!(url, title);
          }
          if (this._menuItems[id]?.onClick != null) {
            this._menuItems[id]?.onClick!(WebUri(url), title);
          }
        }
        break;
      case "onSecondaryItemActionPerform":
        final clickableIDs = this._secondaryToolbar?.clickableIDs;
        if (clickableIDs != null) {
          WebUri? url = call.arguments["url"] != null
              ? WebUri(call.arguments["url"])
              : null;
          String name = call.arguments["name"];
          for (final clickable in clickableIDs) {
            var clickableFullname = clickable.id.name;
            if (clickable.id.defType != null &&
                !clickableFullname.contains("/")) {
              clickableFullname = "${clickable.id.defType}/$clickableFullname";
            }
            if (clickable.id.defPackage != null &&
                !clickableFullname.contains(":")) {
              clickableFullname =
                  "${clickable.id.defPackage}:$clickableFullname";
            }
            if (clickableFullname == name) {
              if (clickable.onClick != null) {
                clickable.onClick!(url);
              }
              break;
            }
          }
        }
        break;
      case "onMessageChannelReady":
        eventHandler?.onMessageChannelReady();
        break;
      case "onPostMessage":
        final String message = call.arguments["message"];
        eventHandler?.onPostMessage(message);
        break;
      case "onVerticalScrollEvent":
        final bool isDirectionUp = call.arguments["isDirectionUp"];
        eventHandler?.onVerticalScrollEvent(isDirectionUp);
        break;
      case "onGreatestScrollPercentageIncreased":
        final int scrollPercentage = call.arguments["scrollPercentage"];
        eventHandler?.onGreatestScrollPercentageIncreased(scrollPercentage);
        break;
      case "onSessionEnded":
        final bool didUserInteract = call.arguments["didUserInteract"];
        eventHandler?.onSessionEnded(didUserInteract);
        break;
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }
  }

  @override
  Future<void> open(
      {WebUri? url,
      Map<String, String>? headers,
      List<WebUri>? otherLikelyURLs,
      WebUri? referrer,
      @Deprecated('Use settings instead')
      // ignore: deprecated_member_use_from_same_package
      ChromeSafariBrowserClassOptions? options,
      ChromeSafariBrowserSettings? settings}) async {
    assert(!_isOpened, 'The browser is already opened.');
    _isOpened = true;

    if (Util.isIOS) {
      assert(url != null, 'The specified URL must not be null on iOS.');
      assert(['http', 'https'].contains(url!.scheme),
          'The specified URL has an unsupported scheme. Only HTTP and HTTPS URLs are supported on iOS.');
    }
    if (url != null) {
      assert(url.toString().isNotEmpty, 'The specified URL must not be empty.');
    }

    _init();

    List<Map<String, dynamic>> menuItemList = [];
    _menuItems.forEach((key, value) {
      menuItemList.add(value.toMap());
    });

    var initialSettings = settings?.toMap() ??
        options?.toMap() ??
        ChromeSafariBrowserSettings().toMap();

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('id', () => id);
    args.putIfAbsent('url', () => url?.toString());
    args.putIfAbsent('headers', () => headers);
    args.putIfAbsent('otherLikelyURLs',
        () => otherLikelyURLs?.map((e) => e.toString()).toList());
    args.putIfAbsent('referrer', () => referrer?.toString());
    args.putIfAbsent('settings', () => initialSettings);
    args.putIfAbsent('actionButton', () => _actionButton?.toMap());
    args.putIfAbsent('secondaryToolbar', () => _secondaryToolbar?.toMap());
    args.putIfAbsent('menuItemList', () => menuItemList);
    await _staticChannel.invokeMethod('open', args);
  }

  @override
  Future<void> launchUrl({
    required WebUri url,
    Map<String, String>? headers,
    List<WebUri>? otherLikelyURLs,
    WebUri? referrer,
  }) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('url', () => url.toString());
    args.putIfAbsent('headers', () => headers);
    args.putIfAbsent('otherLikelyURLs',
        () => otherLikelyURLs?.map((e) => e.toString()).toList());
    args.putIfAbsent('referrer', () => referrer?.toString());
    await channel?.invokeMethod("launchUrl", args);
  }

  @override
  Future<bool> mayLaunchUrl(
      {WebUri? url, List<WebUri>? otherLikelyURLs}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('url', () => url?.toString());
    args.putIfAbsent('otherLikelyURLs',
        () => otherLikelyURLs?.map((e) => e.toString()).toList());
    return await channel?.invokeMethod<bool>("mayLaunchUrl", args) ?? false;
  }

  @override
  Future<bool> validateRelationship(
      {required CustomTabsRelationType relation,
      required WebUri origin}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('relation', () => relation.toNativeValue());
    args.putIfAbsent('origin', () => origin.toString());
    return await channel?.invokeMethod<bool>("validateRelationship", args) ??
        false;
  }

  @override
  Future<void> close() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod("close", args);
  }

  @override
  void setActionButton(ChromeSafariBrowserActionButton actionButton) {
    this._actionButton = actionButton;
  }

  @override
  Future<void> updateActionButton(
      {required Uint8List icon, required String description}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('icon', () => icon);
    args.putIfAbsent('description', () => description);
    await channel?.invokeMethod("updateActionButton", args);
    _actionButton?.icon = icon;
    _actionButton?.description = description;
  }

  @override
  void setSecondaryToolbar(
      ChromeSafariBrowserSecondaryToolbar secondaryToolbar) {
    this._secondaryToolbar = secondaryToolbar;
  }

  @override
  Future<void> updateSecondaryToolbar(
      ChromeSafariBrowserSecondaryToolbar secondaryToolbar) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('secondaryToolbar', () => secondaryToolbar.toMap());
    await channel?.invokeMethod("updateSecondaryToolbar", args);
    this._secondaryToolbar = secondaryToolbar;
  }

  @override
  void addMenuItem(ChromeSafariBrowserMenuItem menuItem) {
    this._menuItems[menuItem.id] = menuItem;
  }

  @override
  void addMenuItems(List<ChromeSafariBrowserMenuItem> menuItems) {
    menuItems.forEach((menuItem) {
      this._menuItems[menuItem.id] = menuItem;
    });
  }

  @override
  Future<bool> requestPostMessageChannel(
      {required WebUri sourceOrigin, WebUri? targetOrigin}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("sourceOrigin", () => sourceOrigin.toString());
    args.putIfAbsent("targetOrigin", () => targetOrigin.toString());
    return await channel?.invokeMethod<bool>(
            "requestPostMessageChannel", args) ??
        false;
  }

  @override
  Future<CustomTabsPostMessageResultType> postMessage(String message) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("message", () => message);
    return CustomTabsPostMessageResultType.fromNativeValue(
            await channel?.invokeMethod<int>("postMessage", args)) ??
        CustomTabsPostMessageResultType.FAILURE_MESSAGING_ERROR;
  }

  @override
  Future<bool> isEngagementSignalsApiAvailable() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>(
            "isEngagementSignalsApiAvailable", args) ??
        false;
  }

  @override
  Future<bool> isAvailable() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await _staticChannel.invokeMethod<bool>("isAvailable", args) ??
        false;
  }

  @override
  Future<int> getMaxToolbarItems() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await _staticChannel.invokeMethod<int>("getMaxToolbarItems", args) ??
        0;
  }

  @override
  Future<String?> getPackageName(
      {List<String>? packages, bool ignoreDefault = false}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("packages", () => packages);
    args.putIfAbsent("ignoreDefault", () => ignoreDefault);
    return await _staticChannel.invokeMethod<String?>("getPackageName", args);
  }

  @override
  bool isOpened() {
    return _isOpened;
  }

  @override
  @mustCallSuper
  void dispose() {
    super.dispose();
    disposeChannel();
  }
}
