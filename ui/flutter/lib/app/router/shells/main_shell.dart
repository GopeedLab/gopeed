import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../application/app_deep_link_controller.dart';
import '../../application/app_notification_controller.dart';
import '../../application/app_runtime_controller.dart';
import '../../application/app_platform_controller.dart';
import '../../../features/settings/presentation/widgets/app_update_dialog.dart';
import '../../../l10n/l10n.dart';

class MainShell extends ConsumerStatefulWidget {
  const MainShell({super.key, required this.child});

  final Widget child;

  @override
  ConsumerState<MainShell> createState() => _MainShellState();
}

class _MainShellState extends ConsumerState<MainShell> {
  String? _shownUpdateVersion;

  @override
  Widget build(BuildContext context) {
    ref.listen(appPlatformControllerProvider, (_, next) {
      if (!kReleaseMode) return;
      final update = next.value;
      final version = update?.latestVersion;
      if (update?.updateStatus != AppUpdateStatus.available ||
          version == null ||
          _shownUpdateVersion == version.version) {
        return;
      }
      _shownUpdateVersion = version.version;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        unawaited(
          showAppUpdateDialog(
            context,
            versionInfo: version,
            onUpdate: ref.read(appPlatformControllerProvider.notifier).startUpdate,
          ),
        );
      });
    });
    final runtime = ref.watch(appRuntimeControllerProvider);
    return runtime.when(
      loading: () => const shad.Scaffold(child: Center(child: shad.CircularProgressIndicator())),
      error: (error, _) => shad.Scaffold(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 420),
            child: Text(context.l10n.runtimeInitializationFailed(error.toString()), textAlign: TextAlign.center),
          ),
        ),
      ),
      data: (_) {
        ref.watch(appPlatformControllerProvider);
        ref.watch(appDeepLinkControllerProvider);
        ref.watch(appNotificationControllerProvider);
        return widget.child;
      },
    );
  }
}
