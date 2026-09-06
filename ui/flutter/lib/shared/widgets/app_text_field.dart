import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

/// The app-wide text field.
///
/// shadcn_flutter's custom mobile context menu is rendered in a full-screen
/// overlay. On Android and iOS that overlay can cover the app with an opaque
/// surface when text is long-pressed. Use Flutter's adaptive native toolbar on
/// mobile, while preserving shadcn_flutter's menu on desktop and web.
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

@visibleForTesting
bool appTextFieldUsesNativeContextMenu({TargetPlatform? platform, bool isWeb = kIsWeb}) {
  final effectivePlatform = platform ?? defaultTargetPlatform;
  return !isWeb && (effectivePlatform == TargetPlatform.android || effectivePlatform == TargetPlatform.iOS);
}

Widget appTextFieldContextMenuBuilder(BuildContext context, EditableTextState editableTextState) {
  if (appTextFieldUsesNativeContextMenu()) {
    return shad.TextField.nativeContextMenuBuilder()(context, editableTextState);
  }
  return shad.TextField.defaultContextMenuBuilder(context, editableTextState);
}
