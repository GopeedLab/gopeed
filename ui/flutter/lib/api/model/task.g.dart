// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'task.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Task _$TaskFromJson(Map<String, dynamic> json) => Task(
      id: json['id'] as String,
      meta: Meta.fromJson(json['meta'] as Map<String, dynamic>),
      status: $enumDecode(_$StatusEnumMap, json['status']),
      progress: Progress.fromJson(json['progress'] as Map<String, dynamic>),
      createdAt: DateTime.parse(json['createdAt'] as String),
      updatedAt: DateTime.parse(json['updatedAt'] as String),
    );

Map<String, dynamic> _$TaskToJson(Task instance) => <String, dynamic>{
      'id': instance.id,
      'meta': instance.meta.toJson(),
      'status': _$StatusEnumMap[instance.status]!,
      'progress': instance.progress.toJson(),
      'createdAt': instance.createdAt.toIso8601String(),
      'updatedAt': instance.updatedAt.toIso8601String(),
    };

const _$StatusEnumMap = {
  Status.ready: 'ready',
  Status.running: 'running',
  Status.pause: 'pause',
  Status.wait: 'wait',
  Status.error: 'error',
  Status.done: 'done',
};

Progress _$ProgressFromJson(Map<String, dynamic> json) => Progress(
      used: (json['used'] as num).toInt(),
      speed: (json['speed'] as num).toInt(),
      downloaded: (json['downloaded'] as num).toInt(),
    );

Map<String, dynamic> _$ProgressToJson(Progress instance) => <String, dynamic>{
      'used': instance.used,
      'speed': instance.speed,
      'downloaded': instance.downloaded,
    };
