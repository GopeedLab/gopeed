import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart' show Colors, Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_primary_button.dart';
import '../../../../shared/widgets/window/desktop_window_header.dart';
import '../../../../l10n/l10n.dart';

class TasksTopBar extends StatelessWidget {
  const TasksTopBar({
    super.key,
    required this.searchController,
    required this.onAddTask,
    required this.batchMode,
    required this.selectedBatchCount,
    required this.canPauseSelected,
    required this.canResumeSelected,
    required this.onToggleBatchMode,
    required this.onPauseSelected,
    required this.onResumeSelected,
    required this.onDeleteSelected,
  });

  final TextEditingController searchController;
  final VoidCallback onAddTask;
  final bool batchMode;
  final int selectedBatchCount;
  final bool canPauseSelected;
  final bool canResumeSelected;
  final VoidCallback onToggleBatchMode;
  final VoidCallback onPauseSelected;
  final VoidCallback onResumeSelected;
  final VoidCallback onDeleteSelected;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final headerBorder = palette.border;
    final showCaptionControls =
        !kIsWeb && (defaultTargetPlatform == TargetPlatform.windows || defaultTargetPlatform == TargetPlatform.linux);

    return SizedBox(
      height: AppDesignTokens.contentHeaderHeight,
      child: Align(
        alignment: Alignment.center,
        child: Padding(
          padding: const EdgeInsets.only(left: 32, right: 32),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              SizedBox(
                width: 240,
                child: _SearchField(controller: searchController, palette: palette, borderColor: headerBorder),
              ),
              const Spacer(),
              _AddTaskButton(onPressed: onAddTask),
              const SizedBox(width: 16),
              Row(
                key: const ValueKey('tasks-top-action-buttons'),
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  _ActionIconButton(
                    icon: Icons.pause_outlined,
                    palette: palette,
                    onPressed: !batchMode || canPauseSelected ? onPauseSelected : null,
                  ),
                  const SizedBox(width: 4),
                  _ActionIconButton(
                    icon: Icons.play_arrow_outlined,
                    palette: palette,
                    onPressed: !batchMode || canResumeSelected ? onResumeSelected : null,
                  ),
                  const SizedBox(width: 4),
                  _ActionIconButton(
                    icon: Icons.delete_outline,
                    palette: palette,
                    onPressed: selectedBatchCount > 0 ? onDeleteSelected : null,
                  ),
                  const SizedBox(width: 4),
                  _ActionIconButton(icon: Icons.checklist_rtl_outlined, palette: palette, onPressed: onToggleBatchMode),
                  if (showCaptionControls) ...[
                    const SizedBox(width: 12),
                    Container(width: 1, height: 24, color: palette.headerDivider),
                    const SizedBox(width: 4),
                    const WindowCaptionControls(),
                  ],
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SearchField extends StatelessWidget {
  const _SearchField({required this.controller, required this.palette, required this.borderColor});

  final TextEditingController controller;
  final AppPalette palette;
  final Color borderColor;

  @override
  Widget build(BuildContext context) {
    return shad.ComponentTheme<shad.FocusOutlineTheme>(
      data: const shad.FocusOutlineTheme(
        align: 0,
        border: Border.fromBorderSide(BorderSide(color: Colors.transparent, width: 0)),
      ),
      child: SizedBox(
        height: 40,
        child: shad.TextField(
          controller: controller,
          style: TextStyle(color: palette.textPrimary, fontSize: 14, height: 1),
          placeholder: Text(
            context.l10n.searchDownloads,
            style: TextStyle(color: palette.searchHint, fontSize: 14, height: 1),
          ),
          features: [shad.InputFeature.leading(Icon(Icons.search_rounded, size: 14, color: palette.textMuted))],
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 0),
          decoration: BoxDecoration(
            color: palette.inputBg,
            borderRadius: BorderRadius.circular(4),
            border: Border.all(color: borderColor),
          ),
        ),
      ),
    );
  }
}

class _AddTaskButton extends StatelessWidget {
  const _AddTaskButton({required this.onPressed});

  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return AppPrimaryButton(onPressed: onPressed, leading: const Icon(Icons.add), child: Text(context.l10n.addTask));
  }
}

class _ActionIconButton extends StatelessWidget {
  const _ActionIconButton({required this.icon, required this.palette, required this.onPressed});

  final IconData icon;
  final AppPalette palette;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    return SizedBox.square(
      dimension: 30,
      child: shad.ButtonStyleOverride(
        decoration: (context, states, value) {
          final disabled = states.contains(WidgetState.disabled);
          final hovered = states.contains(WidgetState.hovered);
          return BoxDecoration(
            color: hovered && !disabled ? palette.cardHoverBg : Colors.transparent,
            borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
            border: Border.all(color: disabled ? palette.border.withValues(alpha: 0.5) : palette.border),
          );
        },
        iconTheme: (context, states, value) {
          final disabled = states.contains(WidgetState.disabled);
          return value.copyWith(
            color: disabled ? palette.textMuted.withValues(alpha: 0.6) : palette.textSecondary,
            size: 18,
          );
        },
        child: shad.IconButton.outline(size: shad.ButtonSize.xSmall, onPressed: onPressed, icon: Icon(icon, size: 18)),
      ),
    );
  }
}
