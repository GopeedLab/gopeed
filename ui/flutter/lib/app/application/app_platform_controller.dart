import 'dart:async';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:launch_at_startup/launch_at_startup.dart';
import 'package:tray_manager/tray_manager.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:window_manager/window_manager.dart';

import '../../api/api.dart' as api;
import '../../api/model/downloader_config.dart';
import '../../api/model/resolve_task.dart';
import '../../core/libgopeed_boot.dart';
import '../../core/capabilities/app_capabilities.dart';
import '../../core/window/app_window_launcher.dart';
import '../../core/window/window_capability_transport.dart';
import '../../database/database.dart';
import '../../database/entity.dart';
import '../../features/tasks/application/pending_create_task.dart';
import '../../features/tasks/application/pending_update_task.dart';
import '../../l10n/l10n.dart';
import '../../util/package_info.dart';
import '../../util/updater.dart';
import '../../util/util.dart';
import '../../util/browser_extension_host/browser_extension_host.dart';
import '../../util/log_util.dart';
import '../../util/scheme_register/scheme_register.dart';
import '../rpc/host_rpc_service.dart';
import '../router/app_router.dart';
import 'app_runtime_controller.dart';

final appPlatformControllerProvider = AsyncNotifierProvider<AppPlatformController, AppPlatformState>(
  AppPlatformController.new,
);

class AppPlatformState {
  const AppPlatformState({
    this.started = false,
    this.launchAtStartup = false,
    this.runAsMenubarApp = false,
    this.updateStatus = AppUpdateStatus.idle,
    this.latestVersion,
    this.updateError,
  });

  final bool started;
  final bool launchAtStartup;
  final bool runAsMenubarApp;
  final AppUpdateStatus updateStatus;
  final VersionInfo? latestVersion;
  final String? updateError;
}

enum AppUpdateStatus { idle, checking, upToDate, available, failed }

class AppPlatformController extends AsyncNotifier<AppPlatformState> with WindowListener, TrayListener {
  bool _started = false;
  String? _trayLocaleConfig;
  bool _updateInitialized = false;
  bool _launchAtStartup = false;
  bool _runAsMenubarApp = false;
  AppUpdateStatus _updateStatus = AppUpdateStatus.idle;
  VersionInfo? _latestVersion;
  String? _updateError;
  Future<VersionInfo?>? _checkUpdateFuture;
  final _windowsResizeSave = Util.debounce(() async {
    final size = await windowManager.getSize();
    Database.instance.saveWindowState(WindowStateEntity(width: size.width, height: size.height));
  }, 500);

  @override
  Future<AppPlatformState> build() async {
    final runtime = ref.watch(appRuntimeControllerProvider).value;
    if (runtime == null || kIsWeb) {
      return _snapshot();
    }
    if (!Util.isDesktop()) {
      _scheduleInitialUpdateCheck();
      return _snapshot();
    }
    if (_started) {
      final localeConfig = runtime.downloaderConfig.extra.locale;
      if (_trayLocaleConfig != localeConfig) await _initTray();
      return _snapshot();
    }
    await _start();
    _scheduleInitialUpdateCheck();
    ref.onDispose(() {
      windowManager.removeListener(this);
      trayManager.removeListener(this);
      unawaited(HostRpcService.instance.stop());
      unawaited(AppWindowCapabilityHost.instance.stop());
    });
    return _snapshot();
  }

  void _scheduleInitialUpdateCheck() {
    if (_updateInitialized) return;
    _updateInitialized = true;
    Timer.run(() => unawaited(_initCheckUpdate()));
  }

  Future<void> _start() async {
    _started = true;
    await AppWindowCapabilityHost.instance.start(LocalAppCapabilities.instance.registry);
    windowManager.addListener(this);
    await windowManager.setPreventClose(true);
    if (Util.isMacos()) {
      _runAsMenubarApp = Database.instance.getRunAsMenubarApp();
      await windowManager.setSkipTaskbar(_runAsMenubarApp);
    }
    await _initTray();
    await _initHostRpc();
    unawaited(_initDesktopIntegrations());
    await _initLaunchAtStartup();
  }

