import 'package:json_annotation/json_annotation.dart';

part 'switch_extension.g.dart';

@JsonSerializable()
class SwitchExtension {
  bool status;

  SwitchExtension({
    required this.status,
  });

  factory SwitchExtension.fromJson(Map<String, dynamic> json) =>
      _$SwitchExtensionFromJson(json);
  Map<String, dynamic> toJson() => _$SwitchExtensionToJson(this);
}
