import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../core/utils/breakpoints.dart';
import '../services/download_directory_picker.dart';
import 'app_tooltip.dart';

enum AppPathPickerButtonStyle { ghost, outline }

class AppPathPickerField extends StatefulWidget {
  const AppPathPickerField({
    super.key,
    required this.controller,
    required this.onPick,
    this.fieldKey,
    this.pickerKey,
    this.hintText,
    this.desktopWidth,
    this.filled,
    this.border,
    this.borderRadius,
    this.padding,
    this.onChanged,
    this.pickerButtonStyle = AppPathPickerButtonStyle.ghost,
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
    this.filled,
    this.border,
    this.borderRadius,
    this.padding,
    this.onChanged,
    this.pickerButtonStyle = AppPathPickerButtonStyle.ghost,
  }) : onPick = null,
       platformDownloadDirectory = true;

  final TextEditingController controller;
  final VoidCallback? onPick;
  final Key? fieldKey;
  final Key? pickerKey;
  final String? hintText;
  final double? desktopWidth;
  final bool? filled;
  final Border? border;
  final BorderRadiusGeometry? borderRadius;
  final EdgeInsetsGeometry? padding;
  final ValueChanged<String>? onChanged;
  final AppPathPickerButtonStyle pickerButtonStyle;
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
                      !DownloadDirectoryPicker.canEditManually(allowAndroidEditing: widget.allowAndroidEditing),
                  filled: widget.filled,
                  border: widget.border,
                  borderRadius: widget.borderRadius,
                  padding: widget.padding,
                  onChanged: widget.onChanged,
                  features: [
                    if (onPick != null && widget.pickerButtonStyle == AppPathPickerButtonStyle.outline)
                      _PathPickerSuffixFeature(pickerKey: widget.pickerKey, onPressed: onPick),
                  ],
                ),
              ),
            ),
          ),
          if (onPick != null && widget.pickerButtonStyle == AppPathPickerButtonStyle.ghost) ...[
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

class _PathPickerSuffixFeature extends shad.InputFeature {
  const _PathPickerSuffixFeature({required this.pickerKey, required this.onPressed}) : super(skipFocusTraversal: false);

  final Key? pickerKey;
  final VoidCallback onPressed;

  @override
  shad.InputFeatureState createState() => _PathPickerSuffixFeatureState();
}

class _PathPickerSuffixFeatureState extends shad.InputFeatureState<_PathPickerSuffixFeature> {
  @override
  Iterable<Widget> buildSuffix() sync* {
    yield shad.IconButton.outline(
      key: feature.pickerKey,
      onPressed: feature.onPressed,
      icon: const Icon(Icons.folder_open_outlined, size: 17),
    );
  }
}
