// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_task_batch.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CreateTaskBatch _$CreateTaskBatchFromJson(Map<String, dynamic> json) =>
    CreateTaskBatch(
      reqs: (json['reqs'] as List<dynamic>?)
              ?.map((e) => Request.fromJson(e as Map<String, dynamic>))
              .toList() ??
          const [],
      opt: json['opt'] == null
          ? null
          : Option.fromJson(json['opt'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$CreateTaskBatchToJson(CreateTaskBatch instance) {
  final val = <String, dynamic>{
    'reqs': instance.reqs,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('opt', instance.opt);
  return val;
}
