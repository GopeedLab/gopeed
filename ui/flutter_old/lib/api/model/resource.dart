import 'package:gopeed/api/model/request.dart';
import 'package:json_annotation/json_annotation.dart';

part 'resource.g.dart';

@JsonSerializable(explicitToJson: true)
class Resource {
  String name;
  int size;
  bool range;
  List<FileInfo> files;
  String hash;

  Resource(
      {this.name = "",
      this.size = 0,
      this.range = false,
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
  Request? req;

  FileInfo({
    this.path = "",
    required this.name,
    this.size = 0,
    this.req,
  });

  factory FileInfo.fromJson(Map<String, dynamic> json) =>
      _$FileInfoFromJson(json);

  Map<String, dynamic> toJson() => _$FileInfoToJson(this);
}
