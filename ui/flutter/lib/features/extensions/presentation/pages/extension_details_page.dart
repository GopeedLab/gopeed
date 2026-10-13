import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/detail/app_detail_surface.dart';
import '../../application/extensions_controller.dart';
import '../widgets/extension_detail_view.dart';
import '../widgets/extension_update_dialog.dart';
import '../widgets/extension_update_status.dart';

class ExtensionDetailsPage extends ConsumerWidget {
  const ExtensionDetailsPage({super.key, required this.extensionId, this.initialItem});

  final String extensionId;
  final ExtensionListItem? initialItem;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final stateAsync = ref.watch(extensionsControllerProvider);
    final item = stateAsync.value?.findItem(extensionId) ?? initialItem;
    final palette = AppPalette.of(context);

    return AppDetailPage(
      title: item?.title ?? context.l10n.extensions,
      onBack: () => context.canPop() ? context.pop() : context.go('/extensions'),
      titleTrailing: _canUpdate(stateAsync.value, item)
          ? ExtensionUpdateStatus(
              key: const ValueKey('extension-details-update-mobile'),
              label: context.l10n.extensionCanUpdate,
              onTap: item?.installed == null
                  ? null
                  : () => showExtensionUpdateDialog(context, item!.installed!),
            )
          : null,
      child: item == null
          ? Center(
              child: stateAsync.isLoading
                  ? const shad.CircularProgressIndicator()
                  : Text(context.l10n.noExtensions, style: TextStyle(color: palette.textMuted, fontSize: 13)),
            )
          : ExtensionDetailView(key: ValueKey(item.id), item: item, mobile: true),
    );
  }

  bool _canUpdate(ExtensionsState? state, ExtensionListItem? item) {
    final installed = item?.installed;
    return item != null && installed != null && (state?.updateFlags.containsKey(installed.identity) ?? false);
  }
}
