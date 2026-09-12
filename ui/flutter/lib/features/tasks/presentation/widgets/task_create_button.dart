import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/widgets/app_primary_button.dart';

/// Shared creation action for task headers and empty-list guidance.
class TaskCreateButton extends StatelessWidget {
  const TaskCreateButton({super.key, required this.onPressed, this.minHeight = 0});

  final VoidCallback onPressed;
  final double minHeight;

  @override
  Widget build(BuildContext context) {
    return ConstrainedBox(
      constraints: BoxConstraints(minHeight: minHeight),
      child: AppPrimaryButton(
        onPressed: onPressed,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.add),
            const SizedBox(width: AppDesignTokens.space8),
            Flexible(child: Text(context.l10n.create, maxLines: 1, overflow: TextOverflow.ellipsis)),
          ],
        ),
      ),
    );
  }
}
