// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'task.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Task _$TaskFromJson(Map<String, dynamic> json) => Task(
      id: json['id'] as String,
      res: Resource.fromJson(json['res'] as Map<String, dynamic>),
      opts: Options.fromJson(json['opts'] as Map<String, dynamic>),
      status: $enumDecode(_$StatusEnumMap, json['status']),
      progress: Progress.fromJson(json['progress'] as Map<String, dynamic>),
      size: json['size'] as int,
      createdAt: DateTime.parse(json['createdAt'] as String),
    );

Map<String, dynamic> _$TaskToJson(Task instance) => <String, dynamic>{
      'id': instance.id,
      'res': instance.res,
      'opts': instance.opts,
      'status': _$StatusEnumMap[instance.status]!,
      'progress': instance.progress,
      'size': instance.size,
      'createdAt': instance.createdAt.toIso8601String(),
    };

const _$StatusEnumMap = {
  Status.ready: 'ready',
  Status.running: 'running',
  Status.pause: 'pause',
  Status.error: 'error',
  Status.done: 'done',
};

Progress _$ProgressFromJson(Map<String, dynamic> json) => Progress(
      used: json['used'] as int,
      speed: json['speed'] as int,
      downloaded: json['downloaded'] as int,
    );

Map<String, dynamic> _$ProgressToJson(Progress instance) => <String, dynamic>{
      'used': instance.used,
      'speed': instance.speed,
      'downloaded': instance.downloaded,
    };
