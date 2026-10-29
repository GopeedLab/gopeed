import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter/widgets.dart' as flutter show Flexible;

import '../../../../core/capabilities/gopeed_capability.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_tooltip.dart';
import '../../application/pending_download.dart';
import 'create_task_action_buttons.dart';
import 'resolve_file_tree.dart';

Future<bool> showResolveDownloadsDialog(
  BuildContext context, {
  required List<PendingDownload> sources,
  required GopeedService service,
  ValueChanged<List<PendingDownload>>? onBatchCreated,
}) async {
  final dialog = const DialogOverlayHandler().show<bool>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (_) => _ResolveDownloadsDialog(sources: sources, service: service, onBatchCreated: onBatchCreated),
  );
  return await dialog.future == true;
}

class _ResolveDownloadsDialog extends StatefulWidget {
  const _ResolveDownloadsDialog({required this.sources, required this.service, required this.onBatchCreated});

  final List<PendingDownload> sources;
  final GopeedService service;
  final ValueChanged<List<PendingDownload>>? onBatchCreated;

  @override
  State<_ResolveDownloadsDialog> createState() => _ResolveDownloadsDialogState();
}

class _ResolveDownloadsDialogState extends State<_ResolveDownloadsDialog> {
  bool _cancelled = false;
  bool _submitting = false;
  String? _errorText;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _resolveAll());
  }

  @override
  void dispose() {
    _cancelled = true;
    super.dispose();
  }

  Future<void> _resolveAll() async {
    for (final source in widget.sources) {
      if (_cancelled || !mounted) return;
      if (source.result != null || source.resolveError != null) continue;
      final resolving = source.resolve(widget.service);
      setState(() {});
      await resolving;
      if (_cancelled || !mounted) return;
      setState(() {});
    }
  }

  void _cancel() {
    _cancelled = true;
    closeOverlay(context, false);
  }

  Future<void> _submit() async {
    if (_submitting || widget.sources.any((source) => source.loading)) return;
    final batches = planDownloads(widget.sources);
    if (batches.isEmpty) return;
    setState(() {
      _submitting = true;
      _errorText = null;
      for (final source in widget.sources) {
        source.createError = null;
      }
    });
    for (final batch in batches) {
      await batch.submit(widget.service);
      if (!mounted) return;
      if (batch.sources.every((source) => source.created)) {
        widget.onBatchCreated?.call(batch.sources);
      }
    }
    final errors = widget.sources.map((source) => source.createError).whereType<String>().toSet();
    if (errors.isEmpty) {
      await closeOverlay(context, true);
    } else {
      setState(() {
        _submitting = false;
        _errorText = errors.join('\n');
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final screen = MediaQuery.sizeOf(context);
    final canCreate =
        !_submitting &&
        !widget.sources.any((source) => source.loading) &&
        widget.sources.any((source) => source.hasSelection);
    return AlertDialog(
      padding: const EdgeInsets.all(18),
      title: Row(
        children: [
          Expanded(child: Text(context.l10n.resolvedFiles)),
          AppTooltip(
            message: context.l10n.close,
            child: GhostButton(
              key: const ValueKey('resolve-close-button'),
              onPressed: _submitting ? null : _cancel,
              child: const Icon(Icons.close, size: 18),
            ),
          ),
        ],
      ),
      content: SizedBox(
        width: screen.width < 760 ? screen.width - 68 : 720,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SizedBox(
              height: (screen.height * 0.48).clamp(120.0, 440.0),
              child: IgnorePointer(
                ignoring: _submitting,
                child: ResolveFileTree(
                  sources: List.of(widget.sources),
                  onSelectionChanged: (selection) {
                    final selectedBySource = <String, List<int>>{};
                    for (final (sourceId, index) in selection) {
                      selectedBySource.putIfAbsent(sourceId, () => []).add(index);
                    }
                    setState(() {
                      for (final source in widget.sources) {
                        if (source.result == null || source.created) continue;
                        source.selected
                          ..clear()
                          ..addAll(selectedBySource[source.id] ?? const []);
                      }
                    });
                  },
                ),
              ),
            ),
            if (_errorText != null)
              ConstrainedBox(
                constraints: BoxConstraints(maxHeight: screen.height * 0.15),
                child: SingleChildScrollView(
                  child: Text(_errorText!, style: TextStyle(color: AppPalette.of(context).error, fontSize: 12)),
                ),
              ),
          ],
        ),
      ),
      actions: [
        flutter.Flexible(
          child: CreateTaskActionButtons(
            submitting: _submitting,
            onCancel: _cancel,
            onSubmit: canCreate ? _submit : null,
            cancelLabel: context.l10n.cancel,
            submitLabel: context.l10n.createAction,
            cancelButtonKey: const ValueKey('resolve-cancel-button'),
            submitButtonKey: const ValueKey('resolve-create-button'),
          ),
        ),
      ],
    );
  }
}
