import 'package:flutter/material.dart';

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
    int length = items.length + items.length - 1;
    return Wrap(
      spacing: 8.0,
      runSpacing: 4.0,
      alignment: WrapAlignment.start,
      children: List.generate(
        length,
        (index) {
          final isEven = index % 2 == 0;
          if (isEven) {
            final isLast = index == length - 1;
            final item = items[index ~/ 2];
            return MouseRegion(
              cursor:
                  !isLast ? SystemMouseCursors.click : SystemMouseCursors.basic,
              child: GestureDetector(
                child: Text(
                  item,
                  style: isLast ? activeTextStyle : textStyle,
                ),
                onTap: () {
                  if (onItemTap != null) {
                    onItemTap!(index);
                  }
                },
              ),
            );
          } else {
            return const Text(" > ");
          }
        },
      ),
    );
  }
}
