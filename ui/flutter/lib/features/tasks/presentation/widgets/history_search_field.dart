import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_text_field.dart';

class HistorySearchField extends StatelessWidget {
  const HistorySearchField({super.key, required this.controller, required this.onChanged});

  final TextEditingController controller;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    const textStyle = TextStyle(fontSize: 13);
    return ConstrainedBox(
      // Allow large accessibility text to grow instead of clipping the line.
      constraints: const BoxConstraints(minHeight: 38),
      child: AppTextField(
        key: const ValueKey('create-history-filter'),
        controller: controller,
        style: textStyle,
        padding: AppDesignTokens.compactTextFieldPadding,
        placeholder: Text(
          context.l10n.searchHistory,
          // Match the editable text metrics so the cursor and hint align.
          style: textStyle.copyWith(color: palette.searchHint),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        features: [InputFeature.leading(Icon(Icons.search_rounded, size: 15, color: palette.textMuted))],
        onChanged: onChanged,
      ),
    );
  }
}
