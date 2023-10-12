// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_task.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CreateTask _$CreateTaskFromJson(Map<String, dynamic> json) => CreateTask(
      rid: json['rid'] as String?,
      req: json['req'] == null
          ? null
          : Request.fromJson(json['req'] as Map<String, dynamic>),
      opt: json['opt'] == null
          ? null
          : Options.fromJson(json['opt'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$CreateTaskToJson(CreateTask instance) {
  final val = <String, dynamic>{};

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('rid', instance.rid);
  writeNotNull('req', instance.req?.toJson());
  writeNotNull('opt', instance.opt?.toJson());
  return val;
}
