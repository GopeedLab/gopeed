findIcon(String filename) {
  String res = 'file';
  String ext = filename.substring(filename.lastIndexOf('.') + 1);
  for (var iconMap in iconMaps) {
    if (iconMap['extensions'].contains(ext)) {
      res = iconMap['thumbnail'];
      break;
    }
  }
  return res;
}

const List iconMaps = [
  //media
  {
    'extensions': [
      'jpg',
      'png',
      'gif',
      'bmp',
      'jpeg',
      'jpe',
      'jif',
      'jfif',
      'jfi',
      'webp',
      'tiff',
      'tif',
      'ico',
      'svg',
      'webp'
    ],
    'type': 'Image',
    'thumbnail': 'file_image'
  },
  {
    'extensions': [
      'mp4',
      'webm',
      'mpg',
      'mp2',
      'mpeg',
      'mpe',
      'mpv',
      'ocg',
      'm4p',
      'm4v',
      'avi',
      'wmv',
      'mov',
      'qt',
      'flv',
      'swf',
      'mkv',
      'rmvb',
      'rm',
      'vob',
      '3gp'
    ],
    'type': 'Video',
    'thumbnail': 'file_video'
  },
  {
    'extensions': [
      'mp3',
      'ogg',
      'ogm',
      'wav',
      '.aac',
      '.ape',
      '.flac',
      '.flav',
      '.m4a',
      '.wma'
    ],
    'type': 'Audio',
    'thumbnail': 'file_audio'
  },
  //compressed
  {
    'extensions': [
      '7z',
      'brotli',
      'bzip2',
      'gz',
      'gzip',
      'rar',
      'tgz',
      'xz',
      'zip',
      'zstd',
      'deb',
      'msi',
      'snap',
      'iso',
      'dmg',
      'dockerfile',
      'dockerignore'
    ],
    'type': 'Archive',
    'thumbnail': 'file_archive'
  },
  //office relative
  {
    'extensions': ['pdf'],
    'type': 'Portable Document Format',
    'thumbnail': 'file_pdf'
  },
  {
    'extensions': ['txt', 'docb', 'rtf'],
    'type': 'Document',
    'thumbnail': 'doc_text'
  },
  {
    'extensions': ['doc', 'docm', 'dot', 'dotm', 'docx'],
    'type': 'Word Document',
    'thumbnail': 'file_word'
  },

  {
    'extensions': ['xlsx', 'xls', 'xlsb', 'xls', 'ods', 'fods', 'csv'],
    'type': 'Excel Document',
    'thumbnail': 'file_excel'
  },
  {
    'extensions': [
      'pot',
      'potm',
      'potx',
      'ppam',
      'pps',
      'ppsm',
      'ppsx',
      'ppt',
      'pptn',
      'pptx'
    ],
    'type': 'Powerpoint Document',
    'thumbnail': 'file_powerpoint'
  },
  //windows
  {
    'extensions': ['exe'],
    'type': 'Microsoft',
    'thumbnail': 'windows'
  },
  //android
  {
    'extensions': ['apk'],
    'type': 'Android',
    'thumbnail': 'android'
  },
  //other
  {
    'extensions': ['lnk'],
    'type': 'Shortcut',
    'thumbnail': 'link_ext'
  },
  {
    'extensions': [
      'ini',
      'dlc',
      'config',
      'conf',
      'properties',
      'prop',
      'settings',
      'option',
      'props',
      'toml',
      'prefs',
      'sln.dotsettings',
      'sln.dotsettings.user',
      'cfg'
    ],
    'type': 'Settings',
    'thumbnail': 'cog_alt'
  },
  {
    'extensions': ['html', 'htm', 'xhtml', 'html_vm'],
    'type': 'html',
    'thumbnail': 'html5'
  },
  //code relative
  // {
  //   'extensions': ['sh', 'bat'],
  //   'type': 'Batch',
  //   'thumbnail': 'power_shell'
  // },
  // {
  //   'extensions': [
  //     'js',
  //     'jsx',
  //     'json',
  //     'tsbuildinfo',
  //     'json5',
  //     'jsonl',
  //     'ndjson'
  //   ],
  //   'type': 'JavaScript',
  //   'thumbnail': 'js'
  // },
  // {
  //   'extensions': ['ts', 'tsx'],
  //   'type': 'TypeScript',
  //   'thumbnail': 'type_script_language'
  // },
  //
  // {
  //   'extensions': ['asp', 'aspx', 'php', 'jsp'],
  //   'type': 'HyperText Markup Language',
  //   'thumbnail': 'file_a_s_p_x'
  // },
  // {
  //   'extensions': [
  //     'accdb',
  //     'db',
  //     'db3',
  //     'mdb',
  //     'pdb',
  //     'pgsql',
  //     'pkb',
  //     'pks',
  //     'postgres',
  //     'psql',
  //     'sql',
  //     'sqlite',
  //     'sqlite3'
  //   ],
  //   'type': 'Database',
  //   'thumbnail': 'database'
  // },
  //
  // {
  //   'extensions': ['exe'],
  //   'type': 'Executable',
  //   'thumbnail': 'product'
  // },
  // {
  //   'extensions': [
  //     'c',
  //     'h',
  //     'cc',
  //     'cpp',
  //     'cxx',
  //     'c++',
  //     'cp',
  //     'mm',
  //     'mii',
  //     'ii',
  //     'dart',
  //     'go'
  //   ],
  //   'type': 'other Program language',
  //   'thumbnail': 'file_code'
  // },
  // {
  //   'extensions': ['py', 'py3', 'pyc', 'pylintrc', 'python-version'],
  //   'type': 'Python Program',
  //   'thumbnail': 'py'
  // },
  // {
  //   'extensions': ['md', 'markdown', 'rst'],
  //   'type': 'Markdown',
  //   'thumbnail': 'mark_down_language'
  // },
  // {
  //   'extensions': ['yml', 'yaml'],
  //   'type': 'Yet Anoter Markup Language',
  //   'thumbnail': 'file_y_m_l'
  // },
  //
  // {
  //   'type': 'Visual Studio',
  //   'extensions': [
  //     'csproj',
  //     'ruleset',
  //     'sln',
  //     'suo',
  //     'vb',
  //     'vbs',
  //     'vcxitems',
  //     'vcxitems.filters',
  //     'vcxproj',
  //     'vcxproj.filters'
  //   ],
  //   'thumbnail': 'visual_studio_for_windows'
  // },
  // {
  //   'extensions': ['java'],
  //   'type': 'Java Program',
  //   'thumbnail': 'file_j_a_v_a'
  // },
];
