import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';

import '../../shared/navigation/app_exit_confirmation_controller.dart';

/// Handles exit at the shell route after the nested Navigator declines a pop.
/// Child routes and dialogs still pop through GoRouter before reaching this guard.
class MobileExitGuard extends StatefulWidget {
  const MobileExitGuard({super.key, required this.child});

  final Widget child;

  @override
  State<MobileExitGuard> createState() => _MobileExitGuardState();
}

class _MobileExitGuardState extends State<MobileExitGuard> with WidgetsBindingObserver {
  final _controller = AppExitConfirmationController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) _controller.dispose();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (kIsWeb || defaultTargetPlatform != TargetPlatform.android) return widget.child;

    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) unawaited(_controller.handleBack(context));
      },
      child: NotificationListener<NavigationNotification>(
        onNotification: (notification) {
          if (notification.canHandlePop) return false;
          // An empty nested back stack still has this shell's exit guard.
          // Otherwise its startup notification disables Android's back callback.
          const NavigationNotification(canHandlePop: true).dispatch(context);
          return true;
        },
        child: widget.child,
      ),
    );
  }
}
