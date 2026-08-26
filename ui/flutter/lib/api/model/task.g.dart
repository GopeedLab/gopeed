// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'task.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Task _$TaskFromJson(Map<String, dynamic> json) => Task(
  id: json['id'] as String,
  name: json['name'] as String,
  meta: Meta.fromJson(json['meta'] as Map<String, dynamic>),
  status: $enumDecode(_$StatusEnumMap, json['status']),
  uploading: json['uploading'] as bool,
  progress: Progress.fromJson(json['progress'] as Map<String, dynamic>),
  createdAt: DateTime.parse(json['createdAt'] as String),
  updatedAt: DateTime.parse(json['updatedAt'] as String),
)..protocol = $enumDecodeNullable(_$ProtocolEnumMap, json['protocol']);

Map<String, dynamic> _$TaskToJson(Task instance) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'protocol': ?_$ProtocolEnumMap[instance.protocol],
  'meta': instance.meta.toJson(),
  'status': _$StatusEnumMap[instance.status]!,
  'uploading': instance.uploading,
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

const _$ProtocolEnumMap = {Protocol.http: 'http', Protocol.bt: 'bt', Protocol.ed2k: 'ed2k', Protocol.gblob: 'gblob'};

Progress _$ProgressFromJson(Map<String, dynamic> json) => Progress(
  used: (json['used'] as num).toInt(),
  speed: (json['speed'] as num).toInt(),
  downloaded: (json['downloaded'] as num).toInt(),
  uploadSpeed: (json['uploadSpeed'] as num).toInt(),
  uploaded: (json['uploaded'] as num).toInt(),
  extractStatus: $enumDecodeNullable(_$ExtractStatusEnumMap, json['extractStatus']) ?? ExtractStatus.none,
  extractProgress: (json['extractProgress'] as num?)?.toInt() ?? 0,
);

Map<String, dynamic> _$ProgressToJson(Progress instance) => <String, dynamic>{
  'used': instance.used,
  'speed': instance.speed,
  'downloaded': instance.downloaded,
  'uploadSpeed': instance.uploadSpeed,
  'uploaded': instance.uploaded,
  'extractStatus': _$ExtractStatusEnumMap[instance.extractStatus]!,
  'extractProgress': instance.extractProgress,
};

const _$ExtractStatusEnumMap = {
  ExtractStatus.none: '',
  ExtractStatus.queued: 'queued',
  ExtractStatus.extracting: 'extracting',
  ExtractStatus.done: 'done',
  ExtractStatus.error: 'error',
  ExtractStatus.waitingParts: 'waitingParts',
};

FileRuntimeStatus _$FileRuntimeStatusFromJson(Map<String, dynamic> json) => FileRuntimeStatus(
  index: (json['index'] as num).toInt(),
  size: (json['size'] as num).toInt(),
  downloaded: (json['downloaded'] as num).toInt(),
);

Map<String, dynamic> _$FileRuntimeStatusToJson(FileRuntimeStatus instance) => <String, dynamic>{
  'index': instance.index,
  'size': instance.size,
  'downloaded': instance.downloaded,
};

TaskRuntimeStatus _$TaskRuntimeStatusFromJson(Map<String, dynamic> json) => TaskRuntimeStatus(
  status: $enumDecode(_$StatusEnumMap, json['status']),
  used: (json['used'] as num).toInt(),
  speed: (json['speed'] as num).toInt(),
  downloaded: (json['downloaded'] as num).toInt(),
  total: (json['total'] as num).toInt(),
  uploadSpeed: (json['uploadSpeed'] as num).toInt(),
  uploaded: (json['uploaded'] as num).toInt(),
  extractStatus: $enumDecodeNullable(_$ExtractStatusEnumMap, json['extractStatus']) ?? ExtractStatus.none,
  extractProgress: (json['extractProgress'] as num?)?.toInt() ?? 0,
  files: (json['files'] as List<dynamic>).map((e) => FileRuntimeStatus.fromJson(e as Map<String, dynamic>)).toList(),
);

Map<String, dynamic> _$TaskRuntimeStatusToJson(TaskRuntimeStatus instance) => <String, dynamic>{
  'status': _$StatusEnumMap[instance.status]!,
  'used': instance.used,
  'speed': instance.speed,
  'downloaded': instance.downloaded,
  'total': instance.total,
  'uploadSpeed': instance.uploadSpeed,
  'uploaded': instance.uploaded,
  'extractStatus': _$ExtractStatusEnumMap[instance.extractStatus]!,
  'extractProgress': instance.extractProgress,
  'files': instance.files.map((e) => e.toJson()).toList(),
};
