import 'package:file_picker/file_picker.dart';
import 'package:device_info_plus/device_info_plus.dart';
import 'package:external_path/external_path.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../l10n/l10n.dart';
import '../theme/app_palette.dart';
import 'download_directory_file_stub.dart' if (dart.library.io) 'download_directory_file_io.dart';
import 'download_directory_host_platform_stub.dart' if (dart.library.io) 'download_directory_host_platform_io.dart';

/// Keeps directory selection compatible with the raw paths consumed by the Go
/// downloader. Android therefore exposes only paths whose writability can be
/// verified without relying on a persisted Storage Access Framework URI.
class DownloadDirectoryPicker {
  DownloadDirectoryPicker._();

  @visibleForTesting
  static TargetPlatform? debugPlatformOverride;

  @visibleForTesting
  static bool? debugWebOverride;

  @visibleForTesting
  static Future<Map<String, String>?> Function()? debugAndroidLocationsLoader;

  @visibleForTesting
  static Future<String> Function(String path)? debugDownloadsPreparer;

  static TargetPlatform get _platform {
    final override = debugPlatformOverride;
    if (override != null) return override;
    return downloadDirectoryHostPlatform;
  }

  static bool get isWeb => debugWebOverride ?? kIsWeb;

  static bool get isAndroid => !isWeb && _platform == TargetPlatform.android;

  static bool get isIOS => !isWeb && _platform == TargetPlatform.iOS;

  static bool get isMobile => isAndroid || isIOS;

  static bool get canPick => !isWeb && _platform != TargetPlatform.iOS;

  static bool canEditManually({bool allowAndroidEditing = false}) {
    return isWeb || !isMobile || (isAndroid && allowAndroidEditing);
  }

  static Future<String?> pick(BuildContext context, {required String currentPath}) async {
    if (isWeb) return null;
    if (_platform == TargetPlatform.android) {
      return _pickAndroid(context, currentPath: currentPath);
    }
    if (_platform == TargetPlatform.iOS) return null;
    return FilePicker.platform.getDirectoryPath();
  }

  static Future<String?> _pickAndroid(BuildContext context, {required String currentPath}) async {
    final locations = await _loadAndroidLocations();
    if (!context.mounted || locations == null) return null;
    final applicationPath = locations['application'];
    final downloadsPath = locations['downloads'];
    if (applicationPath == null || applicationPath.isEmpty || downloadsPath == null || downloadsPath.isEmpty) {
      return null;
    }

    String? errorMessage;
    var checkingDownloads = false;
    final overlay = const shad.DialogOverlayHandler().show<String?>(
      context: context,
      alignment: Alignment.center,
      barrierDismissable: false,
      builder: (dialogContext) => StatefulBuilder(
        builder: (dialogContext, setDialogState) {
          Future<void> chooseDownloads() async {
            if (checkingDownloads) return;
            setDialogState(() {
              checkingDownloads = true;
              errorMessage = null;
            });
            try {
              final verifiedPath = await _prepareDownloads(downloadsPath);
              if (dialogContext.mounted && verifiedPath.isNotEmpty) {
                shad.closeOverlay(dialogContext, verifiedPath);
              }
            } catch (_) {
              if (dialogContext.mounted) {
                setDialogState(() => errorMessage = dialogContext.l10n.downloadsDirectoryUnavailable);
              }
            } finally {
              if (dialogContext.mounted) setDialogState(() => checkingDownloads = false);
            }
          }

          return shad.AlertDialog(
            key: const ValueKey('android-download-directory-dialog'),
            title: Text(dialogContext.l10n.chooseDownloadDirectory),
            content: SizedBox(
              width: (MediaQuery.sizeOf(dialogContext).width - 64).clamp(260.0, 400.0),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _DirectoryOption(
                    key: const ValueKey('android-app-directory-option'),
                    icon: Icons.smartphone_outlined,
                    title: dialogContext.l10n.applicationStorage,
                    description: dialogContext.l10n.applicationStorageDescription,
                    selected: currentPath == applicationPath,
                    onPressed: checkingDownloads ? null : () => shad.closeOverlay(dialogContext, applicationPath),
                  ),
                  const SizedBox(height: 8),
                  _DirectoryOption(
                    key: const ValueKey('android-downloads-directory-option'),
                    icon: Icons.download_outlined,
                    title: 'Download/Gopeed',
                    description: dialogContext.l10n.publicDownloadsDescription,
                    selected: currentPath == downloadsPath,
                    loading: checkingDownloads,
                    onPressed: checkingDownloads ? null : chooseDownloads,
                  ),
                  if (errorMessage != null) ...[
                    const SizedBox(height: 8),
                    Text(
                      errorMessage!,
                      style: TextStyle(color: AppPalette.of(dialogContext).error, fontSize: 12, height: 1.3),
                    ),
                  ],
                ],
              ),
            ),
            actions: [
              shad.SecondaryButton(
                onPressed: checkingDownloads ? null : () => shad.closeOverlay(dialogContext),
                child: Text(dialogContext.l10n.cancel),
              ),
            ],
          );
        },
      ),
    );
    return overlay.future;
  }

  static Future<Map<String, String>?> _loadAndroidLocations() async {
    final debugLoader = debugAndroidLocationsLoader;
    if (debugLoader != null) return debugLoader();
    final applicationDirectory = await getExternalStorageDirectory() ?? await getApplicationDocumentsDirectory();
    final downloadsDirectory = await ExternalPath.getExternalStoragePublicDirectory(ExternalPath.DIRECTORY_DOWNLOAD);
    if (downloadsDirectory.isEmpty) return null;
    return {'application': applicationDirectory.path, 'downloads': path.join(downloadsDirectory, 'Gopeed')};
  }

  static Future<String> _prepareDownloads(String downloadsPath) async {
    final debugPreparer = debugDownloadsPreparer;
    if (debugPreparer != null) return debugPreparer(downloadsPath);
    final androidInfo = await DeviceInfoPlugin().androidInfo;
    if (androidInfo.version.sdkInt < 30) {
      var permission = await Permission.storage.status;
      if (!permission.isGranted) permission = await Permission.storage.request();
      if (!permission.isGranted) throw StateError('Storage permission was denied');
    }
    await verifyDownloadDirectoryWritable(downloadsPath);
    return downloadsPath;
  }
}

class _DirectoryOption extends StatelessWidget {
  const _DirectoryOption({
    super.key,
    required this.icon,
    required this.title,
    required this.description,
    required this.selected,
    required this.onPressed,
    this.loading = false,
  });

  final IconData icon;
  final String title;
  final String description;
  final bool selected;
  final VoidCallback? onPressed;
  final bool loading;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return shad.SecondaryButton(
      onPressed: onPressed,
      child: SizedBox(
        width: double.infinity,
        child: Row(
          children: [
            Icon(icon, size: 20, color: palette.textSecondary),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title, style: TextStyle(color: palette.textPrimary, fontSize: 13)),
                  const SizedBox(height: 2),
                  Text(description, style: TextStyle(color: palette.textSecondary, fontSize: 11, height: 1.25)),
                ],
              ),
            ),
            const SizedBox(width: 8),
            if (loading)
              const SizedBox.square(dimension: 15, child: shad.CircularProgressIndicator())
            else if (selected)
              Icon(Icons.check_circle_outline, size: 18, color: palette.success),
          ],
        ),
      ),
    );
  }
}
