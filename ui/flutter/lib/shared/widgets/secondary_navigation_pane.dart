import 'package:flutter/material.dart' show Colors, Icons;
import 'package:flutter/widgets.dart';

import '../theme/app_design_tokens.dart';
import '../theme/app_palette.dart';

class SecondaryNavigationPaneItem<T> {
  const SecondaryNavigationPaneItem({required this.value, required this.label, this.icon, this.trailing});

  final T value;
  final String label;
  final IconData? icon;
  final String? trailing;
}

class SecondaryNavigationPane<T> extends StatelessWidget {
  const SecondaryNavigationPane({
    super.key,
    required this.title,
    required this.items,
    required this.selectedValue,
    required this.onSelected,
    this.footer,
    this.mobile = false,
    this.showDisclosure = false,
    this.width = AppDesignTokens.filterSidebarWidth,
  });

  final String title;
  final List<SecondaryNavigationPaneItem<T>> items;
  final T selectedValue;
  final ValueChanged<T> onSelected;
  final Widget? footer;
  final bool mobile;
  final bool showDisclosure;
  final double width;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      key: const ValueKey('secondary-navigation-pane'),
      width: mobile ? double.infinity : width,
      color: palette.sideBg,
      padding: EdgeInsets.only(top: mobile ? 0 : AppDesignTokens.windowHeaderHeight),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            height: AppDesignTokens.contentHeaderHeight,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text(
                  title.toUpperCase(),
                  style: TextStyle(
                    color: palette.textMuted,
                    fontSize: 12,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 2.4,
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(height: AppDesignTokens.space8),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Column(
              children: [
                for (final (index, item) in items.indexed)
                  _SecondaryNavigationPaneItem<T>(
                    key: ValueKey('secondary-navigation-item-$index'),
                    item: item,
                    active: item.value == selectedValue,
                    showDisclosure: showDisclosure,
                    onTap: () => onSelected(item.value),
                  ),
              ],
            ),
          ),
          if (footer != null) ...[const Spacer(), footer!] else const Spacer(),
        ],
      ),
    );
  }
}

class _SecondaryNavigationPaneItem<T> extends StatelessWidget {
  const _SecondaryNavigationPaneItem({
    super.key,
    required this.item,
    required this.active,
    required this.showDisclosure,
    required this.onTap,
  });

  final SecondaryNavigationPaneItem<T> item;
  final bool active;
  final bool showDisclosure;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        child: Container(
          margin: const EdgeInsets.only(bottom: 4),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: active ? palette.itemActiveBg : Colors.transparent,
            borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
          ),
          child: Row(
            children: [
              if (item.icon != null) ...[
                Icon(item.icon, size: 16, color: active ? palette.textPrimary : palette.textSecondary),
                const SizedBox(width: 12),
              ],
              Expanded(
                child: Text(
                  item.label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: active ? palette.textPrimary : palette.textSecondary,
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
              if (item.trailing != null)
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: palette.cardHoverBg,
                    borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
                  ),
                  child: Text(
                    item.trailing!,
                    style: TextStyle(
                      color: active ? palette.textPrimary : palette.textSecondary,
                      fontSize: 10,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                )
              else if (showDisclosure)
                Icon(Icons.chevron_right, size: 15, color: palette.textMuted),
            ],
          ),
        ),
      ),
    );
  }
}
