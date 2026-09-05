// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_task.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CreateTask _$CreateTaskFromJson(Map<String, dynamic> json) => CreateTask(
  rid: json['rid'] as String?,
  req: json['req'] == null ? null : Request.fromJson(json['req'] as Map<String, dynamic>),
  opts: json['opts'] == null ? null : Options.fromJson(json['opts'] as Map<String, dynamic>),
);

Map<String, dynamic> _$CreateTaskToJson(CreateTask instance) => <String, dynamic>{
  'rid': ?instance.rid,
  'req': ?instance.req?.toJson(),
  'opts': ?instance.opts?.toJson(),
};
