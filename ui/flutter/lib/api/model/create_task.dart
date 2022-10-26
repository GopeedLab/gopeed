import 'package:json_annotation/json_annotation.dart';

import 'options.dart';
import 'resource.dart';

part 'create_task.g.dart';

@JsonSerializable(explicitToJson: true)
class CreateTask {
  Resource res;
  Options? opts;

  CreateTask({
    required this.res,
    this.opts,
  });

  factory CreateTask.fromJson(Map<String, dynamic> json) =>
      _$CreateTaskFromJson(json);

  Map<String, dynamic> toJson() => _$CreateTaskToJson(this);
}
