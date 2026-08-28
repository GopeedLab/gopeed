import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../core/utils/breakpoints.dart';
import '../services/download_directory_picker.dart';
import 'app_tooltip.dart';

class AppPathPickerField extends StatefulWidget {
  const AppPathPickerField({
    super.key,
    required this.controller,
    required this.onPick,
    this.fieldKey,
    this.pickerKey,
    this.hintText,
    this.desktopWidth,
  }) : platformDownloadDirectory = false,
       onDirectoryPicked = null,
       allowAndroidEditing = false;

  const AppPathPickerField.downloadDirectory({
    super.key,
    required this.controller,
    this.fieldKey,
    this.pickerKey,
    this.hintText,
    this.desktopWidth,
    this.onDirectoryPicked,
    this.allowAndroidEditing = false,
  }) : onPick = null,
       platformDownloadDirectory = true;

  final TextEditingController controller;
  final VoidCallback? onPick;
  final Key? fieldKey;
  final Key? pickerKey;
  final String? hintText;
  final double? desktopWidth;
  final bool platformDownloadDirectory;
  final ValueChanged<String>? onDirectoryPicked;
  final bool allowAndroidEditing;

  @override
  State<AppPathPickerField> createState() => _AppPathPickerFieldState();
}

class _AppPathPickerFieldState extends State<AppPathPickerField> {
  late String _tooltipPath = widget.controller.text.trim();

  @override
  void initState() {
    super.initState();
    widget.controller.addListener(_syncTooltipPath);
  }

  @override
  void didUpdateWidget(covariant AppPathPickerField oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      oldWidget.controller.removeListener(_syncTooltipPath);
      widget.controller.addListener(_syncTooltipPath);
      _tooltipPath = widget.controller.text.trim();
    }
  }

  @override
  void dispose() {
    widget.controller.removeListener(_syncTooltipPath);
    super.dispose();
  }

  void _syncTooltipPath([String? value]) {
    final path = (value ?? widget.controller.text).trim();
    if (path == _tooltipPath || !mounted) return;
    setState(() => _tooltipPath = path);
  }

  Future<void> _pickPlatformDirectory() async {
    final path = await DownloadDirectoryPicker.pick(context, currentPath: widget.controller.text.trim());
    if (!mounted || path == null || path.isEmpty || path == widget.controller.text) return;
    widget.controller.text = path;
    _syncTooltipPath(path);
    widget.onDirectoryPicked?.call(path);
  }

  @override
  Widget build(BuildContext context) {
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    final onPick = widget.platformDownloadDirectory
        ? (DownloadDirectoryPicker.canPick ? _pickPlatformDirectory : null)
        : widget.onPick;
    return SizedBox(
      width: desktop ? widget.desktopWidth : double.infinity,
      child: Row(
        children: [
          Expanded(
            child: MouseRegion(
              onEnter: (_) => _syncTooltipPath(),
              child: AppTooltip(
                message: _tooltipPath,
                child: shad.TextField(
                  key: widget.fieldKey,
                  controller: widget.controller,
                  hintText: widget.hintText,
                  readOnly:
                      widget.platformDownloadDirectory &&
                      DownloadDirectoryPicker.isMobile &&
                      !(DownloadDirectoryPicker.isAndroid && widget.allowAndroidEditing),
                ),
              ),
            ),
          ),
          if (onPick != null) ...[
            const SizedBox(width: 2),
            shad.GhostButton(
              key: widget.pickerKey,
              density: shad.ButtonDensity.icon,
              onPressed: onPick,
              child: const Icon(Icons.folder_open_outlined, size: 17),
            ),
          ],
        ],
      ),
    );
  }
}
