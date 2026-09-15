import 'package:flutter/widgets.dart';

import '../../l10n/l10n.dart';
import '../theme/app_design_tokens.dart';
import '../theme/app_palette.dart';

/// A single editable name/value pair in a narrow headers form.
class AppHttpHeaderGroup extends StatelessWidget {
  const AppHttpHeaderGroup({super.key, required this.name, required this.value, required this.remove});

  final Widget name;
  final Widget value;
  final Widget remove;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      padding: const EdgeInsets.all(AppDesignTokens.space12),
      decoration: BoxDecoration(
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Expanded(child: Text(context.l10n.httpHeaderName)),
              remove,
            ],
          ),
          const SizedBox(height: AppDesignTokens.space4),
          name,
          const SizedBox(height: AppDesignTokens.space12),
          Text(context.l10n.httpHeaderValue),
          const SizedBox(height: AppDesignTokens.space8),
          value,
        ],
      ),
    );
  }
}
