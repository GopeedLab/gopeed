import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';

import '../../l10n/l10n.dart';
import '../widgets/app_toast.dart';

class AppExitConfirmationController {
  static const confirmationWindow = Duration(seconds: 2);

  DateTime? _lastBackAt;

  Future<void> handleBack(BuildContext context) async {
    final now = DateTime.now();
    final lastBackAt = _lastBackAt;
    if (lastBackAt != null && now.difference(lastBackAt) <= confirmationWindow) {
      _lastBackAt = null;
      await SystemNavigator.pop();
      return;
    }

    _lastBackAt = now;
    showAppToast(context, context.l10n.pressBackAgainToExit);
  }
}
