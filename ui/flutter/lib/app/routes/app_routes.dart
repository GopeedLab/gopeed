part of 'app_pages.dart';
// DO NOT EDIT. This is code generated via package:get_cli/get_cli.dart

abstract class Routes {
  Routes._();

  static const ROOT = _Paths.ROOT;
  static const HOME = _Paths.HOME;
  static const CREATE = _Paths.CREATE;
  static const TASK = _Paths.HOME + _Paths.TASK;
  static const TASK_FILES = TASK + _Paths.TASK_FILES;
  static const EXTENSION = _Paths.HOME + _Paths.EXTENSION;
  static const SETTING = _Paths.HOME + _Paths.SETTING;
  static const REDIRECT = _Paths.REDIRECT;
}

abstract class _Paths {
  _Paths._();

  static const ROOT = '/';
  static const HOME = '/home';
  static const CREATE = '/create';
  static const TASK = '/task';
  static const TASK_FILES = '/files';
  static const EXTENSION = '/extension';
  static const SETTING = '/setting';
  static const REDIRECT = '/redirect';
}
