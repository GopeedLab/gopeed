import 'package:flutter/material.dart';
import 'package:get/get.dart';

Locale toLocale(String key) {
  final arr = key.split('_');
  return Locale(arr[0], arr[1]);
}

String getLocaleKey(Locale locale) {
  return '${locale.languageCode}_${locale.countryCode}';
}

final messages = _Messages();
const fallbackLocale = Locale('en', 'US');

class _Messages extends Translations {
  @override
  Map<String, Map<String, String>> get keys => {
        'zh_CN': {
          'error': '错误',
          'confirm': '确认',
          'cancel': '取消',
          'home.task': '任务',
          'home.setting': '设置',
          'create.title': '创建任务',
          'create.downloadLink': '下载链接',
          'create.downloadLinkValid': '请输入下载链接',
          'create.downloadLinkHit': '请输入下载链接，支持 HTTP/HTTPS/MAGNET@append',
          'create.downloadLinkHitDesktop': '，也可以直接拖拽种子文件到此处',
          'create.download': '下载',
          'create.error.noStoragePermission': '需要开启存储权限',
          'create.selectDir': '选择目录',
          'setting.title': '设置',
          'setting.basic': '基础设置',
          'setting.theme': '主题',
          'setting.themeSystem': '跟随系统',
          'setting.themeLight': '明亮主题',
          'setting.themeDark': '暗黑主题',
          'setting.downloadDir': '下载目录',
          'setting.downloadDirValid': '请选择下载目录',
          'setting.connections': '连接数',
          'setting.locale': '语言',
          'setting.locale.zh_CN': '中文（简体）',
          'setting.locale.en_US': '英文（美国）',
          'setting.locale.ru_RU': '俄语 (俄罗斯) ',
          'task.deleteTask': '删除任务',
          'task.deleteTaskTip': '保留已下载的文件',
          'task.delete': '删除',
        },
        'en_US': {
          'error': 'Error',
          'confirm': 'Confirm',
          'cancel': 'Cancel',
          'home.task': 'Task',
          'home.setting': 'Setting',
          'create.title': 'Create Task',
          'create.downloadLink': 'Download Link',
          'create.downloadLinkValid': 'Please enter the download link',
          'create.downloadLinkHit':
              'Please enter the download link, HTTP/HTTPS/MAGNET supported@append',
          'create.downloadLinkHitDesktop':
              ', or drag the torrent file here directly',
          'create.download': 'Download',
          'create.error.noStoragePermission': 'Storage permission required',
          'create.selectDir': 'Select Directory',
          'setting.title': 'Setting',
          'setting.basic': 'Basic',
          'setting.theme': 'Theme',
          'setting.themeSystem': 'System',
          'setting.themeLight': 'Light',
          'setting.themeDark': 'Dark',
          'setting.downloadDir': 'Download Directory',
          'setting.downloadDirValid': 'Please select the download directory',
          'setting.connections': 'Connections',
          'setting.locale': 'Language',
          'setting.locale.zh_CN': 'Chinese(Simplified)',
          'setting.locale.en_US': 'English(US)',
          'setting.locale.ru_RU': 'Russian(Russia)',
          'task.deleteTask': 'Delete Task',
          'task.deleteTaskTip': 'Keep downloaded files',
          'task.delete': 'Delete',
        },
        'ru_RU': {
          'error': 'Ошибка',
          'confirm': 'Подтвердить',
          'cancel': 'Отмена',
          'home.task': 'Задачи',
          'home.setting': 'Настройки',
          'create.title': 'Создать задачу',
          'create.downloadLink': 'Ссылка для скачивание',
          'create.downloadLinkValid': 'Пожалуйста, введите ссылку для скачивания',
          'create.downloadLinkHit':
              'Пожалуйста, введите ссылку для скачивания, HTTP/HTTPS/MAGNET supported@append',
          'create.downloadLinkHitDesktop':
              ', или перетащите сюда торрент-файл',
          'create.download': 'Скачать',
          'create.error.noStoragePermission': 'Требуется доступ к хранилищу',
          'create.selectDir': 'Выберите папку',
          'setting.title': 'Настройки',
          'setting.basic': 'Основные параметры',
          'setting.theme': 'Тема',
          'setting.themeSystem': 'Системная',
          'setting.themeLight': 'Светлая',
          'setting.themeDark': 'Тёмная',
          'setting.downloadDir': 'Папка загрузки',
          'setting.downloadDirValid': 'Пожалуйста, выберите папку загрузки',
          'setting.connections': 'Количество подключений',
          'setting.locale': 'Язык',
          'setting.locale.zh_CN': 'Китайский(Упрощенный)',
          'setting.locale.en_US': 'Английский(США)',
          'setting.locale.ru_RU': 'Русский(Россия)',
          'task.deleteTask': 'Удалить задачу',
          'task.deleteTaskTip': 'Сохранить загруженные файлы',
          'task.delete': 'Удалить',
        },
      };
}
