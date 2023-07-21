// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'resolve_result.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResolveResult _$ResolveResultFromJson(Map<String, dynamic> json) =>
    ResolveResult(
      id: json['id'] as String? ?? "",
      res: Resource.fromJson(json['res'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$ResolveResultToJson(ResolveResult instance) =>
    <String, dynamic>{
      'id': instance.id,
      'res': instance.res.toJson(),
    };
