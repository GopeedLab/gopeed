import 'package:flutter/material.dart';
import 'gradient_button.dart';
import 'task_tabs.dart';

class TaskScreen extends StatefulWidget {
  const TaskScreen({super.key});

  @override
  State<TaskScreen> createState() => _TaskScreenState();
}

class _TaskScreenState extends State<TaskScreen> {
  int _selectedTabIndex = 0;
  final List<String> _tabs = ['下载中', '已完成', '任务失败'];
  // 示例数据，实际应用中应该从数据源获取
  final List<int> _counts = [1, 12, 313];

  void _handleTabSelected(int index) {
    setState(() {
      _selectedTabIndex = index;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16.0),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(left: 16.0, top: 16.0),
            child: Row(
              children: [
                GradientButton(
                  text: '创建任务',
                  icon: Icons.add, // 添加加号图标
                  onPressed: () {
                    // 处理创建任务按钮点击事件
                  },
                ),
                const SizedBox(width: 20), // 添加间距
                TaskTabs(
                  tabs: _tabs,
                  selectedIndex: _selectedTabIndex,
                  onTabSelected: _handleTabSelected,
                  counts: _counts, // 添加计数数据
                ),
              ],
            ),
          ),
          const SizedBox(height: 20),
          Expanded(child: _buildTaskList(_selectedTabIndex)),
        ],
      ),
    );
  }

  Widget _buildTaskList(int tabIndex) {
    // 根据选中的标签页显示不同的任务列表
    switch (tabIndex) {
      case 0:
        return const Center(
          child: Text('下载中的任务列表', style: TextStyle(color: Colors.white)),
        );
      case 1:
        return const Center(
          child: Text('已完成的任务列表', style: TextStyle(color: Colors.white)),
        );
      case 2:
        return const Center(
          child: Text('失败的任务列表', style: TextStyle(color: Colors.white)),
        );
      default:
        return Container();
    }
  }
}
