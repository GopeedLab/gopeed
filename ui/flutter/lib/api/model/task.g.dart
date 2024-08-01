// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'task.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Task _$TaskFromJson(Map<String, dynamic> json) => Task(
      id: json['id'] as String,
      meta: Meta.fromJson(json['meta'] as Map<String, dynamic>),
      status: $enumDecode(_$StatusEnumMap, json['status']),
      uploading: json['uploading'] as bool,
      progress: Progress.fromJson(json['progress'] as Map<String, dynamic>),
      createdAt: DateTime.parse(json['createdAt'] as String),
      updatedAt: DateTime.parse(json['updatedAt'] as String),
    )..protocol = $enumDecodeNullable(_$ProtocolEnumMap, json['protocol']);

Map<String, dynamic> _$TaskToJson(Task instance) {
  final val = <String, dynamic>{
    'id': instance.id,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('protocol', _$ProtocolEnumMap[instance.protocol]);
  val['meta'] = instance.meta.toJson();
  val['status'] = _$StatusEnumMap[instance.status]!;
  val['uploading'] = instance.uploading;
  val['progress'] = instance.progress.toJson();
  val['createdAt'] = instance.createdAt.toIso8601String();
  val['updatedAt'] = instance.updatedAt.toIso8601String();
  return val;
}

const _$StatusEnumMap = {
  Status.ready: 'ready',
  Status.running: 'running',
  Status.pause: 'pause',
  Status.wait: 'wait',
  Status.error: 'error',
  Status.done: 'done',
};

const _$ProtocolEnumMap = {
  Protocol.http: 'http',
  Protocol.bt: 'bt',
};

Progress _$ProgressFromJson(Map<String, dynamic> json) => Progress(
      used: json['used'] as int,
      speed: json['speed'] as int,
      downloaded: json['downloaded'] as int,
      uploadSpeed: json['uploadSpeed'] as int,
      uploaded: json['uploaded'] as int,
    );

Map<String, dynamic> _$ProgressToJson(Progress instance) => <String, dynamic>{
      'used': instance.used,
      'speed': instance.speed,
      'downloaded': instance.downloaded,
      'uploadSpeed': instance.uploadSpeed,
      'uploaded': instance.uploaded,
    };
