import 'package:flutter/widgets.dart' as flutter;
import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/widgets/app_loading_button.dart';

class CreateTaskActionButtons extends StatelessWidget {
  const CreateTaskActionButtons({
    super.key,
    required this.submitting,
    required this.onCancel,
    required this.onSubmit,
    required this.cancelLabel,
    required this.submitLabel,
    this.cancelButtonKey,
    this.submitButtonKey,
  });

  final bool submitting;
  final VoidCallback onCancel;
  final VoidCallback onSubmit;
  final String cancelLabel;
  final String submitLabel;
  final Key? cancelButtonKey;
  final Key? submitButtonKey;

  @override
  Widget build(BuildContext context) {
    // Both actions follow the widest translation and retain their dimensions
    // while the submit label is replaced by the loading indicator.
    return IntrinsicWidth(
      child: flutter.Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          flutter.Expanded(
            child: ConstrainedBox(
              constraints: const BoxConstraints(minWidth: AppDesignTokens.dialogActionMinWidth),
              child: SecondaryButton(
                key: cancelButtonKey,
                onPressed: submitting ? null : onCancel,
                child: Text(cancelLabel, maxLines: 1, softWrap: false),
              ),
            ),
          ),
          const SizedBox(width: AppDesignTokens.space12),
          flutter.Expanded(
            child: ConstrainedBox(
              constraints: const BoxConstraints(minWidth: AppDesignTokens.dialogActionMinWidth),
              child: AppLoadingButton(
                key: submitButtonKey,
                onPressed: onSubmit,
                loading: submitting,
                variant: AppLoadingButtonVariant.primary,
                child: Text(submitLabel, maxLines: 1, softWrap: false),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
