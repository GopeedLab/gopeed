import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../core/utils/breakpoints.dart';
import '../theme/app_palette.dart';
import 'secondary_navigation_pane.dart';

class ResponsiveMenuItem<T> {
  const ResponsiveMenuItem({required this.value, required this.label, required this.icon, this.trailing});

  final T value;
  final String label;
  final IconData icon;
  final String? trailing;
}

class ResponsiveMenuLayout<T> extends StatefulWidget {
  const ResponsiveMenuLayout({
    super.key,
    required this.title,
    required this.items,
    required this.selectedValue,
    required this.onSelected,
    required this.contentBuilder,
    this.mobileContentTitleBuilder,
    this.sidebarFooter,
    this.breakpoint = Breakpoints.mobile,
  });

  final String title;
  final List<ResponsiveMenuItem<T>> items;
  final T selectedValue;
  final ValueChanged<T> onSelected;
  final Widget Function(BuildContext context, T selectedValue) contentBuilder;
  final String Function(T selectedValue)? mobileContentTitleBuilder;
  final Widget? sidebarFooter;
  final double breakpoint;

  @override
  State<ResponsiveMenuLayout<T>> createState() => _ResponsiveMenuLayoutState<T>();
}

class _ResponsiveMenuLayoutState<T> extends State<ResponsiveMenuLayout<T>> {
  T? _mobileSelectedValue;

  @override
  Widget build(BuildContext context) {
    final isDesktop = MediaQuery.sizeOf(context).width >= widget.breakpoint;
    if (isDesktop) {
      return Row(
        children: [
          SecondaryNavigationPane<T>(
            title: widget.title,
            items: widget.items
                .map(
                  (item) => SecondaryNavigationPaneItem<T>(
                    value: item.value,
                    label: item.label,
                    icon: item.icon,
                    trailing: item.trailing,
                  ),
                )
                .toList(),
            selectedValue: widget.selectedValue,
            onSelected: widget.onSelected,
            footer: widget.sidebarFooter,
          ),
          Expanded(child: widget.contentBuilder(context, widget.selectedValue)),
        ],
      );
    }

    final selected = _mobileSelectedValue;
    if (selected == null) {
      return SecondaryNavigationPane<T>(
        title: widget.title,
        items: widget.items
            .map(
              (item) => SecondaryNavigationPaneItem<T>(
                value: item.value,
                label: item.label,
                icon: item.icon,
                trailing: item.trailing,
              ),
            )
            .toList(),
        selectedValue: widget.selectedValue,
        onSelected: (value) {
          widget.onSelected(value);
          setState(() => _mobileSelectedValue = value);
        },
        footer: widget.sidebarFooter,
        mobile: true,
        showDisclosure: true,
      );
    }

    return Column(
      children: [
        _MobileContentHeader(
          title: widget.mobileContentTitleBuilder?.call(selected) ?? widget.title,
          onBack: () => setState(() => _mobileSelectedValue = null),
        ),
        Expanded(child: widget.contentBuilder(context, selected)),
      ],
    );
  }
}

class _MobileContentHeader extends StatelessWidget {
  const _MobileContentHeader({required this.title, required this.onBack});

  final String title;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      height: 52,
      decoration: BoxDecoration(
        color: palette.bg,
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 8),
      child: Row(
        children: [
          shad.GhostButton(
            density: shad.ButtonDensity.icon,
            onPressed: onBack,
            child: Icon(Icons.arrow_back, size: 18, color: palette.textPrimary),
          ),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: palette.textPrimary, fontSize: 16, fontWeight: FontWeight.w800),
            ),
          ),
        ],
      ),
    );
  }
}
