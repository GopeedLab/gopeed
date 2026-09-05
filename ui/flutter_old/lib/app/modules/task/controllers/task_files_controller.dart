import 'package:get/get.dart';
import '../../../../api/api.dart';
import '../../../../api/model/resource.dart';
import '../../../../api/model/task.dart';

class FileItem {
  final bool isDirectory;
  final String path;
  final String name;
  final int size;

  FileItem(this.isDirectory, this.path, this.name, this.size);

  String get fullPath => "${path == "/" ? path : "$path/"}$name";

  String filePath(String optName) {
    return optName.isEmpty
        ? fullPath
        : "${path == "/" ? path : "$path/"}$optName";
  }
}

class TaskFilesController extends GetxController {
  final Map<String, List<FileItem>> _dirMap = {};

  final task = Rx<Task?>(null);
  final fileList = <FileItem>[].obs;

  @override
  void onInit() async {
    super.onInit();

    final taskId = Get.rootDelegate.parameters['id'];
    final tasks = await getTasks([]);
    task.value = tasks.firstWhere((element) => element.id == taskId);
    parseDirMap(task.value!.meta.res!.files);
    toDir("/");
  }

  void parseDirMap(List<FileInfo> fileList) {
    for (final file in fileList) {
      String dir = file.path;
      if (!dir.startsWith("/")) {
        dir = "/$dir";
      }
      if (!_dirMap.containsKey(dir)) {
        _dirMap[dir] = [];
      }

      _dirMap[dir]!.add(FileItem(false, dir, file.name, file.size));

      void findParent(String dir) {
        final parentDirIndex = dir.lastIndexOf("/");
        String parentDir =
            parentDirIndex == 0 ? "/" : dir.substring(0, parentDirIndex);
        if (!_dirMap.containsKey(parentDir)) {
          _dirMap[parentDir] = [];
        }
        String dirName = dir.substring(dir.lastIndexOf("/") + 1);
        if (!_dirMap[parentDir]!.any((element) => element.name == dirName)) {
          _dirMap[parentDir]!.add(FileItem(true, parentDir, dirName, 0));
        }
      }

      while (dir != "/" && dir != "") {
        findParent(dir);
        dir = dir.substring(0, dir.lastIndexOf("/"));
      }
    }

    // sort children, directories first then files, alphabetically
    _dirMap.forEach((key, value) {
      value.sort((a, b) {
        if (a.isDirectory && !b.isDirectory) {
          return -1;
        } else if (!a.isDirectory && b.isDirectory) {
          return 1;
        } else {
          return a.name.compareTo(b.name);
        }
      });
    });
  }

  void toDir(String dir) {
    fileList.value = _dirMap[dir]?.toList() ?? [];
  }

  int dirItemCount(String dir) {
    return _dirMap[dir]?.length ?? 0;
  }
}
