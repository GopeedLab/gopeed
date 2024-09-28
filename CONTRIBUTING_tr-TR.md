# Gopeed Katkıda Bulunma Kılavuzu

Öncelikle, Gopeed’e katkıda bulunmak için ilgilendiğinizden ötürü teşekkür ederiz.
Bu kılavuz, Gopeed’in geliştirilmesinde yer almanız konusunda size yardımcı olacaktır.


## Branch Açıklaması

Bu proje yalnızca bir ana branch içerir: `main`.
Gopeed’in geliştirilmesine katkıda bulunmak istiyorsanız, önce bu projeyi fork etmeniz ve ardından kendi fork projenizde geliştirme yapmanız gerekmektedir.
Geliştirmeyi tamamladıktan sonra, bu projeye bir PR (Pull Request) gönderip geliştirmenizi `main` dalına dahil edebilirsiniz.

## Lokal Geliştirme

Geliştirme ve hata ayıklama için web üzerinden çalışmanız önerilir. 
Öncelikle, back-end hizmetini komut satırında şu komutu çalıştırarak başlatın: `go run cmd/api/main.go`
Hizmetin varsayılan portu `9999`'dur. Ardından front-end flutter projesini `debug` modunda başlatarak çalıştırın.

## Dil Çevirisi

Gopeed’in dil yerelleştirme (i18n) dosyaları `ui/flutter/lib/i18n/langs` dizininde yer almaktadır.

The internationalization files of Gopeed are located in the `ui/flutter/lib/i18n/langs` directory.
Bu dizine yalnızca ilgili dil dosyasını eklemeniz yeterlidir.

Çeviri için `en_us.dart` dosyası kullanılmaktadır. 
`@` ifadesi ile başlayan kelimeler çevrilmemelidir.


## Flutter Geliştirme

Geliştirmeleri commit etmeden önce kodlarınızı standart dart formatında tutmak için `dart format ./ui/flutter` komutunu çalıştırmayı unutmayın.

api/models dosyalarını düzenlemek istiyorsanız build_runner watcher'ı açın.

```
flutter pub run build_runner watch
```