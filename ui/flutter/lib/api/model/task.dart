import 'package:json_annotation/json_annotation.dart';

import 'meta.dart';

part 'task.g.dart';

enum Status { ready, running, pause, wait, error, done }

enum Protocol { http, bt, ed2k }

// ExtractStatus enum matching Go backend
enum ExtractStatus {
  @JsonValue('')
  none,
  @JsonValue('queued')
  queued,
  @JsonValue('extracting')
  extracting,
  @JsonValue('done')
  done,
  @JsonValue('error')
  error,
  @JsonValue('waitingParts')
  waitingParts,
}

@JsonSerializable(explicitToJson: true)
class Task {
  String id;
  String name;
  Protocol? protocol;
  Meta meta;
  Status status;
  bool uploading;
  Progress progress;
  DateTime createdAt;
  DateTime updatedAt;

  Task({
    required this.id,
    required this.name,
    required this.meta,
    required this.status,
    required this.uploading,
    required this.progress,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Task.fromJson(Map<String, dynamic> json) => _$TaskFromJson(json);

  Map<String, dynamic> toJson() => _$TaskToJson(this);
}

@JsonSerializable()
class Progress {
  int used;
  int speed;
  int downloaded;
  int uploadSpeed;
  int uploaded;
  ExtractStatus extractStatus;
  int extractProgress;

  Progress({
    required this.used,
    required this.speed,
    required this.downloaded,
    required this.uploadSpeed,
    required this.uploaded,
    this.extractStatus = ExtractStatus.none,
    this.extractProgress = 0,
  });

  factory Progress.fromJson(Map<String, dynamic> json) => _$ProgressFromJson(json);

  Map<String, dynamic> toJson() => _$ProgressToJson(this);
}

@JsonSerializable()
class FileRuntimeStatus {
  int index;
  int size;
  int downloaded;

  FileRuntimeStatus({required this.index, required this.size, required this.downloaded});

  factory FileRuntimeStatus.fromJson(Map<String, dynamic> json) => _$FileRuntimeStatusFromJson(json);

  Map<String, dynamic> toJson() => _$FileRuntimeStatusToJson(this);
}

@JsonSerializable(explicitToJson: true)
class TaskRuntimeStatus {
  Status status;
  int used;
  int speed;
  int downloaded;
  int total;
  int uploadSpeed;
  int uploaded;
  ExtractStatus extractStatus;
  int extractProgress;
  List<FileRuntimeStatus> files;

  TaskRuntimeStatus({
    required this.status,
    required this.used,
    required this.speed,
    required this.downloaded,
    required this.total,
    required this.uploadSpeed,
    required this.uploaded,
    this.extractStatus = ExtractStatus.none,
    this.extractProgress = 0,
    required this.files,
  });

  factory TaskRuntimeStatus.fromJson(Map<String, dynamic> json) => _$TaskRuntimeStatusFromJson(json);

  Map<String, dynamic> toJson() => _$TaskRuntimeStatusToJson(this);
}
