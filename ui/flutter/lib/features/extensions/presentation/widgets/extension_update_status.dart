import 'package:flutter/widgets.dart';

import '../../../../shared/theme/app_palette.dart';

/// Green status dot with an optional label, marking an available update.
/// Pass [onTap] to make it open the update prompt.
class ExtensionUpdateStatus extends StatelessWidget {
  const ExtensionUpdateStatus({super.key, this.label, this.onTap});

  final String? label;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final content = Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 6,
          height: 6,
          decoration: BoxDecoration(color: palette.brand, shape: BoxShape.circle),
        ),
        if (label != null) ...[
          const SizedBox(width: 5),
          Flexible(
            child: Text(
              label!,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: palette.textPrimary, fontSize: 11.5, fontWeight: FontWeight.w600),
            ),
          ),
        ],
      ],
    );
    if (onTap == null) return content;
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 6),
          child: content,
        ),
      ),
    );
  }
}
