import 'package:flutter/material.dart';
import '../../../components/gp_gradient_button.dart';
import '../../../components/gp_outline_button.dart';
import '../../../components/gp_icon_button.dart';
import '../widgets/task_item.dart';
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
                GpGradientButton(
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
          const SizedBox(height: 16),
          // 操作按钮列表
          Padding(
            padding: const EdgeInsets.only(right: 24.0),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                GpOutlineButton(
                  icon: 'assets/icons/resume.svg',
                  onTap: () {
                    // 处理开始按钮点击事件
                  },
                ),
                const SizedBox(width: 12),
                GpOutlineButton(
                  icon: 'assets/icons/pause.svg',
                  onTap: () {
                    // 处理暂停按钮点击事件
                  },
                ),
                const SizedBox(width: 12),
                GpOutlineButton(
                  icon: 'assets/icons/delete.svg',
                  onTap: () {
                    // 处理删除按钮点击事件
                  },
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          Expanded(child: _buildTaskList(_selectedTabIndex)),
        ],
      ),
    );
  }

  Widget _buildTaskList(int tabIndex) {
    // 根据选中的标签页显示不同的任务列表
    switch (tabIndex) {
      case 0:
        return ListView.separated(
          itemCount: 3, // 示例显示3个任务项
          separatorBuilder: (context, index) => const SizedBox(height: 12),
          itemBuilder:
              (context, index) => TaskItem(
                icon: Icons.download, // 添加下载图标
                taskName: '下载任务 ${index + 1}', // 添加任务名称
                progress: (index + 1) * 0.25, // 添加进度值 (0.25, 0.5, 0.75)
                fileSizeText:
                    '${(2.1 + index * 0.1).toStringAsFixed(1)}MB/2.4MB', // 示例文件大小
                bottomRightIcon: Icons.speed, // 添加速度图标
                bottomRightGradientText:
                    '${(1.5 + index * 0.3).toStringAsFixed(1)}MB/s', // 下载速度
                bottomRightStatusText: '下载中', // 状态文本
                actionButtons: [
                  GpIconButton(
                    icon: 'assets/icons/item_pause.svg',
                    onTap: () => print('暂停任务 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_delete.svg',
                    onTap: () => print('删除任务 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_reveal.svg',
                    onTap: () => print('打开所在·位置 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_info.svg',
                    onTap: () => print('查看信息 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_more.svg',
                    onTap: () => print('更多操作 $index'),
                  ),
                ],
                onTap: () {
                  print('任务项 $index 被点击');
                },
              ),
        );
      case 1:
        return ListView.separated(
          itemCount: 2, // 示例显示2个任务项
          separatorBuilder: (context, index) => const SizedBox(height: 12),
          itemBuilder:
              (context, index) => TaskItem(
                icon: Icons.check_circle, // 添加完成图标
                taskName: '已完成任务 ${index + 1}', // 添加任务名称
                progress: 1.0, // 已完成任务进度为100%
                fileSizeText: '2.4MB/2.4MB - 已完成', // 添加完成信息
                bottomRightIcon: Icons.check, // 添加完成图标
                bottomRightGradientText: '完成时间', // 完成时间文本
                bottomRightStatusText: '已完成', // 状态文本
                actionButtons: [
                  GpIconButton(
                    icon: 'assets/icons/item_resume.svg',
                    onTap: () => print('重新下载任务 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_delete.svg',
                    onTap: () => print('删除任务 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_reveal.svg',
                    onTap: () => print('打开所在位置 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_info.svg',
                    onTap: () => print('查看信息 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_more.svg',
                    onTap: () => print('更多操作 $index'),
                  ),
                ],
                onTap: () {
                  print('已完成任务项 $index 被点击');
                },
              ),
        );
      case 2:
        return ListView.separated(
          itemCount: 1, // 示例显示1个任务项
          separatorBuilder: (context, index) => const SizedBox(height: 12),
          itemBuilder:
              (context, index) => TaskItem(
                icon: Icons.error, // 添加错误图标
                taskName: '失败任务 ${index + 1}', // 添加任务名称
                progress: 0.3, // 失败任务的部分进度
                fileSizeText: '1.2MB/4.0MB - 任务失败', // 添加失败信息
                bottomRightIcon: Icons.error_outline, // 添加错误图标
                bottomRightGradientText: '重试', // 重试文本
                bottomRightStatusText: '失败', // 状态文本
                actionButtons: [
                  GpIconButton(
                    icon: 'assets/icons/item_resume.svg',
                    onTap: () => print('重试任务 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_delete.svg',
                    onTap: () => print('删除任务 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_reveal.svg',
                    onTap: () => print('打开所在位置 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_info.svg',
                    onTap: () => print('查看信息 $index'),
                  ),
                  GpIconButton(
                    icon: 'assets/icons/item_more.svg',
                    onTap: () => print('更多操作 $index'),
                  ),
                ],
                onTap: () {
                  print('失败任务项 $index 被点击');
                },
              ),
        );
      default:
        return Container();
    }
  }
}
