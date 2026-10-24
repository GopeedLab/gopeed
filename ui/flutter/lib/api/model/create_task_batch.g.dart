// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_task_batch.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CreateTaskBatch _$CreateTaskBatchFromJson(Map<String, dynamic> json) => CreateTaskBatch(
  reqs: (json['reqs'] as List<dynamic>?)?.map((e) => CreateTaskBatchItem.fromJson(e as Map<String, dynamic>)).toList(),
  opts: json['opts'] == null ? null : Options.fromJson(json['opts'] as Map<String, dynamic>),
  group: json['group'] == null ? null : CreateTaskGroup.fromJson(json['group'] as Map<String, dynamic>),
);

Map<String, dynamic> _$CreateTaskBatchToJson(CreateTaskBatch instance) => <String, dynamic>{
  'reqs': ?instance.reqs?.map((e) => e.toJson()).toList(),
  'opts': ?instance.opts?.toJson(),
  'group': ?instance.group?.toJson(),
};

CreateTaskGroup _$CreateTaskGroupFromJson(Map<String, dynamic> json) =>
    CreateTaskGroup(id: json['id'] as String, name: json['name'] as String, path: json['path'] as String);

Map<String, dynamic> _$CreateTaskGroupToJson(CreateTaskGroup instance) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'path': instance.path,
};

CreateTaskBatchItem _$CreateTaskBatchItemFromJson(Map<String, dynamic> json) => CreateTaskBatchItem(
  req: json['req'] == null ? null : Request.fromJson(json['req'] as Map<String, dynamic>),
  opts: json['opts'] == null ? null : Options.fromJson(json['opts'] as Map<String, dynamic>),
);

Map<String, dynamic> _$CreateTaskBatchItemToJson(CreateTaskBatchItem instance) => <String, dynamic>{
  'req': ?instance.req,
  'opts': ?instance.opts,
};
