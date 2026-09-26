import 'package:json_annotation/json_annotation.dart';

part 'options.g.dart';

@JsonSerializable(explicitToJson: true)
class Options {
  String name;
  String path;
  bool asDefaultPath;
  List<int> selectFiles;
  Object? extra;

  Options({this.name = '', this.path = '', this.asDefaultPath = false, this.selectFiles = const [], this.extra});

  factory Options.fromJson(Map<String, dynamic> json) => _$OptionsFromJson(json);

  Map<String, dynamic> toJson() => _$OptionsToJson(this);
}

@JsonSerializable()
class ChecksumOption {
  String algorithm;
  String expected;

  ChecksumOption({
    this.algorithm = '',
    this.expected = '',
  });

  factory ChecksumOption.fromJson(Map<String, dynamic> json) => _$ChecksumOptionFromJson(json);

  Map<String, dynamic> toJson() => _$ChecksumOptionToJson(this);
}

@JsonSerializable()
class OptsExtraHttp {
  int connections;
  bool? autoTorrent;
  bool? deleteTorrentAfterDownload;
  bool? autoExtract;
  String archivePassword;
  bool deleteAfterExtract;
  ChecksumOption? checksum;

  OptsExtraHttp({
    this.connections = 0,
    this.autoTorrent,
    this.deleteTorrentAfterDownload,
    this.autoExtract,
    this.archivePassword = '',
    this.deleteAfterExtract = false,
    this.checksum,
  });

  factory OptsExtraHttp.fromJson(Map<String, dynamic> json) => _$OptsExtraHttpFromJson(json);

  Map<String, dynamic> toJson() => _$OptsExtraHttpToJson(this);
}
