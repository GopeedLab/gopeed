Future<void> verifyDownloadDirectoryWritable(String directoryPath) =>
    Future<void>.error(UnsupportedError('Directory write checks are unavailable on this platform'));
