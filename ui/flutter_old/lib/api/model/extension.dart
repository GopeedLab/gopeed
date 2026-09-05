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
  Repository? repository;
  List<Setting>? settings;
  bool disabled;
  bool devMode;
  String devPath;

  Extension({
    required this.identity,
    required this.name,
    required this.author,
    required this.title,
    required this.description,
    required this.icon,
    required this.version,
    required this.homepage,
    required this.repository,
    required this.disabled,
    required this.devMode,
    required this.devPath,
  });

  factory Extension.fromJson(Map<String, dynamic> json) =>
      _$ExtensionFromJson(json);
  Map<String, dynamic> toJson() => _$ExtensionToJson(this);
}

@JsonSerializable()
class Repository {
  String url;
  String directory;

  Repository({
    required this.url,
    required this.directory,
  });

  factory Repository.fromJson(Map<String, dynamic> json) =>
      _$RepositoryFromJson(json);
  Map<String, dynamic> toJson() => _$RepositoryToJson(this);
}

@JsonSerializable()
class Setting {
  String name;
  String title;
  String description;
  bool required;
  SettingType type;
  Object? value;
  List<Option>? options;

  Setting({
    required this.name,
    required this.title,
    required this.description,
    required this.required,
    required this.type,
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
