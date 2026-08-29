import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path/path.dart' as path;
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:url_launcher/url_launcher.dart';

import '../../../../api/model/store_extension.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_loading_button.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../../../shared/widgets/detail/app_detail_surface.dart';
import '../../../../util/util.dart';
import '../../application/extensions_controller.dart';

class ExtensionDetailDrawer extends StatelessWidget {
  const ExtensionDetailDrawer({super.key, required this.item, required this.onClose});

  final ExtensionListItem? item;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    return AppDetailDrawer(
      open: item != null,
      title: item?.title ?? '',
      onClose: onClose,
      drawerKey: const ValueKey('extension-details-drawer'),
      child: item == null ? const SizedBox.shrink() : ExtensionDetailView(item: item!),
    );
  }
}

class ExtensionDetailView extends ConsumerWidget {
  const ExtensionDetailView({super.key, required this.item, this.mobile = false});

  final ExtensionListItem item;
  final bool mobile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final palette = AppPalette.of(context);
    final state = ref.watch(extensionsControllerProvider).value;
    final current = state?.findItem(item.id) ?? item;
    final installed = current.installed;
    final store = current.store;
    final busy = state?.busyExtensionIds.contains(current.id) ?? false;
    final canUpdate = ref.read(extensionsControllerProvider.notifier).canUpdateItem(current);
    final horizontalPadding = mobile ? AppDesignTokens.space16 : AppDesignTokens.space32;

    return ListView(
      key: const ValueKey('extension-details-content'),
      padding: EdgeInsets.fromLTRB(horizontalPadding, mobile ? 24 : 28, horizontalPadding, mobile ? 40 : 48),
      children: [
        Row(
          key: const ValueKey('extension-details-hero'),
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _ExtensionDetailIcon(item: current, size: mobile ? 60 : 68),
            SizedBox(width: mobile ? 16 : 20),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    current.author,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: palette.textPrimary,
                      fontSize: 12,
                      fontWeight: FontWeight.w700,
                      letterSpacing: 0.15,
                    ),
                  ),
                  const SizedBox(height: 7),
                  Text(
                    current.description,
                    style: TextStyle(color: palette.textSecondary, fontSize: mobile ? 14 : 13.5, height: 1.5),
                  ),
                  const SizedBox(height: 10),
                  Wrap(
                    crossAxisAlignment: WrapCrossAlignment.center,
                    spacing: 9,
                    runSpacing: 5,
                    children: [
                      Text(
                        'v${current.version}',
                        style: TextStyle(color: palette.textMuted, fontSize: 11.5, fontWeight: FontWeight.w600),
                      ),
                      if (installed != null)
                        _ExtensionStatus(
                          label: canUpdate ? context.l10n.extensionCanUpdate : context.l10n.extensionInstalled,
                          emphasized: canUpdate,
                        ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
        SizedBox(height: mobile ? 24 : 28),
        Wrap(
          key: const ValueKey('extension-details-actions'),
          spacing: 4,
          runSpacing: 6,
          crossAxisAlignment: WrapCrossAlignment.center,
          children: [
            if (installed == null && store != null)
              AppLoadingButton(
                key: const ValueKey('extension-details-install'),
                loading: busy,
                variant: AppLoadingButtonVariant.primary,
                icon: const Icon(Icons.download_outlined, size: 16),
                onPressed: () =>
                    _runAction(context, () => ref.read(extensionsControllerProvider.notifier).installFromStore(store)),
                child: Text(context.l10n.extensionInstall),
              ),
            if (installed != null && canUpdate)
              AppLoadingButton(
                key: const ValueKey('extension-details-update'),
                loading: busy,
                variant: AppLoadingButtonVariant.primary,
                icon: const Icon(Icons.refresh, size: 16),
                onPressed: () => _runAction(
                  context,
                  () => ref.read(extensionsControllerProvider.notifier).upgradeExtension(installed),
                  successMessage: context.l10n.extensionUpdateSuccess,
                ),
                child: Text(context.l10n.newVersionUpdate),
              ),
            if ((current.homepage ?? '').isNotEmpty)
              shad.GhostButton(
                onPressed: () => unawaited(_openUrl(current.homepage!)),
                child: _LinkButtonContent(icon: Icons.home_outlined, label: context.l10n.homepage),
              ),
            if ((current.repoUrl ?? '').isNotEmpty)
              shad.GhostButton(
                onPressed: () => unawaited(_openUrl(current.repoUrl!)),
                child: const _LinkButtonContent(icon: Icons.code, label: 'GitHub'),
              ),
          ],
        ),
        if (store != null) ...[
          SizedBox(height: mobile ? 24 : 28),
          _ExtensionMetadata(store: store),
          _ExtensionReadme(item: current, mobile: mobile),
        ],
      ],
    );
  }

  Future<void> _runAction(BuildContext context, Future<void> Function() action, {String? successMessage}) async {
    try {
      await action();
      if (context.mounted && successMessage != null) {
        showAppToast(context, successMessage, type: AppToastType.success);
      }
    } catch (error) {
      if (context.mounted) showAppToast(context, error.toString(), type: AppToastType.error);
    }
  }
}

class _ExtensionReadme extends StatefulWidget {
  const _ExtensionReadme({required this.item, required this.mobile});

