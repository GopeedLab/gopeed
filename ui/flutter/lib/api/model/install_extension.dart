import 'package:json_annotation/json_annotation.dart';

part 'install_extension.g.dart';

@JsonSerializable()
class InstallExtension {
  bool devMode;
  String url;

  InstallExtension({
    this.devMode = false,
    required this.url,
  });

  factory InstallExtension.fromJson(Map<String, dynamic> json) =>
      _$InstallExtensionFromJson(json);
  Map<String, dynamic> toJson() => _$InstallExtensionToJson(this);
}
