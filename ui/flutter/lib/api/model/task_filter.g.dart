// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'task_filter.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

TaskFilter _$TaskFilterFromJson(Map<String, dynamic> json) => TaskFilter(
      id: (json['ids'] as List<dynamic>?)?.map((e) => e as String).toList(),
      status: (json['statuses'] as List<dynamic>?)
          ?.map((e) => $enumDecode(_$StatusEnumMap, e))
          .toList(),
      notStatus: (json['notStatuses'] as List<dynamic>?)
          ?.map((e) => $enumDecode(_$StatusEnumMap, e))
          .toList(),
    );

Map<String, dynamic> _$TaskFilterToJson(TaskFilter instance) {
  final val = <String, dynamic>{};

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('ids', instance.id);
  writeNotNull(
      'statuses', instance.status?.map((e) => _$StatusEnumMap[e]!).toList());
  writeNotNull('notStatuses',
      instance.notStatus?.map((e) => _$StatusEnumMap[e]!).toList());
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
