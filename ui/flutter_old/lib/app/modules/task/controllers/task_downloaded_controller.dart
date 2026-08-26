import 'package:gopeed/app/modules/task/controllers/task_list_controller.dart';

import '../../../../api/model/task.dart';

class TaskDownloadedController extends TaskListController {
  TaskDownloadedController()
      : super([Status.done], (a, b) => b.updatedAt.compareTo(a.updatedAt));
}
