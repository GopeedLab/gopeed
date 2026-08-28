import 'package:flutter/material.dart' show Divider, Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../api/model/downloader_config.dart';
import '../../../../core/utils/breakpoints.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_primary_button.dart';
import '../../../../l10n/l10n.dart';

class DownloadCategoryDraft {
  const DownloadCategoryDraft({required this.name, required this.path});

  final String name;
  final String path;
}

class DownloadCategoriesControl extends StatelessWidget {
  const DownloadCategoriesControl({
    super.key,
    required this.categories,
    required this.displayName,
    required this.onAdd,
    required this.onEdit,
    required this.onDelete,
  });

  final List<DownloadCategory> categories;
  final String Function(DownloadCategory category) displayName;
  final VoidCallback onAdd;
  final ValueChanged<DownloadCategory> onEdit;
  final ValueChanged<DownloadCategory> onDelete;

  @override
  Widget build(BuildContext context) {
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    final visibleCategories = categories.where((category) => !category.isDeleted).toList(growable: false);
    final palette = AppPalette.of(context);

    return SizedBox(
      key: const ValueKey('download-categories-editor'),
      width: desktop ? AppDesignTokens.settingsFormControlWidth : double.infinity,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          for (var index = 0; index < visibleCategories.length; index++) ...[
            if (index > 0)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Divider(color: palette.border),
              ),
            _CategoryRow(
              category: visibleCategories[index],
              name: displayName(visibleCategories[index]),
              onEdit: () => onEdit(visibleCategories[index]),
              onDelete: () => onDelete(visibleCategories[index]),
            ),
          ],
          if (visibleCategories.isNotEmpty) const SizedBox(height: 10),
          Align(
            alignment: desktop ? Alignment.centerRight : Alignment.centerLeft,
            child: shad.SecondaryButton(
              key: const ValueKey('add-download-category'),
              onPressed: onAdd,
              leading: const Icon(Icons.add, size: 17),
              child: Text(context.l10n.add),
            ),
          ),
        ],
      ),
    );
  }
}

class _CategoryRow extends StatelessWidget {
  const _CategoryRow({required this.category, required this.name, required this.onEdit, required this.onDelete});

  final DownloadCategory category;
  final String name;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Row(
      key: ValueKey('download-category-${category.nameKey ?? category.name}-${category.path}'),
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                name,
                style: TextStyle(color: palette.textPrimary, fontSize: 13, fontWeight: FontWeight.w700),
              ),
              const SizedBox(height: 3),
              Text(
                category.path,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(color: palette.textSecondary, fontSize: 12),
              ),
            ],
          ),
        ),
        const SizedBox(width: 8),
        shad.GhostButton(
          key: ValueKey('edit-download-category-${category.nameKey ?? category.name}'),
          density: shad.ButtonDensity.icon,
          onPressed: onEdit,
          child: const Icon(Icons.edit_outlined, size: 17),
        ),
        shad.GhostButton(
          key: ValueKey('delete-download-category-${category.nameKey ?? category.name}'),
          density: shad.ButtonDensity.icon,
          onPressed: onDelete,
          child: const Icon(Icons.delete_outline, size: 17),
        ),
      ],
    );
  }
}

Future<DownloadCategoryDraft?> showDownloadCategoryDialog(
  BuildContext context, {
  DownloadCategory? category,
  required String initialName,
  required Future<String?> Function() pickDirectory,
}) async {
  final nameController = TextEditingController(text: initialName);
  final pathController = TextEditingController(text: category?.path ?? '');
  String? validationMessage;

  final overlay = const shad.DialogOverlayHandler().show<DownloadCategoryDraft?>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) => StatefulBuilder(
      builder: (dialogContext, setDialogState) {
        final palette = AppPalette.of(dialogContext);

        void submit() {
          final name = nameController.text.trim();
          final path = pathController.text.trim();
          if (name.isEmpty || path.isEmpty) {
            setDialogState(() => validationMessage = dialogContext.l10n.categoryFieldsRequired);
            return;
          }
          shad.closeOverlay(dialogContext, DownloadCategoryDraft(name: name, path: path));
        }

        return shad.AlertDialog(
          title: Text(category == null ? dialogContext.l10n.addCategory : dialogContext.l10n.editCategory),
          content: SizedBox(
            key: const ValueKey('download-category-dialog'),
            width: (MediaQuery.sizeOf(dialogContext).width - 64).clamp(260.0, 420.0),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(dialogContext.l10n.categoryName, style: TextStyle(color: palette.textSecondary, fontSize: 12)),
                const SizedBox(height: 6),
                shad.TextField(key: const ValueKey('download-category-name'), controller: nameController),
                const SizedBox(height: 14),
                Text(dialogContext.l10n.categoryPath, style: TextStyle(color: palette.textSecondary, fontSize: 12)),
                const SizedBox(height: 6),
                shad.TextField(
                  key: const ValueKey('download-category-path'),
                  controller: pathController,
                  features: [
                    shad.InputFeature.trailing(
                      shad.GhostButton(
                        density: shad.ButtonDensity.icon,
                        onPressed: () async {
                          final path = await pickDirectory();
                          if (path != null && path.isNotEmpty) pathController.text = path;
                        },
                        child: const Icon(Icons.folder_open_outlined, size: 17),
                      ),
                    ),
                  ],
                ),
                if (validationMessage != null) ...[
                  const SizedBox(height: 8),
                  Text(validationMessage!, style: TextStyle(color: palette.error, fontSize: 12)),
                ],
              ],
            ),
          ),
          actions: [
            shad.SecondaryButton(
              onPressed: () => shad.closeOverlay(dialogContext),
              child: Text(dialogContext.l10n.cancel),
            ),
            AppPrimaryButton(onPressed: submit, child: Text(dialogContext.l10n.confirm)),
          ],
        );
      },
    ),
  );

  try {
    return await overlay.future;
  } finally {
    nameController.dispose();
    pathController.dispose();
  }
}

Future<bool> showDeleteDownloadCategoryDialog(BuildContext context, String name) async {
  final overlay = const shad.DialogOverlayHandler().show<bool>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) => shad.AlertDialog(
      title: Text(dialogContext.l10n.deleteCategoryTitle),
      content: Text(dialogContext.l10n.confirmDeleteNamed(name)),
      actions: [
        shad.SecondaryButton(
          onPressed: () => shad.closeOverlay(dialogContext, false),
          child: Text(dialogContext.l10n.cancel),
        ),
        shad.DestructiveButton(
          onPressed: () => shad.closeOverlay(dialogContext, true),
          child: Text(dialogContext.l10n.delete),
        ),
      ],
    ),
  );
  return await overlay.future ?? false;
}
