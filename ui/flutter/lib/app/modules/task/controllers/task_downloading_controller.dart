import '../../../../api/model/task.dart';
import 'task_list_controller.dart';

class TaskDownloadingController extends TaskListController {
  TaskDownloadingController()
      : super([
          Status.ready,
          Status.running,
          Status.pause,
          Status.wait,
          Status.error
        ]);
}
