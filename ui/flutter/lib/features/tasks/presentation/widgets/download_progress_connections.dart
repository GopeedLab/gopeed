import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../shared/theme/app_design_tokens.dart';
import '../../domain/task_record.dart';
import 'task_statistics/task_statistics_tab.dart';

class DownloadProgressConnections extends StatefulWidget {
  const DownloadProgressConnections({super.key, required this.task});

  final TaskRecord task;

  @override
  State<DownloadProgressConnections> createState() => _DownloadProgressConnectionsState();
}

class _DownloadProgressConnectionsState extends State<DownloadProgressConnections> {
  final _scrollController = ScrollController();

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scrollbar(
      controller: _scrollController,
      thumbVisibility: false,
      child: SingleChildScrollView(
        controller: _scrollController,
        padding: const EdgeInsetsDirectional.only(end: AppDesignTokens.space12),
        child: TaskStatisticsTab(task: widget.task, mobile: true, inline: true),
      ),
    );
  }
}
