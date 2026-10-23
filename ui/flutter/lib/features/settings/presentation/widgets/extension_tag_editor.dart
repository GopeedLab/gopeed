import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../api/model/downloader_config.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_text_field.dart';

/// Tag style editor for file extension lists.
///
/// Typed text stays in the input until the submit action (Enter) commits it;
/// commas and whitespace split the committed text into several tags, so
/// `a b,c` becomes `a`, `b` and `c`. Every token goes through
/// [normalizeCategoryExtensions], so the tags are trimmed, lowercased, dot
/// free and deduplicated while keeping the input order. While typing, chips
/// whose tag matches the pending input are highlighted to surface
/// duplicates before they are committed. When the editor is empty and
/// [restoreDefaults] is provided, an outline restore action sits at the end
/// of the [labelText] row and refills the tags. [takenByCategory] lists
/// extensions already claimed by another category: those tokens are never
/// committed, their chips render with the error border, and the editor
/// reports the conflict through [onConflictChanged] so the host can block
/// submission while one is pending.
class ExtensionTagEditor extends StatefulWidget {
  const ExtensionTagEditor({
    super.key,
    this.inputKey,
    required this.initialTags,
    this.labelText,
    this.placeholderText,
    this.usagePlaceholderText,
    this.restoreDefaults,
    this.takenByCategory,
    this.onConflictChanged,
  });

  /// Key attached to the underlying text field so tests can target it.
  final Key? inputKey;

  final List<String> initialTags;

  /// Field label rendered at the top of the editor with the restore action
  /// aligned to its end. Null renders no label row unless the restore action
  /// itself is visible.
  final String? labelText;

  /// Shown inside the text field while there is neither a tag nor typed text.
  final String? placeholderText;

  /// Shown inside the text field while there is no typed text but at least
  /// one tag, carrying the usage rules once the empty-state text no longer
  /// applies.
  final String? usagePlaceholderText;

  /// Default extensions offered by the restore action while the editor is
  /// empty. Null or empty hides the action (e.g. custom categories).
  final List<String>? restoreDefaults;

  /// Normalized extensions already owned by another category, mapped to
  /// that category's display name. Tokens in this map are rejected with an
  /// inline error naming the owner instead of being added to the tags.
  final Map<String, String>? takenByCategory;

  /// Reports whether the editor currently holds an extension claimed by
  /// another category so the host can disable its submit action.
  final ValueChanged<bool>? onConflictChanged;

  @override
  State<ExtensionTagEditor> createState() => ExtensionTagEditorState();
}

class ExtensionTagEditorState extends State<ExtensionTagEditor> {
  late final List<String> _tags;
  late final TextEditingController _controller;
  late bool _lastConflictNotified;

