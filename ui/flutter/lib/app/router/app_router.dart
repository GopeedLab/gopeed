import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';

import '../../features/extensions/presentation/pages/extensions_page.dart';
import '../../features/extensions/presentation/pages/extension_details_page.dart';
import '../../features/extensions/application/extensions_controller.dart';
import '../../features/home/presentation/pages/home_page.dart';
import '../../features/settings/presentation/pages/settings_page.dart';
import '../../features/tasks/presentation/pages/create_task_window_page.dart';
import '../../features/tasks/domain/task_record.dart';
import '../../features/tasks/presentation/pages/task_details_page.dart';
import '../../features/tasks/presentation/pages/task_files_page.dart';
import 'shells/main_shell.dart';

class AppRouter {
  static final GlobalKey<NavigatorState> rootNavigatorKey = GlobalKey<NavigatorState>();

  static GoRouter build() {
    return GoRouter(
      navigatorKey: rootNavigatorKey,
      routes: [
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
}
