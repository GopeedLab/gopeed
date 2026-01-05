import 'package:json_annotation/json_annotation.dart';

import 'meta.dart';

part 'task.g.dart';

enum Status { ready, running, pause, wait, error, done }

enum Protocol { http, bt }

// ExtractStatus enum matching Go backend
enum ExtractStatus {
  @JsonValue('')
  none,
  @JsonValue('extracting')
  extracting,
  @JsonValue('done')
  done,
  @JsonValue('error')
  error,
  @JsonValue('waitingParts')
  waitingParts
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

  factory Progress.fromJson(Map<String, dynamic> json) =>
      _$ProgressFromJson(json);

  Map<String, dynamic> toJson() => _$ProgressToJson(this);
}
