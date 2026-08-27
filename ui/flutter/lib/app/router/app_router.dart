import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';

import '../../features/extensions/presentation/pages/extensions_page.dart';
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
        GoRoute(
          path: '/',
          builder: (context, state) => MainShell(child: const HomePage()),
        ),
        GoRoute(
          path: '/extensions',
          builder: (context, state) => MainShell(child: const ExtensionsPage()),
        ),
        GoRoute(
          path: '/settings',
          builder: (context, state) => MainShell(child: const SettingsPage()),
        ),
        GoRoute(
          path: '/create',
          builder: (context, state) => const MainShell(child: CreateTaskWindowPage()),
        ),
        GoRoute(
          path: '/tasks/:id',
          builder: (context, state) => MainShell(
            child: TaskDetailsPage(
              taskId: state.pathParameters['id'] ?? '',
              initialTask: state.extra is TaskRecord ? state.extra as TaskRecord : null,
            ),
          ),
        ),
        GoRoute(
          path: '/tasks/:id/files',
          builder: (context, state) => MainShell(
            child: TaskFilesPage(
              taskId: state.pathParameters['id'] ?? '',
              initialTask: state.extra is TaskRecord ? state.extra as TaskRecord : null,
            ),
          ),
        ),
      ],
    );
  }
}
