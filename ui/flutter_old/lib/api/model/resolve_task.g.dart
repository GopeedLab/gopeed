// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'resolve_task.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResolveTask _$ResolveTaskFromJson(Map<String, dynamic> json) => ResolveTask(
      req: json['req'] == null
          ? null
          : Request.fromJson(json['req'] as Map<String, dynamic>),
      opts: json['opts'] == null
          ? null
          : Options.fromJson(json['opts'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$ResolveTaskToJson(ResolveTask instance) {
  final val = <String, dynamic>{};

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('req', instance.req);
  writeNotNull('opts', instance.opts);
  return val;
}
