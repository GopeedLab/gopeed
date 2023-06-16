import 'package:json_annotation/json_annotation.dart';

part 'options.g.dart';

@JsonSerializable(explicitToJson: true, genericArgumentFactories: true)
class Options {
  String name;
  String path;
  List<int> selectFiles;
  Map<String, dynamic>? extra;

  Options({
    this.name = "",
    this.path = "",
    this.selectFiles = const [],
  });

  factory Options.fromJson(Map<String, dynamic> json) =>
      _$OptionsFromJson(json);

  Map<String, dynamic> toJson() => _$OptionsToJson(this);
}

@JsonSerializable()
class OptsExtraHttp {
  int connections = 0;

  OptsExtraHttp();

  factory OptsExtraHttp.fromJson(Map<String, dynamic> json) =>
      _$OptsExtraHttpFromJson(json);
  Map<String, dynamic> toJson() => _$OptsExtraHttpToJson(this);
}
