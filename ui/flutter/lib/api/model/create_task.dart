import 'package:gopeed/api/model/resolve_result.dart';
import 'package:gopeed/api/model/resolved_request.dart';
import 'package:json_annotation/json_annotation.dart';

import 'options.dart';

part 'create_task.g.dart';

@JsonSerializable(explicitToJson: true)
class CreateTask {
  String? rid;
  ResolvedRequest? req;
  Options? opts;

  CreateTask({
    this.rid,
    this.req,
    this.opts,
  });

  factory CreateTask.fromJson(
    Map<String, dynamic> json,
  ) =>
      _$CreateTaskFromJson(json);

  Map<String, dynamic> toJson() => _$CreateTaskToJson(this);
}
