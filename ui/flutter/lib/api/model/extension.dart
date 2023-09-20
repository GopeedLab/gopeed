import 'package:json_annotation/json_annotation.dart';

part 'extension.g.dart';

@JsonSerializable(explicitToJson: true)
class Extension {
  String identity;
  String name;
  String author;
  String title;
  String description;
  String icon;
  String version;
  String homepage;
  String installUrl;
  String repository;
  List<Setting>? settings;
  bool disabled;

  Extension({
    required this.identity,
    required this.name,
    required this.author,
    required this.title,
    required this.description,
    required this.icon,
    required this.version,
    required this.homepage,
    required this.installUrl,
    required this.repository,
    required this.disabled,
  });

  factory Extension.fromJson(Map<String, dynamic> json) =>
      _$ExtensionFromJson(json);
  Map<String, dynamic> toJson() => _$ExtensionToJson(this);
}

@JsonSerializable()
class Setting {
  String name;
  String title;
  String description;
  bool required;
  SettingType type;
  Object? value;
  bool multiple;
  List<Option>? options;

  Setting({
    required this.name,
    required this.title,
    required this.description,
    required this.required,
    required this.type,
    required this.multiple,
  });

  factory Setting.fromJson(Map<String, dynamic> json) =>
      _$SettingFromJson(json);
  Map<String, dynamic> toJson() => _$SettingToJson(this);
}

@JsonSerializable()
class Option {
  String label;
  Object value;

  Option({
    required this.label,
    required this.value,
  });

  factory Option.fromJson(Map<String, dynamic> json) => _$OptionFromJson(json);
  Map<String, dynamic> toJson() => _$OptionToJson(this);
}

enum SettingType {
  string,
  number,
  boolean,
}