  final ExtensionListItem item;
  final bool mobile;

  @override
  State<_ExtensionReadme> createState() => _ExtensionReadmeState();
}

class _ExtensionReadmeState extends State<_ExtensionReadme> {
  late Future<_ReadmeInfo> _readme;

  @override
  void initState() {
    super.initState();
    _readme = _loadReadme(widget.item);
  }

  @override
  void didUpdateWidget(covariant _ExtensionReadme oldWidget) {
    super.didUpdateWidget(oldWidget);
    final previous = oldWidget.item.installed;
    final current = widget.item.installed;
    if (previous?.identity != current?.identity ||
        previous?.devMode != current?.devMode ||
        previous?.devPath != current?.devPath ||
        oldWidget.item.store?.readme != widget.item.store?.readme) {
      _readme = _loadReadme(widget.item);
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return FutureBuilder<_ReadmeInfo>(
      future: _readme,
      builder: (context, snapshot) {
        final info = snapshot.data;
        if (info == null) {
          return const SizedBox.shrink();
        }
        if (info.content.trim().isEmpty) {
          return const SizedBox.shrink();
        }
        return Padding(
          key: const ValueKey('extension-details-readme'),
          padding: EdgeInsets.only(top: widget.mobile ? 32 : 36),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'README',
                style: TextStyle(
                  color: palette.textMuted,
                  fontSize: 10.5,
                  fontWeight: FontWeight.w800,
                  letterSpacing: 1.2,
                ),
              ),
              const SizedBox(height: 16),
              MarkdownBody(
                data: info.content,
                selectable: true,
                styleSheet: _markdownStyle(palette, mobile: widget.mobile),
                onTapLink: (_, href, _) {
                  final resolved = _resolveReadmeUrl(widget.item, href, info: info, forImage: false);
                  if (resolved != null) unawaited(_openUrl(resolved));
                },
                imageBuilder: (uri, title, alt) {
                  final resolved = _resolveReadmeUrl(widget.item, uri.toString(), info: info, forImage: true);
                  if (resolved == null) return const SizedBox.shrink();
                  final image = resolved.startsWith('file:')
                      ? Image.file(
                          File(Uri.parse(resolved).toFilePath()),
                          fit: BoxFit.contain,
                          errorBuilder: (_, _, _) => const SizedBox.shrink(),
                        )
                      : Image.network(
                          resolved,
                          fit: BoxFit.contain,
                          errorBuilder: (_, _, _) => const SizedBox.shrink(),
                        );
                  return Padding(
                    padding: const EdgeInsets.symmetric(vertical: 10),
                    child: ClipRRect(borderRadius: BorderRadius.circular(6), child: image),
                  );
                },
              ),
            ],
          ),
        );
      },
    );
  }
}

class _ExtensionDetailIcon extends StatelessWidget {
  const _ExtensionDetailIcon({required this.item, required this.size});

  final ExtensionListItem item;
  final double size;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final icon = item.icon;
    final fallback = Container(
      width: size,
      height: size,
      padding: EdgeInsets.all(size * 0.16),
      decoration: BoxDecoration(color: palette.surfaceSoft, borderRadius: BorderRadius.circular(14)),
      child: Image.asset('assets/extension/default_icon.png'),
    );
    if (icon == null || icon.isEmpty) return fallback;
    return ClipRRect(
      borderRadius: BorderRadius.circular(14),
      child: Image.network(icon, width: size, height: size, fit: BoxFit.cover, errorBuilder: (_, _, _) => fallback),
    );
  }
}

class _ExtensionStatus extends StatelessWidget {
  const _ExtensionStatus({required this.label, required this.emphasized});

  final String label;
  final bool emphasized;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final dotColor = emphasized ? palette.brand : palette.success;
    final textColor = emphasized ? palette.textPrimary : palette.success;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 5,
          height: 5,
          decoration: BoxDecoration(color: dotColor, shape: BoxShape.circle),
        ),
        const SizedBox(width: 5),
        Text(
          label,
          style: TextStyle(color: textColor, fontSize: 11.5, fontWeight: FontWeight.w600),
        ),
      ],
    );
  }
}

class _LinkButtonContent extends StatelessWidget {
  const _LinkButtonContent({required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 15, color: palette.textMuted),
        const SizedBox(width: 7),
        Text(label),
      ],
    );
  }
}

class _ExtensionMetadata extends StatelessWidget {
  const _ExtensionMetadata({required this.store});

