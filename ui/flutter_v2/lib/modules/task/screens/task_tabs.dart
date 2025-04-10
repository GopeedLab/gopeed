import 'package:flutter/material.dart';

class TaskTabs extends StatelessWidget {
  final List<String> tabs;
  final int selectedIndex;
  final ValueChanged<int> onTabSelected;
  final List<int>? counts;

  const TaskTabs({
    super.key,
    required this.tabs,
    required this.selectedIndex,
    required this.onTabSelected,
    this.counts,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 366,
      height: 40,
      decoration: BoxDecoration(
        color: const Color.fromRGBO(76, 111, 130, 0.76),
        borderRadius: BorderRadius.circular(40),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceEvenly,
        children: List.generate(tabs.length, (index) {
          final isSelected = index == selectedIndex;
          final selectedFontColor =
              isSelected ? const Color(0xFFFFFFFE) : const Color(0xFFCCCCCC);
          return GestureDetector(
            onTap: () => onTabSelected(index),
            child: Container(
              width: 114,
              height: 32,
              decoration: BoxDecoration(
                color:
                    isSelected ? const Color(0xFF63889C) : Colors.transparent,
                borderRadius: BorderRadius.circular(40),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min, // 行只占用需要的空间
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    tabs[index],
                    style: TextStyle(
                      fontSize: 16.0,
                      fontWeight: FontWeight.bold,
                      color: selectedFontColor,
                    ),
                    maxLines: 1, // 确保文本不换行
                  ),
                  const SizedBox(width: 4), // 文字和计数器之间的固定间距
                  Container(
                    margin: const EdgeInsets.only(bottom: 10), // 使计数器上移
                    child: Text(
                      '${counts![index]}',
                      style: TextStyle(color: selectedFontColor, fontSize: 12),
                    ),
                  ),
                ],
              ),
            ),
          );
        }),
      ),
    );
  }
}
