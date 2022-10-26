import 'request.dart';
import 'package:json_annotation/json_annotation.dart';

part 'resource.g.dart';

@JsonSerializable(explicitToJson: true)
class Resource {
  Request req;
  String name;
  int size;
  bool range;
  List<FileInfo> files;
  Map<String, dynamic>? extra;

  Resource({
    required this.req,
    required this.name,
    required this.size,
    required this.range,
    required this.files,
    this.extra,
  });

  factory Resource.fromJson(Map<String, dynamic> json) =>
      _$ResourceFromJson(json);
  Map<String, dynamic> toJson() => _$ResourceToJson(this);
}

@JsonSerializable()
class FileInfo {
  String name;
  String path;
  int size;

  FileInfo({
    required this.name,
    required this.path,
    required this.size,
  });

  factory FileInfo.fromJson(Map<String, dynamic> json) =>
      _$FileInfoFromJson(json);
  Map<String, dynamic> toJson() => _$FileInfoToJson(this);
}
