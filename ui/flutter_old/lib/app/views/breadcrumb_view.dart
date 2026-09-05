import 'package:flutter/material.dart';
import 'package:get/get.dart';

class Breadcrumb extends StatelessWidget {
  final List<String> items;
  final Function(int)? onItemTap;
  final TextStyle textStyle;
  final TextStyle activeTextStyle;

  const Breadcrumb({
    super.key,
    required this.items,
    this.onItemTap,
    this.textStyle = const TextStyle(fontSize: 16),
    this.activeTextStyle =
        const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
  });

  @override
  Widget build(BuildContext context) {
    List<Widget> children = [];
    for (int i = 0; i < items.length; i++) {
      children.add(
        GestureDetector(
          onTap: () {
            if (onItemTap != null) {
              onItemTap!(i);
            }
          },
          child: Text(
            items[i],
            style: i == items.length - 1 ? activeTextStyle : textStyle,
          ),
        ),
      );
      if (i != items.length - 1) {
        children.add(const Text(" > "));
      }
    }
    return Row(
      children: [
        ...(children.length == 1
            ? children.sublist(0, 1)
            : children.sublist(0, 2)),
        children.length > 2
            ? Expanded(
                child: SingleChildScrollView(
                  reverse: true,
                  scrollDirection: Axis.horizontal,
                  child: Row(
                    children: children.sublist(2),
                  ),
                ),
              )
            : null,
      ].where((e) => e != null).map((e) => e!).toList(),
    ).paddingOnly(right: 12);
  }
}
