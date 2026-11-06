import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../features/tasks/application/task_list_navigation.dart';

import '../../features/extensions/presentation/pages/extensions_page.dart';
import '../../features/extensions/presentation/pages/extension_details_page.dart';
import '../../features/extensions/application/extensions_controller.dart';
import '../../features/auth/application/web_auth_controller.dart';
import '../../features/auth/presentation/pages/login_page.dart';
import '../../features/home/presentation/pages/home_page.dart';
import '../../features/settings/presentation/pages/settings_page.dart';
import '../../api/model/create_task.dart';
import '../../core/window/app_window_chrome.dart';
import '../../core/window/app_window_launcher.dart';
import '../../core/window/app_window_payload.dart';
import '../../features/tasks/application/pending_create_task.dart';
import 'app_window_content.dart';
import 'app_window_page.dart';
import '../../features/tasks/domain/task_record.dart';
import '../../features/tasks/presentation/pages/task_details_page.dart';
import '../../features/tasks/presentation/pages/task_files_page.dart';
import 'shells/main_shell.dart';
import 'mobile_exit_guard.dart';
import '../../shared/theme/app_design_tokens.dart';
import 'runtime_page.dart';

class AppRouter {
  static final GlobalKey<NavigatorState> rootNavigatorKey = GlobalKey<NavigatorState>();

  static bool get _isMobile =>
      !kIsWeb && (defaultTargetPlatform == TargetPlatform.android || defaultTargetPlatform == TargetPlatform.iOS);

  static Future<void> openCreateTask({CreateTask? createTask}) =>
      openWindow(AppWindowPayload.createTask(createTask: createTask?.toJson()));

  /// Opens a desktop child window, a Web modal, or a native mobile page.
  /// Callers never need to choose a presentation or implement a fallback.
  static Future<void> openWindow(AppWindowPayload payload) async {
    final location = AppWindowContent.location(payload.type);
    if (AppWindowChrome.isDesktopWindow && await AppWindowLauncher.open(payload)) return;
    final context = rootNavigatorKey.currentContext;
    if (context == null || !context.mounted) return;
    final router = GoRouter.of(context);
    if (router.state.uri.path == location) {
      // Native share intents may arrive while the creation page is already open.
      if (payload.type == AppWindowType.createTask && payload.createTask != null) {
        ProviderScope.containerOf(
          context,
          listen: false,
        ).read(pendingCreateTaskProvider.notifier).set(CreateTask.fromJson(payload.createTask!));
      }
      return;
    }
    if (_isMobile) {
      router.go(location, extra: payload.toRaw());
    } else {
      unawaited(router.push<void>(location, extra: payload.toRaw()));
    }
  }

  static void showDownloadingTasks() {
    final context = rootNavigatorKey.currentContext;
    if (context == null || !context.mounted) return;
    ProviderScope.containerOf(context, listen: false).read(taskListNavigationProvider.notifier).showDownloading();
    GoRouter.of(context).go('/');
  }

  static GoRouter build(WebAuthController webAuthController) {
    // Keep desktop and web sections as peers; native mobile retains home below them.
    final isMobile = _isMobile;
    final sectionRoutes = [
      GoRoute(
        path: isMobile ? 'extensions' : '/extensions',
        builder: _runtimePage((context, state) => const ExtensionsPage()),
        routes: [
          GoRoute(
            path: ':id',
            builder: _runtimePage(
              (context, state) => ExtensionDetailsPage(
                extensionId: state.pathParameters['id'] ?? '',
                initialItem: state.extra is ExtensionListItem ? state.extra as ExtensionListItem : null,
              ),
            ),
          ),
        ],
      ),
      GoRoute(
        path: isMobile ? 'settings' : '/settings',
        builder: _runtimePage((context, state) => const SettingsPage()),
        routes: [
          GoRoute(
            path: ':section',
            builder: _runtimePage((context, state) => SettingsPage(sectionKey: state.pathParameters['section'])),
          ),
        ],
      ),
    ];
    return GoRouter(
      navigatorKey: rootNavigatorKey,
      refreshListenable: webAuthController,
      redirect: (context, state) {
        final onLoginPage = state.matchedLocation == '/login';
        if (webAuthController.loginRequired) {
          if (onLoginPage) return null;
          return Uri(path: '/login', queryParameters: {'from': state.uri.toString()}).toString();
        }
        if (onLoginPage) {
          return _safeReturnLocation(state.uri.queryParameters['from']);
        }
        return null;
      },
      routes: [
        GoRoute(path: '/login', parentNavigatorKey: rootNavigatorKey, builder: (context, state) => const LoginPage()),
        ShellRoute(
          builder: (context, state, child) => MobileExitGuard(child: MainShell(child: child)),
          routes: [
            GoRoute(
              path: '/',
              builder: _runtimePage((context, state) => const HomePage()),
              routes: [
                GoRoute(
                  path: 'download-progress',
                  parentNavigatorKey: rootNavigatorKey,
                  pageBuilder: (context, state) {
                    final payload = AppWindowPayload.fromRaw(state.extra is String ? state.extra! as String : '');
                    return AppWindowPage<void>(
                      key: state.pageKey,
                      preferredSize: AppDesignTokens.downloadProgressWindowSize,
                      child: RuntimePage(
                        builder: (context) => AppWindowContent.build(
                          AppWindowPayload.downloadProgress(payload.taskId),
                          modal: true,
                          onClose: (_) => context.canPop() ? context.pop() : context.go('/'),
                        ),
                      ),
                    );
                  },
                ),
                GoRoute(
                  path: 'create',
                  parentNavigatorKey: rootNavigatorKey,
                  pageBuilder: (context, state) {
                    final payload = state.extra is String
                        ? AppWindowPayload.fromRaw(state.extra! as String)
                        : AppWindowPayload.createTask();
                    final child = RuntimePage(
                      builder: (context) => AppWindowContent.build(
                        payload,
                        modal: !isMobile,
                        onClose: (created) {
                          if (created) {
                            showDownloadingTasks();
                          } else if (context.canPop()) {
                            context.pop();
                          } else {
                            context.go('/');
                          }
                        },
                      ),
                    );
                    return isMobile
                        ? NoTransitionPage<void>(key: state.pageKey, child: child)
                        : AppWindowPage<void>(key: state.pageKey, child: child);
                  },
                ),
                GoRoute(
                  path: 'tasks/:id',
                  builder: _runtimePage(
                    (context, state) => TaskDetailsPage(
                      taskId: state.pathParameters['id'] ?? '',
                      initialTask: state.extra is TaskRecord ? state.extra as TaskRecord : null,
                    ),
                  ),
                  routes: [
                    GoRoute(
                      path: 'files',
                      builder: _runtimePage(
                        (context, state) => TaskFilesPage(
                          taskId: state.pathParameters['id'] ?? '',
                          initialTask: state.extra is TaskRecord ? state.extra as TaskRecord : null,
                        ),
                      ),
                    ),
                  ],
                ),
                if (isMobile) ...sectionRoutes,
              ],
            ),
            if (!isMobile) ...sectionRoutes,
          ],
        ),
      ],
    );
  }

  static GoRouterWidgetBuilder _runtimePage(GoRouterWidgetBuilder builder) {
    return (context, state) => RuntimePage(builder: (context) => builder(context, state));
  }

  static String _safeReturnLocation(String? location) {
    if (location == null || location.isEmpty) return '/';
    final uri = Uri.tryParse(location);
    if (uri == null || uri.hasScheme || uri.host.isNotEmpty || !location.startsWith('/') || uri.path == '/login') {
      return '/';
    }
    return location;
  }
}
