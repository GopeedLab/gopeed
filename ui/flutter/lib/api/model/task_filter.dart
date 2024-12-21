import 'package:gopeed/api/model/task.dart';
import 'package:json_annotation/json_annotation.dart';

part 'task_filter.g.dart';

@JsonSerializable()
class TaskFilter {
  List<String>? id;
  List<Status>? status;
  List<Status>? notStatus;

  TaskFilter({this.id, this.status, this.notStatus});

  factory TaskFilter.fromJson(Map<String, dynamic> json) =>
      _$TaskFilterFromJson(json);
  Map<String, dynamic> toJson() => _$TaskFilterToJson(this);
}
