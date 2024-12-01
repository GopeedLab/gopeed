import 'package:gopeed/api/model/task.dart';
import 'package:json_annotation/json_annotation.dart';

part 'task_filter.g.dart';

@JsonSerializable()
class TaskFilter {
  List<String>? ids;
  List<Status>? statuses;
  List<Status>? notStatuses;

  TaskFilter({this.ids, this.statuses, this.notStatuses});

  factory TaskFilter.fromJson(Map<String, dynamic> json) =>
      _$TaskFilterFromJson(json);
  Map<String, dynamic> toJson() => _$TaskFilterToJson(this);
}
