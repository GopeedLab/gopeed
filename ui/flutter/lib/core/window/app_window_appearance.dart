class AppWindowAppearance {
  const AppWindowAppearance({required this.themeMode, required this.themeColor, required this.locale});

  const AppWindowAppearance.defaults() : themeMode = 'system', themeColor = 'green', locale = '';

  factory AppWindowAppearance.fromJson(Map<String, dynamic> json) {
    return AppWindowAppearance(
      themeMode: (json['themeMode'] ?? 'system').toString(),
      themeColor: (json['themeColor'] ?? 'green').toString(),
      locale: (json['locale'] ?? '').toString(),
    );
  }

  final String themeMode;
  final String themeColor;
  final String locale;

  Map<String, dynamic> toJson() => {'themeMode': themeMode, 'themeColor': themeColor, 'locale': locale};

  @override
  bool operator ==(Object other) {
    return other is AppWindowAppearance &&
        other.themeMode == themeMode &&
        other.themeColor == themeColor &&
        other.locale == locale;
  }

  @override
  int get hashCode => Object.hash(themeMode, themeColor, locale);
}
