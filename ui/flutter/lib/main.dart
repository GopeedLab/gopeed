import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app/app.dart';
import 'app/child_window_app.dart';
import 'core/capabilities/app_capabilities.dart';
import 'core/entry/app_initializer.dart';
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
  final hidden = args.contains('--hidden');

  if (launchContext.isSubWindow) {
    final windowController = launchContext.buildController()!;
    final session = await ChildWindowSession.connect(windowController);
    await AppWindowBootstrap.setupSubWindow(launchContext.payload.type);
    runApp(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(session.capabilities)],
        child: ChildWindowApp(payload: launchContext.payload, session: session),
      ),
    );
    return;
  }

  await AppInitializer.ensureDatabaseInitialized();
  await AppWindowBootstrap.setupMainWindow(hidden: hidden);
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
