import 'dart:async';
import 'dart:convert';

import 'package:app_links/app_links.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:share_handler/share_handler.dart';
import 'package:window_manager/window_manager.dart';

import '../../api/model/create_task.dart';
import '../../api/model/install_extension.dart';
import '../../api/model/request.dart';
import '../../core/entry/app_startup_options.dart';
import '../../core/utils/content_uri_resolver.dart';
import '../../core/window/app_window_launcher.dart';
import '../../features/extensions/application/pending_extension_install.dart';
import '../../features/tasks/application/pending_create_task.dart';
import '../../util/util.dart';
import '../router/app_router.dart';
import 'app_runtime_controller.dart';

final appDeepLinkControllerProvider = AsyncNotifierProvider<AppDeepLinkController, AppDeepLinkState>(
  AppDeepLinkController.new,
);

class AppDeepLinkState {
  const AppDeepLinkState({this.started = false});

  final bool started;
}

class AppDeepLinkController extends AsyncNotifier<AppDeepLinkState> {
  StreamSubscription<Uri>? _subscription;
  StreamSubscription<SharedMedia>? _sharedMediaSubscription;
  bool _started = false;

  @override
  Future<AppDeepLinkState> build() async {
    final runtime = ref.watch(appRuntimeControllerProvider).value;
    if (runtime == null || _started || kIsWeb) {
      return AppDeepLinkState(started: _started);
    }
    _started = true;
    final appLinks = AppLinks();
    _subscription = appLinks.uriLinkStream.listen((uri) {
      unawaited(_handleUri(uri));
    });
    if (Util.isMobile()) {
      final shareHandler = ShareHandlerPlatform.instance;
      _sharedMediaSubscription = shareHandler.sharedMediaStream.listen((media) {
        unawaited(_handleSharedMedia(media, ignoreContentUri: true));
      });
      try {
        final media = await shareHandler.getInitialSharedMedia();
        if (media != null) {
          await _handleSharedMedia(media);
        }
      } catch (_) {}
    }
    ref.onDispose(() {
      _subscription?.cancel();
      _sharedMediaSubscription?.cancel();
    });
    try {
      final initial = await appLinks.getInitialLink();
      if (initial != null) {
        await _handleUri(initial);
      }
    } catch (_) {}
    return const AppDeepLinkState(started: true);
  }

  Future<void> _handleSharedMedia(SharedMedia media, {bool ignoreContentUri = false}) async {
    final uri = sharedMediaUri(media);
    if (uri == null || (ignoreContentUri && uri.scheme == 'content')) return;
    await _handleUri(uri);
  }

  Future<void> _handleUri(Uri uri) async {
    if (uri.scheme == 'gopeed') {
      await _handleGopeedUri(uri);
      return;
    }
    final createTask = CreateTask(req: Request(url: await _uriToTaskUrl(uri)));
    await _openCreate(createTask);
  }

  Future<void> _handleGopeedUri(Uri uri) async {
    if (isSilentGopeedWakeUri(uri)) {
      return;
    }
    final route = gopeedDeepLinkRoute(uri);
    if (route == '/create') {
      final params = uri.queryParameters['params'];
      if (params?.isNotEmpty == true) {
        await _openCreate(CreateTask.fromJson(_decodeParams(params!)));
        return;
      }
      await _openCreate(null);
      return;
    }

    if (route == '/extension') {
      final params = uri.queryParameters['params'];
      if (params?.isNotEmpty == true) {
        ref.read(pendingExtensionInstallProvider.notifier).set(InstallExtension.fromJson(_decodeParams(params!)));
      }
      await _go('/extensions');
      return;
    }

    await _go('/');
  }

  Future<void> _openCreate(CreateTask? createTask) async {
    if (Util.isDesktop()) {
      final opened = await AppWindowLauncher.openCreateTaskWindow(createTask: createTask);
      if (!opened) {
        ref.read(pendingCreateTaskProvider.notifier).set(createTask);
        await _go('/create');
      }
      return;
    }
    ref.read(pendingCreateTaskProvider.notifier).set(createTask);
    await _go('/create');
  }

  Future<void> _go(String route) async {
    if (Util.isDesktop()) {
      await windowManager.show();
      await windowManager.focus();
    }
    final context = AppRouter.rootNavigatorKey.currentContext;
    if (context != null && context.mounted) {
      context.go(route);
    }
  }

  Map<String, dynamic> _decodeParams(String params) {
    final safeParams = params.replaceAll('"', '').replaceAll(' ', '+');
    final paramsJson = String.fromCharCodes(base64Decode(base64.normalize(safeParams)));
    return jsonDecode(paramsJson) as Map<String, dynamic>;
  }

  Future<String> _uriToTaskUrl(Uri uri) async {
    if (uri.scheme == 'magnet' || uri.scheme == 'http' || uri.scheme == 'https') {
      return uri.toString();
    }
    if (uri.scheme == 'file') {
      return Util.isWindows() ? Uri.decodeFull(uri.path.substring(1)) : uri.path;
    }
    if (uri.scheme == 'content' && Util.isAndroid()) {
      return ContentUriResolver.copyToCache(uri);
    }
    throw ArgumentError.value(uri, 'uri', 'Unsupported task URI scheme');
  }
}

/// Picks the actionable URI from content received through the platform share UI.
///
/// File attachments take precedence over an optional text caption. Both the
/// Android and iOS share-handler implementations expose attachments as local
/// paths that are ready for the app to read.
Uri? sharedMediaUri(SharedMedia media) {
  final attachments = media.attachments;
  if (attachments != null) {
    for (final attachment in attachments) {
      final path = attachment?.path.trim();
      if (path != null && path.isNotEmpty) return Uri.file(path);
    }
  }

  final content = media.content?.trim();
  if (content == null || content.isEmpty) return null;
  return Uri.tryParse(content);
}

/// Gopeed's established links use `gopeed:///create`, where the action is the
/// URI path. Do not interpret the URI host as an action: the host-style
/// `gopeed://create` form is intentionally unsupported.
String gopeedDeepLinkRoute(Uri uri) {
  return uri.path;
}
