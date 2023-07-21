// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_task_batch.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CreateTaskBatch _$CreateTaskBatchFromJson(Map<String, dynamic> json) =>
    CreateTaskBatch(
      reqs: (json['reqs'] as List<dynamic>?)
          ?.map((e) => Request.fromJson(e as Map<String, dynamic>))
          .toList(),
      opts: json['opts'] == null
          ? null
          : Options.fromJson(json['opts'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$CreateTaskBatchToJson(CreateTaskBatch instance) {
  final val = <String, dynamic>{};

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('reqs', instance.reqs?.map((e) => e.toJson()).toList());
  writeNotNull('opts', instance.opts?.toJson());
  return val;
}
