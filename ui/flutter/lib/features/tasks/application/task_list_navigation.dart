import 'package:flutter_riverpod/flutter_riverpod.dart';

final taskListNavigationProvider = NotifierProvider<TaskListNavigation, int>(TaskListNavigation.new);

// A request also resets an already-visible list's filter, search and selection.
class TaskListNavigation extends Notifier<int> {
  @override
  int build() => 0;

  void showDownloading() => state++;
}
