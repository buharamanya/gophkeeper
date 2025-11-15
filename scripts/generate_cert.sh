#!/bin/bash

# Генерация тестовых TLS сертификатов для разработки

mkdir -p tls
cd tls

# Генерация приватного ключа
openssl genrsa -out server.key 2048

# Генерация CSR (Certificate Signing Request)
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"

# Генерация самоподписанного сертификата
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt

# Очистка временных файлов
rm server.csr

echo "TLS certificates generated in tls/ directory"
echo "Certificate: tls/server.crt"
echo "Private key: tls/server.key"