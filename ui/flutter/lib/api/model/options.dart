import 'package:json_annotation/json_annotation.dart';

part 'options.g.dart';

@JsonSerializable(explicitToJson: true)
class Options {
  String name;
  String path;
  List<int> selectFiles;
  Object? extra;

  Options({
    this.name = '',
    this.path = '',
    this.selectFiles = const [],
    this.extra,
  });

  factory Options.fromJson(Map<String, dynamic> json) =>
      _$OptionsFromJson(json);

  Map<String, dynamic> toJson() => _$OptionsToJson(this);
}

@JsonSerializable()
class OptsExtraHttp {
  int connections;
  bool autoTorrent;
  bool autoExtract;
  String archivePassword;

  OptsExtraHttp({
    this.connections = 0,
    this.autoTorrent = false,
    this.autoExtract = false,
    this.archivePassword = '',
  });

  factory OptsExtraHttp.fromJson(Map<String, dynamic> json) =>
      _$OptsExtraHttpFromJson(json);

  Map<String, dynamic> toJson() => _$OptsExtraHttpToJson(this);
}
