import 'package:flutter/widgets.dart';

import '../../application/task_batch_selection_controller.dart';

typedef TaskBatchSelectionWidgetBuilder = Widget Function(BuildContext context, bool selected, bool allSelected);

class TaskBatchSelectionBuilder extends StatefulWidget {
  const TaskBatchSelectionBuilder({
    super.key,
    required this.controller,
    required this.taskId,
    required this.visibleTaskIds,
    required this.builder,
  });

  final TaskBatchSelectionController controller;
  final String taskId;
  final List<String> visibleTaskIds;
  final TaskBatchSelectionWidgetBuilder builder;

  @override
  State<TaskBatchSelectionBuilder> createState() => _TaskBatchSelectionBuilderState();
}

class _TaskBatchSelectionBuilderState extends State<TaskBatchSelectionBuilder> {
  late bool _selected;
  late bool _allSelected;

  @override
  void initState() {
    super.initState();
    _readSelection();
    widget.controller.addListener(_handleSelectionChanged);
  }

  @override
  void didUpdateWidget(TaskBatchSelectionBuilder oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      oldWidget.controller.removeListener(_handleSelectionChanged);
      widget.controller.addListener(_handleSelectionChanged);
    }
    _readSelection();
  }

  @override
  void dispose() {
    widget.controller.removeListener(_handleSelectionChanged);
    super.dispose();
  }

  void _readSelection() {
    _selected = widget.controller.contains(widget.taskId);
    _allSelected = widget.controller.areAllSelected(widget.visibleTaskIds);
  }

  void _handleSelectionChanged() {
    final selected = widget.controller.contains(widget.taskId);
    final allSelected = widget.controller.areAllSelected(widget.visibleTaskIds);
    if (selected == _selected && allSelected == _allSelected) return;
    setState(() {
      _selected = selected;
      _allSelected = allSelected;
    });
  }

  @override
  Widget build(BuildContext context) => widget.builder(context, _selected, _allSelected);
}