  final StoreExtension store;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      key: const ValueKey('extension-details-metadata'),
      padding: const EdgeInsets.symmetric(vertical: 14),
      decoration: BoxDecoration(
        border: Border(
          top: BorderSide(color: palette.border),
          bottom: BorderSide(color: palette.border),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Wrap(
            spacing: 18,
            runSpacing: 8,
            children: [
              _MetadataItem(icon: Icons.star_rounded, value: store.stars.toString()),
              _MetadataItem(icon: Icons.download_outlined, value: store.installCount.toString()),
            ],
          ),
          if (store.topics.isNotEmpty) ...[
            const SizedBox(height: 12),
            Wrap(
              spacing: 12,
              runSpacing: 6,
              children: [
                for (final topic in store.topics)
                  Text(
                    topic.startsWith('#') ? topic : '#$topic',
                    style: TextStyle(color: palette.textMuted, fontSize: 11.5, fontWeight: FontWeight.w600),
                  ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}

class _MetadataItem extends StatelessWidget {
  const _MetadataItem({required this.icon, required this.value});

  final IconData icon;
  final String value;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 14, color: palette.textMuted),
        const SizedBox(width: 6),
        Text(
          value,
          style: TextStyle(color: palette.textSecondary, fontSize: 12, fontWeight: FontWeight.w600),
        ),
      ],
    );
  }
}

MarkdownStyleSheet _markdownStyle(AppPalette palette, {required bool mobile}) {
  final paragraph = TextStyle(color: palette.textSecondary, fontSize: mobile ? 14 : 13.5, height: 1.62);
  final heading = TextStyle(color: palette.textPrimary, fontWeight: FontWeight.w800, height: 1.25);
  return MarkdownStyleSheet(
    a: paragraph.copyWith(color: palette.brand, decoration: TextDecoration.underline),
    p: paragraph,
    code: paragraph.copyWith(fontFamily: 'monospace', fontSize: 12, backgroundColor: palette.surfaceSoft),
    h1: heading.copyWith(fontSize: mobile ? 21 : 20),
    h2: heading.copyWith(fontSize: mobile ? 18 : 17),
    h3: heading.copyWith(fontSize: mobile ? 16 : 15),
    h4: heading.copyWith(fontSize: 14),
    h5: heading.copyWith(fontSize: 13),
    h6: heading.copyWith(fontSize: 12),
    em: paragraph.copyWith(fontStyle: FontStyle.italic),
    strong: paragraph.copyWith(fontWeight: FontWeight.w800, color: palette.textPrimary),
    blockSpacing: 12,
    listIndent: 22,
    listBullet: paragraph,
    blockquote: paragraph,
    blockquotePadding: const EdgeInsets.fromLTRB(12, 8, 10, 8),
    blockquoteDecoration: BoxDecoration(
      color: palette.surfaceSoft,
      border: Border(left: BorderSide(color: palette.brand, width: 3)),
    ),
    codeblockPadding: const EdgeInsets.all(12),
    codeblockDecoration: BoxDecoration(
      color: palette.surfaceSoft,
      borderRadius: BorderRadius.circular(6),
      border: Border.all(color: palette.border),
    ),
    horizontalRuleDecoration: BoxDecoration(
      border: Border(top: BorderSide(color: palette.border)),
    ),
    tableHead: paragraph.copyWith(color: palette.textPrimary, fontWeight: FontWeight.w800),
    tableBody: paragraph,
    tableBorder: TableBorder.all(color: palette.border),
    tableCellsPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
  );
}

Future<_ReadmeInfo> _loadReadme(ExtensionListItem item) async {
  final installed = item.installed;
  final remote = item.store?.readme ?? '';
  if (installed == null || Util.isWeb()) return _ReadmeInfo(content: remote);

  try {
    final rootDirectory = installed.devMode
        ? installed.devPath
        : path.join(Util.getStorageDir(), 'extensions', installed.identity);
    for (final name in const ['README.md', 'readme.md', 'README.MD']) {
      final filePath = path.join(rootDirectory, name);
      final file = File(filePath);
      if (file.existsSync()) {
        return _ReadmeInfo(content: file.readAsStringSync(), localPath: filePath);
      }
    }
  } catch (_) {
    // Fall back to the store copy when local storage is unavailable or unreadable.
  }
  return _ReadmeInfo(content: remote);
}

String? _resolveReadmeUrl(ExtensionListItem item, String? raw, {required _ReadmeInfo info, required bool forImage}) {
  final store = item.store;
  final value = raw?.trim() ?? '';
  if (value.isEmpty) return null;
  final parsed = Uri.tryParse(value);
  if (parsed != null && parsed.hasScheme) return parsed.toString();

  if (forImage && info.localPath != null) {
    final clean = value.split('#').first;
    return Uri.file(path.normalize(path.join(path.dirname(info.localPath!), clean))).toString();
  }
  if (store == null) return null;

  final ref = store.commitSha?.isNotEmpty == true ? store.commitSha! : 'HEAD';
  final directory = (store.directory ?? '').split('/').where((segment) => segment.isNotEmpty).join('/');
  final suffix = directory.isEmpty ? '' : '$directory/';
  final base = forImage
      ? Uri.https('raw.githubusercontent.com', '/${store.repoFullName}/$ref/$suffix')
      : Uri.https('github.com', '/${store.repoFullName}/blob/$ref/$suffix');
  return base.resolve(value).toString();
}

class _ReadmeInfo {
  const _ReadmeInfo({required this.content, this.localPath});

  final String content;
  final String? localPath;
}

Future<void> _openUrl(String value) async {
  final uri = Uri.tryParse(value);
  if (uri == null) return;
  await launchUrl(uri, mode: LaunchMode.externalApplication);
}
