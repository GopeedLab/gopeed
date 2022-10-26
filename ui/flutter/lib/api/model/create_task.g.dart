// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_task.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CreateTask _$CreateTaskFromJson(Map<String, dynamic> json) => CreateTask(
      res: Resource.fromJson(json['res'] as Map<String, dynamic>),
      opts: json['opts'] == null
          ? null
          : Options.fromJson(json['opts'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$CreateTaskToJson(CreateTask instance) {
  final val = <String, dynamic>{
    'res': instance.res.toJson(),
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('opts', instance.opts?.toJson());
  return val;
}
