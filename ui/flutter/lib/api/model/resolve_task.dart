import 'package:json_annotation/json_annotation.dart';

import 'options.dart';
import 'request.dart';

part 'resolve_task.g.dart';

@JsonSerializable()
class ResolveTask {
  Request? req;
  Options? opts;

  ResolveTask({
    this.req,
    this.opts,
  });

  factory ResolveTask.fromJson(Map<String, dynamic> json) =>
      _$ResolveTaskFromJson(json);
  Map<String, dynamic> toJson() => _$ResolveTaskToJson(this);
}
