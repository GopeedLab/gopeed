class AppStartupOptions {
  const AppStartupOptions({required this.hidden});

  factory AppStartupOptions.fromArgs(List<String> args) {
    return AppStartupOptions(
      hidden: args.contains('--hidden') || args.map(Uri.tryParse).whereType<Uri>().any(isSilentGopeedWakeUri),
    );
  }

  final bool hidden;

  AppStartupOptions withInitialUri(Uri? uri) {
    return AppStartupOptions(hidden: hidden || (uri != null && isSilentGopeedWakeUri(uri)));
  }
}

bool isSilentGopeedWakeUri(Uri uri) {
  if (uri.scheme.toLowerCase() != 'gopeed' || uri.queryParameters['hidden'] != 'true') {
    return false;
  }
  return uri.path.isEmpty || uri.path == '/';
}
