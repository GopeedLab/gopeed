import 'package:flutter/material.dart' show Colors, Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../theme/app_palette.dart';
import 'app_text_field.dart';

/// Keeps the search icon outside the editable text's horizontal viewport.
class AppSearchField extends StatefulWidget {
  const AppSearchField({
    super.key,
    required this.controller,
    this.focusNode,
    this.placeholder,
    this.style,
    this.padding,
    this.decoration,
    this.iconSize = 16,
    this.onChanged,
    this.onSubmitted,
  });

  final TextEditingController controller;
  final FocusNode? focusNode;
  final Widget? placeholder;
  final TextStyle? style;
  final EdgeInsetsGeometry? padding;
  final BoxDecoration? decoration;
  final double iconSize;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;

  @override
  State<AppSearchField> createState() => _AppSearchFieldState();
}

class _AppSearchFieldState extends State<AppSearchField> {
  late final FocusNode _ownedFocusNode = FocusNode();

  FocusNode get _focusNode => widget.focusNode ?? _ownedFocusNode;

  @override
  void dispose() {
    _ownedFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = shad.Theme.of(context);
    final fieldTheme = shad.ComponentTheme.maybeOf<shad.TextFieldTheme>(context);
    final palette = AppPalette.of(context);
    final padding = widget.padding ?? fieldTheme?.padding ?? const EdgeInsets.symmetric(horizontal: 12, vertical: 8);
    final decoration =
        widget.decoration ??
        BoxDecoration(
          borderRadius: fieldTheme?.borderRadius ?? BorderRadius.circular(theme.radiusMd),
          border: fieldTheme?.border ?? Border.all(color: theme.colorScheme.border),
          color: fieldTheme?.filled == true ? theme.colorScheme.muted : theme.colorScheme.input.withValues(alpha: 0.3),
        );
    return ListenableBuilder(
      listenable: _focusNode,
      builder: (context, _) => shad.FocusOutline(
        focused: _focusNode.hasFocus,
        borderRadius: decoration.borderRadius,
        child: GestureDetector(
          behavior: HitTestBehavior.translucent,
          onTap: _focusNode.requestFocus,
          child: Container(
            decoration: decoration,
            clipBehavior: Clip.hardEdge,
            padding: padding,
            child: LayoutBuilder(
              builder: (context, constraints) => Row(
                children: [
                  // A compressed desktop toolbar may leave almost no search
                  // width just before switching to its mobile layout.
                  if (constraints.maxWidth > widget.iconSize + theme.density.baseGap * theme.scaling) ...[
                    Icon(Icons.search_rounded, size: widget.iconSize, color: palette.textMuted),
                    SizedBox(width: theme.density.baseGap * theme.scaling),
                  ],
                  Expanded(
                    // The underlying field allows text to paint outside its own
                    // viewport. Clip only the editable area, leaving the icon fixed.
                    child: ClipRect(
                      clipper: const _SearchTextClipper(),
                      child: shad.ComponentTheme<shad.FocusOutlineTheme>(
                        data: const shad.FocusOutlineTheme(
                          align: 0,
                          border: Border.fromBorderSide(BorderSide(color: Colors.transparent, width: 0)),
                        ),
                        child: AppTextField(
                          controller: widget.controller,
                          focusNode: _focusNode,
                          placeholder: widget.placeholder,
                          style: widget.style,
                          padding: EdgeInsets.zero,
                          decoration: const BoxDecoration(),
                          onChanged: widget.onChanged,
                          onSubmitted: widget.onSubmitted,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _SearchTextClipper extends CustomClipper<Rect> {
  const _SearchTextClipper();

  // Keep horizontal scrolling out of the icon area without cutting glyphs or
  // tall cursors in compact fields. The outer field clips its own border.
  @override
  Rect getClip(Size size) => Rect.fromLTRB(0, -size.height, size.width, size.height * 2);

  @override
  bool shouldReclip(_SearchTextClipper oldClipper) => false;
}
