// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_task_batch.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CreateTaskBatch _$CreateTaskBatchFromJson(Map<String, dynamic> json) => CreateTaskBatch(
  reqs: (json['reqs'] as List<dynamic>?)?.map((e) => CreateTaskBatchItem.fromJson(e as Map<String, dynamic>)).toList(),
  opts: json['opts'] == null ? null : Options.fromJson(json['opts'] as Map<String, dynamic>),
  group: json['group'] as bool? ?? false,
  groupName: json['groupName'] as String? ?? '',
);

Map<String, dynamic> _$CreateTaskBatchToJson(CreateTaskBatch instance) => <String, dynamic>{
  'reqs': ?instance.reqs?.map((e) => e.toJson()).toList(),
  'opts': ?instance.opts?.toJson(),
  'group': instance.group,
  'groupName': instance.groupName,
};

CreateTaskBatchItem _$CreateTaskBatchItemFromJson(Map<String, dynamic> json) => CreateTaskBatchItem(
  req: json['req'] == null ? null : Request.fromJson(json['req'] as Map<String, dynamic>),
  opts: json['opts'] == null ? null : Options.fromJson(json['opts'] as Map<String, dynamic>),
);

Map<String, dynamic> _$CreateTaskBatchItemToJson(CreateTaskBatchItem instance) => <String, dynamic>{
  'req': ?instance.req,
  'opts': ?instance.opts,
};

CreateTaskBatchResult _$CreateTaskBatchResultFromJson(Map<String, dynamic> json) => CreateTaskBatchResult(
  taskIds: (json['taskIds'] as List<dynamic>).map((e) => e as String).toList(),
  groupId: json['groupId'] as String? ?? '',
  name: json['name'] as String? ?? '',
  path: json['path'] as String? ?? '',
  items:
      (json['items'] as List<dynamic>?)
          ?.map((e) => CreateTaskBatchItemResult.fromJson(e as Map<String, dynamic>))
          .toList() ??
      const [],
);

Map<String, dynamic> _$CreateTaskBatchResultToJson(CreateTaskBatchResult instance) => <String, dynamic>{
  'taskIds': instance.taskIds,
  'groupId': instance.groupId,
  'name': instance.name,
  'path': instance.path,
  'items': instance.items.map((e) => e.toJson()).toList(),
};

CreateTaskBatchItemResult _$CreateTaskBatchItemResultFromJson(Map<String, dynamic> json) => CreateTaskBatchItemResult(
  index: (json['index'] as num).toInt(),
  taskId: json['taskId'] as String? ?? '',
  error: json['error'] as String? ?? '',
);

Map<String, dynamic> _$CreateTaskBatchItemResultToJson(CreateTaskBatchItemResult instance) => <String, dynamic>{
  'index': instance.index,
  'taskId': instance.taskId,
  'error': instance.error,
};
