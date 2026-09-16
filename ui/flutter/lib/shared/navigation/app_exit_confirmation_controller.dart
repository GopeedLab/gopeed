import 'dart:async';

import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';

import '../../l10n/l10n.dart';
import '../widgets/app_toast.dart';

class AppExitConfirmationController {
  static const confirmationWindow = Duration(seconds: 2);

  OverlayEntry? _toast;
  Timer? _timer;

  Future<void> handleBack(BuildContext context) async {
    if (_toast != null) {
      dispose();
      await SystemNavigator.pop();
      return;
    }

    final message = context.l10n.pressBackAgainToExit;
    final themes = InheritedTheme.capture(from: context, to: Overlay.of(context, rootOverlay: true).context);
    _toast = OverlayEntry(
      builder: (_) => IgnorePointer(
        child: SafeArea(
          child: Align(
            alignment: Alignment.bottomCenter,
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: themes.wrap(AppToastContent(message: message)),
            ),
          ),
        ),
      ),
    );
    Overlay.of(context, rootOverlay: true).insert(_toast!);
    _timer = Timer(confirmationWindow, dispose);
  }

  void dispose() {
    _timer?.cancel();
    _timer = null;
    _toast?.remove();
    _toast?.dispose();
    _toast = null;
  }
}
