import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../api/model/extension.dart' as api;
import '../../../../l10n/l10n.dart';
import '../../../../shared/widgets/app_loading_button.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../application/extensions_controller.dart';

Future<void> showExtensionUpdateDialog(BuildContext context, api.Extension extension) async {
  final overlay = const DialogOverlayHandler().show<bool>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (_) => _ExtensionUpdateDialog(extension: extension),
  );
  final updated = await overlay.future;
  if (updated == true && context.mounted) {
    showAppToast(context, context.l10n.extensionUpdateSuccess);
  }
}

class _ExtensionUpdateDialog extends ConsumerStatefulWidget {
  const _ExtensionUpdateDialog({required this.extension});

  final api.Extension extension;

  @override
  ConsumerState<_ExtensionUpdateDialog> createState() => _ExtensionUpdateDialogState();
}

class _ExtensionUpdateDialogState extends ConsumerState<_ExtensionUpdateDialog> {
  bool _updating = false;
  bool _failed = false;

  Future<void> _update() async {
    if (_updating) return;
    setState(() {
      _updating = true;
      _failed = false;
    });
    try {
      await ref.read(extensionsControllerProvider.notifier).upgradeExtension(widget.extension);
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _updating = false;
        _failed = true;
      });
      return;
    }
    if (mounted) await closeOverlay(context, true);
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: !_updating,
      child: AlertDialog(
        key: const ValueKey('extension-update-dialog'),
        title: Text(context.l10n.extensionCanUpdate),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(context.l10n.extensionUpdateConfirm(widget.extension.title)),
            if (_failed) ...[
              const SizedBox(height: 12),
              Text(context.l10n.updateFailed, key: const ValueKey('extension-update-error')),
            ],
          ],
        ),
        actions: [
          SecondaryButton(
            key: const ValueKey('cancel-update-extension-button'),
            onPressed: _updating ? null : () => closeOverlay(context, false),
            child: Text(context.l10n.cancel),
          ),
          AppLoadingButton(
            key: const ValueKey('confirm-update-extension-button'),
            variant: AppLoadingButtonVariant.primary,
            loading: _updating,
            onPressed: _update,
            child: Text(context.l10n.newVersionUpdate),
          ),
        ],
      ),
    );
  }
}