  @override
  void initState() {
    super.initState();
    _tags = normalizeCategoryExtensions(widget.initialTags);
    _controller = TextEditingController();
    // The host seeds its own initial state from the same data, so only
    // subsequent changes need to be reported.
    _lastConflictNotified = _firstTakenConflict() != null;
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  /// Commits any pending input text to the tags and returns the full list.
  /// Extensions claimed by [takenByCategory] are left out.
  List<String> flush() {
    _update(_commitPending);
    final taken = widget.takenByCategory;
    return List<String>.unmodifiable(taken == null ? _tags : _tags.where((tag) => !taken.containsKey(tag)));
  }

  void _update(VoidCallback fn) {
    setState(fn);
    final conflicted = _firstTakenConflict() != null;
    if (conflicted != _lastConflictNotified) {
      _lastConflictNotified = conflicted;
      widget.onConflictChanged?.call(conflicted);
    }
  }

  void _commitPending() {
    final rejected = normalizeCategoryExtensions([
      _controller.text,
    ]).where((token) => widget.takenByCategory?.containsKey(token) ?? false).toList(growable: false);
    _addTokens(_controller.text);
    if (rejected.isEmpty) {
      _controller.clear();
    } else {
      // Keep the rejected tokens in the input so the error stays visible
      // until the user edits them away.
      _controller.text = rejected.join(' ');
      _controller.selection = TextSelection.collapsed(offset: _controller.text.length);
    }
  }

  void _addTokens(String raw) {
    for (final token in normalizeCategoryExtensions([raw])) {
      if (_tags.contains(token)) continue;
      if (widget.takenByCategory?.containsKey(token) ?? false) continue;
      _tags.add(token);
    }
  }

  String? _firstTakenConflict() {
    final taken = widget.takenByCategory;
    if (taken == null || taken.isEmpty) return null;
    for (final token in normalizeCategoryExtensions([_controller.text])) {
      if (taken.containsKey(token)) return token;
    }
    for (final tag in _tags) {
      if (taken.containsKey(tag)) return tag;
    }
    return null;
  }

  void _handleChanged(String value) {
    // Keep the placeholder and the duplicate highlight in sync while typing;
    // nothing is committed until the submit action.
    _update(() {});
  }

  void _handleSubmitted(String value) {
    _update(_commitPending);
  }

  void _removeTag(String tag) {
    _update(() => _tags.remove(tag));
  }

  void _restoreDefaults() {
    _update(() {
      _tags
        ..clear()
        ..addAll(normalizeCategoryExtensions(widget.restoreDefaults ?? const []));
    });
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final showPlaceholder = widget.placeholderText != null && _tags.isEmpty && _controller.text.isEmpty;
    final showRestore = showPlaceholder && (widget.restoreDefaults?.isNotEmpty ?? false);
    String? placeholder;
    if (_controller.text.isEmpty) {
      if (showPlaceholder) {
        placeholder = widget.placeholderText;
      } else if (_tags.isNotEmpty && widget.usagePlaceholderText != null) {
        placeholder = widget.usagePlaceholderText;
      }
    }
    final pendingDuplicates = normalizeCategoryExtensions([_controller.text]).toSet();
    final conflict = _firstTakenConflict();
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (widget.labelText != null || showRestore) ...[
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              if (widget.labelText != null)
                Text(widget.labelText!, style: TextStyle(color: palette.textSecondary, fontSize: 12)),
              if (showRestore) _buildRestoreButton(palette),
            ],
          ),
          const SizedBox(height: 6),
        ],
        if (_tags.isNotEmpty) ...[
          Wrap(
            spacing: AppDesignTokens.space4,
            runSpacing: AppDesignTokens.space4,
            children: [
              for (final tag in _tags)
                _buildChip(
                  palette,
                  tag,
                  duplicate: pendingDuplicates.contains(tag) || (widget.takenByCategory?.containsKey(tag) ?? false),
                ),
            ],
          ),
          const SizedBox(height: 6),
        ],
        AppTextField(
          key: widget.inputKey,
          controller: _controller,
          placeholder: placeholder != null
              ? Text(placeholder, style: TextStyle(fontSize: 12, color: palette.textMuted))
              : null,
          onChanged: _handleChanged,
          onSubmitted: _handleSubmitted,
        ),
        if (conflict != null) ...[
          const SizedBox(height: 6),
          Text(
            context.l10n.categoryExtensionTaken(conflict, widget.takenByCategory![conflict]!),
            style: TextStyle(color: palette.error, fontSize: 12),
          ),
        ],
      ],
    );
  }

  Widget _buildRestoreButton(AppPalette palette) {
    return shad.OutlineButton(
      key: const ValueKey('extension-restore-defaults'),
      onPressed: _restoreDefaults,
      size: shad.ButtonSize.small,
      density: shad.ButtonDensity.compact,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.restart_alt, size: 14, color: palette.brand),
          const SizedBox(width: 4),
          Text(context.l10n.categoryExtensionsRestoreDefaults, style: TextStyle(fontSize: 12, color: palette.brand)),
        ],
      ),
    );
  }

  Widget _buildChip(AppPalette palette, String tag, {required bool duplicate}) {
    return Container(
      key: ValueKey('extension-chip-$tag'),
      padding: const EdgeInsets.only(left: 8, right: 1),
      decoration: BoxDecoration(
        border: Border.all(color: duplicate ? palette.error : palette.border),
        borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(tag, style: TextStyle(fontSize: 11, color: palette.textPrimary)),
          const SizedBox(width: 2),
          GestureDetector(
            key: ValueKey('extension-chip-remove-$tag'),
            onTap: () => _removeTag(tag),
            behavior: HitTestBehavior.opaque,
            child: SizedBox(width: 32, height: 32, child: Icon(Icons.close, size: 13, color: palette.textSecondary)),
          ),
        ],
      ),
    );
  }
}
