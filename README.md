# whatsapp-ws

whatsapp-ws, whatsmeow kütüphanesini kullanarak bir WebSocket arayüzü ve bir veritabanına günlük mesajları göndermek için endpointler sağlayan ve aynı zamanda bir WhatsApp köprüsü olarak hareket eden bir projedir. Kullanıcıların WhatsApp ile etkileşimde bulunmak ve çeşitli eylemler gerçekleştirmek için WebSocket aracılığıyla komutlar göndermesine olanak tanır.

whatsapp-ws'nin birincil amacı, WhatsApp mesajlaşma yeteneklerinin diğer uygulamalara ve sistemlere entegrasyonunu kolaylaştırmaktır. WebSocket arayüzünü kullanarak, kullanıcılar gerçek zamanlı bir bağlantı kurabilir ve WhatsApp ile programlı olarak etkileşim kurmak için komutlar gönderebilir.

## Build

```bash
go build -ldflags '-extldflags "-static"'
```

## Endpoints

- `/ws` - websocket endpoint
- `/status` - status endpoint