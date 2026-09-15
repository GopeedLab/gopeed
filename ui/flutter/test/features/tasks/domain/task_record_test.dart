import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/icons/gopeed_icons.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';

void main() {
  test('file names preserve hash and question mark characters for type detection', () {
    expect(
      taskFileTypeIcon('Slow English Podcast #2 _ Eating, Talking, and Learning Natural English.mp4'),
      GopeedIcons.fileVideo,
    );
    expect(taskFileTypeIcon('What is this? #2.MP4'), GopeedIcons.fileVideo);
    expect(taskFileTypeIcon('Episode #2.mp3'), GopeedIcons.fileAudio);
  });

  test('URL fallback ignores query and fragment when detecting file type', () {
    expect(taskFileTypeIcon('https://example.com/video.mp4?token=abc#part'), GopeedIcons.fileVideo);
    expect(taskFileTypeIcon('https://example.com/download?name=video.mp4'), GopeedIcons.file);
    expect(taskFileTypeIcon('Playlist #2.mp4', isFolder: true), GopeedIcons.folder);
  });
}
