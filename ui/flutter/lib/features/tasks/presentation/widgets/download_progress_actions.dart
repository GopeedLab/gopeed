import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../core/window/app_window_chrome.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../domain/task_record.dart';

class DownloadProgressActions extends StatelessWidget {
  const DownloadProgressActions({
    super.key,
    required this.task,
    required this.busy,
    required this.onClose,
    required this.onCancel,
    required this.onPause,
    required this.onResume,
    required this.onOpenFile,
    required this.onOpenDirectory,
    this.onRetry,
  });

  final TaskRecord? task;
  final bool busy;
  final VoidCallback onClose, onCancel, onPause, onResume, onOpenFile, onOpenDirectory;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final completed = task?.status == TaskStatus.completed;
    final actions = Wrap(
      alignment: WrapAlignment.end,
      spacing: AppDesignTokens.space8,
      runSpacing: AppDesignTokens.space8,
      children: [
        if (onRetry != null)
          OutlineButton(size: ButtonSize.small, onPressed: busy ? null : onRetry, child: Text(context.l10n.retry)),
        if (!completed && task != null) ...[
          OutlineButton(
            key: const ValueKey('download-progress-cancel'),
            size: ButtonSize.small,
            onPressed: busy ? null : onCancel,
            child: Text(context.l10n.cancel),
          ),
          if (task!.canPause)
            PrimaryButton(
              key: const ValueKey('download-progress-pause'),
              size: ButtonSize.small,
              onPressed: busy ? null : onPause,
              child: Text(context.l10n.pause),
            ),
          if (task!.canResume)
            PrimaryButton(
              key: const ValueKey('download-progress-resume'),
              size: ButtonSize.small,
              onPressed: busy ? null : onResume,
              child: Text(context.l10n.continueAction),
            ),
        ],
        OutlineButton(
          key: const ValueKey('download-progress-close'),
          size: ButtonSize.small,
          onPressed: busy ? null : onClose,
          child: Text(task != null && !completed ? context.l10n.runInBackground : context.l10n.close),
        ),
        if (completed && AppWindowChrome.isDesktopWindow) ...[
          OutlineButton(
            key: const ValueKey('download-progress-open-folder'),
            size: ButtonSize.small,
            onPressed: busy ? null : onOpenDirectory,
            child: Text(context.l10n.openDirectory),
          ),
          if (!task!.isFolder && !task!.isGroup)
            PrimaryButton(
              key: const ValueKey('download-progress-open-file'),
              size: ButtonSize.small,
              onPressed: busy ? null : onOpenFile,
              child: Text(context.l10n.openFile),
            ),
        ],
      ],
    );
    final hint = Text(context.l10n.downloadBackgroundHint).xSmall().muted();
    return Container(
      color: AppPalette.of(context).footerBg,
      padding: const EdgeInsets.symmetric(horizontal: AppDesignTokens.space16, vertical: AppDesignTokens.space8),
      child: LayoutBuilder(
        builder: (context, constraints) => constraints.maxWidth >= 680
            ? Row(
                children: [
                  Expanded(child: hint),
                  const SizedBox(width: AppDesignTokens.space12),
                  Expanded(
                    flex: 2,
                    child: Align(alignment: AlignmentDirectional.centerEnd, child: actions),
                  ),
                ],
              )
            : Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  hint,
                  const SizedBox(height: AppDesignTokens.space8),
                  actions,
                ],
              ),
      ),
    );
  }
}
