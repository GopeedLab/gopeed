import 'package:json_annotation/json_annotation.dart';

import 'request.dart';
import 'options.dart';

part 'create_task_batch.g.dart';

@JsonSerializable(explicitToJson: true)
class CreateTaskBatch {
  List<Request>? reqs;
  Options? opts;

  CreateTaskBatch({
    this.reqs,
    this.opts,
  });

  factory CreateTaskBatch.fromJson(Map<String, dynamic> json) =>
      _$CreateTaskBatchFromJson(json);

  Map<String, dynamic> toJson() => _$CreateTaskBatchToJson(this);
}