  Future<void> _initTray() async {
    final localeConfig = ref.read(appRuntimeControllerProvider).value?.downloaderConfig.extra.locale ?? '';
    final locale = appLocalizationsFor(localeConfig);
    if (Util.isWindows()) {
      await trayManager.setIcon('assets/tray_icon/icon.ico');
    } else if (Util.isMacos()) {
      await trayManager.setIcon('assets/tray_icon/icon_mac.png', isTemplate: true);
    } else if (Platform.environment.containsKey('FLATPAK_ID') || Platform.environment.containsKey('SNAP')) {
      await trayManager.setIcon('com.gopeed.Gopeed');
    } else {
      await trayManager.setIcon('assets/tray_icon/icon.png');
    }

    final version = _safeVersionLabel();
    final menu = Menu(
      items: [
        MenuItem(label: locale.show, onClick: (_) => _showMainWindow()),
        MenuItem.separator(),
        MenuItem(label: locale.create, onClick: (_) => _openCreateTask()),
        MenuItem(
          label: locale.startAll,
          onClick: (_) => unawaited(ref.read(gopeedServiceProvider).continueAllTasks(null)),
        ),
        MenuItem(
          label: locale.pauseAll,
          onClick: (_) => unawaited(ref.read(gopeedServiceProvider).pauseAllTasks(null)),
        ),
        MenuItem(label: locale.setting, onClick: (_) => _go('/settings')),
        MenuItem.separator(),
        MenuItem(
          label: locale.donate,
          onClick: (_) => unawaited(launchUrl(Uri.parse('https://gopeed.com/docs/donate'))),
        ),
        MenuItem(label: version),
        MenuItem.separator(),
        MenuItem(label: locale.exit, onClick: (_) => _exit()),
      ],
    );

    if (!Util.isLinux()) {
      await trayManager.setToolTip('Gopeed');
    }
    await trayManager.setContextMenu(menu);
    trayManager.removeListener(this);
    trayManager.addListener(this);
    _trayLocaleConfig = localeConfig;
  }

  Future<void> _initHostRpc() async {
    await HostRpcService.instance.start(
      onCreate: (createTask, silent) async {
        final pendingUpdate = ref.read(pendingUpdateTaskProvider);
        if (pendingUpdate != null && createTask.req != null) {
          await ref.read(gopeedServiceProvider).patchTask(pendingUpdate.id, ResolveTask(req: createTask.req));
          await ref.read(gopeedServiceProvider).continueTask(pendingUpdate.id);
          ref.read(pendingUpdateTaskProvider.notifier).clear();
          await _showMainWindow();
          return;
        }
        if (silent) {
          await ref.read(gopeedServiceProvider).createTask(createTask);
          return;
        }
        await _showMainWindow();
        final opened = await AppWindowLauncher.openCreateTaskWindow(createTask: createTask);
        if (!opened) {
          ref.read(pendingCreateTaskProvider.notifier).set(createTask);
          await _go('/create');
        }
      },
      onForward: ({required path, required method, data, query}) {
        return api.forward(path, method: method, data: data, queryParameters: query);
      },
    );
  }

  Future<void> _initDesktopIntegrations() async {
    try {
      registerUrlScheme('gopeed');
      final runtime = ref.read(appRuntimeControllerProvider).value;
      if (runtime?.downloaderConfig.extra.defaultBtClient == true) {
        registerDefaultTorrentClient();
      }
    } catch (error, stackTrace) {
      logger.w('register scheme failed', error, stackTrace);
    }

    try {
      await installHost();
      for (final browser in Browser.values) {
        await installManifest(browser);
      }
    } catch (error, stackTrace) {
      logger.w('browser extension host integration failed', error, stackTrace);
    }

    try {
      await installUpdater();
    } catch (error, stackTrace) {
      // Native helper assets are generated by release builds, so their absence
      // in a source/debug bundle must not prevent the app from starting.
      logger.w('updater integration failed', error, stackTrace);
    }
  }

  Future<void> _initLaunchAtStartup() async {
    if (!Util.isWindows() && !Util.isLinux()) {
      return;
    }
    launchAtStartup.setup(appName: packageInfo.appName, appPath: Platform.resolvedExecutable, args: const ['--hidden']);
    _launchAtStartup = await launchAtStartup.isEnabled();
  }

