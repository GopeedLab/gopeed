// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'task_filter.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

TaskFilter _$TaskFilterFromJson(Map<String, dynamic> json) => TaskFilter(
      ids: (json['ids'] as List<dynamic>?)?.map((e) => e as String).toList(),
      statuses: (json['statuses'] as List<dynamic>?)
          ?.map((e) => $enumDecode(_$StatusEnumMap, e))
          .toList(),
      notStatuses: (json['notStatuses'] as List<dynamic>?)
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

  writeNotNull('ids', instance.ids);
  writeNotNull(
      'statuses', instance.statuses?.map((e) => _$StatusEnumMap[e]!).toList());
  writeNotNull('notStatuses',
      instance.notStatuses?.map((e) => _$StatusEnumMap[e]!).toList());
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
