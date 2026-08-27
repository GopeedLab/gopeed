abstract final class DurationFormatter {
  static String format(Duration duration) {
    final microseconds = duration.inMicroseconds;
    final totalSeconds = microseconds <= 0
        ? 0
        : (microseconds + Duration.microsecondsPerSecond - 1) ~/ Duration.microsecondsPerSecond;
    final hours = totalSeconds ~/ Duration.secondsPerHour;
    final minutes = (totalSeconds ~/ Duration.secondsPerMinute) % Duration.minutesPerHour;
    final seconds = totalSeconds % Duration.secondsPerMinute;

    if (hours > 0) {
      return '$hours:${_twoDigits(minutes)}:${_twoDigits(seconds)}';
    }
    return '${totalSeconds ~/ Duration.secondsPerMinute}:${_twoDigits(seconds)}';
  }

  static String _twoDigits(int value) => value.toString().padLeft(2, '0');
}
