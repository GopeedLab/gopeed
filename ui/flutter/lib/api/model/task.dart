import 'package:json_annotation/json_annotation.dart';

import 'meta.dart';
import 'options.dart';
import 'resource.dart';

part 'task.g.dart';

enum Status { ready, running, pause, error, done }

@JsonSerializable(explicitToJson: true)
class Task {
  String id;
  Meta meta;
  Status status;
  Progress progress;
  int size;
  DateTime createdAt;

  Task({
    required this.id,
    required this.meta,
    required this.status,
    required this.progress,
    required this.size,
    required this.createdAt,
  });

  factory Task.fromJson(Map<String, dynamic> json) => _$TaskFromJson(json);

  Map<String, dynamic> toJson() => _$TaskToJson(this);
}

@JsonSerializable()
class Progress {
  int used;
  int speed;
  int downloaded;

  Progress({
    required this.used,
    required this.speed,
    required this.downloaded,
  });

  factory Progress.fromJson(Map<String, dynamic> json) =>
      _$ProgressFromJson(json);

  Map<String, dynamic> toJson() => _$ProgressToJson(this);
}
