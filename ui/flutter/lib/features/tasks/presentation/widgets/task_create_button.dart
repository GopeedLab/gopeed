import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/widgets/app_primary_button.dart';

/// Shared creation action for task headers and empty-list guidance.
class TaskCreateButton extends StatelessWidget {
  const TaskCreateButton({super.key, required this.onPressed, this.minHeight = 0, this.compact = false});

  final VoidCallback onPressed;
  final double minHeight;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    return ConstrainedBox(
      constraints: BoxConstraints(minHeight: minHeight),
      child: AppPrimaryButton(
        onPressed: onPressed,
        size: compact ? shad.ButtonSize.small : shad.ButtonSize.normal,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.add, size: compact ? 16 : null),
            SizedBox(width: compact ? AppDesignTokens.space4 : AppDesignTokens.space8),
            Flexible(
              child: Text(
                context.l10n.create,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: compact ? const TextStyle(fontSize: 13) : null,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
