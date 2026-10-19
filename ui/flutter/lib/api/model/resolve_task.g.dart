// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'resolve_task.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResolveTask _$ResolveTaskFromJson(Map<String, dynamic> json) => ResolveTask(
  req: json['req'] == null
      ? null
      : Request.fromJson(json['req'] as Map<String, dynamic>),
);

Map<String, dynamic> _$ResolveTaskToJson(ResolveTask instance) =>
    <String, dynamic>{'req': ?instance.req};
