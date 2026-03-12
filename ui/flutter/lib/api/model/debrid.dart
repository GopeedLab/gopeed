class DebridConfig {
  String active;       // "torbox" | "realdebrid" | ""
  String torBoxKey;
  String realDebridKey;

  DebridConfig({
    this.active = '',
    this.torBoxKey = '',
    this.realDebridKey = '',
  });

  factory DebridConfig.fromJson(Map<String, dynamic> json) => DebridConfig(
        active: json['active'] as String? ?? '',
        torBoxKey: json['torBoxKey'] as String? ?? '',
        realDebridKey: json['realDebridKey'] as String? ?? '',
      );

  Map<String, dynamic> toJson() => {
        'active': active,
        'torBoxKey': torBoxKey,
        'realDebridKey': realDebridKey,
      };
}
