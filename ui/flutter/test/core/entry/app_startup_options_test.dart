import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/entry/app_startup_options.dart';

void main() {
  test('starts hidden for the launch-at-startup flag', () {
    expect(AppStartupOptions.fromArgs(const ['--hidden']).hidden, isTrue);
  });

  test('starts hidden for the browser host silent wake URI', () {
    expect(AppStartupOptions.fromArgs(const ['gopeed:?hidden=true']).hidden, isTrue);
    expect(AppStartupOptions.fromArgs(const ['gopeed:///?hidden=true']).hidden, isTrue);
  });

  test('does not hide task and extension deep links', () {
    expect(AppStartupOptions.fromArgs(const ['gopeed:///create?hidden=true']).hidden, isFalse);
    expect(AppStartupOptions.fromArgs(const ['gopeed:///extension']).hidden, isFalse);
  });

  test('merges a platform-provided initial URI before the window is shown', () {
    final options = AppStartupOptions.fromArgs(const []);

    expect(options.withInitialUri(Uri.parse('gopeed:?hidden=true')).hidden, isTrue);
    expect(options.withInitialUri(Uri.parse('gopeed:///create')).hidden, isFalse);
  });

  test('recognizes only the root hidden wake URI', () {
    expect(isSilentGopeedWakeUri(Uri.parse('gopeed:?hidden=true')), isTrue);
    expect(isSilentGopeedWakeUri(Uri.parse('gopeed:///?hidden=true')), isTrue);
    expect(isSilentGopeedWakeUri(Uri.parse('gopeed:///create?hidden=true')), isFalse);
    expect(isSilentGopeedWakeUri(Uri.parse('https://example.com/?hidden=true')), isFalse);
  });
}
