import 'dart:math' as math;

typedef DownloadSpeedSample = ({DateTime time, int bytesPerSecond});

class DownloadSpeedHistory {
  const DownloadSpeedHistory([this.samples = const []]);

  final List<DownloadSpeedSample> samples;
  Duration get elapsed => samples.length < 2 ? Duration.zero : samples.last.time.difference(samples.first.time);

  double position(DateTime time) => elapsed.inMicroseconds <= 0
      ? 0
      : (time.difference(samples.first.time).inMicroseconds / elapsed.inMicroseconds).clamp(0.0, 1.0);

  int get peak => samples.fold(0, (peak, sample) => math.max(peak, sample.bytesPerSecond));

  DownloadSpeedHistory record(DateTime time, int bytesPerSecond) {
    // Keep one actual observation per second, including explicit zero speeds.
    final second = time.millisecondsSinceEpoch ~/ 1000;
    return DownloadSpeedHistory(
      List.unmodifiable([
        ...samples.where((sample) => sample.time.millisecondsSinceEpoch ~/ 1000 < second),
        (time: time, bytesPerSecond: math.max(0, bytesPerSecond)),
      ]),
    );
  }
}
