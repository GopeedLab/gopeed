import 'package:flutter/widgets.dart';

import '../../core/utils/breakpoints.dart';
import '../theme/app_palette.dart';
import 'secondary_navigation_pane.dart';

class ResponsiveNavigationItem<T> {
  const ResponsiveNavigationItem({required this.value, required this.label, this.icon, this.count});

  final T value;
  final String label;
  final IconData? icon;
  final int? count;
}

class ResponsiveNavigationLayout<T> extends StatelessWidget {
  const ResponsiveNavigationLayout({
    super.key,
    required this.items,
    required this.selectedValue,
    required this.onSelected,
    required this.child,
    required this.desktopTitle,
    this.desktopFooter,
    this.mobileHeader,
    this.breakpoint = Breakpoints.mobile,
  });

  final List<ResponsiveNavigationItem<T>> items;
  final T selectedValue;
  final ValueChanged<T> onSelected;
  final Widget child;
  final String desktopTitle;
  final Widget? desktopFooter;
  final Widget? mobileHeader;
  final double breakpoint;

  @override
  Widget build(BuildContext context) {
    final isDesktop = MediaQuery.sizeOf(context).width >= breakpoint;
    if (isDesktop) {
      return Row(
        children: [
          SecondaryNavigationPane<T>(
            title: desktopTitle,
            items: items
                .map(
                  (item) => SecondaryNavigationPaneItem<T>(
                    value: item.value,
                    label: item.label,
                    icon: item.icon,
                    trailing: item.count?.toString(),
                  ),
                )
                .toList(),
            selectedValue: selectedValue,
            onSelected: onSelected,
            footer: desktopFooter,
          ),
          Expanded(child: child),
        ],
      );
    }

    return Column(
      children: [
        ?mobileHeader,
        _MobileNavigationTabs<T>(
          items: items,
          selectedValue: selectedValue,
          onSelected: onSelected,
          reserveTopSafeArea: mobileHeader == null,
        ),
        Expanded(child: child),
      ],
    );
  }
}

class _MobileNavigationTabs<T> extends StatelessWidget {
  const _MobileNavigationTabs({
    required this.items,
    required this.selectedValue,
    required this.onSelected,
    required this.reserveTopSafeArea,
  });

  final List<ResponsiveNavigationItem<T>> items;
  final T selectedValue;
  final ValueChanged<T> onSelected;
  final bool reserveTopSafeArea;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      width: double.infinity,
      decoration: BoxDecoration(
        color: palette.bg,
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: SafeArea(
        top: reserveTopSafeArea,
        bottom: false,
        child: SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 10),
          child: Container(
            padding: const EdgeInsets.all(3),
            decoration: BoxDecoration(
              color: palette.surfaceSoft,
              borderRadius: BorderRadius.circular(6),
              border: Border.all(color: palette.border),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                for (final item in items)
                  _MobileNavigationTab<T>(
                    item: item,
                    active: item.value == selectedValue,
                    onTap: () => onSelected(item.value),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _MobileNavigationTab<T> extends StatelessWidget {
  const _MobileNavigationTab({required this.item, required this.active, required this.onTap});

  final ResponsiveNavigationItem<T> item;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final label = item.count == null ? item.label : '${item.label} ${item.count}';
    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 140),
        curve: Curves.easeOutCubic,
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        decoration: BoxDecoration(color: active ? palette.cardBg : null, borderRadius: BorderRadius.circular(4)),
        child: Text(
          label,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(
            color: active ? palette.textPrimary : palette.textSecondary,
            fontSize: 12,
            fontWeight: active ? FontWeight.w700 : FontWeight.w500,
          ),
        ),
      ),
    );
  }
}
