import 'package:json_annotation/json_annotation.dart';

import 'options.dart';
import 'request.dart';

part 'create_task_batch.g.dart';

@JsonSerializable(explicitToJson: true)
class CreateTaskBatch {
  List<CreateTaskBatchItem>? reqs;
  Options? opt;

  CreateTaskBatch({
    this.reqs,
    this.opt,
  });

  factory CreateTaskBatch.fromJson(
    Map<String, dynamic> json,
  ) =>
      _$CreateTaskBatchFromJson(json);

  Map<String, dynamic> toJson() => _$CreateTaskBatchToJson(this);
}

@JsonSerializable()
class CreateTaskBatchItem {
  Request? req;
  Options? opts;

  CreateTaskBatchItem({
    this.req,
    this.opts,
  });

  factory CreateTaskBatchItem.fromJson(Map<String, dynamic> json) =>
      _$CreateTaskBatchItemFromJson(json);
  Map<String, dynamic> toJson() => _$CreateTaskBatchItemToJson(this);
}
