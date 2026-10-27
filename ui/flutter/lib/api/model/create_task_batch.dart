import 'package:json_annotation/json_annotation.dart';

import 'options.dart';
import 'request.dart';

part 'create_task_batch.g.dart';

@JsonSerializable(explicitToJson: true)
class CreateTaskBatch {
  List<CreateTaskBatchItem>? reqs;
  Options? opts;

  bool group;
  String groupName;

  CreateTaskBatch({this.reqs, this.opts, this.group = false, this.groupName = ''});

  factory CreateTaskBatch.fromJson(Map<String, dynamic> json) => _$CreateTaskBatchFromJson(json);

  Map<String, dynamic> toJson() => _$CreateTaskBatchToJson(this);
}

@JsonSerializable()
class CreateTaskBatchItem {
  Request? req;
  Options? opts;

  CreateTaskBatchItem({this.req, this.opts});

  factory CreateTaskBatchItem.fromJson(Map<String, dynamic> json) => _$CreateTaskBatchItemFromJson(json);
  Map<String, dynamic> toJson() => _$CreateTaskBatchItemToJson(this);
}

@JsonSerializable(explicitToJson: true)
class CreateTaskBatchResult {
  final List<String> taskIds;
  final String groupId;
  final String name;
  final String path;
  final List<CreateTaskBatchItemResult> items;

  CreateTaskBatchResult({
    required this.taskIds,
    this.groupId = '',
    this.name = '',
    this.path = '',
    this.items = const [],
  });

  factory CreateTaskBatchResult.fromResponse(Object? value) => value is List
      ? CreateTaskBatchResult(taskIds: value.cast<String>())
      : CreateTaskBatchResult.fromJson(value! as Map<String, dynamic>);
  factory CreateTaskBatchResult.fromJson(Map<String, dynamic> json) => _$CreateTaskBatchResultFromJson(json);
  Map<String, dynamic> toJson() => _$CreateTaskBatchResultToJson(this);
}

@JsonSerializable()
class CreateTaskBatchItemResult {
  final int index;
  final String taskId;
  final String error;

  CreateTaskBatchItemResult({required this.index, this.taskId = '', this.error = ''});
  factory CreateTaskBatchItemResult.fromJson(Map<String, dynamic> json) => _$CreateTaskBatchItemResultFromJson(json);
  Map<String, dynamic> toJson() => _$CreateTaskBatchItemResultToJson(this);
}
