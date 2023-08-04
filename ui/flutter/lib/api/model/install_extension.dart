import 'package:json_annotation/json_annotation.dart';

part 'install_extension.g.dart';

@JsonSerializable()
class InstallExtension {
  String url;

  InstallExtension({
    required this.url,
  });

  factory InstallExtension.fromJson(Map<String, dynamic> json) =>
      _$InstallExtensionFromJson(json);
  Map<String, dynamic> toJson() => _$InstallExtensionToJson(this);
}
