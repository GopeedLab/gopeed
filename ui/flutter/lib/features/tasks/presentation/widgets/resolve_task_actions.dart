import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter/widgets.dart' as flutter;

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/widgets/app_loading_button.dart';

class ResolveTaskActions extends StatelessWidget {
  const ResolveTaskActions({super.key, required this.submitting, required this.onCancel, required this.onCreate});

  final bool submitting;
  final VoidCallback onCancel;
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) {
    // Size both actions from the wider translated label, including while the
    // create button replaces its label with a loader.
    return IntrinsicWidth(
      child: flutter.Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          flutter.Expanded(
            child: ConstrainedBox(
              constraints: const BoxConstraints(minWidth: AppDesignTokens.dialogActionMinWidth),
              child: SecondaryButton(
                key: const ValueKey('resolve-cancel-button'),
                onPressed: submitting ? null : onCancel,
                child: Text(context.l10n.cancel, maxLines: 1, softWrap: false),
              ),
            ),
          ),
          const SizedBox(width: AppDesignTokens.space8),
          flutter.Expanded(
            child: ConstrainedBox(
              constraints: const BoxConstraints(minWidth: AppDesignTokens.dialogActionMinWidth),
              child: AppLoadingButton(
                key: const ValueKey('resolve-create-button'),
                onPressed: onCreate,
                loading: submitting,
                variant: AppLoadingButtonVariant.primary,
                child: Text(context.l10n.createAction, maxLines: 1, softWrap: false),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
