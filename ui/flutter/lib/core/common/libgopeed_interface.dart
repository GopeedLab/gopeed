import 'package:gopeed/core/common/start_config.dart';

abstract class LibgopeedInterface {
  Future<int> start(StartConfig cfg);

  Future<void> stop();
}
