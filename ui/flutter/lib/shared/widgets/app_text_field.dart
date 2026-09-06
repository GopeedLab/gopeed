import 'package:flutter/material.dart' as material show AdaptiveTextSelectionToolbar;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

/// The app-wide text field.
///
/// shadcn_flutter's custom context menu is rendered in a full-screen overlay
/// that can cover the app with an opaque surface. Use Flutter's adaptive
/// platform toolbar everywhere so long-press and right-click text actions have
/// consistent native behavior on mobile, desktop, and web.
class AppTextField extends shad.TextField {
  const AppTextField({
    super.key,
    super.groupId,
    super.controller,
    super.initialValue,
    super.focusNode,
    super.undoController,
    super.decoration,
    super.padding,
    super.placeholder,
    super.crossAxisAlignment,
    super.clearButtonSemanticLabel,
    super.keyboardType,
    super.textInputAction,
    super.textCapitalization,
    super.style,
    super.strutStyle,
    super.textAlign,
    super.textAlignVertical,
    super.textDirection,
    super.readOnly,
    super.showCursor,
    super.autofocus,
    super.obscuringCharacter,
    super.obscureText,
    super.autocorrect,
    super.smartDashesType,
    super.smartQuotesType,
    super.enableSuggestions,
    super.maxLines,
    super.minLines,
    super.expands,
    super.maxLength,
    super.maxLengthEnforcement,
    super.onChanged,
    super.onEditingComplete,
    super.onSubmitted,
    super.onTapOutside,
    super.onTapUpOutside,
    super.inputFormatters,
    super.enabled,
    super.cursorWidth,
    super.cursorHeight,
    super.cursorRadius,
    super.cursorOpacityAnimates,
    super.cursorColor,
    super.selectionHeightStyle,
    super.selectionWidthStyle,
    super.keyboardAppearance,
    super.scrollPadding,
    super.enableInteractiveSelection,
    super.selectionControls,
    super.dragStartBehavior,
    super.scrollController,
    super.scrollPhysics,
    super.onTap,
    super.autofillHints,
    super.clipBehavior,
    super.restorationId,
    super.stylusHandwritingEnabled,
    super.enableIMEPersonalizedLearning,
    super.contentInsertionConfiguration,
    super.contextMenuBuilder = appTextFieldContextMenuBuilder,
    super.hintText,
    super.border,
    super.borderRadius,
    super.filled,
    super.statesController,
    super.magnifierConfiguration,
    super.spellCheckConfiguration,
    super.features,
    super.submitFormatters,
    super.skipInputFeatureFocusTraversal,
    super.onDragSelectionStart,
    super.onDragSelectionUpdate,
    super.onDragSelectionEnd,
  });
}

Widget appTextFieldContextMenuBuilder(BuildContext context, EditableTextState editableTextState) {
  // Match Flutter's platform text-menu policy without falling back to
  // shadcn_flutter's full-screen overlay. EditableText bypasses this builder
  // on Web while the browser context menu is enabled.
  if (SystemContextMenu.isSupportedByField(editableTextState)) {
    return SystemContextMenu.editableText(editableTextState: editableTextState);
  }
  return material.AdaptiveTextSelectionToolbar.editableText(editableTextState: editableTextState);
}
