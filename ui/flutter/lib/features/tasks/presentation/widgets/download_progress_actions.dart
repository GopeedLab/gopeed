import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../core/window/app_window_chrome.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_primary_button.dart';
import '../../domain/task_record.dart';

class DownloadProgressActions extends StatelessWidget {
  const DownloadProgressActions({
    super.key,
    required this.task,
    required this.busy,
    required this.connectionsExpanded,
    required this.onToggleConnections,
    required this.onCancel,
    required this.onPause,
    required this.onResume,
    required this.onOpenFile,
    required this.onOpenDirectory,
  });

  final TaskRecord task;
  final bool busy;
  final bool connectionsExpanded;
  final VoidCallback? onToggleConnections;
  final VoidCallback onCancel, onPause, onResume, onOpenFile, onOpenDirectory;

  @override
  Widget build(BuildContext context) {
    final completed = task.status == TaskStatus.completed;
    final palette = AppPalette.of(context);
    final actions = Wrap(
      alignment: WrapAlignment.end,
      crossAxisAlignment: WrapCrossAlignment.center,
      spacing: AppDesignTokens.space8,
      runSpacing: AppDesignTokens.space8,
      children: [
        if (!completed) ...[
          SecondaryButton(
            key: const ValueKey('download-progress-cancel'),
            size: ButtonSize.small,
            density: ButtonDensity.dense,
            onPressed: busy ? null : onCancel,
            child: Text(context.l10n.cancel),
          ),
          if (task.canPause)
            AppPrimaryButton(
              key: const ValueKey('download-progress-pause'),
              size: ButtonSize.small,
              density: ButtonDensity.dense,
              onPressed: busy ? null : onPause,
              child: Text(context.l10n.pause),
            )
          else if (task.canResume)
            AppPrimaryButton(
              key: const ValueKey('download-progress-resume'),
              size: ButtonSize.small,
              density: ButtonDensity.dense,
              onPressed: busy ? null : onResume,
              child: Text(context.l10n.continueAction),
            ),
        ] else if (AppWindowChrome.isDesktopWindow) ...[
          GhostButton(
            key: const ValueKey('download-progress-open-folder'),
            size: ButtonSize.small,
            density: ButtonDensity.dense,
            onPressed: busy ? null : onOpenDirectory,
            child: Text(context.l10n.openDirectory),
          ),
          if (!task.isFolder && !task.isGroup)
            AppPrimaryButton(
              key: const ValueKey('download-progress-open-file'),
              size: ButtonSize.small,
              density: ButtonDensity.dense,
              onPressed: busy ? null : onOpenFile,
              child: Text(context.l10n.openFile),
            ),
        ],
      ],
    );
    if (task.isGroup) return Align(alignment: AlignmentDirectional.centerEnd, child: actions);
    return LayoutBuilder(
      builder: (context, constraints) => Row(
        children: [
          Expanded(
            child: Semantics(
              expanded: connectionsExpanded,
              child: ButtonStyleOverride(
                padding: (context, states, value) => const EdgeInsets.symmetric(vertical: AppDesignTokens.space8),
                child: GhostButton(
                  key: const ValueKey('download-progress-details'),
                  onPressed: onToggleConnections,
                  child: Row(
                    children: [
                      Icon(
                        connectionsExpanded ? LucideIcons.chevronUp : LucideIcons.chevronDown,
                        size: 14,
                        color: palette.textSecondary,
                      ),
                      const SizedBox(width: AppDesignTokens.space4),
                      Flexible(
                        child: Text(
                          context.l10n.connectionStatus,
                          style: TextStyle(color: palette.textSecondary, fontSize: 12),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(width: AppDesignTokens.space8),
          ConstrainedBox(
            constraints: BoxConstraints(maxWidth: constraints.maxWidth * .75),
            child: actions,
          ),
        ],
      ),
    );
  }
}
