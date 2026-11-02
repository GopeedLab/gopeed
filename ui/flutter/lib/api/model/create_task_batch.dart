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
  String? rid;
  Request? req;
  Options? opts;

  CreateTaskBatchItem({this.rid, this.req, this.opts});

  factory CreateTaskBatchItem.fromJson(Map<String, dynamic> json) => _$CreateTaskBatchItemFromJson(json);
  Map<String, dynamic> toJson() => _$CreateTaskBatchItemToJson(this);
}

@JsonSerializable()
class CreateTaskBatchResult {
  final List<String> taskIds;
  final String? parentId;

  CreateTaskBatchResult({required this.taskIds, this.parentId});

  factory CreateTaskBatchResult.fromJson(Map<String, dynamic> json) => _$CreateTaskBatchResultFromJson(json);
  Map<String, dynamic> toJson() => _$CreateTaskBatchResultToJson(this);
}
