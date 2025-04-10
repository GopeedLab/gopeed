import 'package:go_router/go_router.dart';

import '../modules/home/screens/home.dart';
import '../modules/task/screens/task.dart';
import '../modules/settings/screens/settings.dart';
import 'route_names.dart';

/// 应用路由配置
class AppRoutes {
  /// 获取路由配置
  static GoRouter getRouter() {
    final GoRouter router = GoRouter(
      initialLocation: RouteNames.task,
      routes: [
        ShellRoute(
          builder: (context, state, child) => HomeScreen(child: child),
          routes: [
            GoRoute(
              path: RouteNames.task,
              builder: (context, state) => const TaskScreen(),
            ),
            GoRoute(
              path: RouteNames.settings,
              builder: (context, state) => const SettingsScreen(),
            ),
          ],
        ),
      ],
    );

    return router;
  }

  // 防止实例化
  AppRoutes._();
}
