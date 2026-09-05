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
        ], (a, b) {
          if (a.status == Status.running && b.status != Status.running) {
            return -1;
          } else if (a.status != Status.running && b.status == Status.running) {
            return 1;
          } else {
            return b.updatedAt.compareTo(a.updatedAt);
          }
        });
}
