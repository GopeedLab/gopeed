import 'package:json_annotation/json_annotation.dart';

import 'options.dart';
import 'request.dart';

part 'create_task.g.dart';

@JsonSerializable(explicitToJson: true)
class CreateTask {
  String? rid;
  Request? req;
  Options? opt;

  CreateTask({
    this.rid,
    this.req,
    this.opt,
  });

  factory CreateTask.fromJson(
    Map<String, dynamic> json,
  ) =>
      _$CreateTaskFromJson(json);

  Map<String, dynamic> toJson() => _$CreateTaskToJson(this);
}
