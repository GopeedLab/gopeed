import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';

import '../../features/extensions/presentation/pages/extensions_page.dart';
import '../../features/extensions/presentation/pages/extension_details_page.dart';
import '../../features/extensions/application/extensions_controller.dart';
import '../../features/auth/application/web_auth_controller.dart';
import '../../features/auth/presentation/pages/login_page.dart';
import '../../features/home/presentation/pages/home_page.dart';
import '../../features/settings/presentation/pages/settings_page.dart';
import '../../features/tasks/presentation/pages/create_task_window_page.dart';
import '../../features/tasks/domain/task_record.dart';
import '../../features/tasks/presentation/pages/task_details_page.dart';
import '../../features/tasks/presentation/pages/task_files_page.dart';
import 'shells/main_shell.dart';

class AppRouter {
  static final GlobalKey<NavigatorState> rootNavigatorKey = GlobalKey<NavigatorState>();

  static GoRouter build(WebAuthController webAuthController) {
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
          builder: (context, state, child) => MainShell(child: child),
          routes: [
            GoRoute(
              path: '/',
              builder: (context, state) => const HomePage(),
              routes: [
                GoRoute(path: 'create', builder: (context, state) => const CreateTaskWindowPage()),
                GoRoute(
                  path: 'tasks/:id',
                  builder: (context, state) => TaskDetailsPage(
                    taskId: state.pathParameters['id'] ?? '',
                    initialTask: state.extra is TaskRecord ? state.extra as TaskRecord : null,
                  ),
                  routes: [
                    GoRoute(
                      path: 'files',
                      builder: (context, state) => TaskFilesPage(
                        taskId: state.pathParameters['id'] ?? '',
                        initialTask: state.extra is TaskRecord ? state.extra as TaskRecord : null,
                      ),
                    ),
                  ],
                ),
              ],
            ),
            GoRoute(
              path: '/extensions',
              builder: (context, state) => const ExtensionsPage(),
              routes: [
                GoRoute(
                  path: ':id',
                  builder: (context, state) => ExtensionDetailsPage(
                    extensionId: state.pathParameters['id'] ?? '',
                    initialItem: state.extra is ExtensionListItem ? state.extra as ExtensionListItem : null,
                  ),
                ),
              ],
            ),
            GoRoute(
              path: '/settings',
              builder: (context, state) => const SettingsPage(),
              routes: [
                GoRoute(
                  path: ':section',
                  builder: (context, state) => SettingsPage(sectionKey: state.pathParameters['section']),
                ),
              ],
            ),
          ],
        ),
      ],
    );
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
