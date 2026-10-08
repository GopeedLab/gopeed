import 'package:json_annotation/json_annotation.dart';

import 'request.dart';

part 'resolve_task.g.dart';

@JsonSerializable()
class ResolveTask {
  Request? req;

  ResolveTask({this.req});

  factory ResolveTask.fromJson(Map<String, dynamic> json) => _$ResolveTaskFromJson(json);
  Map<String, dynamic> toJson() => _$ResolveTaskToJson(this);
}
