import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../domain/task_record.dart';
import '../task_record_localizations.dart';
import 'task_card/task_card_parts.dart';
import 'task_info_summary.dart';

class DownloadProgressHeader extends StatefulWidget {
  const DownloadProgressHeader({super.key, required this.task, required this.onHeightDelta});

  final TaskRecord? task;
  final ValueChanged<double> onHeightDelta;

  @override
  State<DownloadProgressHeader> createState() => _DownloadProgressHeaderState();
}

class _DownloadProgressHeaderState extends State<DownloadProgressHeader> {
  final _fieldsKey = GlobalKey();

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final task = widget.task;
    final progress = task?.status == TaskStatus.completed ? 1.0 : task?.progress;
    final percentage = progress == null ? null : '${(progress * 100).toStringAsFixed(1)}%';

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Flexible(
          child: LayoutBuilder(
            builder: (context, constraints) {
              WidgetsBinding.instance.addPostFrameCallback((_) {
                final size = _fieldsKey.currentContext?.size;
                // Include wrapped lines and give back spare height when a field disappears.
                if (size != null) widget.onHeightDelta(size.height - constraints.maxHeight);
              });
              return SingleChildScrollView(
                child: Column(
                  key: _fieldsKey,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    TaskInfoSummary(task: task, compact: true),
                    if (task?.error case final error? when error.isNotEmpty)
                      Text(
                        error,
                        maxLines: 3,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(color: AppPalette.of(context).error, fontSize: 12),
                      ),
                  ],
                ),
              );
            },
          ),
        ),
        const SizedBox(height: AppDesignTokens.space8),
        if (task == null)
          SizedBox(height: 3, child: ColoredBox(color: AppPalette.of(context).brandTrack))
        else
          Semantics(
            label: l10n.progress,
            value: percentage ?? task.localizedStatus(l10n),
            child: TaskCardProgressBar(task: task, height: 3),
          ),
      ],
    );
  }
}