  Future<void> setLaunchAtStartup(bool enabled) async {
    if (!Util.isWindows() && !Util.isLinux()) {
      return;
    }
    if (enabled) {
      await launchAtStartup.enable();
    } else {
      await launchAtStartup.disable();
    }
    _launchAtStartup = enabled;
    _emit();
  }

  Future<void> setRunAsMenubarApp(bool enabled) async {
    if (!Util.isMacos()) {
      return;
    }
    Database.instance.saveRunAsMenubarApp(enabled);
    _runAsMenubarApp = enabled;
    await windowManager.setSkipTaskbar(enabled);
    await Future<void>.delayed(const Duration(milliseconds: 100));
    await _showMainWindow();
    _emit();
  }

  Future<void> _initCheckUpdate() async {
    final runtime = ref.read(appRuntimeControllerProvider).value;
    if (runtime?.downloaderConfig.extra.notifyWhenNewVersion == false) return;
    await checkForUpdate();
  }

  Future<VersionInfo?> checkForUpdate() {
    return _checkUpdateFuture ??= _performUpdateCheck().whenComplete(() => _checkUpdateFuture = null);
  }

  Future<VersionInfo?> _performUpdateCheck() async {
    _updateStatus = AppUpdateStatus.checking;
    _updateError = null;
    _emit();
    try {
      _latestVersion = await checkUpdate();
      _updateStatus = _latestVersion == null ? AppUpdateStatus.upToDate : AppUpdateStatus.available;
      _emit();
      return _latestVersion;
    } catch (error, stackTrace) {
      _latestVersion = null;
      _updateStatus = AppUpdateStatus.failed;
      _updateError = error.toString();
      _emit();
      logger.w('check update failed', error, stackTrace);
      return null;
    }
  }

  Future<void> startUpdate(VersionInfo versionInfo, UpdateProgressCallback onProgress) async {
    final runtime = ref.read(appRuntimeControllerProvider).value;
    await updateApp(
      versionInfo,
      githubMirror: runtime?.downloaderConfig.extra.githubMirror ?? ExtraConfigGithubMirror(enabled: false),
      onProgress: onProgress,
    );
  }

  AppPlatformState _snapshot() => AppPlatformState(
    started: _started,
    launchAtStartup: _launchAtStartup,
    runAsMenubarApp: _runAsMenubarApp,
    updateStatus: _updateStatus,
    latestVersion: _latestVersion,
    updateError: _updateError,
  );

  void _emit() {
    state = AsyncValue.data(_snapshot());
  }

  String _safeVersionLabel() {
    try {
      return 'Version ${packageInfo.version}';
    } catch (_) {
      return 'Version';
    }
  }

  Future<void> _showMainWindow() async {
    await windowManager.show();
    await windowManager.focus();
  }

  Future<void> _openCreateTask() async {
    await _showMainWindow();
    final opened = await AppWindowLauncher.openCreateTaskWindow();
    if (!opened) {
      _go('/create');
    }
  }

  Future<void> _go(String route) async {
    final context = AppRouter.rootNavigatorKey.currentContext;
    await _showMainWindow();
    if (context != null && context.mounted) {
      context.go(route);
    }
  }

  Future<void> _exit() async {
    try {
      await HostRpcService.instance.stop();
      await AppWindowCapabilityHost.instance.stop();
      await LibgopeedBoot.instance.stop();
    } catch (_) {}
    await windowManager.destroy();
  }

  @override
  void onWindowClose() async {
    final isPreventClose = await windowManager.isPreventClose();
    if (isPreventClose) {
      await windowManager.hide();
    }
  }

  @override
  void onWindowFocus() {
    if (Util.isMacos() && _runAsMenubarApp) {
      unawaited(windowManager.setSkipTaskbar(true));
    }
  }

  @override
  void onTrayIconMouseDown() {
    unawaited(_showMainWindow());
  }

  @override
  void onTrayIconRightMouseDown() {
    trayManager.popUpContextMenu();
  }

  @override
  void onWindowMaximize() {
    Database.instance.saveWindowState(WindowStateEntity(isMaximized: true));
  }

  @override
  void onWindowUnmaximize() {
    Database.instance.saveWindowState(WindowStateEntity(isMaximized: false));
  }

  @override
  void onWindowResize() {
    _windowsResizeSave();
  }
}
