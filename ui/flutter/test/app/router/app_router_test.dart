import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:gopeed/app/router/app_router.dart';
import 'package:gopeed/features/auth/application/web_auth_controller.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  for (final platform in TargetPlatform.values) {
    test('$platform preserves the expected section route stacks', () {
      debugDefaultTargetPlatformOverride = platform;
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      final auth = WebAuthController();
      final router = AppRouter.build(auth);
      addTearDown(router.dispose);
      addTearDown(auth.dispose);
      final isMobile = platform == TargetPlatform.android || platform == TargetPlatform.iOS;

      for (final entry in {
        '/': ['/'],
        '/extensions': isMobile ? ['/', 'extensions'] : ['/extensions'],
        '/extensions/example': isMobile ? ['/', 'extensions', ':id'] : ['/extensions', ':id'],
        '/settings': isMobile ? ['/', 'settings'] : ['/settings'],
        '/settings/general': isMobile ? ['/', 'settings', ':section'] : ['/settings', ':section'],
        '/tasks/example': ['/', 'tasks/:id'],
        '/create': ['/', 'create'],
      }.entries) {
        final match = router.configuration.findMatch(Uri.parse(entry.key));
        expect(match.isError, isFalse);
        final shell = match.matches.single as ShellRouteMatch;
        expect(shell.matches.map((match) => (match.route as GoRoute).path), entry.value);
      }
    });
  }
}
