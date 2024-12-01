import 'package:gopeed/api/model/extension.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:json_annotation/json_annotation.dart';

part 'create_task_batch.g.dart';

@JsonSerializable()
class CreateTaskBatch {
  List<Request> reqs;
  Option? opt;

  CreateTaskBatch({
    this.reqs = const [],
    this.opt,
  });

  factory CreateTaskBatch.fromJson(Map<String, dynamic> json) =>
      _$CreateTaskBatchFromJson(json);
  Map<String, dynamic> toJson() => _$CreateTaskBatchToJson(this);
}
