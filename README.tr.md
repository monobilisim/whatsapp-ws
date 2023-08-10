[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![GPL License][license-shield]][license-url]

[![Readme in English](https://img.shields.io/badge/Readme-English-blue)](README.md)
[![Readme in Turkish](https://img.shields.io/badge/Readme-Turkish-red)](README.tr.md)

<div align="center"> 
<a href="https://mono.net.tr/">
  <img src="https://monobilisim.com.tr/images/mono-bilisim.svg" width="340"/>
</a>

<h2 align="center">whatsapp-ws</h2>
<b>whatsapp-ws</b>, WhatsApp mesajlaşma özelliklerini entegre etmek ve çeşitli işlemleri gerçekleştirmek için whatsmeow kütüphanesini kullanarak WebSocket arayüzleri oluşturur.

</div>

---

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Açıklama](#açıklama)
- [API Kullanımı](#api-kullanımı)
  - [/ws Endpoint](#ws-endpoint)
  - [/status Endpoint](#status-endpoint)
  - [/qr Endpoint](#qr-endpoint)
  - [/upload Endpoint](#upload-endpoint)
- [Derleme](#derleme)
- [Uzantılar](#uzantılar)
- [Lisans](#lisans)

---
## Açıklama

whatsapp-ws, whatsmeow kütüphanesini kullanarak bir WebSocket arayüzü ve bir veritabanına günlük mesajları göndermek için endpointler sağlayan ve aynı zamanda bir WhatsApp köprüsü olarak hareket eden bir projedir. Kullanıcıların WhatsApp ile etkileşimde bulunmak ve çeşitli eylemler gerçekleştirmek için WebSocket aracılığıyla komutlar göndermesine olanak tanır.

whatsapp-ws'nin birincil amacı, WhatsApp mesajlaşma yeteneklerinin diğer uygulamalara ve sistemlere entegrasyonunu kolaylaştırmaktır. WebSocket arayüzünü kullanarak, kullanıcılar gerçek zamanlı bir bağlantı kurabilir ve WhatsApp ile programlı olarak etkileşim kurmak için komutlar gönderebilir.

---

## API Kullanımı

### /ws Endpoint

`/ws`, WhatsApp ıle gerçek zamanlı etkileşim sağlamak için bir WebSocket arayüzü sağlar. Kullanıcılar bu uzantıya bağlanabilir ve JSON nesneleri biçiminde komutlar gönderebilir.

```json
{
  "cmd": "string",
  "args": ["string"],
  "user_id": int
}
```

- `cmd`: Çalıştırılacak komut.
- `args`: Komut için gereken argümanlarının lıstesi.
- `user_id`: Kullanıcının Id'sini temsil eden sayı.

### /status Endpoint

`/status`, kullanıcıların oturumunun açık olup olmadığını kontrol etmelerine olanak tanır. Kullanıcı oturum açmış ve kimlik doğrulaması yapmışsa HTTP 200 yanıtı döner.

### /qr Endpoint

`/qr`, WhatsApp'da oturum açmak için gerekli QR kodunu sunar. Kullanıcılar bu uzantıya erişerek WhatsApp'a giriş yapmak için gereken QR kodunu görüntüleyebilir.

### /upload Endpoint

`/upload`, kullanıcıların WhatsApp'a dosya yüklemelerine olanak tanır. Bu, curl gibi araçlarla kullanılabilir. Örnek bir komut:
```sh
curl -X POST -F file=@filepath -F jid=PHONE_NUMBER@s.whatsapp.net -F user_id=1 http://localhost:6023/upload
```

---

## Derleme

whatsapp-ws'yi derlemek için aşağıdaki komutu kullanın:

```bash
go build -ldflags '-extldflags "-static"'
```

---

## Uzantılar

- `/ws` - websocket uzantısı
- `/status` - status uzantısı
- `/qr` - qr uzantısı
- `/upload` - upload uzantısı

---

## Lisans

whatsapp-ws, GPL-3.0 lisansına sahiptir. Detaylar için [LICENSE](LICENSE) dosyasına bakın.

[contributors-shield]: https://img.shields.io/github/contributors/monobilisim/whatsapp-ws.svg?style=for-the-badge
[contributors-url]: https://github.com/monobilisim/whatsapp-ws/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/monobilisim/whatsapp-ws.svg?style=for-the-badge
[forks-url]: https://github.com/monobilisim/whatsapp-ws/network/members
[stars-shield]: https://img.shields.io/github/stars/monobilisim/whatsapp-ws.svg?style=for-the-badge
[stars-url]: https://github.com/monobilisim/whatsapp-ws/stargazers
[issues-shield]: https://img.shields.io/github/issues/monobilisim/whatsapp-ws.svg?style=for-the-badge
[issues-url]: https://github.com/monobilisim/whatsapp-ws/issues
[license-shield]: https://img.shields.io/github/license/monobilisim/whatsapp-ws.svg?style=for-the-badge
[license-url]: https://github.com/monobilisim/whatsapp-ws/blob/master/LICENSE.txt