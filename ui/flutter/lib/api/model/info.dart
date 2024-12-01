import 'package:json_annotation/json_annotation.dart';

part 'info.g.dart';

@JsonSerializable()
class Info {
  String version;
  String runtime;
  String os;
  String arch;
  bool inDocker;

  Info({
    this.version = '',
    this.runtime = '',
    this.os = '',
    this.arch = '',
    this.inDocker = false,
  });

  factory Info.fromJson(Map<String, dynamic> json) => _$InfoFromJson(json);
  Map<String, dynamic> toJson() => _$InfoToJson(this);
}
