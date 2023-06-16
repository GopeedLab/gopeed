import 'package:gopeed/api/model/resolved_request.dart';
import 'package:json_annotation/json_annotation.dart';

part 'resource.g.dart';

@JsonSerializable(explicitToJson: true)
class Resource {
  String name;
  int size;
  bool range;
  String rootDir;
  List<FileInfo> files;
  String hash;

  Resource(
      {required this.name,
      required this.size,
      required this.range,
      this.rootDir = "",
      required this.files,
      this.hash = ""});

  factory Resource.fromJson(Map<String, dynamic> json) =>
      _$ResourceFromJson(json);

  Map<String, dynamic> toJson() => _$ResourceToJson(this);
}

@JsonSerializable(explicitToJson: true)
class FileInfo {
  String path;
  String name;
  int size;
  ResolvedRequest? req;

  FileInfo({
    required this.path,
    required this.name,
    required this.size,
    this.req,
  });

  factory FileInfo.fromJson(Map<String, dynamic> json) =>
      _$FileInfoFromJson(json);

  Map<String, dynamic> toJson() => _$FileInfoToJson(this);
}
