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
    return flutter.Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        SecondaryButton(
          key: cancelButtonKey,
          onPressed: submitting ? null : onCancel,
          child: SizedBox(
            width: _actionContentWidth,
            child: Center(child: Text(cancelLabel)),
          ),
        ),
        const SizedBox(width: AppDesignTokens.space12),
        AppLoadingButton(
          key: submitButtonKey,
          onPressed: onSubmit,
          loading: submitting,
          variant: AppLoadingButtonVariant.primary,
          child: SizedBox(
            width: _actionContentWidth,
            child: Center(child: Text(submitLabel)),
          ),
        ),
      ],
    );
  }
}

const _actionContentWidth = 68.0;
