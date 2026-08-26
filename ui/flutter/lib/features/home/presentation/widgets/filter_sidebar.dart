import 'package:flutter/material.dart' show Colors, Icons;
import 'package:flutter/widgets.dart';

import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../l10n/l10n.dart';
import '../../../tasks/domain/task_record.dart';
import '../../../tasks/presentation/widgets/speed_monitor_card.dart';

enum TaskFilter { all, downloading, completed, paused, failed }

class FilterSidebar extends StatelessWidget {
  const FilterSidebar({super.key, required this.activeFilter, required this.onFilterSelected, required this.tasks});

  final TaskFilter activeFilter;
  final ValueChanged<TaskFilter> onFilterSelected;
  final List<TaskRecord> tasks;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final titleStyle = TextStyle(
      color: palette.textMuted,
      fontSize: 12,
      fontWeight: FontWeight.w700,
      letterSpacing: 2.4,
    );

    return Container(
      width: AppDesignTokens.filterSidebarWidth,
      color: palette.sideBg,
      padding: const EdgeInsets.only(top: AppDesignTokens.windowHeaderHeight),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            height: AppDesignTokens.contentHeaderHeight,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text(context.l10n.filters.toUpperCase(), style: titleStyle),
              ),
            ),
          ),
          const SizedBox(height: 16),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Column(
              children: [
                _FilterItem(
                  label: context.l10n.allTasks,
                  icon: Icons.grid_view_rounded,
                  active: activeFilter == TaskFilter.all,
                  count: tasks.length.toString(),
                  onTap: () => onFilterSelected(TaskFilter.all),
                ),
                _FilterItem(
                  label: context.l10n.downloading,
                  icon: Icons.south_rounded,
                  active: activeFilter == TaskFilter.downloading,
                  count: _count(TaskStatus.downloading).toString(),
                  onTap: () => onFilterSelected(TaskFilter.downloading),
                ),
                _FilterItem(
                  label: context.l10n.completed,
                  icon: Icons.check_circle_outline,
                  active: activeFilter == TaskFilter.completed,
                  onTap: () => onFilterSelected(TaskFilter.completed),
                ),
                _FilterItem(
                  label: context.l10n.pause,
                  icon: Icons.pause_circle_outline,
                  active: activeFilter == TaskFilter.paused,
                  onTap: () => onFilterSelected(TaskFilter.paused),
                ),
                _FilterItem(
                  label: context.l10n.failed,
                  icon: Icons.error_outline,
                  active: activeFilter == TaskFilter.failed,
                  onTap: () => onFilterSelected(TaskFilter.failed),
                ),
              ],
            ),
          ),
          const Spacer(),
          const SpeedMonitorCard(),
        ],
      ),
    );
  }

  int _count(TaskStatus status) {
    return tasks.where((task) => task.status == status).length;
  }
}

class _FilterItem extends StatelessWidget {
  const _FilterItem({required this.label, required this.icon, required this.active, required this.onTap, this.count});

  final String label;
  final IconData icon;
  final bool active;
  final VoidCallback onTap;
  final String? count;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return GestureDetector(
      onTap: onTap,
      child: Container(
        margin: const EdgeInsets.only(bottom: 4),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: active ? palette.itemActiveBg : Colors.transparent,
          borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
        ),
        child: Row(
          children: [
            Icon(icon, size: 16, color: active ? palette.textPrimary : palette.textSecondary),
            const SizedBox(width: 12),
            Expanded(
              child: Text(
                label,
                style: TextStyle(
                  color: active ? palette.textPrimary : palette.textSecondary,
                  fontSize: 14,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ),
            if (count != null)
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(color: palette.cardHoverBg, borderRadius: BorderRadius.circular(4)),
                child: Text(
                  count!,
                  style: TextStyle(
                    color: active ? palette.textPrimary : palette.textSecondary,
                    fontSize: 10,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
