enum TaskEventType {
  done(1, 'task.done'),
  error(2, 'task.error');

  const TaskEventType(this.mask, this.wireName);

  final int mask;
  final String wireName;

  static TaskEventType fromWireName(String value) {
    return values.firstWhere(
      (type) => type.wireName == value,
      orElse: () => throw FormatException('Unknown task event type: $value'),
    );
  }
}

extension TaskEventTypeSet on Set<TaskEventType> {
  int get mask => fold(0, (value, type) => value | type.mask);
}

class TaskEvent {
  const TaskEvent({required this.type, required this.taskId, required this.name, this.error});

  final TaskEventType type;
  final String taskId;
  final String name;
  final String? error;

  factory TaskEvent.fromJson(Map<String, dynamic> json) {
    return TaskEvent(
      type: TaskEventType.fromWireName(json['type'] as String),
      taskId: json['taskId'] as String,
      name: json['name'] as String? ?? '',
      error: json['error'] as String?,
    );
  }
}
