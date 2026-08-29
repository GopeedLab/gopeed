import 'dart:async';

import 'package:app_links/app_links.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app/app.dart';
import 'app/child_window_app.dart';
import 'core/capabilities/app_capabilities.dart';
import 'core/entry/app_initializer.dart';
import 'core/entry/app_startup_options.dart';
import 'core/window/app_window_bootstrap.dart';
import 'core/window/app_window_launch_context.dart';
import 'core/window/window_capability_transport.dart';
import 'database/database.dart';
import 'util/analytics.dart';
import 'util/log_util.dart';
import 'util/util.dart';

Future<void> main(List<String> args) async {
  WidgetsFlutterBinding.ensureInitialized();
  if (Util.isAndroid()) {
    FlutterForegroundTask.initCommunicationPort();
  }

  final launchContext = AppWindowLaunchContext.fromArgs(args);

  if (launchContext.isSubWindow) {
    final windowController = launchContext.buildController()!;
    final session = await ChildWindowSession.connect(windowController);
    await AppWindowBootstrap.setupSubWindow(launchContext.payload.type, localeConfig: session.appearance.value.locale);
    runApp(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(session.capabilities)],
        child: ChildWindowApp(payload: launchContext.payload, session: session),
      ),
    );
    return;
  }

  var startupOptions = AppStartupOptions.fromArgs(args);
  if (Util.isDesktop() && !startupOptions.hidden) {
    try {
      startupOptions = startupOptions.withInitialUri(await AppLinks().getInitialLink());
    } catch (_) {}
  }

  await AppInitializer.ensureDatabaseInitialized();
  await AppWindowBootstrap.setupMainWindow(hidden: startupOptions.hidden);
  runApp(const ProviderScope(child: GopeedApp()));
  unawaited(_logAppOpen());
}

Future<void> _logAppOpen() async {
  if (!AnalyticsConfig.isConfigured || !Database.instance.getAnalyticsEnabled()) return;
  try {
    await Analytics.instance.init();
    await Analytics.instance.logAppOpen();
  } catch (error) {
    logger.w('GA4 init failed: $error');
  }
}
