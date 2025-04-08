// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_task_batch.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CreateTaskBatch _$CreateTaskBatchFromJson(Map<String, dynamic> json) =>
    CreateTaskBatch(
      reqs: (json['reqs'] as List<dynamic>?)
          ?.map((e) => CreateTaskBatchItem.fromJson(e as Map<String, dynamic>))
          .toList(),
      opt: json['opt'] == null
          ? null
          : Options.fromJson(json['opt'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$CreateTaskBatchToJson(CreateTaskBatch instance) {
  final val = <String, dynamic>{};

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('reqs', instance.reqs?.map((e) => e.toJson()).toList());
  writeNotNull('opt', instance.opt?.toJson());
  return val;
}

CreateTaskBatchItem _$CreateTaskBatchItemFromJson(Map<String, dynamic> json) =>
    CreateTaskBatchItem(
      req: json['req'] == null
          ? null
          : Request.fromJson(json['req'] as Map<String, dynamic>),
      opts: json['opts'] == null
          ? null
          : Options.fromJson(json['opts'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$CreateTaskBatchItemToJson(CreateTaskBatchItem instance) {
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
