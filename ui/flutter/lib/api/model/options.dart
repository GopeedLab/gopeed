import 'package:json_annotation/json_annotation.dart';

part 'options.g.dart';

@JsonSerializable()
class Options {
  String name;
  String path;
  int connections;
  List<int> selectFiles;

  Options({
    required this.name,
    required this.path,
    required this.connections,
    required this.selectFiles,
  });

  factory Options.fromJson(Map<String, dynamic> json) =>
      _$OptionsFromJson(json);

  Map<String, dynamic> toJson() => _$OptionsToJson(this);
}
