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
      required this.rootDir,
      required this.files,
      required this.hash});

  factory Resource.fromJson(Map<String, dynamic> json) =>
      _$ResourceFromJson(json);

  Map<String, dynamic> toJson() => _$ResourceToJson(this);
}

@JsonSerializable()
class FileInfo {
  // String rootDir;
  String path;
  String name;
  int size;

  FileInfo({
    // required this.rootDir,
    required this.path,
    required this.name,
    required this.size,
  });

  factory FileInfo.fromJson(Map<String, dynamic> json) =>
      _$FileInfoFromJson(json);

  Map<String, dynamic> toJson() => _$FileInfoToJson(this);
}
