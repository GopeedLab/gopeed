import 'package:flutter/foundation.dart';

class TaskBatchSelectionController extends ChangeNotifier {
  final Set<String> _selectedIds = <String>{};

  int get count => _selectedIds.length;
  bool get isEmpty => _selectedIds.isEmpty;
  bool get isNotEmpty => _selectedIds.isNotEmpty;

  bool contains(String taskId) => _selectedIds.contains(taskId);

  bool areAllSelected(Iterable<String> taskIds) {
    var hasTasks = false;
    for (final taskId in taskIds) {
      hasTasks = true;
      if (!_selectedIds.contains(taskId)) return false;
    }
    return hasTasks;
  }

  List<String> toList() => _selectedIds.toList(growable: false);

  Iterable<String> where(bool Function(String taskId) test) => _selectedIds.where(test);

  void toggle(String taskId) {
    if (!_selectedIds.remove(taskId)) {
      _selectedIds.add(taskId);
    }
    notifyListeners();
  }

  void replaceWith(Iterable<String> taskIds) {
    final next = taskIds.toSet();
    if (setEquals(_selectedIds, next)) return;
    _selectedIds
      ..clear()
      ..addAll(next);
    notifyListeners();
  }

  void removeAll(Iterable<String> taskIds) {
    var changed = false;
    for (final taskId in taskIds) {
      changed = _selectedIds.remove(taskId) || changed;
    }
    if (changed) notifyListeners();
  }

  void clear() {
    if (_selectedIds.isEmpty) return;
    _selectedIds.clear();
    notifyListeners();
  }
}
