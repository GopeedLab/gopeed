import 'package:flutter/widgets.dart';

import '../../../../shared/theme/app_design_tokens.dart';

class SettingsContentFrame extends StatelessWidget {
  const SettingsContentFrame({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: AppDesignTokens.space24),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: AppDesignTokens.settingsContentMaxWidth),
          child: SizedBox(width: double.infinity, child: child),
        ),
      ),
    );
  }
}
