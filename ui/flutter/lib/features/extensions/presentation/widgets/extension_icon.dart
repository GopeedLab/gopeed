import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:path/path.dart' as path;

import '../../../../api/model/extension.dart';
import '../../../../util/util.dart';
import '../../application/extensions_controller.dart';

enum ExtensionIconSourceKind { asset, file, network }

@immutable
class ExtensionIconSource {
  const ExtensionIconSource(this.kind, this.location);

  final ExtensionIconSourceKind kind;
  final String location;
}

@visibleForTesting
ExtensionIconSource resolveExtensionIconSource(ExtensionListItem item, {bool? web, String? storageDirectory}) {
  const fallbackAsset = 'assets/extension/default_icon.png';
  final installed = item.installed;
  if (installed != null) {
    final icon = installed.icon.trim();
    if (icon.isEmpty) {
      return const ExtensionIconSource(ExtensionIconSourceKind.asset, fallbackAsset);
    }

    if (web ?? kIsWeb) {
      return ExtensionIconSource(ExtensionIconSourceKind.network, _installedExtensionIconUrl(installed, icon));
    }

    final rootDirectory = installed.devMode
        ? installed.devPath
        : path.join(storageDirectory ?? Util.getStorageDir(), 'extensions', installed.identity);
    return ExtensionIconSource(ExtensionIconSourceKind.file, path.join(rootDirectory, icon));
  }

  final remoteIcon = item.store?.icon?.trim() ?? '';
  if (remoteIcon.isNotEmpty) {
    return ExtensionIconSource(ExtensionIconSourceKind.network, remoteIcon);
  }
  return const ExtensionIconSource(ExtensionIconSourceKind.asset, fallbackAsset);
}

String _installedExtensionIconUrl(Extension extension, String icon) {
  final iconSegments = icon
      .replaceAll('\\', '/')
      .split('/')
      .where((segment) => segment.isNotEmpty && segment != '.')
      .toList();
  return Uri(pathSegments: ['', 'fs', 'extensions', extension.identity, ...iconSegments]).toString();
}

class ExtensionIcon extends StatelessWidget {
  const ExtensionIcon({
    super.key,
    required this.item,
    required this.size,
    required this.borderRadius,
    this.fallbackPadding = EdgeInsets.zero,
    this.fallbackBackgroundColor,
    this.fallbackFit = BoxFit.contain,
  });

  final ExtensionListItem item;
  final double size;
  final BorderRadius borderRadius;
  final EdgeInsetsGeometry fallbackPadding;
  final Color? fallbackBackgroundColor;
  final BoxFit fallbackFit;

  @override
  Widget build(BuildContext context) {
    final fallback = Container(
      width: size,
      height: size,
      padding: fallbackPadding,
      color: fallbackBackgroundColor,
      child: Image.asset('assets/extension/default_icon.png', fit: fallbackFit),
    );
    final source = resolveExtensionIconSource(item);
    final image = switch (source.kind) {
      ExtensionIconSourceKind.asset => fallback,
      ExtensionIconSourceKind.file => Image.file(
        File(source.location),
        width: size,
        height: size,
        fit: BoxFit.cover,
        errorBuilder: (_, _, _) => fallback,
      ),
      ExtensionIconSourceKind.network => Image.network(
        source.location,
        width: size,
        height: size,
        fit: BoxFit.cover,
        errorBuilder: (_, _, _) => fallback,
      ),
    };
    return ClipRRect(borderRadius: borderRadius, child: image);
  }
}
